package mashup

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/plugins/private"
)

// MashupRenderer handles combining child plugin outputs into mashup layouts
type MashupRenderer struct {
	layout       string
	childResults map[string]ChildRenderResult
	slotConfig   []database.MashupSlotInfo
}

// NewMashupRenderer creates a new mashup renderer
func NewMashupRenderer(layout string, childResults map[string]ChildRenderResult) *MashupRenderer {
	// Generate slot configuration for this layout
	service := database.NewMashupService(database.GetDB())
	slots, _ := service.GetSlotMetadata(layout)
	
	return &MashupRenderer{
		layout:       layout,
		childResults: childResults,
		slotConfig:   slots,
	}
}

// RenderMashup combines child plugin outputs into a single mashup HTML
func (r *MashupRenderer) RenderMashup(ctx plugins.PluginContext) (string, error) {
	logging.Info("[MASHUP_RENDERER] Starting mashup render", "layout", r.layout, "children_count", len(r.childResults))
	
	// Generate individual child HTML
	childHTML := make(map[string]string)
	
	for slot, result := range r.childResults {
		if !result.Success {
			// Use error HTML
			childHTML[slot] = r.wrapChildContent(result.HTML, slot, "error")
			continue
		}
		
		// Render successful child plugin
		html, err := r.renderChildPlugin(result, ctx, slot)
		if err != nil {
			logging.Error("[MASHUP_RENDERER] Failed to render child", "slot", slot, "error", err)
			errorHTML := fmt.Sprintf("<div class='mashup-error'>Render error: %s</div>", err.Error())
			childHTML[slot] = r.wrapChildContent(errorHTML, slot, "error")
		} else {
			childHTML[slot] = r.wrapChildContent(html, slot, "success")
		}
	}
	
	// Combine into final mashup layout
	return r.combineIntoLayout(childHTML, ctx), nil
}

// renderChildPlugin renders a child plugin appropriately based on its type
func (r *MashupRenderer) renderChildPlugin(result ChildRenderResult, ctx plugins.PluginContext, slot string) (string, error) {
	if result.Instance.PluginDefinition.PluginType == "private" {
		return r.renderPrivateChildPlugin(result, ctx, slot)
	} else if result.Instance.PluginDefinition.PluginType == "system" {
		return r.renderSystemChildPlugin(result, ctx, slot)
	}
	
	return "", fmt.Errorf("unsupported child plugin type: %s", result.Instance.PluginDefinition.PluginType)
}

