package plugins

import (
	"encoding/json"
	"fmt"

	"github.com/rmitchellscott/stationmaster/internal/database"
)

// NewPluginContext creates a new plugin context with parsed settings
func NewPluginContext(device *database.Device, userPlugin *database.UserPlugin) (PluginContext, error) {
	settings := make(map[string]interface{})
	
	if userPlugin.Settings != "" {
		if err := json.Unmarshal([]byte(userPlugin.Settings), &settings); err != nil {
			return PluginContext{}, fmt.Errorf("failed to parse plugin settings: %w", err)
		}
	}
	
	return PluginContext{
		Device:     device,
		UserPlugin: userPlugin,
		Settings:   settings,
	}, nil
}

// GetStringSetting returns a string setting value with fallback
func (ctx PluginContext) GetStringSetting(key string, fallback string) string {
	if val, ok := ctx.Settings[key].(string); ok {
		return val
	}
	return fallback
}

// GetIntSetting returns an integer setting value with fallback
func (ctx PluginContext) GetIntSetting(key string, fallback int) int {
	if val, ok := ctx.Settings[key].(float64); ok {
		return int(val)
	}
	if val, ok := ctx.Settings[key].(int); ok {
		return val
	}
	return fallback
}

// GetBoolSetting returns a boolean setting value with fallback
func (ctx PluginContext) GetBoolSetting(key string, fallback bool) bool {
	if val, ok := ctx.Settings[key].(bool); ok {
		return val
	}
	return fallback
}

// GetFloatSetting returns a float setting value with fallback
func (ctx PluginContext) GetFloatSetting(key string, fallback float64) float64 {
	if val, ok := ctx.Settings[key].(float64); ok {
		return val
	}
	return fallback
}

// HasSetting checks if a setting exists
func (ctx PluginContext) HasSetting(key string) bool {
	_, exists := ctx.Settings[key]
	return exists
}