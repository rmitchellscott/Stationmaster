package database

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UnifiedPluginService handles database operations for the unified plugin system
type UnifiedPluginService struct {
	db *gorm.DB
}

// NewUnifiedPluginService creates a new unified plugin service
func NewUnifiedPluginService(db *gorm.DB) *UnifiedPluginService {
	return &UnifiedPluginService{db: db}
}

// PluginDefinition Operations

// CreatePluginDefinition creates a new plugin definition
func (s *UnifiedPluginService) CreatePluginDefinition(definition *PluginDefinition) error {
	return s.db.Create(definition).Error
}

// GetPluginDefinitionByID retrieves a plugin definition by ID
func (s *UnifiedPluginService) GetPluginDefinitionByID(id uuid.UUID) (*PluginDefinition, error) {
	var definition PluginDefinition
	err := s.db.Preload("Owner").First(&definition, "id = ? AND is_active = ?", id, true).Error
	if err != nil {
		return nil, err
	}
	return &definition, nil
}

// GetPluginDefinitionsByType retrieves plugin definitions by type
func (s *UnifiedPluginService) GetPluginDefinitionsByType(pluginType string) ([]PluginDefinition, error) {
	var definitions []PluginDefinition
	err := s.db.Preload("Owner").
		Where("plugin_type = ? AND is_active = ?", pluginType, true).
		Order("name").
		Find(&definitions).Error
	return definitions, err
}

// GetPluginDefinitionsByOwner retrieves plugin definitions owned by a user
func (s *UnifiedPluginService) GetPluginDefinitionsByOwner(ownerID uuid.UUID) ([]PluginDefinition, error) {
	var definitions []PluginDefinition
	err := s.db.Where("owner_id = ? AND is_active = ?", ownerID, true).
		Order("name").
		Find(&definitions).Error
	return definitions, err
}

// GetAllPluginDefinitions retrieves all plugin definitions
func (s *UnifiedPluginService) GetAllPluginDefinitions() ([]PluginDefinition, error) {
	var definitions []PluginDefinition
	err := s.db.Preload("Owner").
		Where("is_active = ?", true).
		Order("plugin_type, name").
		Find(&definitions).Error
	return definitions, err
}

// UpdatePluginDefinition updates an existing plugin definition
func (s *UnifiedPluginService) UpdatePluginDefinition(definition *PluginDefinition) error {
	return s.db.Save(definition).Error
}

