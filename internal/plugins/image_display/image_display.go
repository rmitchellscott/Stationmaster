package image_display

import (
	"bytes"
	"fmt"
	"image/png"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/imageprocessing"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/utils"
)

// ImageDisplayPlugin implements an image plugin that displays a single image from a URL
type ImageDisplayPlugin struct{}

// Type returns the plugin type identifier
func (p *ImageDisplayPlugin) Type() string {
	return "image_display"
}

// PluginType returns that this is an image plugin
func (p *ImageDisplayPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the human-readable name
func (p *ImageDisplayPlugin) Name() string {
	return "Image Display"
}

// Description returns the plugin description
func (p *ImageDisplayPlugin) Description() string {
	return "Displays a single image from a URL, automatically resized and dithered for your device"
}

// Author returns the plugin author
func (p *ImageDisplayPlugin) Author() string {
	return "Stationmaster"
}

// Version returns the plugin version
func (p *ImageDisplayPlugin) Version() string {
	return "1.0.0"
}

// RequiresProcessing returns true since this plugin needs image processing
func (p *ImageDisplayPlugin) RequiresProcessing() bool {
	return true
}

// ConfigSchema returns the JSON schema for configuration
func (p *ImageDisplayPlugin) ConfigSchema() string {
	return `{
		"type": "object",
		"properties": {
			"image_url": {
				"type": "string",
				"title": "Image URL",
				"description": "The URL of the image to display",
				"format": "uri",
				"examples": ["https://example.com/image.png"]
			}
		},
		"required": ["image_url"]
	}`
}

// Validate validates the plugin settings
func (p *ImageDisplayPlugin) Validate(settings map[string]interface{}) error {
	imageURL, ok := settings["image_url"].(string)
	if !ok || imageURL == "" {
		return fmt.Errorf("image_url is required and must be a valid URL")
	}

	// Basic URL validation
	if len(imageURL) < 7 || (imageURL[:7] != "http://" && imageURL[:8] != "https://") {
		return fmt.Errorf("image_url must be a valid HTTP or HTTPS URL")
	}

	if err := utils.ValidateURL(imageURL); err != nil {
		return fmt.Errorf("image_url validation failed: %w", err)
	}

	return nil
}

// Process executes the plugin logic
func (p *ImageDisplayPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Get image URL from settings
	imageURL := ctx.GetStringSetting("image_url", "")
	if imageURL == "" {
		return plugins.CreateErrorResponse("Image URL is required"),
			fmt.Errorf("image_url setting is missing")
	}

	// Validate that we have device model information for processing
	if ctx.Device == nil || ctx.Device.DeviceModel == nil {
		return plugins.CreateErrorResponse("Device model information not available"),
			fmt.Errorf("device model is required for image processing")
	}

	// Load image from URL
	processingOptions := imageprocessing.DefaultProcessingOptions()
	img, _, err := imageprocessing.LoadImageFromURL(imageURL, processingOptions.Timeout)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to load image: %v", err)),
			fmt.Errorf("failed to load image from URL %s: %w", imageURL, err)
	}

	// Process image for the device
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
	filename := fmt.Sprintf("image_display_%s_%dx%d.png",
		time.Now().UTC().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)

	// Return image data response (RenderWorker will handle storage)
	return plugins.CreateImageDataResponse(pngData, filename), nil
}

// Register the plugin when this package is imported
func init() {
	plugins.Register(&ImageDisplayPlugin{})
}
