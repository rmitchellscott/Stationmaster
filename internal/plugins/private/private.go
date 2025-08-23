package private

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// PrivatePlugin implements the Plugin interface for user-created private plugins
type PrivatePlugin struct {
	definition *database.PluginDefinition
	instance   *database.PluginInstance
}

// NewPrivatePlugin creates a new private plugin instance
func NewPrivatePlugin(definition *database.PluginDefinition, instance *database.PluginInstance) plugins.Plugin {
	return &PrivatePlugin{
		definition: definition,
		instance:   instance,
	}
}

// Type returns the plugin type identifier based on the definition
func (p *PrivatePlugin) Type() string {
	return fmt.Sprintf("private_%s", p.definition.ID.String())
}

// PluginType returns that this is an image plugin
func (p *PrivatePlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the instance name if available, otherwise definition name
func (p *PrivatePlugin) Name() string {
	if p.instance != nil {
		return p.instance.Name
	}
	return p.definition.Name
}

// Description returns the plugin description
func (p *PrivatePlugin) Description() string {
	return p.definition.Description
}

// Author returns the plugin author
func (p *PrivatePlugin) Author() string {
	return p.definition.Author
}

// Version returns the plugin version
func (p *PrivatePlugin) Version() string {
	return p.definition.Version
}

// RequiresProcessing returns true since private plugins need HTML rendering
func (p *PrivatePlugin) RequiresProcessing() bool {
	return p.definition.RequiresProcessing
}

// ConfigSchema returns the JSON schema for form fields defined by the user
func (p *PrivatePlugin) ConfigSchema() string {
	if p.definition.FormFields != nil {
		return string(p.definition.FormFields)
	}
	return `{"type": "object", "properties": {}}`
}

// Process executes the plugin logic - converts HTML to image like screenshot plugin
func (p *PrivatePlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Validate device model information
	if ctx.Device == nil || ctx.Device.DeviceModel == nil {
		return plugins.CreateErrorResponse("Device model information not available"),
			fmt.Errorf("device model is required for private plugin processing")
	}
	
	// Get the user's template from the definition
	if p.definition.MarkupFull == nil || *p.definition.MarkupFull == "" {
		return plugins.CreateErrorResponse("No template defined for private plugin"),
			fmt.Errorf("markup_full is empty for private plugin %s", p.definition.ID)
	}
	
	userTemplate := *p.definition.MarkupFull
	
	// Basic template variable substitution
	processedTemplate := userTemplate
	processedTemplate = strings.ReplaceAll(processedTemplate, "{{ timestamp }}", time.Now().Format("2006-01-02 15:04:05"))
	processedTemplate = strings.ReplaceAll(processedTemplate, "{{timestamp}}", time.Now().Format("2006-01-02 15:04:05"))
	
	// Create complete HTML document with TRMNL framework
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s</title>
    <link rel="stylesheet" href="https://usetrmnl.com/css/latest/plugins.css">
    <style>
        body { 
            width: %dpx; 
            height: %dpx; 
            margin: 0; 
            padding: 0;
        }
    </style>
</head>
<body>
    %s
    <script src="https://usetrmnl.com/js/latest/plugins.js"></script>
</body>
</html>`,
		p.Name(),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
		processedTemplate)
	
	// Create browserless renderer
	renderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to create renderer: %v", err)),
			fmt.Errorf("failed to create browserless renderer: %w", err)
	}
	defer renderer.Close()
	
	// Render HTML to image using browserless
	renderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	imageData, err := renderer.RenderHTML(
		renderCtx,
		html,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
	)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render HTML: %v", err)),
			fmt.Errorf("failed to render HTML to image: %w", err)
	}
	
	// Generate filename
	filename := fmt.Sprintf("private_plugin_%s_%dx%d.png",
		time.Now().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)
	
	// Return image data response (RenderWorker will handle storage)
	return plugins.CreateImageDataResponse(imageData, filename), nil
}

// Validate validates the plugin settings against the form fields schema
func (p *PrivatePlugin) Validate(settings map[string]interface{}) error {
	// TODO: Implement JSON schema validation against FormFields
	return nil
}

// Register the private plugin factory when this package is imported
func init() {
	plugins.RegisterPrivatePluginFactory(NewPrivatePlugin)
}