// DeletePluginDefinition soft deletes a plugin definition and cascades to all instances
func (s *UnifiedPluginService) DeletePluginDefinition(id uuid.UUID, ownerID *uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// First verify the plugin definition exists and check ownership
		var definition PluginDefinition
		query := tx.Where("id = ? AND is_active = ?", id, true)
		
		// If ownerID is provided, ensure ownership (for private plugins)
		if ownerID != nil {
			query = query.Where("owner_id = ?", *ownerID)
		}
		
		result := query.First(&definition)
		if result.Error != nil {
			return fmt.Errorf("plugin definition not found or access denied: %w", result.Error)
		}
		
		// Find all active plugin instances for this definition
		var instances []PluginInstance
		if err := tx.Where("plugin_definition_id = ? AND is_active = ?", id, true).Find(&instances).Error; err != nil {
			return fmt.Errorf("failed to find plugin instances: %w", err)
		}
		
		// Delete each plugin instance (which handles cascading to render_queues, etc.)
		for _, instance := range instances {
			// Cancel any active render jobs for this plugin instance
			var activeJobs []RenderQueue
			if err := tx.Where("plugin_instance_id = ? AND status IN (?)", instance.ID, []string{"pending", "processing"}).Find(&activeJobs).Error; err != nil {
				return fmt.Errorf("failed to find active render jobs for instance %s: %w", instance.ID, err)
			}
			
			if len(activeJobs) > 0 {
				// Cancel active jobs
				if err := tx.Model(&RenderQueue{}).
					Where("plugin_instance_id = ? AND status IN (?)", instance.ID, []string{"pending", "processing"}).
					Update("status", "cancelled").Error; err != nil {
					return fmt.Errorf("failed to cancel active render jobs for instance %s: %w", instance.ID, err)
				}
			}
			
			// Get playlist item IDs that will be deleted
			var playlistItemIDs []uuid.UUID
			if err := tx.Model(&PlaylistItem{}).Where("plugin_instance_id = ?", instance.ID).Pluck("id", &playlistItemIDs).Error; err != nil {
				return fmt.Errorf("failed to get playlist item IDs for instance %s: %w", instance.ID, err)
			}
			
			// Advance devices that reference these playlist items to their next valid item
			if err := s.advanceDevicesFromPlaylistItems(tx, playlistItemIDs); err != nil {
				return fmt.Errorf("failed to advance devices from deleted items for instance %s: %w", instance.ID, err)
			}
			
			// Delete playlist items that reference this plugin instance
			if err := tx.Where("plugin_instance_id = ?", instance.ID).Delete(&PlaylistItem{}).Error; err != nil {
				return fmt.Errorf("failed to delete playlist items for instance %s: %w", instance.ID, err)
			}
			
			// Delete all render queue entries (including cancelled ones)
			if err := tx.Where("plugin_instance_id = ?", instance.ID).Delete(&RenderQueue{}).Error; err != nil {
				return fmt.Errorf("failed to delete render queue entries for instance %s: %w", instance.ID, err)
			}
			
			// Delete rendered content records
			if err := tx.Where("plugin_instance_id = ?", instance.ID).Delete(&RenderedContent{}).Error; err != nil {
				return fmt.Errorf("failed to delete rendered content for instance %s: %w", instance.ID, err)
			}
			
			// Finally hard delete the plugin instance
			if err := tx.Delete(&instance).Error; err != nil {
				return fmt.Errorf("failed to delete plugin instance %s: %w", instance.ID, err)
			}
		}
		
		// Finally, hard delete the plugin definition
		if err := tx.Delete(&definition).Error; err != nil {
			return fmt.Errorf("failed to delete plugin definition: %w", err)
		}
		
		return nil
	})
}

// PluginInstance Operations

// CreatePluginInstance creates a new plugin instance
func (s *UnifiedPluginService) CreatePluginInstance(userID, definitionID uuid.UUID, name string, settings map[string]interface{}, refreshInterval int) (*PluginInstance, error) {
	// Verify the plugin definition exists
	definition, err := s.GetPluginDefinitionByID(definitionID)
	if err != nil {
		return nil, fmt.Errorf("plugin definition not found: %w", err)
	}
	
	// For private plugins, ensure the user owns the definition
	if definition.PluginType == "private" && (definition.OwnerID == nil || *definition.OwnerID != userID) {
		return nil, fmt.Errorf("user does not own this private plugin definition")
	}
	
	// Convert settings to JSON
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}
	
	instance := &PluginInstance{
		UserID:             userID,
		PluginDefinitionID: definitionID,
		Name:               name,
		Settings:           settingsJSON,
		RefreshInterval:    refreshInterval,
		IsActive:           true,
		LastSchemaVersion:  definition.SchemaVersion, // Set to current schema version
		NeedsConfigUpdate:  false, // New instances are always up to date
	}
	
	if err := s.db.Create(instance).Error; err != nil {
		return nil, err
	}
	
	return instance, nil
}

