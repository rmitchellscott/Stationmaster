package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/plugins/private"
	"github.com/rmitchellscott/stationmaster/internal/utils"
	"gopkg.in/yaml.v3"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
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
	
	// Mashup plugin specific fields
	MashupLayout       string `json:"mashup_layout,omitempty"`  // "1L1R", "2T1B", "1T2B", "2x2"
	CreatedAt          string `json:"created_at,omitempty"`     // ISO date string
	UpdatedAt          string `json:"updated_at,omitempty"`     // ISO date string
}

// GetAvailablePluginDefinitionsHandler returns both system and private plugins available to the user
func GetAvailablePluginDefinitionsHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	var allPlugins []UnifiedPluginDefinition

	// Filter by plugin_type query parameter if provided
	pluginType := c.Query("plugin_type")

	// Get system plugins from registry (only if not filtering for specific non-system types)
	if pluginType == "" || pluginType == "system" {
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
	}

	// Get user's private plugins from unified plugin_definitions table
	db := database.GetDB()
	
	var privatePlugins []database.PluginDefinition
	query := db.Where("owner_id = ?", userID)
	if pluginType != "" {
		query = query.Where("plugin_type = ?", pluginType)
	} else {
		// Default: get private and mashup plugins (don't mix with system)
		query = query.Where("plugin_type IN (?)", []string{"private", "mashup"})
	}
	
	err := query.Find(&privatePlugins).Error
	if err == nil {
		for _, privatePlugin := range privatePlugins {
			// Count how many instances user has created of this private plugin
			var instanceCount int64
			db.Model(&database.PluginInstance{}).Where("plugin_definition_id = ? AND user_id = ?", privatePlugin.ID, userID).Count(&instanceCount)
			instances := int(instanceCount)
			
			unifiedPlugin := UnifiedPluginDefinition{
				ID:                 privatePlugin.ID.String(),
				Name:               privatePlugin.Name,
				Type:               "private",
				PluginType:         privatePlugin.PluginType,
				Description:        privatePlugin.Description,
				ConfigSchema:       privatePlugin.ConfigSchema,
				Version:            privatePlugin.Version,
				Author:             privatePlugin.Author,
				IsActive:           privatePlugin.IsActive,
				RequiresProcessing: privatePlugin.RequiresProcessing,
				InstanceCount:      &instances,
				CreatedAt:          privatePlugin.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:          privatePlugin.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
			
			// Add mashup-specific fields if this is a mashup plugin
			if privatePlugin.PluginType == "mashup" {
				if privatePlugin.MashupLayout != nil {
					unifiedPlugin.MashupLayout = *privatePlugin.MashupLayout
				}
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

	// If requesting only private plugins, transform to PrivatePluginList format
	if pluginType == "private" {
		var privatePluginList []map[string]interface{}
		for _, plugin := range allPlugins {
			if plugin.Type == "private" {
				// Get the actual plugin definition for additional fields
				var pluginDef database.PluginDefinition
				err := db.Where("id = ?", plugin.ID).First(&pluginDef).Error
				if err == nil {
					privatePlugin := map[string]interface{}{
						"id":                plugin.ID,
						"name":              plugin.Name,
						"description":       plugin.Description,
						"version":           plugin.Version,
						"data_strategy":     func() string {
						if pluginDef.DataStrategy != nil {
							return *pluginDef.DataStrategy
						}
						return "webhook"
					}(),
						"is_published":      pluginDef.IsPublished,
						"created_at":        pluginDef.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
						"updated_at":        pluginDef.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
						"markup_full":       "",
						"markup_half_vert":  "",
						"markup_half_horiz": "",
						"markup_quadrant":   "",
						"shared_markup":     "",
						"polling_config":    nil,
						"form_fields":       nil,
						"sample_data":       nil,
					}
					
					// Add markup fields if they exist
					if pluginDef.MarkupFull != nil {
						privatePlugin["markup_full"] = *pluginDef.MarkupFull
					}
					if pluginDef.MarkupHalfVert != nil {
						privatePlugin["markup_half_vert"] = *pluginDef.MarkupHalfVert
					}
					if pluginDef.MarkupHalfHoriz != nil {
						privatePlugin["markup_half_horiz"] = *pluginDef.MarkupHalfHoriz
					}
					if pluginDef.MarkupQuadrant != nil {
						privatePlugin["markup_quadrant"] = *pluginDef.MarkupQuadrant
					}
					privatePlugin["shared_markup"] = pluginDef.SharedMarkup
					
					// Add sample data if it exists
					if len(pluginDef.SampleData) > 0 {
						var sampleDataMap map[string]interface{}
						if err := json.Unmarshal(pluginDef.SampleData, &sampleDataMap); err == nil {
							privatePlugin["sample_data"] = sampleDataMap
						}
					}
					
					privatePluginList = append(privatePluginList, privatePlugin)
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"plugins": privatePluginList})
		return
	}

	// If requesting only mashup plugins, return just the mashup plugins
	if pluginType == "mashup" {
		var mashupPluginList []UnifiedPluginDefinition
		for _, plugin := range allPlugins {
			if plugin.PluginType == "mashup" {
				mashupPluginList = append(mashupPluginList, plugin)
			}
		}
		c.JSON(http.StatusOK, gin.H{"plugins": mashupPluginList})
		return
	}

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
	
	// Config update status
	NeedsConfigUpdate  bool                   `json:"needs_config_update"`
	LastSchemaVersion  int                    `json:"last_schema_version"`
	
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
		DataStrategy       string `json:"data_strategy"`
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
				NeedsConfigUpdate: pluginInstance.NeedsConfigUpdate,
				LastSchemaVersion: pluginInstance.LastSchemaVersion,
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
				
				// Set data strategy (no fallback - only set if explicitly defined)
				if pluginInstance.PluginDefinition.DataStrategy != nil {
					instance.Plugin.DataStrategy = *pluginInstance.PluginDefinition.DataStrategy
				} else {
					instance.Plugin.DataStrategy = ""
				}
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
	err := db.Preload("PluginDefinition").Where("id = ? AND user_id = ?", instanceID, userID).First(&unifiedInstance).Error
	if err == nil {
		// Update unified instance
		logging.Info("[PLUGIN_UPDATE] Updating plugin instance", "instance_id", instanceID, "name", req.Name)
		unifiedInstance.Name = req.Name
		
		// Convert settings map to datatypes.JSON
		logging.Info("[PLUGIN_UPDATE] Processing settings update", "settings_count", len(req.Settings), "settings", req.Settings)
		
		if len(req.Settings) > 0 {
			settingsJSON, err := json.Marshal(req.Settings)
			if err != nil {
				logging.Error("[PLUGIN_UPDATE] Failed to marshal settings", "instance_id", instanceID, "error", err, "settings", req.Settings)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to process settings: " + err.Error()})
				return
			}
			unifiedInstance.Settings = settingsJSON
			logging.Info("[PLUGIN_UPDATE] Settings marshaled successfully", "instance_id", instanceID, "settings_json", string(settingsJSON))
		} else if req.Settings != nil {
			// Handle the case where settings is an empty map - this should still be saved
			settingsJSON, err := json.Marshal(req.Settings)
			if err != nil {
				logging.Error("[PLUGIN_UPDATE] Failed to marshal empty settings", "instance_id", instanceID, "error", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to process settings: " + err.Error()})
				return
			}
			unifiedInstance.Settings = settingsJSON
			logging.Info("[PLUGIN_UPDATE] Empty settings saved", "instance_id", instanceID)
		}
		if req.RefreshInterval > 0 {
			unifiedInstance.RefreshInterval = req.RefreshInterval
		}

		// Clear config update flag and sync schema version when instance is updated
		if unifiedInstance.NeedsConfigUpdate {
			unifiedInstance.NeedsConfigUpdate = false
			unifiedInstance.LastSchemaVersion = unifiedInstance.PluginDefinition.SchemaVersion
			logging.Info("[PLUGIN_UPDATE] Clearing config update flag, syncing to schema version", "instance_id", instanceID, "schema_version", unifiedInstance.PluginDefinition.SchemaVersion)
		}

		if err := db.Save(&unifiedInstance).Error; err != nil {
			logging.Error("[PLUGIN_UPDATE] Failed to save plugin instance to database", "instance_id", instanceID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plugin instance: " + err.Error()})
			return
		}
		
		logging.Info("[PLUGIN_UPDATE] Plugin instance updated successfully", "instance_id", instanceID, "name", unifiedInstance.Name)

		// Schedule immediate independent render for updated plugin instance if it requires processing
		if unifiedInstance.PluginDefinition.RequiresProcessing {
			renderJob := database.RenderQueue{
				ID:               uuid.New(),
				PluginInstanceID: unifiedInstance.ID,
				Priority:         999, // High priority for immediate processing
				ScheduledFor:     time.Now(),
				Status:           "pending",
				IndependentRender: true, // Plugin updates are independent renders
			}
			
			err = db.Create(&renderJob).Error
			if err != nil {
				logging.Error("[PLUGIN_UPDATE] Failed to schedule immediate render job", "instance_id", unifiedInstance.ID, "error", err)
				// Don't fail the update if render scheduling fails
			} else {
				logging.Info("[PLUGIN_UPDATE] Scheduled immediate render for updated plugin", "instance_id", unifiedInstance.ID, "name", unifiedInstance.Name)
			}
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
			// Determine if this should be an independent render based on refresh rate type
			dailyRates := []int{
				database.RefreshRateDaily,   // Daily
				database.RefreshRate2xDay,   // Twice daily 
				database.RefreshRate3xDay,   // 3 times daily
				database.RefreshRate4xDay,   // 4 times daily
			}
			
			isDaily := false
			for _, dailyRate := range dailyRates {
				if unifiedInstance.RefreshInterval == dailyRate {
					isDaily = true
					break
				}
			}
			
			// Schedule an immediate render job
			renderJob := database.RenderQueue{
				ID:               uuid.New(),
				PluginInstanceID: unifiedInstance.ID,
				Priority:         999, // High priority for force refresh
				ScheduledFor:     time.Now(),
				Status:           "pending",
				IndependentRender: isDaily, // Daily rates: independent, interval rates: reschedule from now
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
	unifiedPluginService := database.NewUnifiedPluginService(db)
	
	// Find the plugin definition by ID
	var pluginDefinition database.PluginDefinition
	err := db.Where("id = ?", req.DefinitionID).First(&pluginDefinition).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin definition not found"})
		return
	}

	// Verify access permissions
	if pluginDefinition.PluginType == "private" {
		// Private plugins: user must be the owner
		if pluginDefinition.OwnerID == nil || *pluginDefinition.OwnerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to private plugin"})
			return
		}
	}
	// System plugins: accessible to all users (no additional check needed)

	// Create the PluginInstance using unified service
	pluginInstance, err := unifiedPluginService.CreatePluginInstance(userID, pluginDefinition.ID, req.Name, req.Settings, req.RefreshInterval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plugin instance: " + err.Error()})
		return
	}

	// Schedule immediate render for new plugin instance if it requires processing
	if pluginDefinition.RequiresProcessing {
		ScheduleRenderForInstances([]uuid.UUID{pluginInstance.ID})
	}

	c.JSON(http.StatusCreated, gin.H{"instance": pluginInstance})
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

// GetPluginDefinitionHandler returns a single plugin definition by ID
func GetPluginDefinitionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	definitionID := c.Param("id")
	if definitionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Definition ID is required"})
		return
	}

	db := database.GetDB()
	var pluginDefinition database.PluginDefinition
	
	// Only allow users to access their own private plugins or system plugins
	err := db.Where("id = ? AND (owner_id = ? OR plugin_type = 'system')", definitionID, userID).First(&pluginDefinition).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin definition not found"})
		return
	}

	logging.Debug("[GetPluginDefinition] Returning plugin definition", "plugin_id", definitionID, "sample_data_length", len(pluginDefinition.SampleData))
	c.JSON(http.StatusOK, gin.H{"plugin_definition": pluginDefinition})
}

// CreatePluginDefinitionHandler creates a new private plugin definition
func CreatePluginDefinitionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	type CreatePluginRequest struct {
		PluginType        string      `json:"plugin_type" binding:"required"`
		Name              string      `json:"name" binding:"required"`
		Description       string      `json:"description"`
		Version           string      `json:"version"`
		Author            string      `json:"author"`
		MarkupFull        string      `json:"markup_full"`
		MarkupHalfVert    string      `json:"markup_half_vert"`
		MarkupHalfHoriz   string      `json:"markup_half_horiz"`
		MarkupQuadrant    string      `json:"markup_quadrant"`
		SharedMarkup      string      `json:"shared_markup"`
		DataStrategy      string      `json:"data_strategy"`
		PollingConfig     interface{} `json:"polling_config"`
		FormFields        interface{} `json:"form_fields"`
		SampleData        interface{} `json:"sample_data"`
		RemoveBleedMargin bool        `json:"remove_bleed_margin"`
		EnableDarkMode    bool        `json:"enable_dark_mode"`
	}

	var req CreatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Only allow private plugin creation
	if req.PluginType != "private" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only private plugins can be created via this endpoint"})
		return
	}

	// Validate polling configuration
	if err := ValidatePollingConfig(req.PollingConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Polling config validation failed", "details": err.Error()})
		return
	}

	// Validate and convert form fields to JSON schema
	configSchema, err := ValidateFormFields(req.FormFields)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form fields validation failed", "details": err.Error()})
		return
	}

	// Convert configs to JSON for storage
	var pollingConfigJSON, formFieldsJSON, sampleDataJSON []byte

	if req.PollingConfig != nil {
		pollingConfigJSON, err = json.Marshal(req.PollingConfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid polling config"})
			return
		}
	}

	if req.FormFields != nil {
		// Normalize form fields before storage to ensure consistency
		normalizedFormFields := NormalizeFormFields(req.FormFields)
		if normalizedFormFields != nil {
			formFieldsJSON, err = json.Marshal(normalizedFormFields)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form fields config"})
				return
			}
		}
		// If normalized to nil, leave formFieldsJSON as nil (empty byte slice)
	}

	if req.SampleData != nil {
		sampleDataJSON, err = json.Marshal(req.SampleData)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sample data"})
			return
		}
	}

	db := database.GetDB()

	pluginDefinition := database.PluginDefinition{
		PluginType:         req.PluginType,
		OwnerID:            &userID,
		Identifier:         uuid.New().String(), // Generate unique identifier
		Name:               req.Name,
		Description:        req.Description,
		Version:            req.Version,
		Author:             req.Author,
		ConfigSchema:       configSchema, // Use converted JSON schema
		RequiresProcessing: true, // Private plugins always require processing
		MarkupFull:         &req.MarkupFull,
		MarkupHalfVert:     &req.MarkupHalfVert,
		MarkupHalfHoriz:    &req.MarkupHalfHoriz,
		MarkupQuadrant:     &req.MarkupQuadrant,
		SharedMarkup:       &req.SharedMarkup,
		DataStrategy:       &req.DataStrategy,
		PollingConfig:      pollingConfigJSON,
		FormFields:         formFieldsJSON,
		SampleData:         sampleDataJSON,
		RemoveBleedMargin:  &req.RemoveBleedMargin,
		EnableDarkMode:     &req.EnableDarkMode,
		IsPublished:        false,
		IsActive:           true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := db.Create(&pluginDefinition).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plugin definition: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"plugin_definition": pluginDefinition})
}

