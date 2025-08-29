package plugins

import (
	"fmt"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"gorm.io/gorm"
)

// PluginFactory creates plugin instances from definitions
type PluginFactory interface {
	CreatePlugin(definition *database.PluginDefinition, instance *database.PluginInstance) (Plugin, error)
}

// UnifiedPluginFactory implements PluginFactory for all plugin types
type UnifiedPluginFactory struct {
	db           *gorm.DB
	registry     map[string]Plugin // System plugins registry
}

// NewUnifiedPluginFactory creates a new unified plugin factory
func NewUnifiedPluginFactory(db *gorm.DB) *UnifiedPluginFactory {
	return &UnifiedPluginFactory{
		db:       db,
		registry: GetAll(), // Get all registered system plugins
	}
}

// CreatePlugin creates a plugin instance based on the definition type
func (f *UnifiedPluginFactory) CreatePlugin(def *database.PluginDefinition, inst *database.PluginInstance) (Plugin, error) {
	if def == nil {
		return nil, fmt.Errorf("plugin definition cannot be nil")
	}
	
	if inst == nil {
		return nil, fmt.Errorf("plugin instance cannot be nil")
	}
	
	switch def.PluginType {
	case "system":
		return f.createSystemPlugin(def, inst)
	case "private":
		return f.createPrivatePlugin(def, inst)
	case "mashup":
		return f.createMashupPlugin(def, inst)
	case "public":
		// Future implementation
		return nil, fmt.Errorf("public plugins not yet implemented")
	default:
		return nil, fmt.Errorf("unknown plugin type: %s", def.PluginType)
	}
}

// createSystemPlugin creates a system plugin instance
func (f *UnifiedPluginFactory) createSystemPlugin(def *database.PluginDefinition, inst *database.PluginInstance) (Plugin, error) {
	// Get the plugin from the registry
	plugin, exists := f.registry[def.Identifier]
	if !exists {
		return nil, fmt.Errorf("system plugin %s not found in registry", def.Identifier)
	}
	
	// System plugins don't need instance-specific configuration at the plugin level
	// The instance settings are handled in the PluginContext during processing
	return plugin, nil
}

// createPrivatePlugin creates a private plugin instance
func (f *UnifiedPluginFactory) createPrivatePlugin(def *database.PluginDefinition, inst *database.PluginInstance) (Plugin, error) {
	// Get the registered private plugin factory
	factory := GetPrivatePluginFactory()
	if factory == nil {
		return nil, fmt.Errorf("private plugin factory not registered")
	}
	
	// Create a private plugin with the definition and instance
	return factory(def, inst), nil
}

// createMashupPlugin creates a mashup plugin instance
func (f *UnifiedPluginFactory) createMashupPlugin(def *database.PluginDefinition, inst *database.PluginInstance) (Plugin, error) {
	// Get the registered mashup plugin factory
	factory := GetMashupPluginFactory()
	if factory == nil {
		return nil, fmt.Errorf("mashup plugin factory not registered")
	}
	
	// Create a mashup plugin with the definition and instance
	return factory(def, inst), nil
}

// RefreshSystemPlugins updates the system plugin registry
func (f *UnifiedPluginFactory) RefreshSystemPlugins() {
	f.registry = GetAll()
}

// GetAvailableSystemPlugins returns all available system plugins
func (f *UnifiedPluginFactory) GetAvailableSystemPlugins() map[string]Plugin {
	result := make(map[string]Plugin)
	for k, v := range f.registry {
		result[k] = v
	}
	return result
}

// ValidateDefinition validates a plugin definition for a specific type
func (f *UnifiedPluginFactory) ValidateDefinition(def *database.PluginDefinition) error {
	if def == nil {
		return fmt.Errorf("plugin definition cannot be nil")
	}
	
	switch def.PluginType {
	case "system":
		return f.validateSystemDefinition(def)
	case "private":
		return f.validatePrivateDefinition(def)
	case "mashup":
		return f.validateMashupDefinition(def)
	case "public":
		// Future implementation
		return fmt.Errorf("public plugins not yet implemented")
	default:
		return fmt.Errorf("unknown plugin type: %s", def.PluginType)
	}
}

// validateSystemDefinition validates a system plugin definition
func (f *UnifiedPluginFactory) validateSystemDefinition(def *database.PluginDefinition) error {
	if def.Identifier == "" {
		return fmt.Errorf("system plugin identifier cannot be empty")
	}
	
	if def.OwnerID != nil {
		return fmt.Errorf("system plugins cannot have owners")
	}
	
	// Check if plugin exists in registry
	if _, exists := f.registry[def.Identifier]; !exists {
		return fmt.Errorf("system plugin %s not found in registry", def.Identifier)
	}
	
	return nil
}

// validatePrivateDefinition validates a private plugin definition
func (f *UnifiedPluginFactory) validatePrivateDefinition(def *database.PluginDefinition) error {
	if def.OwnerID == nil {
		return fmt.Errorf("private plugins must have an owner")
	}
	
	if def.DataStrategy == nil || *def.DataStrategy == "" {
		return fmt.Errorf("private plugins must have a data strategy")
	}
	
	validStrategies := map[string]bool{
		"webhook": true,
		"polling": true,
		"static":  true,
	}
	
	if !validStrategies[*def.DataStrategy] {
		return fmt.Errorf("invalid data strategy: %s", *def.DataStrategy)
	}
	
	// Validate that at least one layout template is provided
	if (def.MarkupFull == nil || *def.MarkupFull == "") &&
		(def.MarkupHalfVert == nil || *def.MarkupHalfVert == "") &&
		(def.MarkupHalfHoriz == nil || *def.MarkupHalfHoriz == "") &&
		(def.MarkupQuadrant == nil || *def.MarkupQuadrant == "") {
		return fmt.Errorf("private plugins must have at least one layout template")
	}
	
	return nil
}

// validateMashupDefinition validates a mashup plugin definition
func (f *UnifiedPluginFactory) validateMashupDefinition(def *database.PluginDefinition) error {
	if def.OwnerID == nil {
		return fmt.Errorf("mashup plugins must have an owner")
	}
	
	if !def.IsMashup {
		return fmt.Errorf("plugin definition marked as mashup but IsMashup is false")
	}
	
	if def.MashupLayout == nil || *def.MashupLayout == "" {
		return fmt.Errorf("mashup plugins must have a layout")
	}
	
	// Validate layout is supported
	validLayouts := map[string]bool{
		"1Lx1R": true,
		"1Tx1B": true,
		"1Lx2R": true,
		"2Lx1R": true,
		"2Tx1B": true,
		"1Tx2B": true,
		"2x2":   true,
	}
	
	if !validLayouts[*def.MashupLayout] {
		return fmt.Errorf("invalid mashup layout: %s", *def.MashupLayout)
	}
	
	return nil
}

// Global factory instance
var globalFactory *UnifiedPluginFactory

// InitPluginFactory initializes the global plugin factory
func InitPluginFactory(db *gorm.DB) {
	globalFactory = NewUnifiedPluginFactory(db)
}

// GetPluginFactory returns the global plugin factory
func GetPluginFactory() *UnifiedPluginFactory {
	return globalFactory
}