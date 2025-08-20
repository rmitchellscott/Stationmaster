package handlers

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// PluginInfo represents plugin information for the API
type PluginInfo struct {
	ID                *uuid.UUID `json:"id,omitempty"`    // Database ID if exists
	Name              string     `json:"name"`
	Type              string     `json:"type"`
	Description       string     `json:"description"`
	ConfigSchema      string     `json:"config_schema"`
	Version           string     `json:"version"`
	Author            string     `json:"author"`
	IsActive          bool       `json:"is_active"`
	RequiresProcessing bool       `json:"requires_processing"`
}

// GetPluginsHandler returns all available system plugins from the registry
func GetPluginsHandler(c *gin.Context) {
	// Get all plugins from the registry
	pluginInfos := plugins.GetAllInfo()
	
	// Convert to API format
	var apiPlugins []PluginInfo
	for _, info := range pluginInfos {
		apiPlugin := PluginInfo{
			Name:              info.Name,
			Type:              info.Type,
			Description:       info.Description,
			ConfigSchema:      info.ConfigSchema,
			Version:           "1.0.0", // Default version
			Author:            "Stationmaster",
			IsActive:          true,
			RequiresProcessing: info.RequiresProcessing,
		}
		apiPlugins = append(apiPlugins, apiPlugin)
	}
	
	// Sort by name for consistent ordering
	sort.Slice(apiPlugins, func(i, j int) bool {
		return apiPlugins[i].Name < apiPlugins[j].Name
	})

	c.JSON(http.StatusOK, gin.H{"plugins": apiPlugins})
}

// UserPluginResponse represents a user plugin with computed status
type UserPluginResponse struct {
	database.UserPlugin
	IsUsedInPlaylists bool `json:"is_used_in_playlists"`
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

	// Create response with playlist usage information and populate RequiresProcessing from registry
	var response []UserPluginResponse
	for _, userPlugin := range userPlugins {
		isUsed := len(userPlugin.PlaylistItems) > 0
		
		// Populate RequiresProcessing field from registry if not already set
		if registryPlugin, exists := plugins.Get(userPlugin.Plugin.Type); exists {
			userPlugin.Plugin.RequiresProcessing = registryPlugin.RequiresProcessing()
		}
		
		response = append(response, UserPluginResponse{
			UserPlugin:        userPlugin,
			IsUsedInPlaylists: isUsed,
		})
	}

	c.JSON(http.StatusOK, gin.H{"user_plugins": response})
}

// CreateUserPluginHandler creates a new plugin instance for the user
func CreateUserPluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID

	var req struct {
		PluginType      string                 `json:"plugin_type" binding:"required"` // Changed from PluginID to PluginType
		Name            string                 `json:"name" binding:"required"`
		Settings        map[string]interface{} `json:"settings"`
		RefreshInterval *int                   `json:"refresh_interval,omitempty"` // Optional refresh interval
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify plugin exists in registry
	plugin, exists := plugins.Get(req.PluginType)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin type not found"})
		return
	}

	// Validate settings against plugin schema
	if req.Settings != nil {
		if err := plugin.Validate(req.Settings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Settings validation failed",
				"details": err.Error(),
			})
			return
		}
	}

	// Handle refresh interval - only for plugins that require processing
	refreshInterval := database.GetDefaultRefreshRate()
	if plugin.RequiresProcessing() {
		// Plugin requires processing, validate refresh interval if provided
		if req.RefreshInterval != nil {
			if !database.IsValidRefreshRate(*req.RefreshInterval) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid refresh interval",
					"details": "Refresh interval must be one of the predefined values",
				})
				return
			}
			refreshInterval = *req.RefreshInterval
		}
	} else {
		// Plugin doesn't require processing, ignore any provided refresh interval
		if req.RefreshInterval != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Refresh interval not applicable",
				"details": "This plugin type doesn't support refresh intervals",
			})
			return
		}
		// Set to 0 for non-processing plugins (won't be used anyway)
		refreshInterval = 0
	}

	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	// Find or create the plugin in database
	dbPlugin, err := pluginService.GetPluginByType(req.PluginType)
	if err != nil {
		// Plugin doesn't exist in database, create it
		dbPlugin, err = pluginService.CreatePluginWithProcessing(
			plugin.Name(),
			plugin.Type(),
			plugin.Description(),
			plugin.ConfigSchema(),
			"1.0.0",
			"Stationmaster",
			plugin.RequiresProcessing(),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plugin record"})
			return
		}
	}

	userPlugin, err := pluginService.CreateUserPlugin(userUUID, dbPlugin.ID, req.Name, req.Settings, refreshInterval)
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
		Name            string                 `json:"name"`
		Settings        map[string]interface{} `json:"settings"`
		IsActive        *bool                  `json:"is_active"`
		RefreshInterval *int                   `json:"refresh_interval,omitempty"` // Optional refresh interval
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

	// Handle refresh interval updates - only for plugins that require processing
	if req.RefreshInterval != nil {
		// Get plugin from registry to check if it requires processing
		plugin, exists := plugins.Get(userPlugin.Plugin.Type)
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Plugin type not found in registry"})
			return
		}

		if !plugin.RequiresProcessing() {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Refresh interval not applicable",
				"details": "This plugin type doesn't support refresh intervals",
			})
			return
		}

		// Validate refresh interval
		if !database.IsValidRefreshRate(*req.RefreshInterval) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid refresh interval",
				"details": "Refresh interval must be one of the predefined values",
			})
			return
		}
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
	if req.RefreshInterval != nil {
		userPlugin.RefreshInterval = *req.RefreshInterval
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

// New Plugin Registry API Handlers

// GetPluginInfoHandler returns detailed plugin information from the registry
func GetPluginInfoHandler(c *gin.Context) {
	pluginInfos := plugins.GetAllInfo()
	
	// Convert to API format with additional details
	var detailedInfos []gin.H
	for _, info := range pluginInfos {
		detailedInfo := gin.H{
			"type":          info.Type,
			"name":          info.Name,
			"description":   info.Description,
			"config_schema": info.ConfigSchema,
			"version":       "1.0.0",
			"author":        "Stationmaster",
			"is_active":     true,
			"plugin_type":   info.PluginType, // "image" or "data"
		}
		detailedInfos = append(detailedInfos, detailedInfo)
	}
	
	// Sort by name
	sort.Slice(detailedInfos, func(i, j int) bool {
		return detailedInfos[i]["name"].(string) < detailedInfos[j]["name"].(string)
	})
	
	c.JSON(http.StatusOK, gin.H{"plugins": detailedInfos})
}

// ForceRefreshUserPluginHandler clears rendered content for a user plugin to force re-render
func ForceRefreshUserPluginHandler(c *gin.Context) {
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

	// Check if plugin requires processing
	plugin, exists := plugins.Get(userPlugin.Plugin.Type)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Plugin type not found in registry"})
		return
	}

	if !plugin.RequiresProcessing() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plugin does not support force refresh"})
		return
	}

	// Clear rendered content for this user plugin
	err = pluginService.ClearRenderedContentForUserPlugin(userPluginID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear rendered content"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rendered content cleared successfully. Plugin will re-render on next request."})
}