// UpdatePluginDefinitionHandler updates an existing plugin definition
func UpdatePluginDefinitionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	definitionID := c.Param("id")
	if definitionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Definition ID is required"})
		return
	}

	type UpdatePluginRequest struct {
		Name              string      `json:"name"`
		Description       string      `json:"description"`
		Version           string      `json:"version"`
		Author            string      `json:"author"`
		MarkupFull        string      `json:"markup_full"`
		MarkupHalfVert    string      `json:"markup_half_vert"`
		MarkupHalfHoriz   string      `json:"markup_half_horiz"`
		MarkupQuadrant    string      `json:"markup_quadrant"`
		SharedMarkup      string      `json:"shared_markup"`
		DataStrategy      string      `json:"data_strategy"`
		PollingConfig     interface{} `json:"polling_config"`
		FormFields        interface{} `json:"form_fields"`
		SampleData        interface{} `json:"sample_data"`
		RemoveBleedMargin bool        `json:"remove_bleed_margin"`
		EnableDarkMode    bool        `json:"enable_dark_mode"`
	}

	var req UpdatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate polling configuration
	if err := ValidatePollingConfig(req.PollingConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Polling config validation failed", "details": err.Error()})
		return
	}

	// Validate and convert form fields to JSON schema
	configSchema, err := ValidateFormFields(req.FormFields)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form fields validation failed", "details": err.Error()})
		return
	}

	// Convert configs to JSON for storage
	var pollingConfigJSON, formFieldsJSON, sampleDataJSON []byte

	if req.PollingConfig != nil {
		pollingConfigJSON, err = json.Marshal(req.PollingConfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid polling config"})
			return
		}
	}

	if req.FormFields != nil {
		// Normalize form fields before storage to ensure consistency
		normalizedFormFields := NormalizeFormFields(req.FormFields)
		if normalizedFormFields != nil {
			formFieldsJSON, err = json.Marshal(normalizedFormFields)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form fields config"})
				return
			}
		}
		// If normalized to nil, leave formFieldsJSON as nil (empty byte slice)
	}

	if req.SampleData != nil {
		sampleDataJSON, err = json.Marshal(req.SampleData)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sample data"})
			return
		}
	}

	db := database.GetDB()
	var pluginDefinition database.PluginDefinition
	
	// Only allow users to update their own private plugins
	err = db.Where("id = ? AND owner_id = ? AND plugin_type = 'private'", definitionID, userID).First(&pluginDefinition).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin definition not found or access denied"})
		return
	}

	// Check if form fields have changed to increment schema version
	formFieldsChanged := CompareFormFieldSchemas(pluginDefinition.FormFields, formFieldsJSON)
	currentSchemaVersion := pluginDefinition.SchemaVersion

	// Update fields
	if req.Name != "" {
		pluginDefinition.Name = req.Name
	}
	pluginDefinition.Description = req.Description
	pluginDefinition.Version = req.Version
	pluginDefinition.Author = req.Author
	pluginDefinition.ConfigSchema = configSchema // Use converted JSON schema
	pluginDefinition.MarkupFull = &req.MarkupFull
	pluginDefinition.MarkupHalfVert = &req.MarkupHalfVert
	pluginDefinition.MarkupHalfHoriz = &req.MarkupHalfHoriz
	pluginDefinition.MarkupQuadrant = &req.MarkupQuadrant
	pluginDefinition.SharedMarkup = &req.SharedMarkup
	pluginDefinition.DataStrategy = &req.DataStrategy
	pluginDefinition.PollingConfig = pollingConfigJSON
	pluginDefinition.FormFields = formFieldsJSON
	pluginDefinition.SampleData = sampleDataJSON
	pluginDefinition.RemoveBleedMargin = &req.RemoveBleedMargin
	pluginDefinition.EnableDarkMode = &req.EnableDarkMode
	pluginDefinition.UpdatedAt = time.Now()

	// Increment schema version if form fields changed
	if formFieldsChanged {
		pluginDefinition.SchemaVersion = currentSchemaVersion + 1
		logging.Info("[PLUGIN_UPDATE] Form fields changed, incrementing schema version", "plugin_id", pluginDefinition.ID, "old_version", currentSchemaVersion, "new_version", pluginDefinition.SchemaVersion)
	}

	if err := db.Save(&pluginDefinition).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plugin definition: " + err.Error()})
		return
	}

	// If form fields changed, flag all instances to need config updates
	if formFieldsChanged {
		result := db.Model(&database.PluginInstance{}).
			Where("plugin_definition_id = ?", pluginDefinition.ID).
			Updates(map[string]interface{}{
				"needs_config_update": true,
				"updated_at": time.Now(),
			})
		
		if result.Error != nil {
			logging.Error("[PLUGIN_UPDATE] Failed to flag instances for config updates", "plugin_id", pluginDefinition.ID, "error", result.Error)
		} else {
			logging.Info("[PLUGIN_UPDATE] Flagged instances for config updates", "plugin_id", pluginDefinition.ID, "affected_instances", result.RowsAffected)
		}
	}

	// Schedule renders for all instances of this updated plugin definition
	unifiedPluginService := database.NewUnifiedPluginService(db)
	instances, err := unifiedPluginService.GetPluginInstancesByDefinition(pluginDefinition.ID)
	if err != nil {
		logging.Error("[PLUGIN_UPDATE] Failed to get plugin instances for render scheduling", "plugin_id", pluginDefinition.ID, "error", err)
	} else if len(instances) > 0 {
		instanceIDs := make([]uuid.UUID, len(instances))
		for i, instance := range instances {
			instanceIDs[i] = instance.ID
		}
		ScheduleRenderForInstances(instanceIDs)
		logging.Info("[PLUGIN_UPDATE] Scheduled renders for plugin instances", "plugin_id", pluginDefinition.ID, "instance_count", len(instances))
	}

	c.JSON(http.StatusOK, gin.H{"plugin_definition": pluginDefinition})
}

