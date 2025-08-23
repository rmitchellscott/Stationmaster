package private

import (
	"context"
	"fmt"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// LayoutType represents different TRMNL layout types
type LayoutType string

const (
	LayoutFull         LayoutType = "full"
	LayoutHalfVertical LayoutType = "half_vertical"
	LayoutHalfHorizontal LayoutType = "half_horizontal"
	LayoutQuadrant     LayoutType = "quadrant"
)

// PrivatePlugin implements the Plugin interface for user-created private plugins
type PrivatePlugin struct {
	dbModel      *database.PrivatePlugin
	liquidEngine *LiquidRenderer
}

// NewPrivatePlugin creates a new private plugin instance from database model
func NewPrivatePlugin(dbModel *database.PrivatePlugin) *PrivatePlugin {
	return &PrivatePlugin{
		dbModel:      dbModel,
		liquidEngine: NewLiquidRenderer(),
	}
}

// Type returns the plugin type identifier
func (p *PrivatePlugin) Type() string {
	return fmt.Sprintf("private_%s", p.dbModel.ID.String())
}

// PluginType returns that this is a data plugin (requires processing)
func (p *PrivatePlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeData
}

// Name returns the user-defined name
func (p *PrivatePlugin) Name() string {
	return p.dbModel.Name
}

// Description returns the user-defined description
func (p *PrivatePlugin) Description() string {
	return p.dbModel.Description
}

// Author returns the plugin author (user who created it)
func (p *PrivatePlugin) Author() string {
	// This would need to be populated from the User association
	// For now, return a default value
	return "User Created"
}

// Version returns the plugin version
func (p *PrivatePlugin) Version() string {
	return p.dbModel.Version
}

// RequiresProcessing returns true since private plugins need HTML rendering
func (p *PrivatePlugin) RequiresProcessing() bool {
	return true
}

// ConfigSchema returns the JSON schema for form fields defined by the user
func (p *PrivatePlugin) ConfigSchema() string {
	if p.dbModel.FormFields != nil {
		// Convert form fields to JSON schema format
		return string(p.dbModel.FormFields)
	}
	// Return empty schema if no form fields defined
	return `{"type": "object", "properties": {}}`
}

// Process executes the plugin logic with layout-aware rendering
func (p *PrivatePlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// 1. Determine the requested layout from context
	layout := p.getLayoutFromContext(ctx)
	
	// 2. Get the appropriate template for the layout
	template, err := p.getTemplateForLayout(layout)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Template error: %v", err)), err
	}
	
	// 3. Fetch data based on the configured data strategy
	data, err := p.fetchData(ctx)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Data fetch error: %v", err)), err
	}
	
	// 4. Render the liquid template with data and layout context
	html, err := p.renderWithLayout(template, data, layout, ctx)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Render error: %v", err)), err
	}
	
	// 5. Convert HTML to image using browserless
	imageData, err := p.renderHTMLToImage(ctx, html)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Image render error: %v", err)), err
	}
	
	// 6. Generate filename
	filename := fmt.Sprintf("private_plugin_%s_%s_%dx%d.png",
		p.dbModel.ID.String()[:8],
		time.Now().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)
	
	// 7. Return image data response
	return plugins.CreateImageDataResponse(imageData, filename), nil
}

// Validate validates the plugin settings against the form fields schema
func (p *PrivatePlugin) Validate(settings map[string]interface{}) error {
	// TODO: Implement JSON schema validation against FormFields
	// For now, return nil (no validation errors)
	return nil
}

// getLayoutFromContext determines the layout type from the plugin context
func (p *PrivatePlugin) getLayoutFromContext(ctx plugins.PluginContext) LayoutType {
	// TODO: Add layout information to PluginContext
	// For now, default to full layout
	return LayoutFull
}

// getTemplateForLayout returns the appropriate template for the given layout
func (p *PrivatePlugin) getTemplateForLayout(layout LayoutType) (string, error) {
	switch layout {
	case LayoutFull:
		if p.dbModel.MarkupFull == "" {
			return "", fmt.Errorf("no template defined for full layout")
		}
		return p.dbModel.MarkupFull, nil
	case LayoutHalfVertical:
		if p.dbModel.MarkupHalfVert == "" {
			return "", fmt.Errorf("no template defined for half vertical layout")
		}
		return p.dbModel.MarkupHalfVert, nil
	case LayoutHalfHorizontal:
		if p.dbModel.MarkupHalfHoriz == "" {
			return "", fmt.Errorf("no template defined for half horizontal layout")
		}
		return p.dbModel.MarkupHalfHoriz, nil
	case LayoutQuadrant:
		if p.dbModel.MarkupQuadrant == "" {
			return "", fmt.Errorf("no template defined for quadrant layout")
		}
		return p.dbModel.MarkupQuadrant, nil
	default:
		return "", fmt.Errorf("unknown layout type: %s", layout)
	}
}

// fetchData fetches data based on the configured data strategy
func (p *PrivatePlugin) fetchData(ctx plugins.PluginContext) (map[string]interface{}, error) {
	switch p.dbModel.DataStrategy {
	case "webhook":
		return p.fetchWebhookData()
	case "polling":
		return p.fetchPollingData()
	case "merge":
		return p.fetchMergeData(ctx)
	default:
		return make(map[string]interface{}), nil
	}
}

