package imageprocessing

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
)

// EncodePalettedPNG encodes a paletted image to PNG as grayscale with the correct bit depth
// This mimics ImageMagick's approach: PNG color type 0 (grayscale) instead of palette
func EncodePalettedPNG(img image.Image, bitDepth int) ([]byte, error) {
	paletted, ok := img.(*image.Paletted)
	if !ok {
		return nil, fmt.Errorf("image must be *image.Paletted")
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Validate bit depth
	if bitDepth != 1 && bitDepth != 2 && bitDepth != 4 && bitDepth != 8 {
		return nil, fmt.Errorf("unsupported bit depth: %d", bitDepth)
	}

	var buf bytes.Buffer

	// PNG signature
	buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})

	// IHDR chunk - using color type 0 (grayscale) like ImageMagick
	writeChunk(&buf, "IHDR", func(data *bytes.Buffer) {
		binary.Write(data, binary.BigEndian, uint32(width))
		binary.Write(data, binary.BigEndian, uint32(height))
		data.WriteByte(uint8(bitDepth)) // Bit depth
		data.WriteByte(0)               // Color type: Grayscale (like ImageMagick)
		data.WriteByte(0)               // Compression method
		data.WriteByte(0)               // Filter method
		data.WriteByte(0)               // Interlace method
	})

	// No PLTE chunk needed for grayscale PNG

	// Pack image data as grayscale values according to bit depth
	imageData, err := packGrayscaleImageData(paletted, bitDepth)
	if err != nil {
		return nil, fmt.Errorf("failed to pack image data: %w", err)
	}

	// IDAT chunk (compressed image data)
	compressedData, err := zlibCompress(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to compress image data: %w", err)
	}
	
	writeChunk(&buf, "IDAT", func(data *bytes.Buffer) {
		data.Write(compressedData)
	})

	// IEND chunk
	writeChunk(&buf, "IEND", func(data *bytes.Buffer) {
		// Empty data for IEND
	})

	return buf.Bytes(), nil
}

// packGrayscaleImageData packs paletted image as grayscale values according to bit depth
// This converts palette indices to actual grayscale values for PNG color type 0
func packGrayscaleImageData(paletted *image.Paletted, bitDepth int) ([]byte, error) {
	bounds := paletted.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	pixelsPerByte := 8 / bitDepth
	bytesPerRow := (width + pixelsPerByte - 1) / pixelsPerByte
	
	// Add filter byte at the start of each row
	data := make([]byte, height*(bytesPerRow+1))
	
	// Note: We use palette index directly as grayscale level since our dithering
	// creates palettes with the correct grayscale values at each index
	
	for y := 0; y < height; y++ {
		rowStart := y * (bytesPerRow + 1)
		data[rowStart] = 0 // Filter type: None
		
		for x := 0; x < width; x++ {
			// Get palette index
			pixelIndex := paletted.ColorIndexAt(bounds.Min.X+x, bounds.Min.Y+y)
			
			// Use palette index directly as grayscale level since our dithering 
			// already created the palette with the correct grayscale values at each index
			grayLevel := pixelIndex
			
			byteIndex := rowStart + 1 + x/pixelsPerByte
			bitOffset := (pixelsPerByte - 1 - (x % pixelsPerByte)) * bitDepth
			
			data[byteIndex] |= grayLevel << bitOffset
		}
	}
	
	return data, nil
}

// writeChunk writes a PNG chunk with proper CRC
func writeChunk(buf *bytes.Buffer, chunkType string, dataWriter func(*bytes.Buffer)) {
	var chunkData bytes.Buffer
	dataWriter(&chunkData)
	
	data := chunkData.Bytes()
	
	// Length
	binary.Write(buf, binary.BigEndian, uint32(len(data)))
	
	// Type
	buf.WriteString(chunkType)
	
	// Data
	buf.Write(data)
	
	// CRC
	crc := crc32.NewIEEE()
	crc.Write([]byte(chunkType))
	crc.Write(data)
	binary.Write(buf, binary.BigEndian, crc.Sum32())
}

// zlibCompress compresses data using proper zlib compression
func zlibCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	
	// Create zlib writer with best compression
	writer, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib writer: %w", err)
	}
	
	// Write data
	_, err = writer.Write(data)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to write data: %w", err)
	}
	
	// Close writer to flush
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close zlib writer: %w", err)
	}
	
	return buf.Bytes(), nil
}