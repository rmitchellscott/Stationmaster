package imageprocessing

import (
	"image"
	"image/color"
)

// ToGrayscale converts an image to grayscale using the luminance formula
// Y = 0.299*R + 0.587*G + 0.114*B
func ToGrayscale(img image.Image) image.Image {
	if img == nil {
		return nil
	}

	bounds := img.Bounds()
	gray := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			grayColor := color.GrayModel.Convert(originalColor)
			gray.Set(x, y, grayColor)
		}
	}

	return gray
}

// ToRGBA converts any image to RGBA format for easier processing
func ToRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	
	return rgba
}

// QuantizeColor reduces the color depth of a grayscale value to match the bit depth
func QuantizeColor(gray uint8, bitDepth int) uint8 {
	switch bitDepth {
	case 1:
		// 1-bit: pure black or white
		if gray >= 128 {
			return 255
		}
		return 0
	case 2:
		// 2-bit: 4 levels (0, 85, 170, 255)
		levels := gray / 64
		if levels > 3 {
			levels = 3
		}
		return uint8(levels * 85)
	case 4:
		// 4-bit: 16 levels
		levels := gray / 16
		if levels > 15 {
			levels = 15
		}
		return uint8(levels * 17) // 17 * 15 = 255
	case 8:
		// 8-bit: full range
		return gray
	default:
		// Default to 8-bit for unknown bit depths
		return gray
	}
}

// GetMaxColorValue returns the maximum color value for a given bit depth
func GetMaxColorValue(bitDepth int) uint8 {
	switch bitDepth {
	case 1:
		return 255 // Black or white only
	case 2:
		return 255 // 0, 85, 170, 255
	case 4:
		return 255 // 16 levels
	case 8:
		return 255 // Full 8-bit range
	default:
		return 255
	}
}

// GetColorLevels returns the number of color levels for a given bit depth
func GetColorLevels(bitDepth int) int {
	switch bitDepth {
	case 1:
		return 2 // Black, white
	case 2:
		return 4 // 4 levels
	case 4:
		return 16 // 16 levels
	case 8:
		return 256 // Full 8-bit
	default:
		return 256
	}
}