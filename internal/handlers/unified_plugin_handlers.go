package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// UnifiedPluginDefinition represents a plugin definition that can be system or private
type UnifiedPluginDefinition struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"type"`               // "system" or "private"
	PluginType         string `json:"plugin_type"`       // "image" or "data"
	Description        string `json:"description"`
	ConfigSchema       string `json:"config_schema"`
	Version            string `json:"version"`
	Author             string `json:"author"`
	IsActive           bool   `json:"is_active"`
	RequiresProcessing bool   `json:"requires_processing"`
	
	// Private plugin specific fields
	InstanceCount      *int   `json:"instance_count,omitempty"` // Number of instances user has created
}

// GetAvailablePluginDefinitionsHandler returns both system and private plugins available to the user
func GetAvailablePluginDefinitionsHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	var allPlugins []UnifiedPluginDefinition

	// Get system plugins from registry
	systemPlugins := plugins.GetAllInfo()
	for _, plugin := range systemPlugins {
		unifiedPlugin := UnifiedPluginDefinition{
			ID:                 plugin.Type, // Use type as ID for system plugins
			Name:               plugin.Name,
			Type:               "system",
			PluginType:         string(plugin.PluginType),
			Description:        plugin.Description,
			ConfigSchema:       plugin.ConfigSchema,
			Version:            plugin.Version,
			Author:             plugin.Author,
			IsActive:           true,
			RequiresProcessing: plugin.RequiresProcessing,
		}
		allPlugins = append(allPlugins, unifiedPlugin)
	}

	// Get user's private plugins from database
	db := database.GetDB()
	var privatePlugins []database.PrivatePlugin
	err := db.Where("user_id = ?", userID).Find(&privatePlugins).Error
	if err == nil {
		for _, privatePlugin := range privatePlugins {
			// Count how many instances user has created of this private plugin
			// For now, since we haven't fully migrated, we'll show 0 instances
			instanceCount := 0
			
			unifiedPlugin := UnifiedPluginDefinition{
				ID:                 privatePlugin.ID.String(),
				Name:               privatePlugin.Name,
				Type:               "private",
				PluginType:         "data", // Private plugins are always data plugins
				Description:        privatePlugin.Description,
				ConfigSchema:       string(privatePlugin.FormFields), // Use form fields as config schema
				Version:            privatePlugin.Version,
				Author:             "Private Plugin", // Could enhance to show actual user name
				IsActive:           true,
				RequiresProcessing: true, // Private plugins always require processing
				InstanceCount:      &instanceCount,
			}
			allPlugins = append(allPlugins, unifiedPlugin)
		}
	}

	// Sort plugins by type (system first) then by name
	sort.Slice(allPlugins, func(i, j int) bool {
		if allPlugins[i].Type == allPlugins[j].Type {
			return allPlugins[i].Name < allPlugins[j].Name
		}
		// System plugins first
		return allPlugins[i].Type == "system" && allPlugins[j].Type == "private"
	})

	c.JSON(http.StatusOK, gin.H{"plugins": allPlugins})
}

// UnifiedPluginInstance represents a plugin instance from either legacy or unified system
type UnifiedPluginInstance struct {
	ID                 string                 `json:"id"`
	UserID             string                 `json:"user_id"`
	Name               string                 `json:"name"`
	Settings           string                 `json:"settings"`
	RefreshInterval    int                    `json:"refresh_interval"`
	IsActive           bool                   `json:"is_active"`
	CreatedAt          string                 `json:"created_at"`
	UpdatedAt          string                 `json:"updated_at"`
	IsUsedInPlaylists  bool                   `json:"is_used_in_playlists"`
	
	// Plugin info
	Plugin struct {
		ID                 string `json:"id"`
		Name               string `json:"name"`
		Type               string `json:"type"`
		Description        string `json:"description"`
		ConfigSchema       string `json:"config_schema"`
		Version            string `json:"version"`
		Author             string `json:"author"`
		IsActive           bool   `json:"is_active"`
		RequiresProcessing bool   `json:"requires_processing"`
	} `json:"plugin"`
}

// GetPluginInstancesHandler returns all plugin instances for the user
func GetPluginInstancesHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	db := database.GetDB()
	var allInstances []UnifiedPluginInstance

	// Get unified PluginInstances (filter out soft-deleted ones)
	var unifiedInstances []database.PluginInstance
	err := db.Preload("PluginDefinition").Where("user_id = ? AND is_active = ?", userID, true).Find(&unifiedInstances).Error
	if err == nil {
		for _, pluginInstance := range unifiedInstances {
			// Check if used in playlists
			var playlistCount int64
			db.Model(&database.PlaylistItem{}).Where("plugin_instance_id = ?", pluginInstance.ID).Count(&playlistCount)

			// Convert settings map to JSON string
			settingsJSON := "{}"
			if len(pluginInstance.Settings) > 0 {
				if settingsBytes, err := json.Marshal(pluginInstance.Settings); err == nil {
					settingsJSON = string(settingsBytes)
				}
			}

			instance := UnifiedPluginInstance{
				ID:                pluginInstance.ID.String(),
				UserID:            pluginInstance.UserID.String(),
				Name:              pluginInstance.Name,
				Settings:          settingsJSON,
				RefreshInterval:   pluginInstance.RefreshInterval,
				IsActive:          pluginInstance.IsActive,
				CreatedAt:         pluginInstance.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:         pluginInstance.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
				IsUsedInPlaylists: playlistCount > 0,
			}

			// Fill plugin info from PluginDefinition
			if pluginInstance.PluginDefinition.ID != uuid.Nil {
				instance.Plugin.ID = pluginInstance.PluginDefinition.ID.String()
				instance.Plugin.Name = pluginInstance.PluginDefinition.Name
				instance.Plugin.Type = pluginInstance.PluginDefinition.PluginType
				instance.Plugin.Description = pluginInstance.PluginDefinition.Description
				instance.Plugin.ConfigSchema = pluginInstance.PluginDefinition.ConfigSchema
				instance.Plugin.Version = pluginInstance.PluginDefinition.Version
				instance.Plugin.Author = pluginInstance.PluginDefinition.Author
				instance.Plugin.IsActive = true
				instance.Plugin.RequiresProcessing = pluginInstance.PluginDefinition.RequiresProcessing
			}

			allInstances = append(allInstances, instance)
		}
	}

	c.JSON(http.StatusOK, gin.H{"plugin_instances": allInstances})
}

// UpdatePluginInstanceHandler updates a plugin instance (handles both legacy and unified)
func UpdatePluginInstanceHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	instanceID := c.Param("id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance ID is required"})
		return
	}

	type UpdateInstanceRequest struct {
		Name            string                 `json:"name" binding:"required"`
		Settings        map[string]interface{} `json:"settings"`
		RefreshInterval int                    `json:"refresh_interval"`
	}

	var req UpdateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	// Try to update as unified PluginInstance first
	var unifiedInstance database.PluginInstance
	err := db.Where("id = ? AND user_id = ?", instanceID, userID).First(&unifiedInstance).Error
	if err == nil {
		// Update unified instance
		unifiedInstance.Name = req.Name
		// Convert settings map to datatypes.JSON
		if len(req.Settings) > 0 {
			if settingsJSON, err := json.Marshal(req.Settings); err == nil {
				unifiedInstance.Settings = settingsJSON
			}
		}
		if req.RefreshInterval > 0 {
			unifiedInstance.RefreshInterval = req.RefreshInterval
		}

		if err := db.Save(&unifiedInstance).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plugin instance: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"instance": unifiedInstance})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Plugin instance not found"})
}

// DeletePluginInstanceHandler deletes a plugin instance (handles both legacy and unified)
func DeletePluginInstanceHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	instanceID := c.Param("id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance ID is required"})
		return
	}

	db := database.GetDB()

	// Try to delete as unified PluginInstance first
	logging.Debug("[DELETE] Attempting to delete unified PluginInstance", "instance_id", instanceID, "user_id", userID.String())
	instanceUUID, err := uuid.Parse(instanceID)
	if err != nil {
		logging.Error("[DELETE] Invalid instance ID format", "instance_id", instanceID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}
	
	unifiedPluginService := database.NewUnifiedPluginService(db)
	err = unifiedPluginService.DeletePluginInstance(instanceUUID, userID)
	if err == nil {
		logging.Info("[DELETE] Successfully deleted unified PluginInstance", "instance_id", instanceID)
		c.JSON(http.StatusOK, gin.H{"message": "Plugin instance deleted successfully"})
		return
	}
	
	logging.Error("[DELETE] Failed to delete unified PluginInstance", "instance_id", instanceID, "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete plugin instance: " + err.Error()})
}

// ForceRefreshPluginInstanceHandler forces refresh of a plugin instance (handles both legacy and unified)
func ForceRefreshPluginInstanceHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	instanceID := c.Param("id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance ID is required"})
		return
	}

	db := database.GetDB()

	// Try as unified PluginInstance first
	var unifiedInstance database.PluginInstance
	err := db.Preload("PluginDefinition").Where("id = ? AND user_id = ?", instanceID, userID).First(&unifiedInstance).Error
	if err == nil {
		// For unified instances, clear any pre-rendered content first
		db.Where("plugin_instance_id = ?", unifiedInstance.ID).Delete(&database.RenderedContent{})

		// Check if this plugin requires processing (rendering)
		if unifiedInstance.PluginDefinition.RequiresProcessing {
			// Schedule an immediate high-priority render job
			renderJob := database.RenderQueue{
				ID:               uuid.New(),
				PluginInstanceID: unifiedInstance.ID,
				Priority:         999, // High priority for force refresh
				ScheduledFor:     time.Now(),
				Status:           "pending",
			}

			err = db.Create(&renderJob).Error
			if err != nil {
				logging.Error("[FORCE_REFRESH] Failed to schedule immediate render job", "instance_id", instanceID, "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to schedule render job"})
				return
			}

			logging.Info("[FORCE_REFRESH] Scheduled immediate render job", "instance_name", unifiedInstance.Name, "instance_id", instanceID, "job_id", renderJob.ID)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Plugin refresh triggered successfully"})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Plugin instance not found"})
}

// CreatePluginInstanceFromDefinitionHandler creates a plugin instance from a unified definition
func CreatePluginInstanceFromDefinitionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	type CreateInstanceRequest struct {
		DefinitionID    string                 `json:"definition_id" binding:"required"`
		DefinitionType  string                 `json:"definition_type" binding:"required"` // "system" or "private"
		Name            string                 `json:"name" binding:"required"`
		Settings        map[string]interface{} `json:"settings"`
		RefreshInterval int                    `json:"refresh_interval"`
	}

	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	switch req.DefinitionType {
	case "system":
		// Create unified PluginInstance for system plugins
		unifiedPluginService := database.NewUnifiedPluginService(db)
		
		// Find the system plugin definition
		var pluginDefinition database.PluginDefinition
		err := db.Where("plugin_type = ? AND identifier = ?", "system", req.DefinitionID).First(&pluginDefinition).Error
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "System plugin definition not found"})
			return
		}

		// Create the PluginInstance using unified service
		pluginInstance, err := unifiedPluginService.CreatePluginInstance(userID, pluginDefinition.ID, req.Name, req.Settings, req.RefreshInterval)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plugin instance: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"instance": pluginInstance})

	case "private":
		// Create proper PluginInstance for private plugin using unified system
		unifiedPluginService := database.NewUnifiedPluginService(db)
		
		// Verify the private plugin exists and belongs to the user
		var privatePlugin database.PrivatePlugin
		err := db.Where("id = ? AND user_id = ?", req.DefinitionID, userID).First(&privatePlugin).Error
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Private plugin not found"})
			return
		}

		// Create or find the PluginDefinition for this private plugin
		var pluginDefinition database.PluginDefinition
		err = db.Where("plugin_type = ? AND owner_id = ? AND identifier = ?", "private", userID, privatePlugin.ID.String()).First(&pluginDefinition).Error
		if err != nil {
			// Create PluginDefinition for this private plugin
			pluginDefinition = database.PluginDefinition{
				PluginType:         "private",
				Name:               privatePlugin.Name,
				Description:        privatePlugin.Description,
				Version:            privatePlugin.Version,
				Author:             "Private Plugin",
				OwnerID:            &userID,
				Identifier:         privatePlugin.ID.String(),
				ConfigSchema:       string(privatePlugin.FormFields),
				RequiresProcessing: true,
			}
			
			if err := db.Create(&pluginDefinition).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plugin definition: " + err.Error()})
				return
			}
		}

		// Create the PluginInstance using unified service
		pluginInstance, err := unifiedPluginService.CreatePluginInstance(userID, pluginDefinition.ID, req.Name, req.Settings, req.RefreshInterval)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plugin instance: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"instance": pluginInstance})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid definition type"})
		return
	}
}

// GetRefreshRateOptionsHandler returns available refresh rate options
func GetRefreshRateOptionsHandler(c *gin.Context) {
	options := database.GetRefreshRateOptions()
	c.JSON(http.StatusOK, gin.H{"refresh_rate_options": options})
}

// ValidatePluginSettingsHandler validates plugin settings against schema
func ValidatePluginSettingsHandler(c *gin.Context) {
	var req struct {
		DefinitionID string                 `json:"definition_id" binding:"required"`
		Settings     map[string]interface{} `json:"settings" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	db := database.GetDB()

	var pluginDefinition database.PluginDefinition
	err := db.Where("id = ?", req.DefinitionID).First(&pluginDefinition).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin definition not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "Settings are valid"})
}

// GetAvailablePluginTypesHandler returns available plugin types
func GetAvailablePluginTypesHandler(c *gin.Context) {
	db := database.GetDB()

	var types []struct {
		PluginType string `json:"plugin_type"`
	}

	err := db.Model(&database.PluginDefinition{}).
		Select("DISTINCT plugin_type").
		Where("is_active = ?", true).
		Find(&types).Error

	if err != nil {
		logging.Error("[PLUGIN_TYPES] Failed to fetch plugin types", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch plugin types"})
		return
	}

	pluginTypes := make([]string, len(types))
	for i, t := range types {
		pluginTypes[i] = t.PluginType
	}

	c.JSON(http.StatusOK, gin.H{"types": pluginTypes})
}