package private

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
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
	return fmt.Sprintf("private_%s", p.definition.ID)
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

	// Prepare template data with external data only
	templateData := make(map[string]interface{})

	// Parse form field values from instance settings for polling variable substitution
	var formFieldValues map[string]interface{}
	if p.instance != nil && p.instance.Settings != nil {
		if err := json.Unmarshal(p.instance.Settings, &formFieldValues); err != nil {
			formFieldValues = make(map[string]interface{})
		}
	} else {
		formFieldValues = make(map[string]interface{})
	}

	// Fetch external data based on data strategy
	switch dataStrategy := p.definition.DataStrategy; {
	case dataStrategy != nil && *dataStrategy == "polling":
		// First check if we have fresh polling data stored
		pollingService := database.NewPollingDataService(database.GetDB())
		
		// Check if stored data is fresh (within 5 minutes of expected refresh)
		maxAge := 5 * time.Minute // Allow some staleness to avoid duplicate polls
		if isFresh, err := pollingService.IsPollingDataFresh(instanceID, maxAge); err == nil && isFresh {
			// Use stored polling data
			if storedData, err := pollingService.GetPollingDataTemplate(instanceID); err == nil {
				for key, value := range storedData {
					templateData[key] = value
				}
				logging.Debug("[PRIVATE_PLUGIN] Using fresh stored polling data", "plugin_id", p.definition.ID, "instance_id", instanceID)
			}
		} else {
			// Poll fresh data and store it
			unifiedRenderer := rendering.NewUnifiedRenderer()
			poller := NewEnhancedDataPoller(unifiedRenderer)
			pollingCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			pollStartTime := time.Now().UTC()
			polledResult, err := poller.PollData(pollingCtx, p.definition, formFieldValues)
			pollDuration := time.Since(pollStartTime)
			
			if err == nil && polledResult.Success {
				// Merge polling data into template data
				for key, value := range polledResult.Data {
					templateData[key] = value
				}
				
				// Store the polling data for future use (including mashup children)
				rawDataJSON, _ := json.Marshal(polledResult.Data)
				mergedDataJSON, _ := json.Marshal(polledResult.Data)
				errorsJSON, _ := json.Marshal(polledResult.Errors)
				
				pollingData := &database.PrivatePluginPollingData{
					ID:               instanceID + "_polling_data",
					PluginInstanceID: instanceID,
					MergedData:       mergedDataJSON,
					RawData:          rawDataJSON,
					PolledAt:         time.Now().UTC(),
					PollDuration:     pollDuration,
					Success:          true,
					Errors:           errorsJSON,
					URLCount:         len(polledResult.Data), // Approximation
				}
				
				if storeErr := pollingService.StorePollingData(pollingData); storeErr != nil {
					logging.WarnWithComponent(logging.ComponentPlugins, "Failed to store polling data", "plugin_id", p.definition.ID, "error", storeErr)
				} else {
					logging.Debug("[PRIVATE_PLUGIN] Stored fresh polling data", "plugin_id", p.definition.ID, "instance_id", instanceID, "duration", pollDuration)
				}
			} else {
				// Store failed polling attempt
				errorsJSON, _ := json.Marshal(polledResult.Errors)
				if err != nil {
					errorsJSON, _ = json.Marshal([]string{err.Error()})
				}
				
				pollingData := &database.PrivatePluginPollingData{
					ID:               instanceID + "_polling_data",
					PluginInstanceID: instanceID,
					MergedData:       []byte("{}"),
					RawData:          []byte("{}"),
					PolledAt:         time.Now().UTC(),
					PollDuration:     pollDuration,
					Success:          false,
					Errors:           errorsJSON,
					URLCount:         0,
				}
				
				pollingService.StorePollingData(pollingData)
				
				// Log error but don't fail - allow template to render with form data only
				if err != nil {
					logging.WarnWithComponent(logging.ComponentPlugins, "Failed to fetch polling data for plugin", "plugin_id", p.definition.ID, "error", err)
				} else if len(polledResult.Errors) > 0 {
					logging.WarnWithComponent(logging.ComponentPlugins, "Polling errors for plugin", "plugin_id", p.definition.ID, "errors", polledResult.Errors)
				}
			}
		}
	case dataStrategy != nil && *dataStrategy == "webhook":
		// Webhook data: retrieve and merge the latest webhook data
		if p.instance != nil && p.instance.ID != uuid.Nil {
			webhookService := database.NewWebhookService(database.GetDB())
			webhookData, err := webhookService.GetWebhookDataTemplate(p.instance.ID.String())
			if err != nil {
				logging.WarnWithComponent(logging.ComponentPlugins, "Failed to fetch webhook data for plugin instance", "instance_id", p.instance.ID, "error", err)
			} else if webhookData != nil {
				// Merge webhook data into template data
				for key, value := range webhookData {
					templateData[key] = value
				}
			}
		}
	case dataStrategy != nil && *dataStrategy == "static":
		// Static strategy: merge both static data (from plugin definition) and form field values (instance settings)
		
		// First, merge static data from plugin definition (SampleData)
		if p.definition.SampleData != nil {
			var staticData map[string]interface{}
			if err := json.Unmarshal(p.definition.SampleData, &staticData); err == nil {
				for key, value := range staticData {
					templateData[key] = value
				}
				logging.Debug("[PRIVATE_PLUGIN] Merged static data from definition", "plugin_id", p.definition.ID, "keys", len(staticData))
			} else {
				logging.WarnWithComponent(logging.ComponentPlugins, "Failed to parse static data from plugin definition", "plugin_id", p.definition.ID, "error", err)
			}
		}
		
		// Then, merge form field values (instance settings) - these can override static data
		for key, value := range formFieldValues {
			templateData[key] = value
		}
		logging.Debug("[PRIVATE_PLUGIN] Merged form field values", "plugin_id", p.definition.ID, "instance_id", instanceID, "keys", len(formFieldValues))
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
	
	// Use the private plugin renderer service with Ruby server-side liquid
	htmlRenderer, err := NewPrivatePluginRenderer(".")
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to initialize private plugin renderer: %v", err)),
			fmt.Errorf("failed to initialize private plugin renderer: %w", err)
	}
	renderOptions := RenderOptions{
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
	
	// Use Ruby server-side rendering (required)
	html, err := htmlRenderer.RenderToServerSideHTML(context.Background(), renderOptions)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Ruby template rendering failed: %v", err)),
			fmt.Errorf("failed to render HTML template with Ruby: %w", err)
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
	
	renderResult, err := browserRenderer.RenderHTMLWithResult(
		renderCtx,
		html,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
	)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render HTML: %v", err)),
			fmt.Errorf("failed to render HTML to image: %w", err)
	}
	
	imageData := renderResult.ImageData
	flags := renderResult.Flags
	
	
	// Generate filename
	filename := fmt.Sprintf("private_plugin_%s_%dx%d.png",
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

// Validate validates the plugin settings against the form fields schema
func (p *PrivatePlugin) Validate(settings map[string]interface{}) error {
	// TODO: Implement JSON schema validation against FormFields
	return nil
}

// GetInstance returns the plugin instance
func (p *PrivatePlugin) GetInstance() *database.PluginInstance {
	return p.instance
}


// Register the private plugin factory when this package is imported
func init() {
	plugins.RegisterPrivatePluginFactory(NewPrivatePlugin)
}