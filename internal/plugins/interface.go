package plugins

import (
	"github.com/gin-gonic/gin"
	"github.com/rmitchellscott/stationmaster/internal/database"
)

// PluginType represents the type of plugin (image or data)
type PluginType string

const (
	PluginTypeImage PluginType = "image" // Returns pre-rendered image URL
	PluginTypeData  PluginType = "data"  // Returns data for rendering
)

// PluginContext provides context information to plugins
type PluginContext struct {
	Device         *database.Device
	PluginInstance *database.PluginInstance
	Settings       map[string]interface{}
}

// PluginResponse is the response format returned by plugins
type PluginResponse = gin.H

// Plugin defines the interface that all plugins must implement
type Plugin interface {
	// Type returns the unique type identifier for this plugin
	Type() string
	
	// PluginType returns whether this is an image or data plugin
	PluginType() PluginType
	
	// Name returns the human-readable name of the plugin
	Name() string
	
	// Description returns a description of what the plugin does
	Description() string
	
	// Author returns the author/creator of the plugin
	Author() string
	
	// Version returns the version of the plugin
	Version() string
	
	// ConfigSchema returns the JSON schema for plugin configuration
	ConfigSchema() string
	
	// RequiresProcessing returns whether this plugin needs processing (HTML rendering/image processing)
	RequiresProcessing() bool
	
	// Process executes the plugin logic and returns a response
	Process(ctx PluginContext) (PluginResponse, error)
	
	// Validate validates the plugin settings
	Validate(settings map[string]interface{}) error
}

// DataPlugin extends Plugin for plugins that return data for rendering
type DataPlugin interface {
	Plugin
	
	// RenderTemplate returns the HTML template for rendering the data
	RenderTemplate() string
	
	// DataSchema returns the schema of the data structure returned
	DataSchema() string
}

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	Type               string     `json:"type"`
	PluginType         PluginType `json:"plugin_type"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	Author             string     `json:"author"`
	Version            string     `json:"version"`
	ConfigSchema       string     `json:"config_schema"`
	DataSchema         string     `json:"data_schema,omitempty"`
	RequiresProcessing bool       `json:"requires_processing"`
}

// GetInfo returns the plugin metadata
func GetInfo(plugin Plugin) PluginInfo {
	info := PluginInfo{
		Type:               plugin.Type(),
		PluginType:         plugin.PluginType(),
		Name:               plugin.Name(),
		Description:        plugin.Description(),
		Author:             plugin.Author(),
		Version:            plugin.Version(),
		ConfigSchema:       plugin.ConfigSchema(),
		RequiresProcessing: plugin.RequiresProcessing(),
	}
	
	if dataPlugin, ok := plugin.(DataPlugin); ok {
		info.DataSchema = dataPlugin.DataSchema()
	}
	
	return info
}