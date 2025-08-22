package storage

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// ImageStorage handles minimal image storage operations for legacy compatibility
type ImageStorage struct {
	basePath string
	baseURL  string
}

// NewImageStorage creates a new image storage instance
func NewImageStorage(basePath, baseURL string) *ImageStorage {
	return &ImageStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// StoreImage stores image data and returns a URL - used by legacy on-demand processing
func (s *ImageStorage) StoreImage(imageData []byte, deviceID uuid.UUID, pluginType string) (string, error) {
	// Ensure the base directory exists
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}
	
	// Generate filename based on content hash and timestamp
	hash := sha256.Sum256(imageData)
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s_%x.png", pluginType, timestamp, hash[:8])
	
	// Create full path
	fullPath := filepath.Join(s.basePath, filename)
	
	// Write image data to file
	if err := os.WriteFile(fullPath, imageData, 0644); err != nil {
		return "", fmt.Errorf("failed to write image file: %w", err)
	}
	
	// Return URL
	imageURL := fmt.Sprintf("%s/%s", s.baseURL, filename)
	
	return imageURL, nil
}

// GetDefaultImageStorage returns a default image storage configuration
func GetDefaultImageStorage() *ImageStorage {
	basePath := filepath.Join(".", "static", "rendered")
	baseURL := "/static/rendered"
	
	// Try to get from environment
	if envPath := os.Getenv("RENDERED_IMAGES_PATH"); envPath != "" {
		basePath = envPath
	}
	if envURL := os.Getenv("RENDERED_IMAGES_URL"); envURL != "" {
		baseURL = envURL
	}
	
	return NewImageStorage(basePath, baseURL)
}

// GetBasePath returns the base path where images are stored
func (s *ImageStorage) GetBasePath() string {
	return s.basePath
}