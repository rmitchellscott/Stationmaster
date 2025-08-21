package imageprocessing

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"net/http"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
)

// ProcessingOptions allows customization of the image processing pipeline
type ProcessingOptions struct {
	Timeout time.Duration
}

// DefaultProcessingOptions returns sensible defaults for image processing
func DefaultProcessingOptions() ProcessingOptions {
	return ProcessingOptions{
		Timeout: 30 * time.Second,
	}
}

// ProcessForDevice applies the full processing pipeline to prepare an image for a specific device
func ProcessForDevice(img image.Image, deviceModel *database.DeviceModel) (image.Image, error) {
	return ProcessForDeviceWithOptions(img, deviceModel, DefaultProcessingOptions())
}

// ProcessForDeviceWithOptions applies the full processing pipeline with custom options
func ProcessForDeviceWithOptions(img image.Image, deviceModel *database.DeviceModel, options ProcessingOptions) (image.Image, error) {
	if img == nil {
		return nil, fmt.Errorf("input image is nil")
	}
	if deviceModel == nil {
		return nil, fmt.Errorf("device model is nil")
	}

	// Step 1: Resize to fill device dimensions while preserving aspect ratio
	resized := ResizeToFill(img, deviceModel.ScreenWidth, deviceModel.ScreenHeight)

	// Step 2: Convert to grayscale if not already
	grayscale := ToGrayscale(resized)

	// Step 3: Apply Floyd-Steinberg dithering based on device bit depth
	dithered := DitherFloydSteinberg(grayscale, deviceModel.BitDepth)

	return dithered, nil
}

// LoadImageFromURL downloads and decodes an image from a URL
func LoadImageFromURL(url string, timeout time.Duration) (image.Image, string, error) {
	client := &http.Client{Timeout: timeout}
	
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}

	// Try to decode the image directly from the response body
	img, format, err := image.Decode(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	return img, format, nil
}

// GetImageBounds returns the bounds of an image
func GetImageBounds(img image.Image) image.Rectangle {
	return img.Bounds()
}

// CreateImageCanvas creates a new RGBA image with the specified dimensions
func CreateImageCanvas(width, height int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, width, height))
}

// CopyImage creates a copy of an image
func CopyImage(src image.Image) *image.RGBA {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}