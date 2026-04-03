package imageprocessing

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
)

// RotateCW90 rotates an image 90 degrees clockwise.
// A portrait-rendered image (HxW) becomes landscape (WxH).
func RotateCW90(img image.Image) image.Image {
	if img == nil {
		return nil
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	rotated := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rotated.Set(h-1-(y-bounds.Min.Y), x-bounds.Min.X, img.At(x, y))
		}
	}

	return rotated
}

// RotateCCW90 rotates an image 90 degrees counter-clockwise.
func RotateCCW90(img image.Image) image.Image {
	if img == nil {
		return nil
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	_ = draw.Src
	rotated := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rotated.Set(y-bounds.Min.Y, w-1-(x-bounds.Min.X), img.At(x, y))
		}
	}

	return rotated
}

// Rotate180 rotates an image 180 degrees.
func Rotate180(img image.Image) image.Image {
	if img == nil {
		return nil
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	rotated := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rotated.Set(w-1-(x-bounds.Min.X), h-1-(y-bounds.Min.Y), img.At(x, y))
		}
	}

	return rotated
}

// RotatePNGBytes decodes PNG bytes, applies the named rotation, and re-encodes.
// Rotation must be "cw90", "ccw90", or "180".
func RotatePNGBytes(data []byte, rotation string) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image for rotation: %w", err)
	}

	var rotated image.Image
	switch rotation {
	case "cw90":
		rotated = RotateCW90(img)
	case "ccw90":
		rotated = RotateCCW90(img)
	case "180":
		rotated = Rotate180(img)
	default:
		return data, nil
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, rotated); err != nil {
		return nil, fmt.Errorf("failed to encode rotated image: %w", err)
	}

	return buf.Bytes(), nil
}

// RotatePNGBytesCW90 decodes PNG bytes, rotates 90° clockwise, and re-encodes.
func RotatePNGBytesCW90(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image for rotation: %w", err)
	}

	rotated := RotateCW90(img)

	var buf bytes.Buffer
	if err := png.Encode(&buf, rotated); err != nil {
		return nil, fmt.Errorf("failed to encode rotated image: %w", err)
	}

	return buf.Bytes(), nil
}
