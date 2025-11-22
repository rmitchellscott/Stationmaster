package screenshot

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"net/url"
	"strings"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/imageprocessing"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
	"github.com/rmitchellscott/stationmaster/internal/utils"
)

// ScreenshotPlugin implements a data plugin that captures screenshots of web pages
type ScreenshotPlugin struct{}

// Type returns the plugin type identifier
func (p *ScreenshotPlugin) Type() string {
	return "screenshot"
}

// PluginType returns that this is an image plugin
func (p *ScreenshotPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the human-readable name
func (p *ScreenshotPlugin) Name() string {
	return "Screenshot"
}

// Description returns the plugin description
func (p *ScreenshotPlugin) Description() string {
	return "Captures a screenshot of any website and displays it on your device"
}

// Author returns the plugin author
func (p *ScreenshotPlugin) Author() string {
	return "Stationmaster"
}

// Version returns the plugin version
func (p *ScreenshotPlugin) Version() string {
	return "1.0.0"
}

// RequiresProcessing returns true since this plugin needs image processing
func (p *ScreenshotPlugin) RequiresProcessing() bool {
	return true
}

// ConfigSchema returns the JSON schema for configuration
func (p *ScreenshotPlugin) ConfigSchema() string {
	return `{
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"title": "Website URL",
				"description": "The URL of the website to screenshot",
				"format": "uri",
				"examples": ["https://example.com", "https://news.ycombinator.com"]
			},
			"wait_time": {
				"type": "integer",
				"title": "Wait Time (seconds)",
				"description": "How long to wait for the page to load before taking screenshot",
				"minimum": 0,
				"maximum": 30,
				"default": 3
			},
			"headers": {
				"type": "string",
				"title": "HTTP Headers",
				"description": "Custom HTTP headers in format: key1=value1&key2=value2 (use %3D for = in values)",
				"examples": ["authorization=bearer token123", "authorization=bearer%20jwt%3D%3D&content-type=application/json"]
			}
		},
		"required": ["url"]
	}`
}

// parseHeaders parses header string in format "key1=value1&key2=value2" into map
func parseHeaders(headerStr string) (map[string]string, error) {
	if headerStr == "" {
		return nil, nil
	}
	
	headers := make(map[string]string)
	pairs := strings.Split(headerStr, "&")
	
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header format: %s (expected key=value)", pair)
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// URL decode the value to handle %3D encoding
		decodedValue, err := url.QueryUnescape(value)
		if err != nil {
			return nil, fmt.Errorf("failed to decode header value %s: %w", value, err)
		}
		
		if key == "" {
			return nil, fmt.Errorf("header key cannot be empty")
		}
		
		headers[key] = decodedValue
	}
	
	return headers, nil
}

// Validate validates the plugin settings
func (p *ScreenshotPlugin) Validate(settings map[string]interface{}) error {
	url, ok := settings["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("url is required and must be a valid URL")
	}

	// Basic URL validation
	if len(url) < 7 || (url[:7] != "http://" && url[:8] != "https://") {
		return fmt.Errorf("url must be a valid HTTP or HTTPS URL")
	}

	if err := utils.ValidateURL(url); err != nil {
		return fmt.Errorf("url validation failed: %w", err)
	}

	// Validate wait_time if provided
	if waitTime, exists := settings["wait_time"]; exists {
		if waitTimeFloat, ok := waitTime.(float64); ok {
			if waitTimeFloat < 0 || waitTimeFloat > 30 {
				return fmt.Errorf("wait time must be between 0 and 30 seconds - this controls how long to wait for the page to fully load before taking the screenshot")
			}
		} else {
			return fmt.Errorf("wait time must be a number (seconds)")
		}
	}

	// Validate headers if provided
	if headerStr, exists := settings["headers"]; exists {
		if headerStrValue, ok := headerStr.(string); ok {
			_, err := parseHeaders(headerStrValue)
			if err != nil {
				return fmt.Errorf("invalid headers format: %w", err)
			}
		} else if headerStr != "" {
			return fmt.Errorf("headers must be a string")
		}
	}

	return nil
}

// Process executes the plugin logic
func (p *ScreenshotPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Get URL from settings
	url := ctx.GetStringSetting("url", "")
	if url == "" {
		return plugins.CreateErrorResponse("URL is required"), 
			fmt.Errorf("url setting is missing")
	}

	// Get optional settings
	waitTimeSeconds := ctx.GetIntSetting("wait_time", 3)
	headerStr := ctx.GetStringSetting("headers", "")
	
	// Parse headers if provided
	headers, err := parseHeaders(headerStr)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Invalid headers: %v", err)),
			fmt.Errorf("failed to parse headers: %w", err)
	}

	// Validate that we have device model information for proper sizing
	if ctx.Device == nil || ctx.Device.DeviceModel == nil {
		return plugins.CreateErrorResponse("Device model information not available"),
			fmt.Errorf("device model is required for screenshot processing")
	}

	// Create browserless renderer
	renderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to create renderer: %v", err)),
			fmt.Errorf("failed to create browserless renderer: %w", err)
	}
	defer renderer.Close()

	// Capture screenshot using browserless with device resolution
	screenshotCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	imageData, err := renderer.CaptureScreenshot(
		screenshotCtx, 
		url, 
		ctx.Device.DeviceModel.ScreenWidth, 
		ctx.Device.DeviceModel.ScreenHeight, 
		waitTimeSeconds,
		headers,
	)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to capture screenshot: %v", err)),
			fmt.Errorf("failed to capture screenshot of %s: %w", url, err)
	}

	// Convert screenshot bytes to image.Image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to decode screenshot: %v", err)),
			fmt.Errorf("failed to decode screenshot image: %w", err)
	}

	// Process image for the device using existing image processing pipeline
	processedImg, err := imageprocessing.ProcessForDevice(img, ctx.Device.DeviceModel)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to process image: %v", err)),
			fmt.Errorf("failed to process image for device: %w", err)
	}

	// Convert processed image to PNG bytes with proper bit depth
	var pngData []byte
	if ctx.Device.DeviceModel.BitDepth <= 2 {
		// Use custom PNG encoder for 1-bit and 2-bit images to ensure proper bit depth
		pngData, err = imageprocessing.EncodePalettedPNG(processedImg, ctx.Device.DeviceModel.BitDepth)
		if err != nil {
			return plugins.CreateErrorResponse(fmt.Sprintf("Failed to encode processed image with custom encoder: %v", err)),
				fmt.Errorf("failed to encode processed image with custom encoder: %w", err)
		}
	} else {
		// Use standard PNG encoder for higher bit depths
		var buf bytes.Buffer
		encoder := &png.Encoder{
			CompressionLevel: png.BestCompression,
		}
		err = encoder.Encode(&buf, processedImg)
		if err != nil {
			return plugins.CreateErrorResponse(fmt.Sprintf("Failed to encode processed image: %v", err)),
				fmt.Errorf("failed to encode processed image: %w", err)
		}
		pngData = buf.Bytes()
	}

	// Generate filename
	filename := fmt.Sprintf("screenshot_%s_%dx%d.png",
		time.Now().UTC().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)

	// Return image data response (RenderWorker will handle storage)
	return plugins.CreateImageDataResponse(pngData, filename), nil
}

// Register the plugin when this package is imported
func init() {
	plugins.Register(&ScreenshotPlugin{})
}