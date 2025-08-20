package imageprocessing

import (
	"image"
	"image/color"

	"github.com/makeworld-the-better-one/dither/v2"
)

// DitherFloydSteinberg applies Floyd-Steinberg dithering using the high-quality dither library
func DitherFloydSteinberg(img image.Image, bitDepth int) image.Image {
	if img == nil {
		return nil
	}

	// Create appropriate color palette based on bit depth
	palette := createGrayscalePalette(bitDepth)
	
	// Create a Floyd-Steinberg ditherer with the palette
	ditherer := dither.NewDitherer(palette)
	ditherer.Matrix = dither.FloydSteinberg
	
	// Apply dithering
	return ditherer.Dither(img)
}

// createGrayscalePalette creates a grayscale color palette for the specified bit depth
func createGrayscalePalette(bitDepth int) color.Palette {
	levels := GetColorLevels(bitDepth)
	palette := make(color.Palette, levels)
	
	if levels == 2 { // 1-bit = 2 levels (black and white)
		palette[0] = color.Gray{Y: 0}   // Black
		palette[1] = color.Gray{Y: 255} // White
		return palette
	}
	
	// Create evenly distributed grayscale levels
	for i := 0; i < levels; i++ {
		value := uint8((i * 255) / (levels - 1))
		palette[i] = color.Gray{Y: value}
	}
	
	return palette
}