// GetAvailablePluginTypesHandler returns available plugin types from the registry
func GetAvailablePluginTypesHandler(c *gin.Context) {
	pluginTypes := plugins.GetAllTypes()
	
	c.JSON(http.StatusOK, gin.H{"types": pluginTypes})
}

// GetPluginByTypeHandler returns information about a specific plugin type
func GetPluginByTypeHandler(c *gin.Context) {
	pluginType := c.Param("type")
	
	plugin, exists := plugins.Get(pluginType)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin type not found"})
		return
	}
	
	pluginInfo := gin.H{
		"type":          plugin.Type(),
		"name":          plugin.Name(),
		"description":   plugin.Description(),
		"config_schema": plugin.ConfigSchema(),
		"version":       "1.0.0",
		"author":        "Stationmaster",
		"is_active":     true,
		"plugin_type":   string(plugin.PluginType()),
	}
	
	c.JSON(http.StatusOK, gin.H{"plugin": pluginInfo})
}

// ValidatePluginSettingsHandler validates plugin settings against the plugin's validation rules
func ValidatePluginSettingsHandler(c *gin.Context) {
	var req struct {
		Type     string                 `json:"type" binding:"required"`
		Settings map[string]interface{} `json:"settings" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	plugin, exists := plugins.Get(req.Type)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin type not found"})
		return
	}
	
	if err := plugin.Validate(req.Settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Validation failed",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// GetRefreshRateOptionsHandler returns available refresh rate options
func GetRefreshRateOptionsHandler(c *gin.Context) {
	options := database.GetRefreshRateOptions()
	c.JSON(http.StatusOK, gin.H{"refresh_rate_options": options})
}

// GetPluginRegistryStatsHandler returns statistics about the plugin registry
func GetPluginRegistryStatsHandler(c *gin.Context) {
	pluginTypes := plugins.GetAllTypes()
	pluginInfos := plugins.GetAllInfo()
	
	// Count by plugin type (image vs data)
	imagePlugins := 0
	dataPlugins := 0
	
	for _, info := range pluginInfos {
		if info.PluginType == "image" {
			imagePlugins++
		} else if info.PluginType == "data" {
			dataPlugins++
		}
	}
	
	stats := gin.H{
		"total_plugins":  plugins.Count(),
		"plugin_types":   len(pluginTypes),
		"image_plugins":  imagePlugins,
		"data_plugins":   dataPlugins,
		"available_types": pluginTypes,
	}
	
	c.JSON(http.StatusOK, stats)
}
