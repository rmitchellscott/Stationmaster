package redirect

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// RedirectPlugin implements a plugin that fetches JSON from external endpoints
type RedirectPlugin struct{}

// Type returns the plugin type identifier
func (p *RedirectPlugin) Type() string {
	return "redirect"
}

// PluginType returns that this is an image plugin
func (p *RedirectPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the human-readable name
func (p *RedirectPlugin) Name() string {
	return "Redirect"
}

// Description returns the plugin description
func (p *RedirectPlugin) Description() string {
	return "Fetches JSON response from an external endpoint and returns the image URL"
}

// Author returns the plugin author
func (p *RedirectPlugin) Author() string {
	return "Stationmaster Team"
}

// Version returns the plugin version
func (p *RedirectPlugin) Version() string {
	return "1.0.0"
}

// RequiresProcessing returns false since this plugin returns direct URLs
func (p *RedirectPlugin) RequiresProcessing() bool {
	return false
}

// ConfigSchema returns the JSON schema for configuration
func (p *RedirectPlugin) ConfigSchema() string {
	return `{
		"type": "object",
		"properties": {
			"endpoint_url": {
				"type": "string",
				"title": "Endpoint URL",
				"description": "The URL to fetch JSON data from",
				"format": "uri"
			},
			"timeout_seconds": {
				"type": "number",
				"title": "Timeout (seconds)",
				"description": "Request timeout in seconds (max 10)",
				"default": 2,
				"minimum": 1,
				"maximum": 10
			}
		},
		"required": ["endpoint_url"]
	}`
}

// Validate validates the plugin settings
func (p *RedirectPlugin) Validate(settings map[string]interface{}) error {
	endpointURL, ok := settings["endpoint_url"].(string)
	if !ok || endpointURL == "" {
		return fmt.Errorf("endpoint_url is required")
	}

	if timeout, ok := settings["timeout_seconds"]; ok {
		if timeoutFloat, ok := timeout.(float64); ok {
			if timeoutFloat < 1 || timeoutFloat > 10 {
				return fmt.Errorf("timeout_seconds must be between 1 and 10")
			}
		}
	}

	return nil
}

// Process executes the plugin logic
func (p *RedirectPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Get endpoint URL from settings
	endpointURL := ctx.GetStringSetting("endpoint_url", "")
	if endpointURL == "" {
		return plugins.CreateErrorResponse("endpoint_url not configured"), 
			fmt.Errorf("endpoint_url not configured in plugin settings")
	}

	// Get timeout (default to 2 seconds)
	timeoutSeconds := ctx.GetIntSetting("timeout_seconds", 2)
	if timeoutSeconds > 10 {
		timeoutSeconds = 10 // Cap at 10 seconds
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	// Fetch JSON from endpoint
	resp, err := client.Get(endpointURL)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to fetch from endpoint: %v", err)),
			fmt.Errorf("failed to fetch from endpoint: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return plugins.CreateErrorResponse(fmt.Sprintf("Endpoint returned status %d", resp.StatusCode)),
			fmt.Errorf("endpoint returned status %d", resp.StatusCode)
	}

	// Parse JSON response
	var pluginResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&pluginResponse); err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to parse JSON response: %v", err)),
			fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Extract image URL
	var imageURL string
	if url, ok := pluginResponse["url"].(string); ok {
		imageURL = url
	} else if url, ok := pluginResponse["image_url"].(string); ok {
		imageURL = url
	} else {
		return plugins.CreateErrorResponse("No image URL found in response"),
			fmt.Errorf("no image URL found in response")
	}

	// Extract filename with fallback
	filename := "display.png"
	if fname, ok := pluginResponse["filename"].(string); ok && fname != "" {
		filename = fname
	}

	// Extract refresh rate with fallback
	refreshRate := 3600 // Default 1 hour
	if rate, ok := pluginResponse["refresh_rate"]; ok {
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
}

// Register the plugin when this package is imported
func init() {
	plugins.Register(&RedirectPlugin{})
}