// DeletePluginDefinitionHandler deletes a plugin definition
func DeletePluginDefinitionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userID := user.ID

	definitionIDStr := c.Param("id")
	if definitionIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Definition ID is required"})
		return
	}

	definitionID, err := uuid.Parse(definitionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid definition ID format"})
		return
	}

	db := database.GetDB()
	service := database.NewUnifiedPluginService(db)
	
	// Use the service method which properly handles cascading deletions
	err = service.DeletePluginDefinition(definitionID, &userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete plugin definition: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plugin definition deleted successfully"})
}

// ValidatePluginDefinitionHandler validates plugin templates
func ValidatePluginDefinitionHandler(c *gin.Context) {
	_, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	type ValidateRequest struct {
		Name             string `json:"name"`
		Description      string `json:"description"`
		MarkupFull       string `json:"markup_full"`
		MarkupHalfVert   string `json:"markup_half_vert"`
		MarkupHalfHoriz  string `json:"markup_half_horiz"`
		MarkupQuadrant   string `json:"markup_quadrant"`
		SharedMarkup     string `json:"shared_markup"`
		DataStrategy     string `json:"data_strategy"`
		PollingConfig    interface{} `json:"polling_config"`
		FormFields       interface{} `json:"form_fields"`
		Version          string `json:"version"`
		PluginType       string `json:"plugin_type"`
	}

	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Add actual template validation logic
	// For now, just return success
	c.JSON(http.StatusOK, gin.H{
		"valid": true, 
		"message": "Templates validated successfully", 
		"warnings": []string{},
		"errors": []string{},
	})
}

