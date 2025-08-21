package core_proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// CoreProxyPlugin implements a plugin that proxies requests to TRMNL's official server
type CoreProxyPlugin struct{}

// Type returns the plugin type identifier
func (p *CoreProxyPlugin) Type() string {
	return "core_proxy"
}

// PluginType returns that this is an image plugin
func (p *CoreProxyPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the human-readable name
func (p *CoreProxyPlugin) Name() string {
	return "TRMNL Core Proxy"
}

// Description returns the plugin description
func (p *CoreProxyPlugin) Description() string {
	return "Proxies requests to TRMNL's official server using your TRMNL device credentials"
}

// Author returns the plugin author
func (p *CoreProxyPlugin) Author() string {
	return "Stationmaster Team"
}

// Version returns the plugin version
func (p *CoreProxyPlugin) Version() string {
	return "1.0.0"
}

// RequiresProcessing returns false since this plugin returns direct URLs
func (p *CoreProxyPlugin) RequiresProcessing() bool {
	return false
}

// ConfigSchema returns the JSON schema for configuration
func (p *CoreProxyPlugin) ConfigSchema() string {
	return `{
		"type": "object",
		"properties": {
			"device_mac": {
				"type": "string",
				"title": "TRMNL Device MAC",
				"description": "MAC address of your TRMNL device",
				"pattern": "^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"
			},
			"access_token": {
				"type": "string",
				"title": "TRMNL Access Token",
				"description": "Your TRMNL device access token"
			},
			"timeout_seconds": {
				"type": "number",
				"title": "Timeout (seconds)",
				"description": "Request timeout in seconds (max 15)",
				"default": 5,
				"minimum": 1,
				"maximum": 15
			},
			"pass_through_refresh_rate": {
				"type": "boolean",
				"title": "Pass through refresh rate from core",
				"description": "Use refresh rate from TRMNL core response",
				"default": false
			}
		},
		"required": ["device_mac", "access_token"]
	}`
}

// Validate validates the plugin settings
func (p *CoreProxyPlugin) Validate(settings map[string]interface{}) error {
	deviceMac, ok := settings["device_mac"].(string)
	if !ok || deviceMac == "" {
		return fmt.Errorf("device_mac is required")
	}

	accessToken, ok := settings["access_token"].(string)
	if !ok || accessToken == "" {
		return fmt.Errorf("access_token is required")
	}

	if timeout, ok := settings["timeout_seconds"]; ok {
		if timeoutFloat, ok := timeout.(float64); ok {
			if timeoutFloat < 1 || timeoutFloat > 15 {
				return fmt.Errorf("timeout_seconds must be between 1 and 15")
			}
		}
	}

	return nil
}

// Process executes the plugin logic
func (p *CoreProxyPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Get TRMNL device MAC and access token from settings
	deviceMac := ctx.GetStringSetting("device_mac", "")
	if deviceMac == "" {
		return plugins.CreateErrorResponse("device_mac not configured"),
			fmt.Errorf("device_mac not configured in plugin settings")
	}

	accessToken := ctx.GetStringSetting("access_token", "")
	if accessToken == "" {
		return plugins.CreateErrorResponse("access_token not configured"),
			fmt.Errorf("access_token not configured in plugin settings")
	}

	// Get timeout (default to 5 seconds)
	timeoutSeconds := ctx.GetIntSetting("timeout_seconds", 5)
	if timeoutSeconds > 15 {
		timeoutSeconds = 15 // Cap at 15 seconds
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	// Create request to TRMNL's API
	req, err := http.NewRequest("GET", "https://usetrmnl.com/api/display", nil)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to create request: %v", err)),
			fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers that TRMNL expects
	req.Header.Set("ID", deviceMac)
	req.Header.Set("Access-Token", accessToken)

	// Forward device status headers if available from our local device
	if ctx.Device.FirmwareVersion != "" {
		req.Header.Set("Fw-Version", ctx.Device.FirmwareVersion)
	}
	if ctx.Device.BatteryVoltage > 0 {
		req.Header.Set("Battery-Voltage", fmt.Sprintf("%.2f", ctx.Device.BatteryVoltage))
	}
	if ctx.Device.RSSI != 0 {
		req.Header.Set("Rssi", fmt.Sprintf("%d", ctx.Device.RSSI))
	}
	if ctx.Device.RefreshRate > 0 {
		req.Header.Set("Refresh-Rate", fmt.Sprintf("%d", ctx.Device.RefreshRate))
	}

	// Make request to TRMNL
	resp, err := client.Do(req)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to fetch from TRMNL API: %v", err)),
			fmt.Errorf("failed to fetch from TRMNL API: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return plugins.CreateErrorResponse(fmt.Sprintf("TRMNL API returned status %d", resp.StatusCode)),
			fmt.Errorf("TRMNL API returned status %d", resp.StatusCode)
	}

	// Parse JSON response from TRMNL
	var trmnlResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&trmnlResponse); err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to parse TRMNL response: %v", err)),
			fmt.Errorf("failed to parse TRMNL response: %w", err)
	}

	// Extract image URL
	var imageURL string
	if url, ok := trmnlResponse["image_url"].(string); ok {
		imageURL = url
	} else if url, ok := trmnlResponse["url"].(string); ok {
		imageURL = url
	} else {
		return plugins.CreateErrorResponse("No image URL found in TRMNL response"),
			fmt.Errorf("no image URL found in TRMNL response")
	}

	// Extract filename with fallback
	filename := time.Now().Format("2006-01-02T15:04:05")
	if fname, ok := trmnlResponse["filename"].(string); ok && fname != "" {
		filename = fname
	}

	// Check if we should pass through refresh rate from core
	passThrough := ctx.GetBoolSetting("pass_through_refresh_rate", false)

	if passThrough {
		// Extract refresh rate with fallback
		refreshRate := 3600 // Default 1 hour
		if rate, ok := trmnlResponse["refresh_rate"]; ok {
			if rateFloat, ok := rate.(float64); ok {
				refreshRate = int(rateFloat)
			} else if rateStr, ok := rate.(string); ok {
				var parsedRate int
				if _, err := fmt.Sscanf(rateStr, "%d", &parsedRate); err == nil {
					refreshRate = parsedRate
				}
			}
		}
		return plugins.CreateImageResponse(imageURL, filename, refreshRate), nil
	} else {
		// Don't use refresh rate from core
		return plugins.CreateImageResponseWithoutRefresh(imageURL, filename), nil
	}
}

// Register the plugin when this package is imported
func init() {
	plugins.Register(&CoreProxyPlugin{})
}
