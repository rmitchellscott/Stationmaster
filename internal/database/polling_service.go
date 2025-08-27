package database

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// PollingDataService handles database operations for polling data
type PollingDataService struct {
	db *gorm.DB
}

// NewPollingDataService creates a new polling service
func NewPollingDataService(db *gorm.DB) *PollingDataService {
	return &PollingDataService{db: db}
}

// StorePollingData stores polling data results
func (s *PollingDataService) StorePollingData(data *PrivatePluginPollingData) error {
	// Validate merged data is valid JSON before storing
	var testParse interface{}
	if err := json.Unmarshal(data.MergedData, &testParse); err != nil {
		return fmt.Errorf("invalid merged data JSON: %w", err)
	}

	// UPSERT: Update existing record or create new one (single record per plugin instance)
	result := s.db.Where("plugin_instance_id = ?", data.PluginInstanceID).
		Assign(map[string]interface{}{
			"merged_data":    data.MergedData,
			"raw_data":       data.RawData,
			"polled_at":      data.PolledAt,
			"poll_duration":  data.PollDuration,
			"success":        data.Success,
			"errors":         data.Errors,
			"url_count":      data.URLCount,
		}).
		FirstOrCreate(&PrivatePluginPollingData{
			ID:               data.ID,
			PluginInstanceID: data.PluginInstanceID,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to store polling data: %w", result.Error)
	}

	return nil
}

// GetLatestPollingData retrieves the latest polling data for a plugin instance
func (s *PollingDataService) GetLatestPollingData(pluginInstanceID string) (*PrivatePluginPollingData, error) {
	var pollingData PrivatePluginPollingData
	
	if err := s.db.Where("plugin_instance_id = ?", pluginInstanceID).Order("polled_at DESC").First(&pollingData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No polling data found
		}
		return nil, fmt.Errorf("failed to get polling data: %w", err)
	}
	
	return &pollingData, nil
}

// GetPollingDataTemplate extracts template-ready data from polling data
func (s *PollingDataService) GetPollingDataTemplate(pluginInstanceID string) (map[string]interface{}, error) {
	pollingData, err := s.GetLatestPollingData(pluginInstanceID)
	if err != nil {
		return nil, err
	}
	
	if pollingData == nil {
		return make(map[string]interface{}), nil
	}
	
	// Parse the merged data JSON
	var templateData map[string]interface{}
	if err := json.Unmarshal(pollingData.MergedData, &templateData); err != nil {
		return nil, fmt.Errorf("failed to parse polling template data: %w", err)
	}
	
	return templateData, nil
}

// IsPollingDataFresh checks if polling data is fresh enough (within the given duration)
func (s *PollingDataService) IsPollingDataFresh(pluginInstanceID string, maxAge time.Duration) (bool, error) {
	pollingData, err := s.GetLatestPollingData(pluginInstanceID)
	if err != nil {
		return false, err
	}
	
	if pollingData == nil {
		return false, nil // No data means not fresh
	}
	
	return time.Since(pollingData.PolledAt) <= maxAge, nil
}

// DeletePollingData deletes polling data for a plugin instance
func (s *PollingDataService) DeletePollingData(pluginInstanceID string) error {
	result := s.db.Where("plugin_instance_id = ?", pluginInstanceID).Delete(&PrivatePluginPollingData{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete polling data: %w", result.Error)
	}
	return nil
}