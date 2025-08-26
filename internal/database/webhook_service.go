package database

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// WebhookService handles database operations for webhook data
type WebhookService struct {
	db *gorm.DB
}

// NewWebhookService creates a new webhook service
func NewWebhookService(db *gorm.DB) *WebhookService {
	return &WebhookService{db: db}
}

// StoreWebhookData stores webhook data with the specified merge strategy
func (s *WebhookService) StoreWebhookData(data *PrivatePluginWebhookData) error {
	// Process merge strategy
	mergedData, err := s.processMergeStrategy(data.PluginInstanceID, data.RawData, data.MergeStrategy)
	if err != nil {
		return fmt.Errorf("failed to process merge strategy: %w", err)
	}

	// Validate merged data is valid JSON before storing
	var testParse interface{}
	if err := json.Unmarshal(mergedData, &testParse); err != nil {
		return fmt.Errorf("invalid merged data JSON: %w", err)
	}

	// Store the merged data
	data.MergedData = mergedData

	// UPSERT: Update existing record or create new one (single record per plugin instance)
	result := s.db.Where("plugin_instance_id = ?", data.PluginInstanceID).
		Assign(map[string]interface{}{
			"merged_data":         data.MergedData,
			"raw_data":            data.RawData,
			"merge_strategy":      data.MergeStrategy,
			"received_at":         data.ReceivedAt,
			"content_type":        data.ContentType,
			"content_size":        data.ContentSize,
			"source_ip":           data.SourceIP,
		}).
		FirstOrCreate(&PrivatePluginWebhookData{
			ID:                 data.ID,
			PluginInstanceID:   data.PluginInstanceID,
			MergedData:         data.MergedData,
			RawData:            data.RawData,
			MergeStrategy:      data.MergeStrategy,
			ReceivedAt:         data.ReceivedAt,
			ContentType:        data.ContentType,
			ContentSize:        data.ContentSize,
			SourceIP:           data.SourceIP,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to store webhook data: %w", result.Error)
	}

	return nil
}

// GetLatestWebhookData retrieves the webhook data for a plugin instance (single record per instance)
func (s *WebhookService) GetLatestWebhookData(pluginInstanceID string) (*PrivatePluginWebhookData, error) {
	var webhookData PrivatePluginWebhookData
	
	if err := s.db.Where("plugin_instance_id = ?", pluginInstanceID).First(&webhookData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No webhook data found
		}
		return nil, fmt.Errorf("failed to get webhook data: %w", err)
	}
	
	return &webhookData, nil
}

// GetWebhookDataTemplate extracts template-ready data from webhook data
func (s *WebhookService) GetWebhookDataTemplate(pluginInstanceID string) (map[string]interface{}, error) {
	webhookData, err := s.GetLatestWebhookData(pluginInstanceID)
	if err != nil {
		return nil, err
	}
	
	if webhookData == nil {
		return make(map[string]interface{}), nil
	}
	
	// Parse the merged data JSON
	var templateData map[string]interface{}
	if err := json.Unmarshal(webhookData.MergedData, &templateData); err != nil {
		return nil, fmt.Errorf("failed to parse merged data: %w", err)
	}
	
	return templateData, nil
}


// processMergeStrategy applies the specified merge strategy to the webhook data
func (s *WebhookService) processMergeStrategy(pluginInstanceID string, rawDataJSON []byte, strategy string) ([]byte, error) {
	// Parse the raw webhook data
	var rawData map[string]interface{}
	if err := json.Unmarshal(rawDataJSON, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse raw webhook data: %w", err)
	}
	
	// Extract merge_variables from the webhook payload
	mergeVariables, ok := rawData["merge_variables"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("webhook payload missing merge_variables object")
	}
	
	switch strategy {
	case "default", "":
		// Default strategy: completely replace existing data
		mergedData, err := json.Marshal(mergeVariables)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal merge variables: %w", err)
		}
		return mergedData, nil
		
	case "deep_merge":
		// Deep merge strategy: recursively merge with existing data
		return s.processDeepMerge(pluginInstanceID, mergeVariables)
		
	case "stream":
		// Stream strategy: accumulate values in arrays
		streamLimit := 10 // Default stream limit
		if limit, ok := rawData["stream_limit"].(float64); ok {
			streamLimit = int(limit)
		}
		return s.processStream(pluginInstanceID, mergeVariables, streamLimit)
		
	default:
		return nil, fmt.Errorf("unsupported merge strategy: %s", strategy)
	}
}

// processDeepMerge recursively merges new data with existing data
func (s *WebhookService) processDeepMerge(pluginInstanceID string, newData map[string]interface{}) ([]byte, error) {
	// Get existing data
	existingData, err := s.GetWebhookDataTemplate(pluginInstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing data: %w", err)
	}
	
	// Perform deep merge
	merged := s.deepMerge(existingData, newData)
	
	// Marshal the merged data
	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged data: %w", err)
	}
	
	return mergedJSON, nil
}

// processStream accumulates values in arrays with the specified limit
func (s *WebhookService) processStream(pluginInstanceID string, newData map[string]interface{}, streamLimit int) ([]byte, error) {
	// Get existing data
	existingData, err := s.GetWebhookDataTemplate(pluginInstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing data: %w", err)
	}
	
	// Process streaming for each field
	for key, newValue := range newData {
		if existingValue, exists := existingData[key]; exists {
			// Convert existing value to array if it isn't already
			var existingArray []interface{}
			if arr, ok := existingValue.([]interface{}); ok {
				existingArray = arr
			} else {
				existingArray = []interface{}{existingValue}
			}
			
			// Append new value(s)
			if newArray, ok := newValue.([]interface{}); ok {
				// New value is an array, append all elements
				existingArray = append(existingArray, newArray...)
			} else {
				// New value is a single item
				existingArray = append(existingArray, newValue)
			}
			
			// Apply stream limit
			if len(existingArray) > streamLimit {
				existingArray = existingArray[len(existingArray)-streamLimit:]
			}
			
			existingData[key] = existingArray
		} else {
			// New field, convert to array if needed
			if newArray, ok := newValue.([]interface{}); ok {
				// Apply stream limit to new array
				if len(newArray) > streamLimit {
					newArray = newArray[len(newArray)-streamLimit:]
				}
				existingData[key] = newArray
			} else {
				// Single value becomes array
				existingData[key] = []interface{}{newValue}
			}
		}
	}
	
	// Marshal the result
	streamedJSON, err := json.Marshal(existingData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal streamed data: %w", err)
	}
	
	return streamedJSON, nil
}

// deepMerge recursively merges two maps
func (s *WebhookService) deepMerge(existing, new map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy existing data
	for k, v := range existing {
		result[k] = v
	}
	
	// Merge new data
	for k, newVal := range new {
		if existingVal, exists := result[k]; exists {
			// Both values exist, check if we can merge recursively
			existingMap, existingIsMap := existingVal.(map[string]interface{})
			newMap, newIsMap := newVal.(map[string]interface{})
			
			if existingIsMap && newIsMap {
				// Both are maps, merge recursively
				result[k] = s.deepMerge(existingMap, newMap)
			} else {
				// Different types or not maps, replace
				result[k] = newVal
			}
		} else {
			// New key, just add it
			result[k] = newVal
		}
	}
	
	return result
}