package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ExternalPluginData represents the structure returned by the external plugin service
type ExternalPluginData struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Version     string            `json:"version"`
	Templates   map[string]string `json:"templates"`     // Layout name -> Liquid template
	FormFields  json.RawMessage   `json:"form_fields"`   // JSON schema for form configuration
	OAuthConfig json.RawMessage   `json:"oauth_config"`  // OAuth provider configuration
	Enabled     bool              `json:"enabled"`
}

// PluginScannerService handles discovery and registration of external plugins
type PluginScannerService struct {
	db         *gorm.DB
	serviceURL string
	client     *http.Client
}

// NewPluginScannerService creates a new plugin scanner service
func NewPluginScannerService(db *gorm.DB) *PluginScannerService {
	serviceURL := os.Getenv("EXTERNAL_PLUGIN_SERVICES")
	if serviceURL == "" {
		serviceURL = "http://stationmaster-plugins:3000"
	}

	return &PluginScannerService{
		db:         db,
		serviceURL: serviceURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ScanAndRegisterPlugins discovers plugins from external services and registers them in the database
func (s *PluginScannerService) ScanAndRegisterPlugins(ctx context.Context) error {
	logging.InfoWithComponent(logging.ComponentPlugins, "Starting external plugin discovery", "service_url", s.serviceURL)

	// Check if service is available first
	if !s.IsServiceAvailable(ctx) {
		// Mark all external plugins as unavailable if service is down
		err := s.markAllExternalPluginsUnavailable()
		if err != nil {
			logging.WarnWithComponent(logging.ComponentPlugins, "Failed to mark external plugins as unavailable", "error", err)
		}
		return fmt.Errorf("external plugin service is unavailable")
	}

	// Fetch plugin metadata from external service
	plugins, err := s.fetchPluginMetadata(ctx)
	if err != nil {
		// Mark all external plugins as unavailable if fetch fails
		if markErr := s.markAllExternalPluginsUnavailable(); markErr != nil {
			logging.WarnWithComponent(logging.ComponentPlugins, "Failed to mark external plugins as unavailable", "error", markErr)
		}
		return fmt.Errorf("failed to fetch plugin metadata: %w", err)
	}

	// Get existing external plugins to check which ones are missing
	existingPlugins, err := s.getExistingExternalPlugins()
	if err != nil {
		logging.WarnWithComponent(logging.ComponentPlugins, "Failed to get existing external plugins", "error", err)
	}

	// Mark plugins found in service as available and register/update them
	foundPlugins := make(map[string]bool)
	for identifier, pluginData := range plugins {
		foundPlugins[identifier] = true
		if err := s.registerPlugin(identifier, pluginData, "available"); err != nil {
			logging.WarnWithComponent(logging.ComponentPlugins, "Failed to register external plugin", 
				"plugin", identifier, "error", err)
			continue
		}
		
		logging.InfoWithComponent(logging.ComponentPlugins, "Registered external plugin", 
			"plugin", identifier, "version", pluginData.Version)
	}

	// Mark plugins not found in service as unavailable
	for _, existingPlugin := range existingPlugins {
		if !foundPlugins[existingPlugin.Identifier] {
			if err := s.markPluginUnavailable(existingPlugin.Identifier); err != nil {
				logging.WarnWithComponent(logging.ComponentPlugins, "Failed to mark plugin as unavailable", 
					"plugin", existingPlugin.Identifier, "error", err)
			} else {
				logging.InfoWithComponent(logging.ComponentPlugins, "Marked external plugin as unavailable", 
					"plugin", existingPlugin.Identifier)
			}
		}
	}

	logging.InfoWithComponent(logging.ComponentPlugins, "External plugin discovery completed", 
		"discovered_count", len(plugins), "unavailable_count", len(existingPlugins)-len(foundPlugins))

	return nil
}

// ExternalServiceResponse represents the Ruby service response structure
type ExternalServiceResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Plugins map[string]*ExternalPluginData `json:"plugins"`
	} `json:"data"`
}

// fetchPluginMetadata retrieves plugin metadata from the external service
func (s *PluginScannerService) fetchPluginMetadata(ctx context.Context) (map[string]*ExternalPluginData, error) {
	url := fmt.Sprintf("%s/api/plugins", s.serviceURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service returned status %d", resp.StatusCode)
	}

	var response ExternalServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("service returned success=false")
	}

	return response.Data.Plugins, nil
}

