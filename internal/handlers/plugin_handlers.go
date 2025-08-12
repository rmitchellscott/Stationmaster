package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
)

// GetPluginsHandler returns all available system plugins
func GetPluginsHandler(c *gin.Context) {
	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	plugins, err := pluginService.GetAllPlugins()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch plugins"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plugins": plugins})
}

// GetUserPluginsHandler returns all plugin instances for the current user
func GetUserPluginsHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID

	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	userPlugins, err := pluginService.GetUserPluginsByUserID(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user plugins"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_plugins": userPlugins})
}

// CreateUserPluginHandler creates a new plugin instance for the user
func CreateUserPluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID

	var req struct {
		PluginID uuid.UUID              `json:"plugin_id" binding:"required"`
		Name     string                 `json:"name" binding:"required"`
		Settings map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	// Verify plugin exists
	_, err := pluginService.GetPluginByID(req.PluginID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	// TODO: Validate settings against plugin schema

	userPlugin, err := pluginService.CreateUserPlugin(userUUID, req.PluginID, req.Name, req.Settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user plugin"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user_plugin": userPlugin})
}

// GetUserPluginHandler returns a specific user plugin instance
func GetUserPluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	userPluginIDStr := c.Param("id")

	userPluginID, err := uuid.Parse(userPluginIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user plugin ID"})
		return
	}

	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	userPlugin, err := pluginService.GetUserPluginByID(userPluginID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User plugin not found"})
		return
	}

	// Verify ownership
	if userPlugin.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_plugin": userPlugin})
}

// UpdateUserPluginHandler updates a user plugin instance
func UpdateUserPluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	userPluginIDStr := c.Param("id")

	userPluginID, err := uuid.Parse(userPluginIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user plugin ID"})
		return
	}

	var req struct {
		Name     string                 `json:"name"`
		Settings map[string]interface{} `json:"settings"`
		IsActive *bool                  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	userPlugin, err := pluginService.GetUserPluginByID(userPluginID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User plugin not found"})
		return
	}

	// Verify ownership
	if userPlugin.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update fields
	if req.Name != "" {
		userPlugin.Name = req.Name
	}
	if req.Settings != nil {
		settingsJSON, err := json.Marshal(req.Settings)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settings format"})
			return
		}
		userPlugin.Settings = string(settingsJSON)
	}
	if req.IsActive != nil {
		userPlugin.IsActive = *req.IsActive
	}

	err = pluginService.UpdateUserPlugin(userPlugin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user plugin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_plugin": userPlugin})
}

// DeleteUserPluginHandler deletes a user plugin instance
func DeleteUserPluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	userPluginIDStr := c.Param("id")

	userPluginID, err := uuid.Parse(userPluginIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user plugin ID"})
		return
	}

	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	userPlugin, err := pluginService.GetUserPluginByID(userPluginID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User plugin not found"})
		return
	}

	// Verify ownership
	if userPlugin.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	err = pluginService.DeleteUserPlugin(userPluginID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user plugin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User plugin deleted successfully"})
}

// Admin plugin handlers

// CreatePluginHandler creates a new system plugin (admin only)
func CreatePluginHandler(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Type         string `json:"type" binding:"required"`
		Description  string `json:"description"`
		ConfigSchema string `json:"config_schema"`
		Version      string `json:"version"`
		Author       string `json:"author"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	plugin, err := pluginService.CreatePlugin(
		req.Name, 
		req.Type, 
		req.Description, 
		req.ConfigSchema, 
		req.Version, 
		req.Author,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plugin"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"plugin": plugin})
}

// UpdatePluginHandler updates a system plugin (admin only)
func UpdatePluginHandler(c *gin.Context) {
	pluginIDStr := c.Param("id")

	pluginID, err := uuid.Parse(pluginIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin ID"})
		return
	}

	var req struct {
		Name         string `json:"name"`
		Type         string `json:"type"`
		Description  string `json:"description"`
		ConfigSchema string `json:"config_schema"`
		Version      string `json:"version"`
		Author       string `json:"author"`
		IsActive     *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	plugin, err := pluginService.GetPluginByID(pluginID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	// Update fields
	if req.Name != "" {
		plugin.Name = req.Name
	}
	if req.Type != "" {
		plugin.Type = req.Type
	}
	if req.Description != "" {
		plugin.Description = req.Description
	}
	if req.ConfigSchema != "" {
		plugin.ConfigSchema = req.ConfigSchema
	}
	if req.Version != "" {
		plugin.Version = req.Version
	}
	if req.Author != "" {
		plugin.Author = req.Author
	}
	if req.IsActive != nil {
		plugin.IsActive = *req.IsActive
	}

	err = pluginService.UpdatePlugin(plugin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plugin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plugin": plugin})
}

// DeletePluginHandler deletes a system plugin (admin only)
func DeletePluginHandler(c *gin.Context) {
	pluginIDStr := c.Param("id")

	pluginID, err := uuid.Parse(pluginIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin ID"})
		return
	}

	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	err = pluginService.DeletePlugin(pluginID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete plugin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plugin deleted successfully"})
}

// GetPluginStatsHandler returns plugin statistics (admin only)
func GetPluginStatsHandler(c *gin.Context) {
	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	stats, err := pluginService.GetPluginStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get plugin statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}