package pollers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// ModelPoller polls for device model updates
type ModelPoller struct {
	*BasePoller
	db     *gorm.DB
	apiURL string
}

// ModelAPIResponse represents the response from the model API
type ModelAPIResponse struct {
	Data []DeviceModelInfo `json:"data"`
}

// DeviceModelInfo represents device model information from the API
type DeviceModelInfo struct {
	Name        string  `json:"name"`  // Maps to model_name
	Label       string  `json:"label"` // Maps to display_name
	Description string  `json:"description"`
	Width       float64 `json:"width"`  // Maps to screen_width
	Height      float64 `json:"height"` // Maps to screen_height
	Colors      float64 `json:"colors"` // Maps to color_depth
	BitDepth    float64 `json:"bit_depth"`
	ScaleFactor float64 `json:"scale_factor"`
	Rotation    float64 `json:"rotation"`
	MimeType    string  `json:"mime_type"`
	OffsetX     float64 `json:"offset_x"`
	OffsetY     float64 `json:"offset_y"`
	PublishedAt string  `json:"published_at"`
}

// NewModelPoller creates a new model poller
func NewModelPoller(db *gorm.DB) *ModelPoller {
	// Get configuration from environment variables
	interval := 24 * time.Hour // Default 24 hours
	if envInterval := config.Get("MODEL_POLLER_INTERVAL", ""); envInterval != "" {
		if d, err := time.ParseDuration(envInterval); err == nil {
			interval = d
		}
	}

	enabled := config.Get("MODEL_POLLER", "true") != "false"
	apiURL := config.Get("TRMNL_MODEL_API_URL", "https://usetrmnl.com/api/models")

	config := PollerConfig{
		Name:       "model",
		Interval:   interval,
		Enabled:    enabled,
		MaxRetries: 3,
		RetryDelay: 30 * time.Second,
		Timeout:    60 * time.Second,
	}

	poller := &ModelPoller{
		db:     db,
		apiURL: apiURL,
	}

	poller.BasePoller = NewBasePoller(config, poller.poll)
	return poller
}

// poll performs the model polling operation
func (p *ModelPoller) poll(ctx context.Context) error {
	logging.InfoWithComponent(logging.ComponentModelPoller, "Starting model information sync")

	// Fetch model information from API
	models, err := p.fetchDeviceModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch device models: %w", err)
	}

	logging.InfoWithComponent(logging.ComponentModelPoller, "Found device models", "count", len(models))

	// Process each model
	processed := 0
	for _, modelInfo := range models {
		if err := p.processDeviceModel(ctx, modelInfo); err != nil {
			logging.ErrorWithComponent(logging.ComponentModelPoller, "Error processing model", "model", modelInfo.Name, "error", err)
			continue
		}
		processed++
	}

	logging.InfoWithComponent(logging.ComponentModelPoller, "Model sync completed", "processed", processed, "total", len(models))
	return nil
}

// fetchDeviceModels fetches device model information from the API
func (p *ModelPoller) fetchDeviceModels(ctx context.Context) ([]DeviceModelInfo, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResponse ModelAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	return apiResponse.Data, nil
}

// processDeviceModel processes a single device model
func (p *ModelPoller) processDeviceModel(ctx context.Context, modelInfo DeviceModelInfo) error {
	// Map API fields to our database model with reasonable defaults
	modelName := modelInfo.Name
	displayName := modelInfo.Label
	if displayName == "" {
		displayName = modelName
	}

	// Set reasonable defaults for fields not provided by API
	hasWiFi := true    // Most modern devices have WiFi
	hasBattery := true // Most TRMNL devices are battery powered
	hasButtons := 0    // Default to no buttons unless specified
	isActive := true   // Default to active
	minFirmware := ""  // No minimum firmware requirement by default

	// Create default capabilities based on device characteristics
	capabilities := []string{"display", "wireless"}
	if int(modelInfo.Colors) > 1 {
		capabilities = append(capabilities, "color")
	}

	// Convert capabilities to JSON string
	capabilitiesJSON := ""
	if capBytes, err := json.Marshal(capabilities); err == nil {
		capabilitiesJSON = string(capBytes)
	}

	// Check if this exact model version already exists (not deleted)
	var existingModel database.DeviceModel
	err := p.db.Where("model_name = ? AND deleted_at IS NULL", modelName).
		Where("display_name = ?", displayName).
		Where("description = ?", modelInfo.Description).
		Where("screen_width = ?", int(modelInfo.Width)).
		Where("screen_height = ?", int(modelInfo.Height)).
		Where("color_depth = ?", int(modelInfo.Colors)).
		Where("bit_depth = ?", int(modelInfo.BitDepth)).
		First(&existingModel).Error

	now := time.Now().UTC()
	
	if err == nil {
		// Exact model exists, just update last seen time
		existingModel.ApiLastSeenAt = &now
		if err := p.db.Save(&existingModel).Error; err != nil {
			return fmt.Errorf("failed to update last seen time: %w", err)
		}
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error: %w", err)
	}

	// Model with these exact specs doesn't exist, create new version
	deviceModel := database.DeviceModel{
		ModelName:     modelName,
		DisplayName:   displayName,
		Description:   modelInfo.Description,
		ScreenWidth:   int(modelInfo.Width),
		ScreenHeight:  int(modelInfo.Height),
		ColorDepth:    int(modelInfo.Colors),
		BitDepth:      int(modelInfo.BitDepth),
		HasWiFi:       hasWiFi,
		HasBattery:    hasBattery,
		HasButtons:    hasButtons,
		Capabilities:  capabilitiesJSON,
		MinFirmware:   minFirmware,
		IsActive:      isActive,
		ApiLastSeenAt: &now,
	}

	if err := p.db.Create(&deviceModel).Error; err != nil {
		return fmt.Errorf("failed to create device model: %w", err)
	}

	logging.Info("[MODEL POLLER] Added new device model version", "name", modelName, "display_name", displayName)
	return nil
}

// GetCapabilities returns the capabilities as a string slice
func (p *ModelPoller) GetCapabilities(model *database.DeviceModel) ([]string, error) {
	if model.Capabilities == "" {
		return []string{}, nil
	}

	var capabilities []string
	if err := json.Unmarshal([]byte(model.Capabilities), &capabilities); err != nil {
		return nil, err
	}

	return capabilities, nil
}

// SetCapabilities sets the capabilities from a string slice
func (p *ModelPoller) SetCapabilities(model *database.DeviceModel, capabilities []string) error {
	capBytes, err := json.Marshal(capabilities)
	if err != nil {
		return err
	}

	model.Capabilities = string(capBytes)
	return nil
}
