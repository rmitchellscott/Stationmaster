package external

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/imageprocessing"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/pluginruntime"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
	"github.com/rmitchellscott/stationmaster/internal/validation"
)

// ExternalPlugin implements the Plugin interface for externally-sourced plugins
type ExternalPlugin struct {
	definition *database.PluginDefinition
	instance   *database.PluginInstance
}

// NewExternalPlugin creates a new external plugin instance
func NewExternalPlugin(definition *database.PluginDefinition, instance *database.PluginInstance) plugins.Plugin {
	return &ExternalPlugin{
		definition: definition,
		instance:   instance,
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
	logging.Debug("[EXTERNAL_PLUGIN] ConfigSchema called", "plugin", p.definition.Identifier)
	
	if p.definition.FormFields == nil {
		logging.Debug("[EXTERNAL_PLUGIN] FormFields is nil", "plugin", p.definition.Identifier)
		return `{"type": "object", "properties": {}}`
	}
	
	logging.Debug("[EXTERNAL_PLUGIN] FormFields found", "plugin", p.definition.Identifier, "formFields", string(p.definition.FormFields))
	
	// Parse the FormFields JSON and convert YAML to JSON schema
	var formFieldsData interface{}
	if err := json.Unmarshal(p.definition.FormFields, &formFieldsData); err != nil {
		logging.Error("[EXTERNAL_PLUGIN] Failed to parse FormFields JSON", "plugin", p.definition.Identifier, "error", err, "formFields", string(p.definition.FormFields))
		return `{"type": "object", "properties": {}}`
	}
	
	logging.Debug("[EXTERNAL_PLUGIN] FormFields JSON parsed successfully", "plugin", p.definition.Identifier, "parsedData", formFieldsData)
	
	// Use the validation function to convert YAML form fields to JSON schema
	jsonSchema, err := validation.ValidateFormFields(formFieldsData)
	if err != nil {
		logging.Error("[EXTERNAL_PLUGIN] Failed to convert form fields to JSON schema", "plugin", p.definition.Identifier, "error", err, "formFieldsData", formFieldsData)
		return `{"type": "object", "properties": {}}`
	}
	
	return jsonSchema
}

// Process executes the plugin logic - fetches fully rendered HTML from Ruby service
func (p *ExternalPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Validate device model information
	if ctx.Device == nil || ctx.Device.DeviceModel == nil {
		return plugins.CreateErrorResponse("Device model information not available"),
			fmt.Errorf("device model is required for external plugin processing")
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

	// For standalone external plugins, use "full" layout
	layout := "full"
	
	// Fetch processed HTML from Ruby service (includes plugin execution + ERB rendering)
	processedContent, err := p.fetchRenderedHTML(formFieldValues, layout, ctx)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to fetch rendered HTML: %v", err)),
			fmt.Errorf("failed to fetch rendered HTML for external plugin %s: %w", p.definition.ID, err)
	}
	
	// Wrap content with same structure as private plugins get from generateHTMLStructure()
	// This provides the .environment.trmnl and .screen wrappers needed for proper CSS layout
	// Note: Don't add extra .view wrapper since external plugin content already has it
	renderWidth, renderHeight := rendering.RenderDimensions(
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
		ctx.Device.ScreenOrientation,
	)

	screenClasses := rendering.BuildScreenClasses(rendering.ScreenClassOptions{
		ModelName:         ctx.Device.DeviceModel.ModelName,
		BitDepth:          ctx.Device.DeviceModel.BitDepth,
		ScreenWidth:       ctx.Device.DeviceModel.ScreenWidth,
		ScreenHeight:      ctx.Device.DeviceModel.ScreenHeight,
		ScreenOrientation: ctx.Device.ScreenOrientation,
	})
	structuredContent := fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
		<div class="%s">
			%s
		</div>
	</div>`, p.instance.ID.String(), screenClasses, processedContent)

	assetsManager := rendering.NewHTMLAssetsManager()
	assetBaseURL := config.GetAssetBaseURL()
	wrappedHTML := assetsManager.WrapWithTRNMLAssets(
		structuredContent,
		p.Name(),
		renderWidth,
		renderHeight,
		false,
		false,
		assetBaseURL,
	)

	browserRenderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to create renderer: %v", err)),
			fmt.Errorf("failed to create browserless renderer: %w", err)
	}
	defer browserRenderer.Close()

	renderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	renderResult, err := browserRenderer.RenderHTMLWithResult(
		renderCtx,
		wrappedHTML,
		renderWidth,
		renderHeight,
	)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render HTML: %v", err)),
			fmt.Errorf("failed to render HTML to image: %w", err)
	}

	imageData := renderResult.ImageData
	flags := renderResult.Flags

	if rotation := rendering.ImageRotation(ctx.Device.DeviceModel.ScreenWidth, ctx.Device.DeviceModel.ScreenHeight, ctx.Device.ScreenOrientation); rotation != "none" {
		rotated, rotErr := imageprocessing.RotatePNGBytes(imageData, rotation)
		if rotErr != nil {
			return plugins.CreateErrorResponse(fmt.Sprintf("Failed to rotate image: %v", rotErr)),
				fmt.Errorf("failed to rotate image: %w", rotErr)
		}
		imageData = rotated
	}

	filename := fmt.Sprintf("external_plugin_%s_%dx%d.png",
		time.Now().UTC().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)
	
	// Return image data response (RenderWorker will handle storage)
	response := plugins.CreateImageDataResponse(imageData, filename)
	// Add flags to response metadata if needed
	if flags.SkipDisplay {
		response["skip_display"] = true
	}
	
	return response, nil
}


// fetchRenderedHTML runs the plugin in the embedded Ruby process and returns its HTML.
func (p *ExternalPlugin) fetchRenderedHTML(settings map[string]interface{}, layout string, ctx plugins.PluginContext) (string, error) {
	trmnlBuilder := rendering.NewTRNMLDataBuilder()
	trmnlData := trmnlBuilder.BuildTRNMLData(ctx, p.instance, settings)

	// Inject OAuth tokens for external service integration
	if ctx.User != nil {
		tokenCtx, tokenCancel := context.WithTimeout(context.Background(), 15*time.Second)
		tokens := auth.AccessTokensForUser(tokenCtx, ctx.User.ID.String())
		tokenCancel()
		if len(tokens) > 0 {
			trmnlData["oauth_tokens"] = tokens
		}
	}

	runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runtime := pluginruntime.New()
	html, err := runtime.Execute(runCtx, p.definition.Identifier, layout, settings, trmnlData)
	if err != nil {
		return "", err
	}

	logging.Debug("[EXTERNAL_PLUGIN] Rendered HTML", "plugin", p.definition.Identifier, "html_length", len(html))
	return html, nil
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