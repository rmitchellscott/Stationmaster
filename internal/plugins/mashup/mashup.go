package mashup

import (
	"context"
	"fmt"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// Register the mashup plugin factory when this package is imported
func init() {
	plugins.RegisterMashupPluginFactory(NewMashupPlugin)
}

// MashupPlugin implements the Plugin interface for mashup plugins
type MashupPlugin struct {
	definition *database.PluginDefinition
	instance   *database.PluginInstance
	mashupService *database.MashupService
}

// NewMashupPlugin creates a new mashup plugin instance
func NewMashupPlugin(definition *database.PluginDefinition, instance *database.PluginInstance) plugins.Plugin {
	db := database.GetDB()
	return &MashupPlugin{
		definition: definition,
		instance:   instance,
		mashupService: database.NewMashupService(db),
	}
}

// Type returns the plugin type identifier
func (p *MashupPlugin) Type() string {
	return fmt.Sprintf("mashup_%s", p.definition.ID)
}

// PluginType returns that this is an image plugin
func (p *MashupPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the instance name
func (p *MashupPlugin) Name() string {
	if p.instance != nil {
		return p.instance.Name
	}
	return p.definition.Name
}

// Description returns the plugin description
func (p *MashupPlugin) Description() string {
	return p.definition.Description
}

// Author returns the plugin author
func (p *MashupPlugin) Author() string {
	return p.definition.Author
}

// Version returns the plugin version
func (p *MashupPlugin) Version() string {
	return p.definition.Version
}

// RequiresProcessing returns true since mashups need HTML rendering
func (p *MashupPlugin) RequiresProcessing() bool {
	return true
}

// ConfigSchema returns the JSON schema for configuration
func (p *MashupPlugin) ConfigSchema() string {
	return p.definition.ConfigSchema
}

// Process executes the mashup logic - combines child plugin outputs
func (p *MashupPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	if p.instance == nil {
		return plugins.CreateErrorResponse("No plugin instance provided"), 
			fmt.Errorf("mashup plugin requires an instance")
	}
	
	// Validate device model information
	if ctx.Device == nil || ctx.Device.DeviceModel == nil {
		return plugins.CreateErrorResponse("Device model information not available"),
			fmt.Errorf("device model is required for mashup processing")
	}
	
	// Get mashup layout
	if p.definition.MashupLayout == nil {
		return plugins.CreateErrorResponse("No mashup layout defined"),
			fmt.Errorf("mashup layout is required")
	}
	layout := *p.definition.MashupLayout
	
	// Get child plugin instances
	children, err := p.mashupService.GetChildren(p.instance.ID)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to get mashup children: %v", err)),
			fmt.Errorf("failed to get mashup children: %w", err)
	}
	
	if len(children) == 0 {
		return plugins.CreateErrorResponse("No child plugins configured"),
			fmt.Errorf("mashup has no child plugins configured")
	}
	
	logging.Info("[MASHUP] Processing mashup", "layout", layout, "children_count", len(children))
	
	// Render each child plugin
	childResults := make(map[string]ChildRenderResult)
	factory := plugins.GetPluginFactory()
	
	for _, child := range children {
		// Create child plugin instance
		var childPlugin plugins.Plugin
		var childErr error
		
		if child.ChildInstance.PluginDefinition.PluginType == "private" {
			childPlugin, childErr = factory.CreatePlugin(&child.ChildInstance.PluginDefinition, &child.ChildInstance)
		} else if child.ChildInstance.PluginDefinition.PluginType == "system" {
			var exists bool
			childPlugin, exists = plugins.Get(child.ChildInstance.PluginDefinition.Identifier)
			if !exists {
				childErr = fmt.Errorf("system plugin %s not found", child.ChildInstance.PluginDefinition.Identifier)
			}
		} else {
			childErr = fmt.Errorf("unsupported child plugin type: %s", child.ChildInstance.PluginDefinition.PluginType)
		}
		
		if childErr != nil {
			logging.Error("[MASHUP] Failed to create child plugin", "slot", child.SlotPosition, "error", childErr)
			childResults[child.SlotPosition] = ChildRenderResult{
				Success: false,
				Error:   childErr.Error(),
				HTML:    fmt.Sprintf("<div class='mashup-error'>Failed to load plugin: %s</div>", childErr.Error()),
			}
			continue
		}
		
		// Create plugin context for child (same user, device, etc.)
		childPluginCtx := plugins.PluginContext{
			Device: ctx.Device,
			User:   ctx.User,
		}
		
		childResponse, childErr := childPlugin.Process(childPluginCtx)
		if childErr != nil {
			logging.Error("[MASHUP] Child plugin processing failed", "slot", child.SlotPosition, "error", childErr)
			childResults[child.SlotPosition] = ChildRenderResult{
				Success: false,
				Error:   childErr.Error(),
				HTML:    fmt.Sprintf("<div class='mashup-error'>Plugin error: %s</div>", childErr.Error()),
			}
			continue
		}
		
		// Store the response for mashup rendering
		childResults[child.SlotPosition] = ChildRenderResult{
			Success:  true,
			Response: childResponse,
			Plugin:   childPlugin,
			Instance: &child.ChildInstance,
		}
		
		logging.Info("[MASHUP] Child plugin processed successfully", "slot", child.SlotPosition, "plugin", child.ChildInstance.Name)
	}
	
	// Generate mashup HTML using the real MashupRenderer with child plugin data
	renderer := NewMashupRenderer(layout, childResults)
	finalHTML, err := renderer.RenderMashup(ctx)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render mashup: %v", err)),
			fmt.Errorf("failed to render mashup: %w", err)
	}
	
	// Create browserless renderer for HTML to image conversion
	browserRenderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to create renderer: %v", err)),
			fmt.Errorf("failed to create browserless renderer: %w", err)
	}
	defer browserRenderer.Close()
	
	// Render HTML to image using browserless (like private plugins do)
	renderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	imageData, err := browserRenderer.RenderHTML(
		renderCtx,
		finalHTML,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
	)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render HTML: %v", err)),
			fmt.Errorf("failed to render HTML to image: %w", err)
	}
	
	// Generate filename
	filename := fmt.Sprintf("mashup_%s_%dx%d.png",
		time.Now().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)
	
	// Return image data response (like private plugins do)
	return plugins.CreateImageDataResponse(imageData, filename), nil
}

// Validate validates the plugin settings (currently no special validation needed)
func (p *MashupPlugin) Validate(settings map[string]interface{}) error {
	return nil
}

// GetInstance returns the plugin instance
func (p *MashupPlugin) GetInstance() *database.PluginInstance {
	return p.instance
}


// ChildRenderResult represents the result of rendering a child plugin
type ChildRenderResult struct {
	Success  bool                         // Whether rendering was successful
	Error    string                       // Error message if failed
	HTML     string                       // Rendered HTML content (for errors)
	Response plugins.PluginResponse       // Full plugin response (for success)
	Plugin   plugins.Plugin               // Plugin instance (for rendering context)
	Instance *database.PluginInstance     // Plugin instance data
}