// GetPluginInstanceByID retrieves a plugin instance by ID
func (s *UnifiedPluginService) GetPluginInstanceByID(instanceID uuid.UUID) (*PluginInstance, error) {
	var instance PluginInstance
	err := s.db.Preload("PluginDefinition").
		Preload("PluginDefinition.Owner").
		First(&instance, "id = ? AND is_active = ?", instanceID, true).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetPluginInstancesByUser retrieves all plugin instances for a user
func (s *UnifiedPluginService) GetPluginInstancesByUser(userID uuid.UUID) ([]PluginInstance, error) {
	var instances []PluginInstance
	err := s.db.Preload("PluginDefinition").
		Preload("PluginDefinition.Owner").
		Preload("PlaylistItems").
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("created_at DESC").
		Find(&instances).Error
	return instances, err
}

// GetPluginInstancesByDefinition retrieves all instances of a specific plugin definition
func (s *UnifiedPluginService) GetPluginInstancesByDefinition(definitionID uuid.UUID) ([]PluginInstance, error) {
	var instances []PluginInstance
	err := s.db.Preload("User").
		Where("plugin_definition_id = ? AND is_active = ?", definitionID, true).
		Order("created_at DESC").
		Find(&instances).Error
	return instances, err
}

// UpdatePluginInstance updates an existing plugin instance
func (s *UnifiedPluginService) UpdatePluginInstance(instance *PluginInstance) error {
	return s.db.Save(instance).Error
}

// UpdatePluginInstanceSettings updates just the settings of a plugin instance
func (s *UnifiedPluginService) UpdatePluginInstanceSettings(instanceID uuid.UUID, settings map[string]interface{}) error {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	
	return s.db.Model(&PluginInstance{}).
		Where("id = ?", instanceID).
		Update("settings", settingsJSON).Error
}

// advanceDevicesFromPlaylistItems advances devices that reference the given playlist items to their next valid item
func (s *UnifiedPluginService) advanceDevicesFromPlaylistItems(tx *gorm.DB, playlistItemIDs []uuid.UUID) error {
	if len(playlistItemIDs) == 0 {
		return nil
	}
	
	// Find devices that reference any of the playlist items being deleted
	var affectedDevices []Device
	if err := tx.Where("last_playlist_item_id IN (?)", playlistItemIDs).Find(&affectedDevices).Error; err != nil {
		return fmt.Errorf("failed to find affected devices: %w", err)
	}
	
	if len(affectedDevices) == 0 {
		return nil // No devices affected
	}
	
	// Process each affected device
	for _, device := range affectedDevices {
		// Get all active playlist items for this device (excluding those being deleted)
		var activeItems []PlaylistItem
		err := tx.Table("playlist_items pi").
			Select("pi.*").
			Joins("JOIN playlists p ON pi.playlist_id = p.id").
			Where("p.device_id = ? AND p.is_default = ? AND pi.is_visible = ? AND pi.id NOT IN (?)", 
				device.ID, true, true, playlistItemIDs).
			Order("pi.order_index ASC").
			Find(&activeItems).Error
		
		if err != nil {
			return fmt.Errorf("failed to get active items for device %s: %w", device.ID, err)
		}
		
		var nextItemID *uuid.UUID
		if len(activeItems) > 0 {
			// Find current item in remaining active items
			var currentItem *PlaylistItem
			if device.LastPlaylistItemID != nil {
				for i := range activeItems {
					if activeItems[i].ID == *device.LastPlaylistItemID {
						currentItem = &activeItems[i]
						break
					}
				}
			}
			
			// Find next item using round-robin logic
			if currentItem != nil {
				// Find next item after current
				for i, item := range activeItems {
					if item.ID == currentItem.ID {
						nextIndex := (i + 1) % len(activeItems)
						nextItemID = &activeItems[nextIndex].ID
						break
					}
				}
			} else {
				// No current item or current item being deleted, use first available
				nextItemID = &activeItems[0].ID
			}
		}
		// If no active items remain, nextItemID stays nil (will clear the reference)
		
		// Update device's last_playlist_item_id
		if err := tx.Model(&Device{}).Where("id = ?", device.ID).Update("last_playlist_item_id", nextItemID).Error; err != nil {
			return fmt.Errorf("failed to update device %s playlist reference: %w", device.ID, err)
		}
	}
	
	return nil
}

// DeletePluginInstance permanently deletes a plugin instance and its references
func (s *UnifiedPluginService) DeletePluginInstance(instanceID, userID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Verify ownership first
		var instance PluginInstance
		result := tx.Where("id = ? AND user_id = ?", instanceID, userID).First(&instance)
		if result.Error != nil {
			return fmt.Errorf("plugin instance not found or access denied: %w", result.Error)
		}
		
		// Cancel any active render jobs for this plugin instance
		var activeJobs []RenderQueue
		if err := tx.Where("plugin_instance_id = ? AND status IN (?)", instanceID, []string{"pending", "processing"}).Find(&activeJobs).Error; err != nil {
			return fmt.Errorf("failed to find active render jobs: %w", err)
		}
		
		if len(activeJobs) > 0 {
			// Cancel active jobs
			if err := tx.Model(&RenderQueue{}).
				Where("plugin_instance_id = ? AND status IN (?)", instanceID, []string{"pending", "processing"}).
				Update("status", "cancelled").Error; err != nil {
				return fmt.Errorf("failed to cancel active render jobs: %w", err)
			}
		}
		
		// Get playlist item IDs that will be deleted
		var playlistItemIDs []uuid.UUID
		if err := tx.Model(&PlaylistItem{}).Where("plugin_instance_id = ?", instanceID).Pluck("id", &playlistItemIDs).Error; err != nil {
			return fmt.Errorf("failed to get playlist item IDs: %w", err)
		}
		
		// Advance devices that reference these playlist items to their next valid item
		if err := s.advanceDevicesFromPlaylistItems(tx, playlistItemIDs); err != nil {
			return fmt.Errorf("failed to advance devices from deleted items: %w", err)
		}
		
		// Delete playlist items that reference this plugin instance
		if err := tx.Where("plugin_instance_id = ?", instanceID).Delete(&PlaylistItem{}).Error; err != nil {
			return fmt.Errorf("failed to delete playlist items: %w", err)
		}
		
		// Delete all render queue entries (including cancelled ones)
		if err := tx.Where("plugin_instance_id = ?", instanceID).Delete(&RenderQueue{}).Error; err != nil {
			return fmt.Errorf("failed to delete render queue entries: %w", err)
		}
		
		// Delete rendered content records and track files for cleanup
		var renderedContent []RenderedContent
		if err := tx.Where("plugin_instance_id = ?", instanceID).Find(&renderedContent).Error; err != nil {
			return fmt.Errorf("failed to find rendered content: %w", err)
		}
		
		// Delete rendered content database records
		if err := tx.Where("plugin_instance_id = ?", instanceID).Delete(&RenderedContent{}).Error; err != nil {
			return fmt.Errorf("failed to delete rendered content: %w", err)
		}
		
		// Finally hard delete the plugin instance
		if err := tx.Delete(&instance).Error; err != nil {
			return fmt.Errorf("failed to delete plugin instance: %w", err)
		}
		
		// Note: File cleanup for rendered content should be handled outside the transaction
		// to avoid blocking the database transaction on filesystem operations
		
		return nil
	})
}