// TestPlugin represents a plugin for testing purposes
type TestPlugin struct {
	Name             string      `json:"name"`
	Description      string      `json:"description"`
	MarkupFull       string      `json:"markup_full"`
	MarkupHalfVert   string      `json:"markup_half_vert"`
	MarkupHalfHoriz  string      `json:"markup_half_horiz"`
	MarkupQuadrant   string      `json:"markup_quadrant"`
	SharedMarkup     string      `json:"shared_markup"`
	DataStrategy     string      `json:"data_strategy"`
	PollingConfig    interface{} `json:"polling_config"`
	FormFields       interface{} `json:"form_fields"`
	Version          string      `json:"version"`
	PluginType       string      `json:"plugin_type"`
}

// extractFormFieldDefaults extracts default values from form field configuration
func extractFormFieldDefaults(formFields interface{}) map[string]interface{} {
	defaults := make(map[string]interface{})
	
	if formFields == nil {
		logging.Debug("[TestPlugin] No form fields provided")
		return defaults
	}
	
	// Try to parse form fields structure
	var fieldList []map[string]interface{}
	
	// Handle different possible structures
	if formFieldsMap, ok := formFields.(map[string]interface{}); ok {
		if yamlField, exists := formFieldsMap["yaml"]; exists {
			// Form fields are stored in "yaml" key as string - parse as YAML
			if yamlStr, ok := yamlField.(string); ok && yamlStr != "" {
				logging.Debug("[TestPlugin] Parsing YAML form fields", "yaml", yamlStr)
				if err := yaml.Unmarshal([]byte(yamlStr), &fieldList); err != nil {
					logging.Error("[TestPlugin] Failed to parse form fields YAML", "error", err, "yaml", yamlStr)
					return defaults
				}
			}
		}
	} else if list, ok := formFields.([]interface{}); ok {
		// Direct list format - convert to expected structure
		for _, item := range list {
			if itemMap, ok := item.(map[string]interface{}); ok {
				fieldList = append(fieldList, itemMap)
			}
		}
	}
	
	// Extract defaults from field list
	for _, field := range fieldList {
		if keyname, exists := field["keyname"]; exists {
			if keynameStr, ok := keyname.(string); ok {
				if defaultVal, exists := field["default"]; exists {
					defaults[keynameStr] = defaultVal
					logging.Info("[TestPlugin] Found form field default", "key", keynameStr, "value", defaultVal)
				}
			}
		}
	}
	
	logging.Info("[TestPlugin] Extracted form field defaults", "count", len(defaults), "defaults", defaults)
	return defaults
}

