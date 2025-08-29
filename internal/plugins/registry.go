package plugins

import (
	"fmt"
	"sort"
	"sync"

	"github.com/rmitchellscott/stationmaster/internal/database"
)

// registry holds all registered plugins
var (
	registry = make(map[string]Plugin)
	mutex    sync.RWMutex
)

// privatePluginFactory holds the factory function for creating private plugins
var (
	privatePluginFactory func(*database.PluginDefinition, *database.PluginInstance) Plugin
	privateFactoryMutex  sync.RWMutex
)

// mashupPluginFactory holds the factory function for creating mashup plugins
var (
	mashupPluginFactory func(*database.PluginDefinition, *database.PluginInstance) Plugin
	mashupFactoryMutex  sync.RWMutex
)

// Register adds a plugin to the registry
func Register(plugin Plugin) error {
	mutex.Lock()
	defer mutex.Unlock()
	
	pluginType := plugin.Type()
	if pluginType == "" {
		return fmt.Errorf("plugin type cannot be empty")
	}
	
	if _, exists := registry[pluginType]; exists {
		return fmt.Errorf("plugin type '%s' already registered", pluginType)
	}
	
	registry[pluginType] = plugin
	return nil
}

// Get retrieves a plugin from the registry by type
func Get(pluginType string) (Plugin, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	
	plugin, exists := registry[pluginType]
	return plugin, exists
}

// GetAll returns all registered plugins
func GetAll() map[string]Plugin {
	mutex.RLock()
	defer mutex.RUnlock()
	
	result := make(map[string]Plugin)
	for k, v := range registry {
		result[k] = v
	}
	return result
}

// GetAllTypes returns a sorted list of all plugin types
func GetAllTypes() []string {
	mutex.RLock()
	defer mutex.RUnlock()
	
	types := make([]string, 0, len(registry))
	for pluginType := range registry {
		types = append(types, pluginType)
	}
	
	sort.Strings(types)
	return types
}

// GetAllInfo returns metadata for all registered plugins
func GetAllInfo() []PluginInfo {
	mutex.RLock()
	defer mutex.RUnlock()
	
	info := make([]PluginInfo, 0, len(registry))
	for _, plugin := range registry {
		info = append(info, GetInfo(plugin))
	}
	
	// Sort by plugin type for consistent output
	sort.Slice(info, func(i, j int) bool {
		return info[i].Type < info[j].Type
	})
	
	return info
}

// Count returns the number of registered plugins
func Count() int {
	mutex.RLock()
	defer mutex.RUnlock()
	
	return len(registry)
}

// Exists checks if a plugin type is registered
func Exists(pluginType string) bool {
	mutex.RLock()
	defer mutex.RUnlock()
	
	_, exists := registry[pluginType]
	return exists
}

// RegisterPrivatePluginFactory registers a factory function for creating private plugins
func RegisterPrivatePluginFactory(factory func(*database.PluginDefinition, *database.PluginInstance) Plugin) {
	privateFactoryMutex.Lock()
	defer privateFactoryMutex.Unlock()
	privatePluginFactory = factory
}

// GetPrivatePluginFactory returns the registered private plugin factory function
func GetPrivatePluginFactory() func(*database.PluginDefinition, *database.PluginInstance) Plugin {
	privateFactoryMutex.RLock()
	defer privateFactoryMutex.RUnlock()
	return privatePluginFactory
}

// RegisterMashupPluginFactory registers a factory function for creating mashup plugins
func RegisterMashupPluginFactory(factory func(*database.PluginDefinition, *database.PluginInstance) Plugin) {
	mashupFactoryMutex.Lock()
	defer mashupFactoryMutex.Unlock()
	mashupPluginFactory = factory
}

// GetMashupPluginFactory returns the registered mashup plugin factory function
func GetMashupPluginFactory() func(*database.PluginDefinition, *database.PluginInstance) Plugin {
	mashupFactoryMutex.RLock()
	defer mashupFactoryMutex.RUnlock()
	return mashupPluginFactory
}