// GetPluginInstanceSettings returns the parsed settings for a plugin instance
func (s *UnifiedPluginService) GetPluginInstanceSettings(instanceID uuid.UUID) (map[string]interface{}, error) {
	instance, err := s.GetPluginInstanceByID(instanceID)
	if err != nil {
		return nil, err
	}
	
	var settings map[string]interface{}
	if instance.Settings != nil {
		if err := json.Unmarshal(instance.Settings, &settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
	} else {
		settings = make(map[string]interface{})
	}
	
	return settings, nil
}

// Migration and Utility Operations

// CreateSystemPluginDefinition creates a plugin definition for a system plugin
func (s *UnifiedPluginService) CreateSystemPluginDefinition(identifier, name, description, configSchema, version, author string, requiresProcessing bool) (*PluginDefinition, error) {
	definition := &PluginDefinition{
		PluginType:         "system",
		OwnerID:            nil, // System plugins have no owner
		Identifier:         identifier,
		Name:               name,
		Description:        description,
		ConfigSchema:       configSchema,
		Version:            version,
		Author:             author,
		RequiresProcessing: requiresProcessing,
		IsActive:           true,
	}
	
	// Use ON CONFLICT to handle existing system plugins
	err := s.db.FirstOrCreate(definition, PluginDefinition{
		PluginType: "system",
		Identifier: identifier,
	}).Error
	
	return definition, err
}


// Statistics and Analytics

// GetPluginStats returns statistics about plugin usage
func (s *UnifiedPluginService) GetPluginStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Plugin definitions by type
	var definitionStats []struct {
		PluginType string `json:"plugin_type"`
		Count      int64  `json:"count"`
	}
	
	err := s.db.Model(&PluginDefinition{}).
		Select("plugin_type, COUNT(*) as count").
		Where("is_active = ?", true).
		Group("plugin_type").
		Find(&definitionStats).Error
	if err != nil {
		return nil, err
	}
	
	stats["definitions_by_type"] = definitionStats
	
	// Plugin instances by definition type
	var instanceStats []struct {
		PluginType string `json:"plugin_type"`
		Count      int64  `json:"count"`
	}
	
	err = s.db.Table("plugin_instances pi").
		Select("pd.plugin_type, COUNT(*) as count").
		Joins("JOIN plugin_definitions pd ON pi.plugin_definition_id = pd.id").
		Where("pi.is_active = ? AND pd.is_active = ?", true, true).
		Group("pd.plugin_type").
		Find(&instanceStats).Error
	if err != nil {
		return nil, err
	}
	
	stats["instances_by_type"] = instanceStats
	
	// Total counts
	var totalDefinitions, totalInstances int64
	s.db.Model(&PluginDefinition{}).Where("is_active = ?", true).Count(&totalDefinitions)
	s.db.Model(&PluginInstance{}).Where("is_active = ?", true).Count(&totalInstances)
	
	stats["total_definitions"] = totalDefinitions
	stats["total_instances"] = totalInstances
	
	return stats, nil
}