// getPollingDataForPreview uses the existing poller to fetch real data for preview
func getPollingDataForPreview(plugin TestPlugin, formDefaults map[string]interface{}) (map[string]interface{}, error) {
	logging.Info("[TestPlugin] Starting polling data fetch", "plugin", plugin.Name, "form_defaults", formDefaults)
	
	// Convert polling config to the expected structure
	pollingConfigBytes, err := json.Marshal(plugin.PollingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal polling config: %v", err)
	}
	
	logging.Info("[TestPlugin] Polling config marshaled", "config_bytes", string(pollingConfigBytes))
	
	var enhancedConfig private.EnhancedPollingConfig
	if err := json.Unmarshal(pollingConfigBytes, &enhancedConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal polling config: %v", err)
	}
	
	logging.Info("[TestPlugin] Polling config parsed", "urls_count", len(enhancedConfig.URLs))
	
	// Create a minimal plugin definition for the poller
	dataStrategy := "polling"
	pluginDefinition := &database.PluginDefinition{
		ID:            uuid.New(),
		Name:          plugin.Name,
		DataStrategy:  &dataStrategy,
		PollingConfig: pollingConfigBytes,
	}
	
	// Create poller and fetch data
	poller := private.NewEnhancedDataPoller()
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	// Use the poller to get real data
	logging.Info("[TestPlugin] Calling poller.PollData")
	result, err := poller.PollData(ctx, pluginDefinition, formDefaults)
	if err != nil {
		logging.Error("[TestPlugin] Poller.PollData failed", "error", err)
		return nil, fmt.Errorf("polling failed: %v", err)
	}
	
	if !result.Success {
		logging.Error("[TestPlugin] Polling was not successful", "errors", result.Errors)
		return nil, fmt.Errorf("polling was not successful: %v", result.Errors)
	}
	
	logging.Info("[TestPlugin] Polling successful", "data_keys", len(result.Data), "duration", result.Duration)
	return result.Data, nil
}

