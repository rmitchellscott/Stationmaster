package plugins

import (
	"time"
	"context"
	"encoding/json"
	"fmt"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/pluginruntime"
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
	db      *gorm.DB
	runtime *pluginruntime.Runtime
}

// NewPluginScannerService creates a new plugin scanner service
func NewPluginScannerService(db *gorm.DB) *PluginScannerService {
	return &PluginScannerService{
		db:      db,
		runtime: pluginruntime.New(),
	}
}

// ScanAndRegisterPlugins discovers plugins from external services and registers them in the database
func (s *PluginScannerService) ScanAndRegisterPlugins(ctx context.Context) error {
	logging.InfoWithComponent(logging.ComponentPlugins, "Starting plugin discovery")

	if err := s.runtime.WaitReady(ctx, 60*time.Second); err != nil {
		return fmt.Errorf("plugin runtime not ready: %w", err)
	}

	plugins, err := s.fetchPluginMetadata(ctx)
	if err != nil {
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

// fetchPluginMetadata asks the embedded runtime to enumerate the plugin tree. The
// payload is the shape the Rails service used to return over HTTP, so registerPlugin
// below is unchanged.
func (s *PluginScannerService) fetchPluginMetadata(ctx context.Context) (map[string]*ExternalPluginData, error) {
	raw, err := s.runtime.DiscoverMetadata(ctx)
	if err != nil {
		return nil, err
	}

	plugins := make(map[string]*ExternalPluginData)
	if err := json.Unmarshal(raw, &plugins); err != nil {
		return nil, fmt.Errorf("failed to decode plugin metadata: %w", err)
	}

	return plugins, nil
}

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

// getExistingExternalPlugins returns all external plugin definitions from the database
func (s *PluginScannerService) getExistingExternalPlugins() ([]database.PluginDefinition, error) {
	var plugins []database.PluginDefinition
	err := s.db.Where("plugin_type = ?", "external").Find(&plugins).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get existing external plugins: %w", err)
	}
	return plugins, nil
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