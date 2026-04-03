package rendering

import (
	"fmt"
	"strings"
)

type ScreenClassOptions struct {
	ModelName         string
	BitDepth          int
	ScreenWidth       int
	ScreenHeight      int
	ScreenOrientation string // "auto", "landscape", "landscape_inverted", "portrait_cw", "portrait_ccw"
	RemoveBleedMargin bool
	EnableDarkMode    bool
}

func BuildScreenClasses(opts ScreenClassOptions) string {
	classes := []string{"screen"}

	classes = append(classes, modelClass(opts.ModelName))
	classes = append(classes, sizeClass(opts.ScreenWidth))
	classes = append(classes, bitDepthClass(opts.BitDepth))

	if isPortrait(opts.ScreenOrientation, opts.ScreenWidth, opts.ScreenHeight) {
		classes = append(classes, "screen--portrait")
	}

	if opts.RemoveBleedMargin {
		classes = append(classes, "screen--no-bleed")
	}
	if opts.EnableDarkMode {
		classes = append(classes, "screen--dark-mode")
	}

	return strings.Join(classes, " ")
}

func isPortrait(orientation string, screenWidth, screenHeight int) bool {
	switch orientation {
	case "portrait_cw", "portrait_ccw":
		return true
	case "landscape", "landscape_inverted":
		return false
	default:
		return screenHeight > screenWidth
	}
}

func modelClass(modelName string) string {
	mapping := map[string]string{
		"og_plus": "screen--og",
		"og_png":  "screen--og",
		"og_bwry": "screen--og",
		"v2":      "screen--v2",
	}
	if class, ok := mapping[modelName]; ok {
		return class
	}
	return "screen--og"
}

func sizeClass(width int) string {
	switch {
	case width > 1200:
		return "screen--lg"
	case width > 800:
		return "screen--md"
	default:
		return "screen--sm"
	}
}

func bitDepthClass(bitDepth int) string {
	if bitDepth < 1 {
		bitDepth = 1
	}
	return fmt.Sprintf("screen--%dbit", bitDepth)
}

// RenderDimensions returns the width and height to use for rendering.
// Portrait orientations swap dimensions; landscape orientations keep them as-is.
func RenderDimensions(screenWidth, screenHeight int, orientation string) (int, int) {
	if isPortrait(orientation, screenWidth, screenHeight) {
		return screenHeight, screenWidth
	}
	return screenWidth, screenHeight
}

// ImageRotation returns the rotation to apply to the rendered image
// so it displays correctly on the device. The user sets how they rotated
// the device; we counter-rotate the image to compensate.
func ImageRotation(screenWidth, screenHeight int, orientation string) string {
	switch orientation {
	case "portrait_cw":
		return "ccw90"
	case "portrait_ccw":
		return "cw90"
	case "landscape_inverted":
		return "180"
	default:
		return "none"
	}
}
