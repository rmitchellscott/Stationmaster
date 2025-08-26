package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PrivatePluginService handles database operations for private plugins
type PrivatePluginService struct {
	db *gorm.DB
}

// NewPrivatePluginService creates a new private plugin service
func NewPrivatePluginService(db *gorm.DB) *PrivatePluginService {
	return &PrivatePluginService{db: db}
}

// CreatePrivatePlugin creates a new private plugin for a user
func (s *PrivatePluginService) CreatePrivatePlugin(userID uuid.UUID, plugin *PrivatePlugin) error {
	// Set the user ID
	plugin.UserID = userID
	
	// Create the plugin in the database
	if err := s.db.Create(plugin).Error; err != nil {
		return fmt.Errorf("failed to create private plugin: %w", err)
	}
	
	return nil
}

// GetPrivatePluginByID retrieves a private plugin by ID, ensuring user ownership
func (s *PrivatePluginService) GetPrivatePluginByID(id uuid.UUID, userID uuid.UUID) (*PrivatePlugin, error) {
	var plugin PrivatePlugin
	
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&plugin).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("private plugin not found")
		}
		return nil, fmt.Errorf("failed to get private plugin: %w", err)
	}
	
	return &plugin, nil
}

// GetPrivatePluginsByUserID retrieves all private plugins for a user
func (s *PrivatePluginService) GetPrivatePluginsByUserID(userID uuid.UUID) ([]PrivatePlugin, error) {
	var plugins []PrivatePlugin
	
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&plugins).Error; err != nil {
		return nil, fmt.Errorf("failed to get private plugins: %w", err)
	}
	
	return plugins, nil
}

// UpdatePrivatePlugin updates an existing private plugin
func (s *PrivatePluginService) UpdatePrivatePlugin(id uuid.UUID, userID uuid.UUID, updates *PrivatePlugin) error {
	// First verify the plugin exists and belongs to the user
	var existingPlugin PrivatePlugin
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&existingPlugin).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("private plugin not found")
		}
		return fmt.Errorf("failed to find private plugin: %w", err)
	}
	
	// Preserve certain fields that shouldn't be updated
	updates.ID = id
	updates.UserID = userID
	
	// Update the plugin
	if err := s.db.Save(updates).Error; err != nil {
		return fmt.Errorf("failed to update private plugin: %w", err)
	}
	
	return nil
}

// DeletePrivatePlugin deletes a private plugin
func (s *PrivatePluginService) DeletePrivatePlugin(id uuid.UUID, userID uuid.UUID) error {
	// Verify ownership and delete in one operation
	result := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&PrivatePlugin{})
	
	if result.Error != nil {
		return fmt.Errorf("failed to delete private plugin: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("private plugin not found or access denied")
	}
	
	return nil
}



// GetPublishedPrivatePlugins retrieves all published private plugins (for recipe marketplace)
func (s *PrivatePluginService) GetPublishedPrivatePlugins() ([]PrivatePlugin, error) {
	var plugins []PrivatePlugin
	
	if err := s.db.Where("is_published = ?", true).Order("created_at DESC").Find(&plugins).Error; err != nil {
		return nil, fmt.Errorf("failed to get published private plugins: %w", err)
	}
	
	return plugins, nil
}

// GetPrivatePluginStats returns statistics about private plugins
func (s *PrivatePluginService) GetPrivatePluginStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Total count
	var totalCount int64
	if err := s.db.Model(&PrivatePlugin{}).Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count private plugins: %w", err)
	}
	stats["total"] = totalCount
	
	// Published count
	var publishedCount int64
	if err := s.db.Model(&PrivatePlugin{}).Where("is_published = ?", true).Count(&publishedCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count published private plugins: %w", err)
	}
	stats["published"] = publishedCount
	
	// Count by data strategy
	var webhookCount, pollingCount, mergeCount int64
	
	s.db.Model(&PrivatePlugin{}).Where("data_strategy = ?", "webhook").Count(&webhookCount)
	s.db.Model(&PrivatePlugin{}).Where("data_strategy = ?", "polling").Count(&pollingCount)
	s.db.Model(&PrivatePlugin{}).Where("data_strategy = ?", "merge").Count(&mergeCount)
	
	stats["by_strategy"] = map[string]int64{
		"webhook": webhookCount,
		"polling": pollingCount,
		"merge":   mergeCount,
	}
	
	return stats, nil
}

// generateWebhookToken generates a cryptographically secure webhook token
func (s *PrivatePluginService) generateWebhookToken() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}