// TestPluginDefinitionHandler tests plugin template rendering
func TestPluginDefinitionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	type TestRequest struct {
		Plugin       TestPlugin             `json:"plugin"`
		Layout       string                 `json:"layout"`
		SampleData   map[string]interface{} `json:"sample_data"`
		DeviceWidth  int                    `json:"device_width"`
		DeviceHeight int                    `json:"device_height"`
	}

	var req TestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Select template based on layout
	var layoutTemplate string
	switch req.Layout {
	case "full":
		layoutTemplate = req.Plugin.MarkupFull
	case "half_vertical":
		layoutTemplate = req.Plugin.MarkupHalfVert
	case "half_horizontal":
		layoutTemplate = req.Plugin.MarkupHalfHoriz
	case "quadrant":
		layoutTemplate = req.Plugin.MarkupQuadrant
	default:
		layoutTemplate = req.Plugin.MarkupFull
	}

	if layoutTemplate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("No template defined for layout: %s", req.Layout)})
		return
	}

	// Generate instance ID for preview
	instanceID := fmt.Sprintf("preview_%d", time.Now().Unix())

	// Determine what data to use for rendering
	var templateData map[string]interface{}
	
	// Try to get real data via polling if configured
	logging.Info("[TestPlugin] Plugin data strategy", "strategy", req.Plugin.DataStrategy, "has_polling_config", req.Plugin.PollingConfig != nil)
	
	if req.Plugin.DataStrategy == "polling" && req.Plugin.PollingConfig != nil {
		logging.Info("[TestPlugin] Attempting to use real polling data", "plugin_name", req.Plugin.Name)
		
		// Log the raw polling config for debugging
		if configBytes, err := json.Marshal(req.Plugin.PollingConfig); err == nil {
			logging.Info("[TestPlugin] Raw polling config", "config", string(configBytes))
		}
		
		// Extract form field defaults
		formDefaults := extractFormFieldDefaults(req.Plugin.FormFields)
		
		// Try to get real polling data
		realData, err := getPollingDataForPreview(req.Plugin, formDefaults)
		if err != nil {
			logging.Error("[TestPlugin] Failed to get polling data, falling back to sample data", "error", err)
			templateData = req.SampleData
		} else {
			logging.Info("[TestPlugin] Successfully got real polling data", "data_keys", len(realData))
			// Log the actual keys in the real data for debugging
			var dataKeys []string
			for key := range realData {
				dataKeys = append(dataKeys, key)
			}
			logging.Info("[TestPlugin] Real data keys", "keys", dataKeys)
			templateData = realData
		}
	} else {
		logging.Info("[TestPlugin] Using sample data (not polling strategy or no polling config)")
		templateData = req.SampleData
	}

	// Create TRMNL data structure to match what real plugins receive
	trmnlData := make(map[string]interface{})
	
	// Add system information - Unix timestamp
	systemData := map[string]interface{}{
		"timestamp_utc": time.Now().Unix(),
	}
	trmnlData["system"] = systemData
	
	// Add user information if available
	if user != nil {
		// Calculate UTC offset in seconds
		utcOffset := int64(0)
		locale := "en" // Default locale
		timezone := "UTC" // Default timezone IANA
		timezoneFriendly := "UTC" // Default friendly name
		
		if user.Timezone != "" {
			timezone = user.Timezone
			timezoneFriendly = utils.GetTimezoneFriendlyName(user.Timezone)
			// Parse timezone and calculate UTC offset
			loc, err := time.LoadLocation(user.Timezone)
			if err == nil {
				_, offset := time.Now().In(loc).Zone()
				utcOffset = int64(offset)
			}
		}
		
		if user.Locale != "" {
			// Convert "en-US" to "en" format if needed
			if len(user.Locale) >= 2 {
				locale = user.Locale[:2]
			}
		}

		// Build user full name
		firstName := user.FirstName
		lastName := user.LastName
		fullName := ""
		if firstName != "" && lastName != "" {
			fullName = firstName + " " + lastName
		} else if firstName != "" {
			fullName = firstName
		} else if lastName != "" {
			fullName = lastName
		} else {
			// Fallback to username if no names available
			fullName = user.Username
		}

		userData := map[string]interface{}{
			"name":           fullName,
			"first_name":     firstName,
			"last_name":      lastName,
			"locale":         locale,
			"time_zone":      timezoneFriendly,
			"time_zone_iana": timezone,
			"utc_offset":     utcOffset,
		}
		
		trmnlData["user"] = userData
	}

	// Add mock device information for preview
	deviceData := map[string]interface{}{
		"friendly_id":     "TEST1",
		"width":           req.DeviceWidth,
		"height":          req.DeviceHeight,
		"percent_charged": 100,
		"wifi_strength":   100,
	}
	trmnlData["device"] = deviceData

	// Add plugin settings metadata
	pluginSettings := map[string]interface{}{
		"instance_name": req.Plugin.Name,
		"strategy":      req.Plugin.DataStrategy,
		"dark_mode":     "no",
		"no_screen_padding": "no",
	}
	trmnlData["plugin_settings"] = pluginSettings

	// Merge TRMNL data with template data
	finalTemplateData := make(map[string]interface{})
	// First add the external/polling/sample data
	for key, value := range templateData {
		finalTemplateData[key] = value
	}
	// Then add TRMNL data
	finalTemplateData["trmnl"] = trmnlData

	logging.Info("[TestPlugin] Created TRMNL context data", "user_available", user != nil, "device_width", req.DeviceWidth, "device_height", req.DeviceHeight)

	// Use the private plugin renderer service
	htmlRenderer := private.NewPrivatePluginRenderer()
	renderedHTML, err := htmlRenderer.RenderToClientSideHTML(private.RenderOptions{
		SharedMarkup:   req.Plugin.SharedMarkup,
		LayoutTemplate: layoutTemplate,
		Data:           finalTemplateData,
		Width:          req.DeviceWidth,
		Height:         req.DeviceHeight,
		PluginName:     req.Plugin.Name,
		InstanceID:     instanceID,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Template render error: %v", err)})
		return
	}

	// Convert HTML to image using BrowserlessRenderer
	browserRenderer, err := rendering.DefaultBrowserlessRenderer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create renderer: %v", err)})
		return
	}
	defer browserRenderer.Close()

	ctx := context.Background()
	imageData, err := browserRenderer.RenderHTML(ctx, renderedHTML, req.DeviceWidth, req.DeviceHeight)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to render image: %v", err)})
		return
	}

	// Convert image data to base64 data URL
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	previewURL := fmt.Sprintf("data:image/png;base64,%s", base64Image)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"preview_url": previewURL,
	})
}