// renderPrivateChildPlugin renders a private plugin child by accessing its stored data directly
func (r *MashupRenderer) renderPrivateChildPlugin(result ChildRenderResult, ctx plugins.PluginContext, slot string) (string, error) {
	// Access stored data directly instead of calling Process() to avoid duplicate API calls
	def := result.Instance.PluginDefinition
	instanceID := result.Instance.ID.String()
	
	// Get the shared markup if available
	sharedMarkup := ""
	if def.SharedMarkup != nil {
		sharedMarkup = *def.SharedMarkup
	}
	
	if def.MarkupFull == nil {
		return "", fmt.Errorf("private child plugin has no template markup")
	}
	
	// Build base template data structure (TRMNL compatibility)
	templateData := make(map[string]interface{})
	
	// Add system information - Unix timestamp
	systemData := map[string]interface{}{
		"timestamp_utc": time.Now().Unix(),
	}
	
	// Add device information
	deviceData := map[string]interface{}{
		"friendly_id": ctx.Device.FriendlyID,
		"width":       ctx.Device.DeviceModel.ScreenWidth,
		"height":      ctx.Device.DeviceModel.ScreenHeight,
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
	
	// Add user information if available
	userData := map[string]interface{}{}
	if ctx.User != nil {
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
			fullName = ctx.User.Username
		}
		
		// Calculate timezone info
		utcOffset := int64(0)
		locale := "en"
		timezone := "UTC"
		timezoneFriendly := "UTC"
		
		if ctx.User.Timezone != "" {
			timezone = ctx.User.Timezone
			timezoneFriendly = timezone // TODO: Get friendly name
			if loc, err := time.LoadLocation(ctx.User.Timezone); err == nil {
				_, offset := time.Now().In(loc).Zone()
				utcOffset = int64(offset)
			}
		}
		
		if ctx.User.Locale != "" && len(ctx.User.Locale) >= 2 {
			locale = ctx.User.Locale[:2]
		}
		
		userData = map[string]interface{}{
			"name":           fullName,
			"first_name":     firstName,
			"last_name":      lastName,
			"locale":         locale,
			"time_zone":      timezoneFriendly,
			"time_zone_iana": timezone,
			"utc_offset":     utcOffset,
		}
	}
	
	// Parse form field values from instance settings
	var formFieldValues map[string]interface{}
	if result.Instance.Settings != nil {
		if err := json.Unmarshal(result.Instance.Settings, &formFieldValues); err != nil {
			formFieldValues = make(map[string]interface{})
		}
	} else {
		formFieldValues = make(map[string]interface{})
	}
	
	// Add plugin settings
	pluginSettings := map[string]interface{}{
		"instance_name":        result.Instance.Name,
		"custom_fields_values": formFieldValues,
		"dark_mode":            "no",
		"no_screen_padding":    "no",
	}
	
	// Add data strategy if available
	if def.DataStrategy != nil {
		pluginSettings["strategy"] = *def.DataStrategy
	}
	
	// Fetch external data based on data strategy - accessing stored data directly
	switch dataStrategy := def.DataStrategy; {
	case dataStrategy != nil && *dataStrategy == "polling":
		// Access stored polling data
		pollingService := database.NewPollingDataService(database.GetDB())
		if storedData, err := pollingService.GetPollingDataTemplate(instanceID); err == nil {
			// Merge stored polling data into template data
			for key, value := range storedData {
				templateData[key] = value
			}
			logging.Debug("[MASHUP_RENDERER] Using stored polling data for child", "instance_id", instanceID, "slot", slot)
		} else {
			logging.Warn("[MASHUP_RENDERER] No stored polling data available for child", "instance_id", instanceID, "slot", slot, "error", err)
		}
		
	case dataStrategy != nil && *dataStrategy == "webhook":
		// Access stored webhook data
		webhookService := database.NewWebhookService(database.GetDB())
		if webhookData, err := webhookService.GetWebhookDataTemplate(instanceID); err == nil && webhookData != nil {
			// Merge stored webhook data into template data
			for key, value := range webhookData {
				templateData[key] = value
			}
			logging.Debug("[MASHUP_RENDERER] Using stored webhook data for child", "instance_id", instanceID, "slot", slot)
		} else {
			logging.Warn("[MASHUP_RENDERER] No stored webhook data available for child", "instance_id", instanceID, "slot", slot, "error", err)
		}
		
	case dataStrategy != nil && *dataStrategy == "static":
		// Static strategy uses only form fields and TRMNL struct - no external data
		logging.Debug("[MASHUP_RENDERER] Using static data strategy for child", "instance_id", instanceID, "slot", slot)
	}
	
	// Build final TRMNL data structure
	trmnlData := map[string]interface{}{
		"system":          systemData,
		"device":          deviceData,
		"user":            userData,
		"plugin_settings": pluginSettings,
	}
	templateData["trmnl"] = trmnlData
	
	// Use the private plugin renderer to generate the HTML
	htmlRenderer := private.NewPrivatePluginRenderer()
	html, err := htmlRenderer.RenderToClientSideHTML(private.RenderOptions{
		SharedMarkup:      sharedMarkup,
		LayoutTemplate:    *def.MarkupFull,
		Data:              templateData,
		Width:             ctx.Device.DeviceModel.ScreenWidth,
		Height:            ctx.Device.DeviceModel.ScreenHeight,
		PluginName:        def.Name,
		InstanceID:        instanceID,
		InstanceName:      result.Instance.Name,
		RemoveBleedMargin: def.RemoveBleedMargin != nil && *def.RemoveBleedMargin,
		EnableDarkMode:    def.EnableDarkMode != nil && *def.EnableDarkMode,
	})
	
	if err != nil {
		return "", fmt.Errorf("failed to render private child plugin HTML: %w", err)
	}
	
	return html, nil
}

// renderSystemChildPlugin renders a system plugin child
func (r *MashupRenderer) renderSystemChildPlugin(result ChildRenderResult, ctx plugins.PluginContext, slot string) (string, error) {
	// For system plugins, we expect they return HTML or image data
	// For now, assume they return HTML that we can use directly
	
	if htmlContent, ok := plugins.GetHTMLContent(result.Response); ok {
		return htmlContent, nil
	}
	
	// If not HTML, try to handle as image response (not implemented yet)
	return "", fmt.Errorf("system child plugin did not return HTML content")
}

// getSlotInfo returns slot information for a given slot position
func (r *MashupRenderer) getSlotInfo(slotPosition string) *database.MashupSlotInfo {
	for i := range r.slotConfig {
		if r.slotConfig[i].Position == slotPosition {
			return &r.slotConfig[i]
		}
	}
	return nil
}