// ClearRenderedContentForInstance deletes all rendered content for a specific plugin instance
func (s *UnifiedPluginService) ClearRenderedContentForInstance(instanceID uuid.UUID) error {
	return s.db.Where("plugin_instance_id = ?", instanceID).Delete(&RenderedContent{}).Error
}

// CleanupOrphanedData removes orphaned records that reference non-existent or inactive plugin instances
func (s *UnifiedPluginService) CleanupOrphanedData() error {
	var cleanupCount int64
	
	// Clean up orphaned render queue entries
	result := s.db.Exec(`
		DELETE FROM render_queues 
		WHERE plugin_instance_id NOT IN (
			SELECT id FROM plugin_instances WHERE is_active = true
		) OR plugin_instance_id IS NULL
	`)
	if result.Error != nil {
		return fmt.Errorf("failed to clean orphaned render queue entries: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		cleanupCount += result.RowsAffected
	}
	
	// Clean up orphaned rendered content
	result = s.db.Exec(`
		DELETE FROM rendered_contents 
		WHERE plugin_instance_id NOT IN (
			SELECT id FROM plugin_instances WHERE is_active = true
		) OR plugin_instance_id IS NULL
	`)
	if result.Error != nil {
		return fmt.Errorf("failed to clean orphaned rendered content: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		cleanupCount += result.RowsAffected
	}
	
	// Clean up orphaned playlist items
	result = s.db.Exec(`
		DELETE FROM playlist_items 
		WHERE plugin_instance_id NOT IN (
			SELECT id FROM plugin_instances WHERE is_active = true
		) OR plugin_instance_id IS NULL
	`)
	if result.Error != nil {
		return fmt.Errorf("failed to clean orphaned playlist items: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		cleanupCount += result.RowsAffected
	}
	
	if cleanupCount > 0 {
		// Log cleanup activity (using fmt since we don't have logging imported here)
		// The caller can log this information
	}
	
	return nil
}

