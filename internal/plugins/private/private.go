package private

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
	"github.com/rmitchellscott/stationmaster/internal/utils"
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

	// Prepare template data with form fields and external data
	templateData := make(map[string]interface{})

	// Add form field values from instance settings
	if p.instance != nil && p.instance.Settings != nil {
		var settings map[string]interface{}
		if err := json.Unmarshal(p.instance.Settings, &settings); err == nil {
			for key, value := range settings {
				templateData[key] = value
			}
		}
	}

	// Fetch external data based on data strategy
	switch dataStrategy := p.definition.DataStrategy; {
	case dataStrategy != nil && *dataStrategy == "polling":
		// Use enhanced data poller for robust polling with retries and error handling
		poller := NewEnhancedDataPoller()
		pollingCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if polledResult, err := poller.PollData(pollingCtx, p.definition, templateData); err == nil && polledResult.Success {
			// Merge polling data into template data
			for key, value := range polledResult.Data {
				templateData[key] = value
			}
		} else {
			// Log error but don't fail - allow template to render with form data only
			if err != nil {
				fmt.Printf("Warning: Failed to fetch polling data for plugin %s: %v\n", p.definition.ID, err)
			} else if len(polledResult.Errors) > 0 {
				fmt.Printf("Warning: Polling errors for plugin %s: %v\n", p.definition.ID, polledResult.Errors)
			}
		}
	case dataStrategy != nil && *dataStrategy == "webhook":
		// Webhook data is handled separately via webhook endpoints
		// No additional data fetching needed here
	case dataStrategy != nil && *dataStrategy == "merge":
		// TODO: Implement plugin merge functionality
		fmt.Printf("Plugin merge data strategy not yet implemented for plugin %s\n", p.definition.ID)
	}

	// Create TRMNL data structure to match official API
	trmnlData := map[string]interface{}{}

	// Add system information - Unix timestamp
	systemData := map[string]interface{}{
		"timestamp_utc": time.Now().Unix(),
	}
	trmnlData["system"] = systemData

	// Add device information if available
	if ctx.Device != nil {
		deviceData := map[string]interface{}{
			"friendly_id": ctx.Device.FriendlyID,
		}

		// Add device model dimensions if available
		if ctx.Device.DeviceModel != nil {
			deviceData["width"] = ctx.Device.DeviceModel.ScreenWidth
			deviceData["height"] = ctx.Device.DeviceModel.ScreenHeight
		}

		// Add battery information if available
		if ctx.Device.BatteryVoltage > 0 {
			batteryPercentage := utils.CalculateBatteryPercentage(ctx.Device.BatteryVoltage)
			deviceData["percent_charged"] = float64(batteryPercentage)
		}

		// Add WiFi information if available  
		if ctx.Device.RSSI != 0 {
			wifiPercentage := utils.CalculateWiFiPercentage(ctx.Device.RSSI)
			deviceData["wifi_strength"] = wifiPercentage
		}

		trmnlData["device"] = deviceData
	}

	// Add user information if available
	if ctx.User != nil {
		// Calculate UTC offset in seconds
		utcOffset := int64(0)
		locale := "en" // Default locale
		timezone := "UTC" // Default timezone IANA
		timezoneFriendly := "UTC" // Default friendly name
		
		if ctx.User.Timezone != "" {
			timezone = ctx.User.Timezone
			timezoneFriendly = utils.GetTimezoneFriendlyName(ctx.User.Timezone)
			// Parse timezone and calculate UTC offset
			loc, err := time.LoadLocation(ctx.User.Timezone)
			if err == nil {
				_, offset := time.Now().In(loc).Zone()
				utcOffset = int64(offset)
			}
		}
		
		if ctx.User.Locale != "" {
			// Convert "en-US" to "en" format if needed
			if len(ctx.User.Locale) >= 2 {
				locale = ctx.User.Locale[:2]
			}
		}

		// Build user full name
		firstName := ctx.User.FirstName
		lastName := ctx.User.LastName
		fullName := ""
		if firstName != "" && lastName != "" {
			fullName = firstName + " " + lastName
		} else if firstName != "" {
			fullName = firstName
		} else if lastName != "" {
			fullName = lastName
		} else {
			// Fallback to username if no names available
			fullName = ctx.User.Username
		}

		userData := map[string]interface{}{
			"name":           fullName,
			"first_name":     firstName,
			"last_name":      lastName,
			"locale":         locale,
			"time_zone":      timezoneFriendly,
			"time_zone_iana": timezone,
			"utc_offset":     utcOffset,
		}
		
		trmnlData["user"] = userData
	}

	// Add plugin settings - this contains plugin metadata, not user form data
	pluginSettings := map[string]interface{}{
		"instance_name": p.Name(),
	}
	
	// Add data strategy if available
	if p.definition.DataStrategy != nil {
		pluginSettings["strategy"] = *p.definition.DataStrategy
	}
	
	// Add polling config if this is a polling plugin
	if p.definition.DataStrategy != nil && *p.definition.DataStrategy == "polling" {
		pluginSettings["polling_url"] = ""
		pluginSettings["polling_headers"] = ""
		
		// Parse polling config if available
		if p.definition.PollingConfig != nil {
			var pollingConfig map[string]interface{}
			if err := json.Unmarshal(p.definition.PollingConfig, &pollingConfig); err == nil {
				if url, ok := pollingConfig["url"].(string); ok {
					pluginSettings["polling_url"] = url
				}
				if headers, ok := pollingConfig["headers"].(string); ok {
					pluginSettings["polling_headers"] = headers
				}
			}
		}
	}
	
	// Add default plugin configuration (these might come from plugin definition or defaults)
	pluginSettings["dark_mode"] = "no"
	pluginSettings["no_screen_padding"] = "no"
	
	trmnlData["plugin_settings"] = pluginSettings
	
	templateData["trmnl"] = trmnlData
	
	// Use the private plugin renderer service with client-side LiquidJS
	htmlRenderer := NewPrivatePluginRenderer()
	html, err := htmlRenderer.RenderToClientSideHTML(RenderOptions{
		SharedMarkup:   sharedMarkup,
		LayoutTemplate: *p.definition.MarkupFull,
		Data:           templateData,
		Width:          ctx.Device.DeviceModel.ScreenWidth,
		Height:         ctx.Device.DeviceModel.ScreenHeight,
		PluginName:     p.definition.Name,
		InstanceID:     instanceID,
		InstanceName:   p.Name(),
	})
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render template: %v", err)),
			fmt.Errorf("failed to render HTML template: %w", err)
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
	
	// Check if image content has changed by comparing image hash
	if p.instance != nil {
		comparator := NewPluginContentComparator()
		comparison := comparator.CompareImage(imageData, p.instance.LastImageHash)
		
		if comparator.ShouldSkipRender(comparison) {
			return plugins.CreateNoChangeResponse("Image content unchanged, skipping storage"), nil
		}
		
		// Image has changed - update the hash (this will be persisted by the render worker)
		p.instance.LastImageHash = &comparison.NewHash
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

// GetInstance returns the plugin instance (used for accessing updated fields like LastImageHash)
func (p *PrivatePlugin) GetInstance() *database.PluginInstance {
	return p.instance
}


// Register the private plugin factory when this package is imported
func init() {
	plugins.RegisterPrivatePluginFactory(NewPrivatePlugin)
}