// wrapChildContent wraps child content in appropriate mashup view classes
func (r *MashupRenderer) wrapChildContent(content string, slot string, status string) string {
	slotInfo := r.getSlotInfo(slot)
	if slotInfo == nil {
		return fmt.Sprintf("<div class='mashup-error'>Unknown slot: %s</div>", slot)
	}
	
	// Apply mashup view classes
	classes := []string{"view", slotInfo.ViewClass}
	if status == "error" {
		classes = append(classes, "mashup-child-error")
	}
	
	return fmt.Sprintf(`<div class="%s">
		<div class="layout">
			%s
		</div>
	</div>`, strings.Join(classes, " "), content)
}

// combineIntoLayout combines child HTML into the final mashup layout structure
func (r *MashupRenderer) combineIntoLayout(childHTML map[string]string, ctx plugins.PluginContext) string {
	// Build the mashup container
	var children []string
	
	// Add children in the correct order based on layout
	for _, slot := range r.slotConfig {
		if html, exists := childHTML[slot.Position]; exists {
			children = append(children, html)
		} else {
			// Empty slot
			placeholder := fmt.Sprintf(`<div class="view %s">
				<div class="layout">
					<div class="mashup-empty-slot">
						<span class="label">Empty Slot: %s</span>
					</div>
				</div>
			</div>`, slot.ViewClass, slot.DisplayName)
			children = append(children, placeholder)
		}
	}
	
	// Combine into mashup structure
	mashupContent := strings.Join(children, "\n")
	
	// Wrap in mashup container with proper CSS classes
	finalHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Mashup</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@100..900&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://usetrmnl.com/css/latest/plugins.css">
    <style>
        body { 
            width: %dpx; 
            height: %dpx; 
            margin: 0; 
            padding: 0;
        }
        .mashup-error {
            padding: 10px;
            background: #fee;
            border: 1px solid #fcc;
            color: #c00;
            font-family: monospace;
            font-size: 12px;
        }
        .mashup-empty-slot {
            display: flex;
            align-items: center;
            justify-content: center;
            height: 100%%;
            background: #f5f5f5;
            border: 2px dashed #ccc;
            color: #999;
        }
        .mashup-child-error {
            background: rgba(255, 0, 0, 0.1);
        }
    </style>
</head>
<body>
    <div class="environment trmnl">
        <div class="screen">
            <div class="mashup mashup--%s">
                %s
            </div>
        </div>
    </div>
</body>
</html>`,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
		r.layout,
		mashupContent)
	
	return finalHTML
}

// createTRNMLData creates TRMNL-compatible data structure for child plugins
func (r *MashupRenderer) createTRNMLData(ctx plugins.PluginContext, instance *database.PluginInstance) map[string]interface{} {
	trmnlData := make(map[string]interface{})
	
	// Add system information
	systemData := map[string]interface{}{
		"timestamp_utc": ctx.Device.LastSeen.Unix(),
	}
	trmnlData["system"] = systemData
	
	// Add device information
	if ctx.Device != nil {
		deviceData := map[string]interface{}{
			"friendly_id": ctx.Device.FriendlyID,
		}
		
		if ctx.Device.DeviceModel != nil {
			deviceData["width"] = ctx.Device.DeviceModel.ScreenWidth
			deviceData["height"] = ctx.Device.DeviceModel.ScreenHeight
		}
		
		if ctx.Device.BatteryVoltage > 0 {
			batteryPercentage := plugins.BatteryVoltageToPercentage(ctx.Device.BatteryVoltage)
			deviceData["percent_charged"] = batteryPercentage
		}
		
		if ctx.Device.RSSI != 0 {
			wifiPercentage := plugins.RSSIToWifiStrengthPercentage(ctx.Device.RSSI)
			deviceData["wifi_strength"] = wifiPercentage
		}
		
		trmnlData["device"] = deviceData
	}
	
	// Add user information
	if ctx.User != nil {
		userData := map[string]interface{}{
			"name":       ctx.User.Username,
			"first_name": ctx.User.FirstName,
			"last_name":  ctx.User.LastName,
			"locale":     ctx.User.Locale,
		}
		trmnlData["user"] = userData
	}
	
	// Add plugin settings
	pluginSettings := map[string]interface{}{
		"instance_name": instance.Name,
		"strategy":      "mashup_child",
		"dark_mode":     "no",
		"no_screen_padding": "no",
	}
	
	// Add form field values from instance settings
	if len(instance.Settings) > 0 {
		var settingsMap map[string]interface{}
		if err := json.Unmarshal(instance.Settings, &settingsMap); err == nil {
			pluginSettings["custom_fields_values"] = settingsMap
		}
	}
	
	trmnlData["plugin_settings"] = pluginSettings
	
	return trmnlData
}