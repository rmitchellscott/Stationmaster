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
			poller := NewEnhancedDataPoller()
			pollingCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			pollStartTime := time.Now()
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
					PolledAt:         time.Now(),
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
					PolledAt:         time.Now(),
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
		// Static strategy uses only form fields and trmnl struct - no external data
		// No additional data fetching needed here
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
			batteryPercentage := plugins.BatteryVoltageToPercentage(ctx.Device.BatteryVoltage)
			deviceData["percent_charged"] = batteryPercentage
		}

		// Add WiFi information if available  
		if ctx.Device.RSSI != 0 {
			wifiPercentage := plugins.RSSIToWifiStrengthPercentage(ctx.Device.RSSI)
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
				// Handle both legacy single URL format and new URLs array format
				if urls, ok := pollingConfig["urls"].([]interface{}); ok && len(urls) > 0 {
					// New format with URLs array
					if urlObj, ok := urls[0].(map[string]interface{}); ok {
						if url, ok := urlObj["url"].(string); ok {
							pluginSettings["polling_url"] = url
						}
						if headers, ok := urlObj["headers"]; ok {
							pluginSettings["polling_headers"] = fmt.Sprintf("%v", headers)
						}
					}
				} else {
					// Legacy format with single URL
					if url, ok := pollingConfig["url"].(string); ok {
						pluginSettings["polling_url"] = url
					}
					if headers, ok := pollingConfig["headers"].(string); ok {
						pluginSettings["polling_headers"] = headers
					}
				}
			}
		}
	}
	
	// Add default plugin configuration (these might come from plugin definition or defaults)
	pluginSettings["dark_mode"] = "no"
	pluginSettings["no_screen_padding"] = "no"
	
	// Add custom_fields_values containing form field values (TRMNL compatibility)
	pluginSettings["custom_fields_values"] = formFieldValues
	
	trmnlData["plugin_settings"] = pluginSettings
	
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
	
	// Use the private plugin renderer service with client-side LiquidJS
	htmlRenderer := NewPrivatePluginRenderer()
	html, err := htmlRenderer.RenderToClientSideHTML(RenderOptions{
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

// GetInstance returns the plugin instance
func (p *PrivatePlugin) GetInstance() *database.PluginInstance {
	return p.instance
}


// Register the private plugin factory when this package is imported
func init() {
	plugins.RegisterPrivatePluginFactory(NewPrivatePlugin)
}