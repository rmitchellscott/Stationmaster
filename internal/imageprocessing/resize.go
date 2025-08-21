package imageprocessing

import (
	"image"
	"image/color"
	"image/draw"
	
	xdraw "golang.org/x/image/draw"
)

// ResizeToFit resizes an image to fit within the specified dimensions while preserving aspect ratio
// If the image doesn't fill the entire target area, it will be centered with a black background
func ResizeToFit(img image.Image, targetWidth, targetHeight int) image.Image {
	if img == nil {
		return nil
	}

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate scaling factor to fit within target dimensions
	scaleX := float64(targetWidth) / float64(srcWidth)
	scaleY := float64(targetHeight) / float64(srcHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Calculate new dimensions
	newWidth := int(float64(srcWidth) * scale)
	newHeight := int(float64(srcHeight) * scale)

	// Create target canvas with black background
	canvas := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	
	// Fill with black background
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.Point{}, draw.Src)

	// Calculate centering offset
	offsetX := (targetWidth - newWidth) / 2
	offsetY := (targetHeight - newHeight) / 2

	// Create resized image using high-quality scaling
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	
	// Use BiLinear interpolation for good quality/speed balance
	xdraw.BiLinear.Scale(resized, resized.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	// Draw the resized image onto the centered canvas
	targetRect := image.Rect(offsetX, offsetY, offsetX+newWidth, offsetY+newHeight)
	draw.Draw(canvas, targetRect, resized, image.Point{}, draw.Src)

	return canvas
}

// GetScaledDimensions calculates the scaled dimensions that fit within the target while preserving aspect ratio
func GetScaledDimensions(srcWidth, srcHeight, targetWidth, targetHeight int) (int, int) {
	scaleX := float64(targetWidth) / float64(srcWidth)
	scaleY := float64(targetHeight) / float64(srcHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(srcWidth) * scale)
	newHeight := int(float64(srcHeight) * scale)
	
	return newWidth, newHeight
}

// ResizeToFill resizes an image to fill the entire target dimensions while preserving aspect ratio
// The image will be scaled to cover the full target area and cropped if necessary
func ResizeToFill(img image.Image, targetWidth, targetHeight int) image.Image {
	if img == nil {
		return nil
	}

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate scaling factor to fill target dimensions (use the larger scale)
	scaleX := float64(targetWidth) / float64(srcWidth)
	scaleY := float64(targetHeight) / float64(srcHeight)
	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	// Calculate new dimensions (will be larger than or equal to target)
	newWidth := int(float64(srcWidth) * scale)
	newHeight := int(float64(srcHeight) * scale)

	// Create resized image using high-quality scaling
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	xdraw.BiLinear.Scale(resized, resized.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	// Create target canvas
	canvas := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Calculate cropping offset to center the image
	offsetX := (newWidth - targetWidth) / 2
	offsetY := (newHeight - targetHeight) / 2

	// Copy the centered portion of the resized image to the canvas
	srcRect := image.Rect(offsetX, offsetY, offsetX+targetWidth, offsetY+targetHeight)
	draw.Draw(canvas, canvas.Bounds(), resized, srcRect.Min, draw.Src)

	return canvas
}