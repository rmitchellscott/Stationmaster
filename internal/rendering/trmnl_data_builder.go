package rendering

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/utils"
)

// TRNMLDataBuilder creates the standardized TRMNL data structure used by both private and mashup plugins
type TRNMLDataBuilder struct{}

// NewTRNMLDataBuilder creates a new TRMNL data builder instance
func NewTRNMLDataBuilder() *TRNMLDataBuilder {
	return &TRNMLDataBuilder{}
}

// BuildTRNMLData creates the complete TRMNL data structure for a plugin instance
// This is the shared logic extracted from both private and mashup plugins
func (b *TRNMLDataBuilder) BuildTRNMLData(ctx plugins.PluginContext, instance *database.PluginInstance, formFieldValues map[string]interface{}) map[string]interface{} {
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
		"instance_name": instance.Name,
	}
	
	// Add data strategy if available
	if instance.PluginDefinition.DataStrategy != nil {
		pluginSettings["strategy"] = *instance.PluginDefinition.DataStrategy
	}
	
	// Add polling config if this is a polling plugin
	if instance.PluginDefinition.DataStrategy != nil && *instance.PluginDefinition.DataStrategy == "polling" {
		pluginSettings["polling_url"] = ""
		pluginSettings["polling_headers"] = ""
		
		// Parse polling config if available
		if instance.PluginDefinition.PollingConfig != nil {
			var pollingConfig map[string]interface{}
			if err := json.Unmarshal(instance.PluginDefinition.PollingConfig, &pollingConfig); err == nil {
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
	
	// Add default plugin configuration
	pluginSettings["dark_mode"] = "no"
	pluginSettings["no_screen_padding"] = "no"
	
	// Add custom_fields_values containing form field values (TRMNL compatibility)
	pluginSettings["custom_fields_values"] = formFieldValues
	
	trmnlData["plugin_settings"] = pluginSettings
	
	return trmnlData
}