// fetchWebhookData retrieves the latest webhook data for this plugin
func (p *PrivatePlugin) fetchWebhookData() (map[string]interface{}, error) {
	// TODO: Implement webhook data storage and retrieval
	// For now, return empty data
	return make(map[string]interface{}), nil
}

// fetchPollingData polls external URLs based on the polling configuration
func (p *PrivatePlugin) fetchPollingData() (map[string]interface{}, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Create data poller
	poller := NewDataPoller()
	
	// Poll data
	polledData, err := poller.PollData(ctx, p.dbModel)
	if err != nil {
		return nil, fmt.Errorf("polling failed: %w", err)
	}
	
	// Return the polled data
	return polledData.Data, nil
}

// fetchMergeData merges data from other plugins
func (p *PrivatePlugin) fetchMergeData(ctx plugins.PluginContext) (map[string]interface{}, error) {
	// TODO: Implement plugin merge functionality
	// For now, return empty data
	return make(map[string]interface{}), nil
}

// renderWithLayout renders the template with the appropriate TRMNL layout wrapper
func (p *PrivatePlugin) renderWithLayout(template string, data map[string]interface{}, layout LayoutType, ctx plugins.PluginContext) (string, error) {
	// Add layout and context information to template data
	templateData := make(map[string]interface{})
	
	// Add user data
	templateData["data"] = data
	
	// Add TRMNL context variables
	templateData["trmnl"] = map[string]interface{}{
		"user": map[string]interface{}{
			// TODO: Add user information from context
			"first_name": "User",
			"email":      "user@example.com",
		},
		"device": map[string]interface{}{
			"name":   ctx.Device.Name,
			"width":  ctx.Device.DeviceModel.ScreenWidth,
			"height": ctx.Device.DeviceModel.ScreenHeight,
		},
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}
	
	// Add layout information
	templateData["layout"] = map[string]interface{}{
		"type":    string(layout),
		"width":   p.getLayoutWidth(layout, ctx),
		"height":  p.getLayoutHeight(layout, ctx),
		"is_split": layout != LayoutFull,
	}
	
	// Add instance ID for containerization
	templateData["instance_id"] = p.dbModel.ID.String()[:8]
	
	// Render the liquid template
	renderedTemplate, err := p.liquidEngine.RenderTemplate(template, templateData)
	if err != nil {
		return "", fmt.Errorf("failed to render liquid template: %w", err)
	}
	
	// Wrap with TRMNL framework and layout structure
	return p.wrapWithTRMNLFramework(renderedTemplate, layout, ctx), nil
}

// wrapWithTRMNLFramework wraps the rendered template with TRMNL CSS/JS framework and layout
func (p *PrivatePlugin) wrapWithTRMNLFramework(content string, layout LayoutType, ctx plugins.PluginContext) string {
	viewClass := p.getViewClass(layout)
	
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="https://usetrmnl.com/css/latest/plugins.css">
    <script src="https://usetrmnl.com/js/latest/plugins.js"></script>
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
    <div class="screen">
        <div class="view %s">
            %s
            %s
        </div>
    </div>
</body>
</html>`,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
		viewClass,
		p.dbModel.SharedMarkup, // Prepend shared markup
		content)
	
	return html
}

// getViewClass returns the appropriate CSS class for the layout
func (p *PrivatePlugin) getViewClass(layout LayoutType) string {
	switch layout {
	case LayoutFull:
		return "view--full"
	case LayoutHalfVertical:
		return "view--half_vertical"
	case LayoutHalfHorizontal:
		return "view--half_horizontal"
	case LayoutQuadrant:
		return "view--quadrant"
	default:
		return "view--full"
	}
}

// getLayoutWidth returns the width for the given layout
func (p *PrivatePlugin) getLayoutWidth(layout LayoutType, ctx plugins.PluginContext) int {
	screenWidth := ctx.Device.DeviceModel.ScreenWidth
	switch layout {
	case LayoutFull:
		return screenWidth
	case LayoutHalfVertical, LayoutQuadrant:
		return screenWidth / 2
	case LayoutHalfHorizontal:
		return screenWidth
	default:
		return screenWidth
	}
}

// getLayoutHeight returns the height for the given layout
func (p *PrivatePlugin) getLayoutHeight(layout LayoutType, ctx plugins.PluginContext) int {
	screenHeight := ctx.Device.DeviceModel.ScreenHeight
	switch layout {
	case LayoutFull:
		return screenHeight
	case LayoutHalfVertical:
		return screenHeight
	case LayoutHalfHorizontal, LayoutQuadrant:
		return screenHeight / 2
	default:
		return screenHeight
	}
}

// renderHTMLToImage converts HTML content to PNG using browserless
func (p *PrivatePlugin) renderHTMLToImage(ctx plugins.PluginContext, html string) ([]byte, error) {
	// Create browserless renderer
	renderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create browserless renderer: %w", err)
	}
	defer renderer.Close()
	
	// Render HTML to image with device dimensions
	renderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	imageData, err := renderer.RenderHTML(
		renderCtx,
		html,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML to image: %w", err)
	}
	
	return imageData, nil
}