// registerPlugin registers or updates a plugin definition in the database
func (s *PluginScannerService) registerPlugin(identifier string, data *ExternalPluginData, status string) error {
	// Check if plugin already exists
	var existing database.PluginDefinition
	err := s.db.Where("identifier = ? AND plugin_type = ?", identifier, "external").First(&existing).Error
	
	isUpdate := err == nil
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing plugin: %w", err)
	}

	// Create or update plugin definition
	plugin := database.PluginDefinition{
		Identifier:         identifier,
		PluginType:         "external",
		Name:               data.Name,
		Description:        data.Description,
		Author:             "TRMNL", // Always TRMNL for external plugins
		Version:            data.Version,
		RequiresProcessing: true, // External plugins always require processing
		FormFields:         datatypes.JSON(data.FormFields),
		OAuthConfig:        datatypes.JSON(data.OAuthConfig), // Store OAuth configuration
		EnableDarkMode:     &[]bool{false}[0], // Default to false
		RemoveBleedMargin:  &[]bool{false}[0], // Default to false
		IsActive:           true,  // External plugins should be active by default
		Status:             status, // Set availability status
	}

	// Set template fields from the templates map
	if template, ok := data.Templates["full"]; ok && template != "" {
		plugin.MarkupFull = &template
	}
	if template, ok := data.Templates["half_vert"]; ok && template != "" {
		plugin.MarkupHalfVert = &template
	}
	if template, ok := data.Templates["half_horiz"]; ok && template != "" {
		plugin.MarkupHalfHoriz = &template
	}
	if template, ok := data.Templates["quadrant"]; ok && template != "" {
		plugin.MarkupQuadrant = &template
	}

	if isUpdate {
		// Update existing plugin
		plugin.ID = existing.ID
		plugin.CreatedAt = existing.CreatedAt
		
		if err := s.db.Save(&plugin).Error; err != nil {
			return fmt.Errorf("failed to update plugin definition: %w", err)
		}
		
		logging.Debug("[PLUGIN_SCANNER] Updated external plugin definition", 
			"plugin", identifier, "version", data.Version)
	} else {
		// Create new plugin
		if err := s.db.Create(&plugin).Error; err != nil {
			return fmt.Errorf("failed to create plugin definition: %w", err)
		}
		
		logging.Debug("[PLUGIN_SCANNER] Created external plugin definition", 
			"plugin", identifier, "version", data.Version)
	}

	return nil
}

// IsServiceAvailable checks if the external plugin service is reachable
func (s *PluginScannerService) IsServiceAvailable(ctx context.Context) bool {
	url := fmt.Sprintf("%s/api/health", s.serviceURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// StartPeriodicScanning starts a background goroutine that periodically scans for plugins
func (s *PluginScannerService) StartPeriodicScanning(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				if err := s.ScanAndRegisterPlugins(ctx); err != nil {
					logging.WarnWithComponent(logging.ComponentPlugins, 
						"Periodic plugin scan failed", "error", err)
				}
				cancel()
			}
		}
	}()
}

// getExistingExternalPlugins returns all external plugin definitions from the database
func (s *PluginScannerService) getExistingExternalPlugins() ([]database.PluginDefinition, error) {
	var plugins []database.PluginDefinition
	err := s.db.Where("plugin_type = ?", "external").Find(&plugins).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get existing external plugins: %w", err)
	}
	return plugins, nil
}

// markAllExternalPluginsUnavailable marks all external plugins as unavailable
func (s *PluginScannerService) markAllExternalPluginsUnavailable() error {
	err := s.db.Model(&database.PluginDefinition{}).
		Where("plugin_type = ?", "external").
		Update("status", "unavailable").Error
	if err != nil {
		return fmt.Errorf("failed to mark external plugins as unavailable: %w", err)
	}
	return nil
}

// markPluginUnavailable marks a specific external plugin as unavailable
func (s *PluginScannerService) markPluginUnavailable(identifier string) error {
	err := s.db.Model(&database.PluginDefinition{}).
		Where("identifier = ? AND plugin_type = ?", identifier, "external").
		Update("status", "unavailable").Error
	if err != nil {
		return fmt.Errorf("failed to mark plugin as unavailable: %w", err)
	}
	return nil
}

// GetAvailablePluginDefinitions returns all available plugin definitions from the database
func (s *PluginScannerService) GetAvailablePluginDefinitions() ([]database.PluginDefinition, error) {
	var plugins []database.PluginDefinition
	err := s.db.Where("is_active = ? AND status = ?", true, "available").Find(&plugins).Error
	return plugins, err
}