// GetPluginInstanceSchemaDiffHandler returns schema differences for an instance that needs config updates
func GetPluginInstanceSchemaDiffHandler(c *gin.Context) {
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
	var pluginInstance database.PluginInstance
	err := db.Preload("PluginDefinition").Where("id = ? AND user_id = ?", instanceID, userID).First(&pluginInstance).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin instance not found"})
		return
	}

	// Only return diff if instance needs config update
	if !pluginInstance.NeedsConfigUpdate {
		c.JSON(http.StatusOK, gin.H{
			"needs_update": false,
			"message":      "Instance is up to date",
		})
		return
	}

	type SchemaDiff struct {
		NeedsUpdate         bool   `json:"needs_update"`
		CurrentSchemaVersion int    `json:"current_schema_version"`
		InstanceSchemaVersion int   `json:"instance_schema_version"`
		Message             string `json:"message"`
		// TODO: Add more detailed field-level diff information when needed
	}

	diff := SchemaDiff{
		NeedsUpdate:          true,
		CurrentSchemaVersion: pluginInstance.PluginDefinition.SchemaVersion,
		InstanceSchemaVersion: pluginInstance.LastSchemaVersion,
		Message:              "This plugin instance needs to be updated because the form configuration has changed. Please review and update your settings.",
	}

	c.JSON(http.StatusOK, diff)
}

// Mashup Management Endpoints

