package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/plugins/external"
	"github.com/rmitchellscott/stationmaster/internal/plugins/private"
	"github.com/rmitchellscott/stationmaster/internal/utils"
	"github.com/rmitchellscott/stationmaster/internal/validation"
	"gopkg.in/yaml.v3"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// UnifiedPluginDefinition represents a plugin definition that can be system, private, or external
type UnifiedPluginDefinition struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"type"`               // "system", "private", or "external"
	PluginType         string `json:"plugin_type"`       // "image" or "data"
	Description        string `json:"description"`
	ConfigSchema       string `json:"config_schema"`
	Version            string `json:"version"`
	Author             string `json:"author"`
	IsActive           bool   `json:"is_active"`
	RequiresProcessing bool   `json:"requires_processing"`
	Status             string `json:"status"`             // "available", "unavailable", "error"
	
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
			Status:             "available", // System plugins are always available
		}
		allPlugins = append(allPlugins, unifiedPlugin)
	}

	// Get external plugins from database (available to all users like system plugins)
	// Only include plugins with status = "available" so unavailable plugins don't show in "Add Plugin" UI
	db := database.GetDB()
	var externalPlugins []database.PluginDefinition
	err := db.Where("plugin_type = ? AND is_active = ? AND status = ?", "external", true, "available").Find(&externalPlugins).Error
	if err == nil {
		for _, extPlugin := range externalPlugins {
			// Create external plugin instance to get properly processed ConfigSchema
			externalPluginInstance := external.NewExternalPlugin(&extPlugin, nil)
			configSchema := externalPluginInstance.ConfigSchema()
			
			unifiedPlugin := UnifiedPluginDefinition{
				ID:                 extPlugin.ID,
				Name:               extPlugin.Name,
				Type:               "external", // Keep true type, UI will handle display
				PluginType:         extPlugin.PluginType,
				Description:        extPlugin.Description,
				ConfigSchema:       configSchema, // Use processed schema from plugin method
				Version:            extPlugin.Version,
				Author:             extPlugin.Author,
				IsActive:           extPlugin.IsActive,
				RequiresProcessing: extPlugin.RequiresProcessing,
				Status:             extPlugin.Status, // Include availability status
				// No InstanceCount for external plugins (like system plugins)
			}
			allPlugins = append(allPlugins, unifiedPlugin)
		}
	}

	// Get user's private plugins from unified plugin_definitions table
	
	// Filter by plugin_type query parameter if provided
	pluginType := c.Query("plugin_type")
	
	var privatePlugins []database.PluginDefinition
	query := db.Where("owner_id = ?", userID)
	if pluginType != "" {
		query = query.Where("plugin_type = ?", pluginType)
	} else {
		// Default: get only private plugins (exclude mashups and don't mix with system)
		query = query.Where("plugin_type = ?", "private")
	}
	
	err = query.Find(&privatePlugins).Error
	if err == nil {
		for _, privatePlugin := range privatePlugins {
			// Count how many instances user has created of this private plugin
			var instanceCount int64
			db.Model(&database.PluginInstance{}).Where("plugin_definition_id = ? AND user_id = ?", privatePlugin.ID, userID).Count(&instanceCount)
			instances := int(instanceCount)
			
			unifiedPlugin := UnifiedPluginDefinition{
				ID:                 privatePlugin.ID,
				Name:               privatePlugin.Name,
				Type:               "private",
				PluginType:         privatePlugin.PluginType,
				Description:        privatePlugin.Description,
				ConfigSchema:       privatePlugin.ConfigSchema,
				Version:            privatePlugin.Version,
				Author:             privatePlugin.Author,
				IsActive:           privatePlugin.IsActive,
				RequiresProcessing: privatePlugin.RequiresProcessing,
				Status:             privatePlugin.Status, // Include availability status
				InstanceCount:      &instances,
			}
			allPlugins = append(allPlugins, unifiedPlugin)
		}
	}

	// Sort plugins by name only
	sort.Slice(allPlugins, func(i, j int) bool {
		return allPlugins[i].Name < allPlugins[j].Name
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
		Status             string `json:"status"`
		ConfigSchema       string `json:"config_schema"`
		Version            string `json:"version"`
		Author             string `json:"author"`
		IsActive           bool   `json:"is_active"`
		RequiresProcessing bool   `json:"requires_processing"`
		DataStrategy       string `json:"data_strategy"`
		IsMashup           bool   `json:"is_mashup"`
	} `json:"plugin"`
	
	// Plugin definition info (for compatibility with frontend expecting plugin_definition)
	PluginDefinition struct {
		ID                 string `json:"id"`
		Name               string `json:"name"`
		PluginType         string `json:"plugin_type"`
		Description        string `json:"description"`
		Status             string `json:"status"`
		ConfigSchema       string `json:"config_schema"`
		Version            string `json:"version"`
		Author             string `json:"author"`
		IsActive           bool   `json:"is_active"`
		RequiresProcessing bool   `json:"requires_processing"`
		DataStrategy       string `json:"data_strategy"`
		IsMashup           bool   `json:"is_mashup"`
	} `json:"plugin_definition"`
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
			// Check if used in playlists directly
			var playlistCount int64
			db.Model(&database.PlaylistItem{}).Where("plugin_instance_id = ?", pluginInstance.ID).Count(&playlistCount)
			
			// Also check if used indirectly as a child of a mashup that's in playlists
			var mashupPlaylistCount int64
			if playlistCount == 0 {
				// Only check mashup parents if not already directly in playlists
				db.Raw(`
					SELECT COUNT(*) 
					FROM playlist_items pi 
					JOIN mashup_children mc ON pi.plugin_instance_id = mc.mashup_instance_id 
					WHERE mc.child_instance_id = ?
				`, pluginInstance.ID).Count(&mashupPlaylistCount)
			}
			
			isUsedInPlaylists := playlistCount > 0 || mashupPlaylistCount > 0

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
				IsUsedInPlaylists: isUsedInPlaylists,
				NeedsConfigUpdate: pluginInstance.NeedsConfigUpdate,
				LastSchemaVersion: pluginInstance.LastSchemaVersion,
			}

			// Fill plugin info from PluginDefinition
			if pluginInstance.PluginDefinition.ID != "" {
				instance.Plugin.ID = pluginInstance.PluginDefinition.ID
				instance.Plugin.Name = pluginInstance.PluginDefinition.Name
				instance.Plugin.Type = pluginInstance.PluginDefinition.PluginType
				instance.Plugin.Description = pluginInstance.PluginDefinition.Description
				instance.Plugin.Status = pluginInstance.PluginDefinition.Status
				
				// For external plugins, generate schema dynamically from YAML form fields
				if pluginInstance.PluginDefinition.PluginType == "external" {
					externalPlugin := external.NewExternalPlugin(&pluginInstance.PluginDefinition, &pluginInstance)
					instance.Plugin.ConfigSchema = externalPlugin.ConfigSchema()
				} else {
					// For other plugins, use database field
					instance.Plugin.ConfigSchema = pluginInstance.PluginDefinition.ConfigSchema
				}
				
				instance.Plugin.Version = pluginInstance.PluginDefinition.Version
				instance.Plugin.Author = pluginInstance.PluginDefinition.Author
				instance.Plugin.IsActive = true
				instance.Plugin.RequiresProcessing = pluginInstance.PluginDefinition.RequiresProcessing
				instance.Plugin.IsMashup = pluginInstance.PluginDefinition.IsMashup
				
				// Set data strategy (no fallback - only set if explicitly defined)
				if pluginInstance.PluginDefinition.DataStrategy != nil {
					instance.Plugin.DataStrategy = *pluginInstance.PluginDefinition.DataStrategy
				} else {
					instance.Plugin.DataStrategy = ""
				}
				
				// Populate plugin_definition for frontend compatibility
				instance.PluginDefinition.ID = pluginInstance.PluginDefinition.ID
				instance.PluginDefinition.Name = pluginInstance.PluginDefinition.Name
				instance.PluginDefinition.PluginType = pluginInstance.PluginDefinition.PluginType
				instance.PluginDefinition.Description = pluginInstance.PluginDefinition.Description
				instance.PluginDefinition.Status = pluginInstance.PluginDefinition.Status
				instance.PluginDefinition.ConfigSchema = instance.Plugin.ConfigSchema // Use the processed schema
				instance.PluginDefinition.Version = pluginInstance.PluginDefinition.Version
				instance.PluginDefinition.Author = pluginInstance.PluginDefinition.Author
				instance.PluginDefinition.IsActive = true
				instance.PluginDefinition.RequiresProcessing = pluginInstance.PluginDefinition.RequiresProcessing
				instance.PluginDefinition.IsMashup = pluginInstance.PluginDefinition.IsMashup
				
				if pluginInstance.PluginDefinition.DataStrategy != nil {
					instance.PluginDefinition.DataStrategy = *pluginInstance.PluginDefinition.DataStrategy
				} else {
					instance.PluginDefinition.DataStrategy = ""
				}
			}

			allInstances = append(allInstances, instance)
		}
	}

	// Sort instances alphabetically by name
	sort.Slice(allInstances, func(i, j int) bool {
		return allInstances[i].Name < allInstances[j].Name
	})

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
		c.JSON(http.StatusNotFound, gin.H{
			"error":         "Plugin definition not found",
			"definition_id": req.DefinitionID,
			"definition_type": req.DefinitionType,
			"details":       "The requested plugin definition does not exist in the database. This may indicate that system plugins have not been bootstrapped or the plugin ID is invalid.",
		})
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
	// Check if frequent refreshes are enabled
	frequentRefreshesEnabled := false
	if enabledStr, err := database.GetSystemSetting("enable_frequent_refreshes"); err == nil {
		frequentRefreshesEnabled = enabledStr == "true"
	}
	
	options := database.GetRefreshRateOptionsWithFrequent(frequentRefreshesEnabled)
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
		c.JSON(http.StatusNotFound, gin.H{
			"error":         "Plugin definition not found",
			"definition_id": req.DefinitionID,
			"details":       "The requested plugin definition does not exist in the database. Check that the plugin ID is correct and that system plugins have been bootstrapped.",
		})
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
		c.JSON(http.StatusNotFound, gin.H{
			"error":         "Plugin definition not found",
			"definition_id": definitionID,
			"details":       "The requested plugin definition does not exist or you don't have access to it. Check that the plugin ID is correct and that you own the private plugin or it's a system plugin.",
		})
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
	configSchema, err := validation.ValidateFormFields(req.FormFields)
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
		logging.Debug("[CREATE_HANDLER] Received and marshalled sample data", "data", string(sampleDataJSON))
	} else {
		logging.Debug("[CREATE_HANDLER] No sample data received from UI")
	}

	db := database.GetDB()

	pluginDefinition := database.PluginDefinition{
		PluginType:         req.PluginType,
		OwnerID:            &userID,
		Identifier:         uuid.New().String(), // Generate unique identifier
		Name:               req.Name,
		Description:        req.Description,
		Version:            req.Version,
		Author:             user.Username,
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

	logging.Debug("[CREATE_HANDLER] Plugin definition created in database", "plugin_id", pluginDefinition.ID, "sample_data_size", len(pluginDefinition.SampleData))

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
	configSchema, err := validation.ValidateFormFields(req.FormFields)
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
		logging.Debug("[CREATE_HANDLER] Received and marshalled sample data", "data", string(sampleDataJSON))
	} else {
		logging.Debug("[CREATE_HANDLER] No sample data received from UI")
	}

	db := database.GetDB()
	var pluginDefinition database.PluginDefinition
	
	// Only allow users to update their own private plugins
	err = db.Where("id = ? AND owner_id = ? AND plugin_type = 'private'", definitionID, userID).First(&pluginDefinition).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":         "Plugin definition not found or access denied",
			"definition_id": definitionID,
			"details":       "The requested private plugin definition does not exist or you are not the owner. Only the plugin owner can update private plugin definitions.",
		})
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
	pluginDefinition.Author = user.Username
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

	// No need to parse as UUID since IDs are now strings
	definitionID := definitionIDStr

	db := database.GetDB()
	service := database.NewUnifiedPluginService(db)
	
	// Use the service method which properly handles cascading deletions
	err := service.DeletePluginDefinition(definitionID, &userID)
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
		ID:            uuid.New().String(),
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
		LayoutWidth  int                    `json:"layout_width"`  // Layout-specific dimensions for content positioning
		LayoutHeight int                    `json:"layout_height"` // Layout-specific dimensions for content positioning
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
	htmlRenderer, err := private.NewPrivatePluginRenderer(".")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to initialize private plugin renderer: %v", err)})
		return
	}
	renderedHTML, err := htmlRenderer.RenderToServerSideHTML(c.Request.Context(), private.RenderOptions{
		SharedMarkup:   req.Plugin.SharedMarkup,
		LayoutTemplate: layoutTemplate,
		Data:           finalTemplateData,
		Width:          req.DeviceWidth,
		Height:         req.DeviceHeight,
		PluginName:     req.Plugin.Name,
		InstanceID:     instanceID,
		Layout:         req.Layout,         // Pass layout info for proper mashup structure
		LayoutWidth:    req.LayoutWidth,    // Layout-specific dimensions
		LayoutHeight:   req.LayoutHeight,   // Layout-specific dimensions
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

// ImportPluginDefinitionHandler imports a TRMNL-compatible ZIP file as a private plugin
func ImportPluginDefinitionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded", "details": err.Error()})
		return
	}
	defer file.Close()

	// Create ZIP service
	zipService := NewTRMNLZipService()

	// Validate ZIP structure
	if err := zipService.ValidateZipStructure(file, header); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ZIP file", "details": err.Error()})
		return
	}

	// Reset file position
	if _, err := file.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process uploaded file"})
		return
	}

	// Extract and validate ZIP contents
	zipData, err := zipService.ExtractTRMNLZip(file, header)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid TRMNL ZIP format", "details": err.Error()})
		return
	}

	// Convert to PluginDefinition
	def, err := zipService.ConvertZipDataToPluginDefinition(zipData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to process plugin data", "details": err.Error()})
		return
	}

	// Template validation removed to match manual plugin creation behavior

	// Set ownership
	def.OwnerID = &user.ID
	def.Author = user.Username

	// Create the plugin in the unified system
	db := database.GetDB()
	unifiedService := database.NewUnifiedPluginService(db)

	if err := unifiedService.CreatePluginDefinition(def); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create private plugin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Private plugin imported successfully",
		"plugin": gin.H{
			"id":          def.ID,
			"name":        def.Name,
			"description": def.Description,
			"version":     def.Version,
		},
	})
}

// safeStringValue safely extracts string value from pointer
func safeStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ExportPluginDefinitionHandler exports a private plugin as a TRMNL-compatible ZIP file
func ExportPluginDefinitionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin ID"})
		return
	}

	db := database.GetDB()

	unifiedService := database.NewUnifiedPluginService(db)
	def, err := unifiedService.GetPluginDefinitionByID(id.String())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	if def.OwnerID == nil || *def.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	zipService := NewTRMNLZipService()
	zipBuffer, err := zipService.CreateTRMNLZip(def)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ZIP file"})
		return
	}

	filename := fmt.Sprintf("%s.zip", strings.ReplaceAll(def.Name, " ", "_"))
	
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/zip", zipBuffer.Bytes())
}

// ========== MASHUP PLUGIN HANDLERS ==========

// CreateMashupHandler creates a new mashup plugin definition
func CreateMashupHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	type CreateMashupRequest struct {
		Name        string `json:"name" binding:"required,min=1,max=255"`
		Description string `json:"description" binding:"max=1000"`
		Layout      string `json:"layout" binding:"required"`
	}

	var req CreateMashupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Validate layout
	validLayouts := []string{"1Lx1R", "1Tx1B", "1Lx2R", "2Lx1R", "2Tx1B", "1Tx2B", "2x2"}
	layoutValid := false
	for _, layout := range validLayouts {
		if req.Layout == layout {
			layoutValid = true
			break
		}
	}
	if !layoutValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid layout", "valid_layouts": validLayouts})
		return
	}

	db := database.GetDB()
	mashupService := database.NewMashupService(db)

	// Create mashup definition
	definition, err := mashupService.CreateMashupDefinition(user.ID, req.Name, req.Layout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create mashup", "details": err.Error()})
		return
	}

	// Get slot metadata for the response
	slots, _ := mashupService.GetSlotMetadata(req.Layout)

	c.JSON(http.StatusCreated, gin.H{
		"mashup": gin.H{
			"id":          definition.ID,
			"name":        definition.Name,
			"description": definition.Description,
			"layout":      req.Layout,
			"slots":       slots,
		},
	})
}

// GetAvailableMashupLayoutsHandler returns available mashup layouts
func GetAvailableMashupLayoutsHandler(c *gin.Context) {
	db := database.GetDB()
	mashupService := database.NewMashupService(db)
	layouts := mashupService.GetAvailableLayouts()

	c.JSON(http.StatusOK, gin.H{"layouts": layouts})
}

// GetMashupSlotsHandler returns slot configuration for a specific layout
func GetMashupSlotsHandler(c *gin.Context) {
	layout := c.Param("layout")
	if layout == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Layout parameter is required"})
		return
	}

	db := database.GetDB()
	mashupService := database.NewMashupService(db)
	slots, err := mashupService.GetSlotMetadata(layout)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid layout", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"layout": layout, "slots": slots})
}

// AssignMashupChildrenHandler assigns child plugin instances to mashup slots
func AssignMashupChildrenHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	mashupInstanceID := c.Param("id")
	if mashupInstanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mashup instance ID is required"})
		return
	}

	instanceUUID, err := uuid.Parse(mashupInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mashup instance ID"})
		return
	}

	type AssignChildrenRequest struct {
		Assignments map[string]string `json:"assignments" binding:"required"`
	}

	var req AssignChildrenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	db := database.GetDB()

	// Verify mashup instance belongs to user
	var mashupInstance database.PluginInstance
	err = db.Preload("PluginDefinition").Where("id = ? AND user_id = ?", instanceUUID, user.ID).First(&mashupInstance).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mashup instance not found"})
		return
	}

	if mashupInstance.PluginDefinition.PluginType != "mashup" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance is not a mashup plugin"})
		return
	}

	// Convert string UUIDs to UUID type and validate child instances
	assignments := make(map[string]uuid.UUID)
	mashupService := database.NewMashupService(db)

	for slot, childIDStr := range req.Assignments {
		if childIDStr == "" {
			continue // Skip empty assignments
		}

		childUUID, err := uuid.Parse(childIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid child instance ID for slot %s", slot)})
			return
		}

		// Validate child instance
		if err := mashupService.ValidateMashupChild(childUUID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid child for slot %s: %s", slot, err.Error())})
			return
		}

		// Verify child instance belongs to the same user
		var childInstance database.PluginInstance
		err = db.Where("id = ? AND user_id = ?", childUUID, user.ID).First(&childInstance).Error
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Child instance not found for slot %s", slot)})
			return
		}

		assignments[slot] = childUUID
	}

	// Assign children to slots
	if err := mashupService.AssignChildren(instanceUUID, assignments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign children", "details": err.Error()})
		return
	}

	// Calculate and update mashup refresh rate based on children
	refreshRate, err := mashupService.CalculateRefreshRate(instanceUUID)
	if err == nil && refreshRate != mashupInstance.RefreshInterval {
		db.Model(&mashupInstance).Update("refresh_interval", refreshRate)
		logging.Info("[MASHUP] Updated mashup refresh rate", "mashup", mashupInstance.Name, "new_rate", refreshRate, "old_rate", mashupInstance.RefreshInterval)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Children assigned successfully"})
}

// GetMashupChildrenHandler returns the current child assignments for a mashup
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

	instanceUUID, err := uuid.Parse(mashupInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mashup instance ID"})
		return
	}

	db := database.GetDB()

	// Verify mashup instance belongs to user
	var mashupInstance database.PluginInstance
	err = db.Preload("PluginDefinition").Where("id = ? AND user_id = ?", instanceUUID, user.ID).First(&mashupInstance).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mashup instance not found"})
		return
	}

	if mashupInstance.PluginDefinition.PluginType != "mashup" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance is not a mashup plugin"})
		return
	}

	// Get children
	mashupService := database.NewMashupService(db)
	children, err := mashupService.GetChildren(instanceUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get mashup children", "details": err.Error()})
		return
	}

	// Format response
	assignments := make(map[string]interface{})
	for _, child := range children {
		assignments[child.SlotPosition] = gin.H{
			"instance_id":   child.ChildInstanceID.String(),
			"instance_name": child.ChildInstance.Name,
			"plugin_name":   child.ChildInstance.PluginDefinition.Name,
			"plugin_type":   child.ChildInstance.PluginDefinition.PluginType,
		}
	}

	// Get slot metadata
	layout := ""
	if mashupInstance.PluginDefinition.MashupLayout != nil {
		layout = *mashupInstance.PluginDefinition.MashupLayout
	}
	slots, _ := mashupService.GetSlotMetadata(layout)

	c.JSON(http.StatusOK, gin.H{
		"layout":      layout,
		"slots":       slots,
		"assignments": assignments,
	})
}

// GetUserPrivatePluginInstancesHandler returns user's private plugin instances available for mashup children
func GetUserPrivatePluginInstancesHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	db := database.GetDB()

	// Get user's private plugin instances (only active ones)
	var instances []database.PluginInstance
	err := db.Preload("PluginDefinition").
		Where("user_id = ? AND is_active = ?", user.ID, true).
		Find(&instances).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get plugin instances"})
		return
	}

	// Filter for private and external plugins that can be used in mashups
	var availableInstances []gin.H
	mashupService := database.NewMashupService(db)

	for _, instance := range instances {
		// Skip if not private or external plugin
		if instance.PluginDefinition.PluginType != "private" && instance.PluginDefinition.PluginType != "external" {
			continue
		}

		// Skip if it's already a mashup (no nesting)
		if instance.PluginDefinition.IsMashup {
			continue
		}

		// Validate that it can be used as mashup child
		if err := mashupService.ValidateMashupChild(instance.ID); err != nil {
			continue
		}

		availableInstances = append(availableInstances, gin.H{
			"id":                instance.ID.String(),
			"name":              instance.Name,
			"plugin_name":       instance.PluginDefinition.Name,
			"plugin_description": instance.PluginDefinition.Description,
			"refresh_interval":  instance.RefreshInterval,
		})
	}

	c.JSON(http.StatusOK, gin.H{"instances": availableInstances})
}

// AdminGetExternalPluginsHandler returns all external plugin definitions for admin management
func AdminGetExternalPluginsHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	db := database.GetDB()
	var externalPlugins []database.PluginDefinition
	err := db.Where("plugin_type = ?", "external").Order("name").Find(&externalPlugins).Error
	if err != nil {
		logging.Error("Failed to fetch external plugins for admin", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch external plugins"})
		return
	}

	// Count instances for each plugin
	var result []gin.H
	for _, plugin := range externalPlugins {
		var instanceCount int64
		db.Model(&database.PluginInstance{}).Where("plugin_definition_id = ?", plugin.ID).Count(&instanceCount)
		
		result = append(result, gin.H{
			"id":                plugin.ID,
			"identifier":        plugin.Identifier,
			"name":              plugin.Name,
			"description":       plugin.Description,
			"version":           plugin.Version,
			"author":            plugin.Author,
			"status":            plugin.Status,
			"is_active":         plugin.IsActive,
			"instance_count":    instanceCount,
			"created_at":        plugin.CreatedAt,
			"updated_at":        plugin.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"plugins": result})
}

// AdminDeleteExternalPluginHandler deletes an external plugin definition and all its instances
func AdminDeleteExternalPluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	pluginID := c.Param("id")
	if pluginID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plugin ID required"})
		return
	}

	db := database.GetDB()
	
	// Verify this is an external plugin
	var plugin database.PluginDefinition
	err := db.Where("id = ? AND plugin_type = ?", pluginID, "external").First(&plugin).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "External plugin not found"})
		return
	}

	// Use the unified plugin service to delete (handles cascading deletes)
	pluginService := database.NewUnifiedPluginService(db)
	err = pluginService.DeletePluginDefinition(pluginID, nil)
	if err != nil {
		logging.Error("Failed to delete external plugin", "plugin_id", pluginID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete plugin"})
		return
	}

	logging.Info("Admin deleted external plugin", "plugin_id", pluginID, "plugin_name", plugin.Name, "admin_user", user.Username)
	c.JSON(http.StatusOK, gin.H{"message": "Plugin deleted successfully"})
}