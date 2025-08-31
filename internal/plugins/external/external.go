package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// ExternalPlugin implements the Plugin interface for externally-sourced plugins
type ExternalPlugin struct {
	definition *database.PluginDefinition
	instance   *database.PluginInstance
	serviceURL string
}

// NewExternalPlugin creates a new external plugin instance
func NewExternalPlugin(definition *database.PluginDefinition, instance *database.PluginInstance) plugins.Plugin {
	serviceURL := os.Getenv("EXTERNAL_PLUGIN_SERVICES")
	if serviceURL == "" {
		serviceURL = "http://stationmaster-plugins:3000"
	}
	
	return &ExternalPlugin{
		definition: definition,
		instance:   instance,
		serviceURL: serviceURL,
	}
}

// Type returns the plugin type identifier based on the definition
func (p *ExternalPlugin) Type() string {
	return fmt.Sprintf("external_%s", p.definition.ID)
}

// PluginType returns that this is an image plugin (uses templates + data rendering)
func (p *ExternalPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the instance name if available, otherwise definition name
func (p *ExternalPlugin) Name() string {
	if p.instance != nil {
		return p.instance.Name
	}
	return p.definition.Name
}

// Description returns the plugin description
func (p *ExternalPlugin) Description() string {
	return p.definition.Description
}

// Author returns the plugin author (always TRMNL for external plugins)
func (p *ExternalPlugin) Author() string {
	return p.definition.Author
}

// Version returns the plugin version
func (p *ExternalPlugin) Version() string {
	return p.definition.Version
}

// RequiresProcessing returns true since external plugins need HTML rendering
func (p *ExternalPlugin) RequiresProcessing() bool {
	return p.definition.RequiresProcessing
}

// ConfigSchema returns the JSON schema for form fields
func (p *ExternalPlugin) ConfigSchema() string {
	if p.definition.FormFields != nil {
		return string(p.definition.FormFields)
	}
	return `{"type": "object", "properties": {}}`
}

// Process executes the plugin logic - fetches data via HTTP and renders with stored templates
func (p *ExternalPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Validate device model information
	if ctx.Device == nil || ctx.Device.DeviceModel == nil {
		return plugins.CreateErrorResponse("Device model information not available"),
			fmt.Errorf("device model is required for external plugin processing")
	}
	
	// Get the stored template from the definition
	if p.definition.MarkupFull == nil || *p.definition.MarkupFull == "" {
		return plugins.CreateErrorResponse("No template defined for external plugin"),
			fmt.Errorf("markup_full is empty for external plugin %s", p.definition.ID)
	}
	
	// Get plugin instance ID for the wrapper
	instanceID := "unknown"
	if p.instance != nil {
		instanceID = p.instance.ID.String()
	}
	
	// Get shared markup if available
	sharedMarkup := ""
	if p.definition.SharedMarkup != nil {
		sharedMarkup = *p.definition.SharedMarkup
	}

	// Parse form field values from instance settings
	var formFieldValues map[string]interface{}
	if p.instance != nil && p.instance.Settings != nil {
		if err := json.Unmarshal(p.instance.Settings, &formFieldValues); err != nil {
			formFieldValues = make(map[string]interface{})
		}
	} else {
		formFieldValues = make(map[string]interface{})
	}

	// Fetch data from external plugin service
	templateData, err := p.fetchPluginData(formFieldValues)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to fetch plugin data: %v", err)),
			fmt.Errorf("failed to fetch data for external plugin %s: %w", p.definition.ID, err)
	}

	// Create TRMNL data structure using shared builder
	trmnlBuilder := rendering.NewTRNMLDataBuilder()
	trmnlData := trmnlBuilder.BuildTRNMLData(ctx, p.instance, formFieldValues)
	
	templateData["trmnl"] = trmnlData
	
	// Get screen options from definition, defaulting to false if nil
	removeBleedMargin := false
	if p.definition.RemoveBleedMargin != nil {
		removeBleedMargin = *p.definition.RemoveBleedMargin
	}
	enableDarkMode := false
	if p.definition.EnableDarkMode != nil {
		enableDarkMode = *p.definition.EnableDarkMode
	}
	
	// Render template using unified renderer
	unifiedRenderer := rendering.NewUnifiedRenderer()
	renderOptions := rendering.PluginRenderOptions{
		SharedMarkup:      sharedMarkup,
		LayoutTemplate:    *p.definition.MarkupFull,
		Data:              templateData,
		Width:             ctx.Device.DeviceModel.ScreenWidth,
		Height:            ctx.Device.DeviceModel.ScreenHeight,
		PluginName:        p.definition.Name,
		InstanceID:        instanceID,
		InstanceName:      p.Name(),
		RemoveBleedMargin: removeBleedMargin,
		EnableDarkMode:    enableDarkMode,
	}
	
	html, err := unifiedRenderer.RenderToHTML(context.Background(), renderOptions)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Template rendering failed: %v", err)),
			fmt.Errorf("failed to render template via external service: %w", err)
	}
	
	// Create browserless renderer
	browserRenderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to create renderer: %v", err)),
			fmt.Errorf("failed to create browserless renderer: %w", err)
	}
	defer browserRenderer.Close()
	
	// Always render HTML to image using browserless
	renderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	imageData, err := browserRenderer.RenderHTML(
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
	filename := fmt.Sprintf("external_plugin_%s_%dx%d.png",
		time.Now().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)
	
	// Return image data response (RenderWorker will handle storage)
	return plugins.CreateImageDataResponse(imageData, filename), nil
}


// fetchPluginData fetches data from the external plugin service
func (p *ExternalPlugin) fetchPluginData(settings map[string]interface{}) (map[string]interface{}, error) {
	// Build URL for plugin execution
	url := fmt.Sprintf("%s/api/plugins/%s/execute", p.serviceURL, p.definition.Identifier)
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// For now, use GET request (similar to private plugin polling)
	// TODO: Support POST with settings if needed
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plugin service returned status %d", resp.StatusCode)
	}
	
	// Parse JSON response
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	logging.Debug("[EXTERNAL_PLUGIN] Fetched data successfully", "plugin", p.definition.Identifier, "data_keys", len(data))
	
	return data, nil
}

// Validate validates the plugin settings against the form fields schema
func (p *ExternalPlugin) Validate(settings map[string]interface{}) error {
	// TODO: Implement JSON schema validation against FormFields
	return nil
}

// GetInstance returns the plugin instance
func (p *ExternalPlugin) GetInstance() *database.PluginInstance {
	return p.instance
}

// Register the external plugin factory when this package is imported
func init() {
	plugins.RegisterExternalPluginFactory(NewExternalPlugin)
}