// CreateMashupDefinitionHandler creates a new mashup plugin definition
func CreateMashupDefinitionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	var request struct {
		Name         string `json:"name" binding:"required"`
		Description  string `json:"description"`
		MashupLayout string `json:"mashup_layout" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Create mashup plugin definition
	db := database.GetDB()
	pluginService := database.NewUnifiedPluginService(db)
	
	definition := &database.PluginDefinition{
		Name:              request.Name,
		Description:       request.Description,
		Identifier:        fmt.Sprintf("mashup_%s_%d", user.ID.String()[:8], time.Now().Unix()),
		PluginType:        "mashup",
		Version:           "1.0.0",
		Author:            user.Username,
		IsActive:          true,
		RequiresProcessing: true,
		OwnerID:           &user.ID,
		MashupLayout:      &request.MashupLayout,
	}

	if err := pluginService.CreatePluginDefinition(definition); err != nil {
		logging.Error("Failed to create mashup definition", "error", err, "user_id", user.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create mashup definition"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      definition.ID.String(),
		"message": "Mashup definition created successfully",
	})
}

// AddMashupChildHandler adds a child plugin instance to a mashup
func AddMashupChildHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	mashupInstanceID := c.Param("id")
	if mashupInstanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mashup instance ID is required"})
		return
	}

	var request struct {
		ChildInstanceID string `json:"child_instance_id" binding:"required"`
		GridPosition    string `json:"grid_position" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Parse UUIDs
	mashupUUID, err := uuid.Parse(mashupInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mashup instance ID"})
		return
	}

	childUUID, err := uuid.Parse(request.ChildInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid child instance ID"})
		return
	}

	// Verify user owns the mashup instance
	db := database.GetDB()
	pluginService := database.NewUnifiedPluginService(db)
	mashupInstance, err := pluginService.GetPluginInstanceByID(mashupUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mashup instance not found"})
		return
	}

	if mashupInstance.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Add child to mashup
	if err := pluginService.AddMashupChild(mashupUUID, childUUID, request.GridPosition); err != nil {
		logging.Error("Failed to add mashup child", "error", err, "mashup_id", mashupInstanceID, "child_id", request.ChildInstanceID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Child plugin added to mashup successfully"})
}

// RemoveMashupChildHandler removes a child plugin instance from a mashup
func RemoveMashupChildHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	mashupInstanceID := c.Param("id")
	childInstanceID := c.Param("childId")

	if mashupInstanceID == "" || childInstanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both mashup and child instance IDs are required"})
		return
	}

	// Parse UUIDs
	mashupUUID, err := uuid.Parse(mashupInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mashup instance ID"})
		return
	}

	childUUID, err := uuid.Parse(childInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid child instance ID"})
		return
	}

	// Verify user owns the mashup instance
	db := database.GetDB()
	pluginService := database.NewUnifiedPluginService(db)
	mashupInstance, err := pluginService.GetPluginInstanceByID(mashupUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mashup instance not found"})
		return
	}

	if mashupInstance.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Remove child from mashup
	if err := pluginService.RemoveMashupChild(mashupUUID, childUUID); err != nil {
		logging.Error("Failed to remove mashup child", "error", err, "mashup_id", mashupInstanceID, "child_id", childInstanceID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Child plugin removed from mashup successfully"})
}

// UpdateMashupChildPositionHandler updates the grid position of a child plugin in a mashup
func UpdateMashupChildPositionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	mashupInstanceID := c.Param("id")
	childInstanceID := c.Param("childId")

	if mashupInstanceID == "" || childInstanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both mashup and child instance IDs are required"})
		return
	}

	var request struct {
		GridPosition string `json:"grid_position" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Parse UUIDs
	mashupUUID, err := uuid.Parse(mashupInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mashup instance ID"})
		return
	}

	childUUID, err := uuid.Parse(childInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid child instance ID"})
		return
	}

	// Verify user owns the mashup instance
	db := database.GetDB()
	pluginService := database.NewUnifiedPluginService(db)
	mashupInstance, err := pluginService.GetPluginInstanceByID(mashupUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mashup instance not found"})
		return
	}

	if mashupInstance.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update child position
	if err := pluginService.UpdateMashupChildPosition(mashupUUID, childUUID, request.GridPosition); err != nil {
		logging.Error("Failed to update mashup child position", "error", err, "mashup_id", mashupInstanceID, "child_id", childInstanceID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Child plugin position updated successfully"})
}

// GetMashupChildrenHandler retrieves all child plugin instances for a mashup
func GetMashupChildrenHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	mashupInstanceID := c.Param("id")
	if mashupInstanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mashup instance ID is required"})
		return
	}

	// Parse UUID
	mashupUUID, err := uuid.Parse(mashupInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mashup instance ID"})
		return
	}

	// Verify user owns the mashup instance
	db := database.GetDB()
	pluginService := database.NewUnifiedPluginService(db)
	mashupInstance, err := pluginService.GetPluginInstanceByID(mashupUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mashup instance not found"})
		return
	}

	if mashupInstance.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get children
	children, err := pluginService.GetMashupChildren(mashupUUID)
	if err != nil {
		logging.Error("Failed to get mashup children", "error", err, "mashup_id", mashupInstanceID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get mashup children"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"children": children})
}

// GetAvailableMashupLayoutsHandler returns available mashup layout types
func GetAvailableMashupLayoutsHandler(c *gin.Context) {
	layouts := []map[string]interface{}{
		{
			"id":          "1L1R",
			"name":        "Left & Right",
			"description": "Two plugins side by side",
			"positions":   []string{"left", "right"},
		},
		{
			"id":          "2T1B",
			"name":        "Two Top, One Bottom",
			"description": "Two plugins on top, one on bottom",
			"positions":   []string{"top-left", "top-right", "bottom"},
		},
		{
			"id":          "1T2B",
			"name":        "One Top, Two Bottom",
			"description": "One plugin on top, two on bottom",
			"positions":   []string{"top", "bottom-left", "bottom-right"},
		},
		{
			"id":          "2x2",
			"name":        "Four Quadrants",
			"description": "Four plugins in a 2x2 grid",
			"positions":   []string{"top-left", "top-right", "bottom-left", "bottom-right"},
		},
	}

	c.JSON(http.StatusOK, gin.H{"layouts": layouts})
}