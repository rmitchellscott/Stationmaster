package alias

import (
	"fmt"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// AliasPlugin implements a plugin that returns a configured static image URL
type AliasPlugin struct{}

// Type returns the plugin type identifier
func (p *AliasPlugin) Type() string {
	return "alias"
}

// PluginType returns that this is an image plugin
func (p *AliasPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the human-readable name
func (p *AliasPlugin) Name() string {
	return "Alias"
}

// Description returns the plugin description
func (p *AliasPlugin) Description() string {
	return "Returns a configured static image URL directly"
}

// Author returns the plugin author
func (p *AliasPlugin) Author() string {
	return "Stationmaster"
}

// Version returns the plugin version
func (p *AliasPlugin) Version() string {
	return "1.0.0"
}

// RequiresProcessing returns false since this plugin returns direct URLs
func (p *AliasPlugin) RequiresProcessing() bool {
	return false
}

// ConfigSchema returns the JSON schema for configuration
func (p *AliasPlugin) ConfigSchema() string {
	return `{
		"type": "object",
		"properties": {
			"image_url": {
				"type": "string",
				"title": "Image URL",
				"description": "The URL of the image to display",
				"format": "uri"
			}
		},
		"required": ["image_url"]
	}`
}

// Validate validates the plugin settings
func (p *AliasPlugin) Validate(settings map[string]interface{}) error {
	imageURL, ok := settings["image_url"].(string)
	if !ok || imageURL == "" {
		return fmt.Errorf("image_url is required")
	}

	return nil
}

// Process executes the plugin logic
func (p *AliasPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Get image URL from settings
	imageURL := ctx.GetStringSetting("image_url", "")
	if imageURL == "" {
		return plugins.CreateErrorResponse("image_url not configured"),
			fmt.Errorf("image_url not configured in plugin settings")
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("alias_%s", time.Now().UTC().Format("2006-01-02T15:04:05"))

	return plugins.CreateImageResponseWithoutRefresh(imageURL, filename), nil
}

// Register the plugin when this package is imported
func init() {
	plugins.Register(&AliasPlugin{})
}
