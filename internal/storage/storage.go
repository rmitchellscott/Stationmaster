package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rmitchellscott/stationmaster/internal/config"
)

// FileInfo represents information about a stored file
type FileInfo struct {
	Key  string
	Size int64
}

// StorageBackendWithInfo defines the interface for storage backends
type StorageBackendWithInfo interface {
	Put(ctx context.Context, key string, reader io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	ListWithInfo(ctx context.Context, prefix string) ([]FileInfo, error)
}

// FilesystemBackend implements storage using local filesystem
type FilesystemBackend struct {
	dataDir string
}

// NewFilesystemBackend creates a new filesystem storage backend
func NewFilesystemBackend(dataDir string) *FilesystemBackend {
	return &FilesystemBackend{
		dataDir: dataDir,
	}
}

// Put stores data in the filesystem
func (f *FilesystemBackend) Put(ctx context.Context, key string, reader io.Reader) error {
	fullPath := filepath.Join(f.dataDir, key)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer file.Close()

	// Copy data
	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", fullPath, err)
	}

	return nil
}

// Get retrieves data from the filesystem
func (f *FilesystemBackend) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(f.dataDir, key)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", fullPath, err)
	}

	return file, nil
}

// Delete removes a file from the filesystem
func (f *FilesystemBackend) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(f.dataDir, key)

	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}

	return nil
}

// ListWithInfo lists files with the given prefix and returns their info
func (f *FilesystemBackend) ListWithInfo(ctx context.Context, prefix string) ([]FileInfo, error) {
	var files []FileInfo

	prefixPath := filepath.Join(f.dataDir, prefix)

	// If prefix is exact file, check if it exists
	if info, err := os.Stat(prefixPath); err == nil && !info.IsDir() {
		relPath, _ := filepath.Rel(f.dataDir, prefixPath)
		files = append(files, FileInfo{
			Key:  relPath,
			Size: info.Size(),
		})
		return files, nil
	}

	// Otherwise, walk the directory
	err := filepath.Walk(filepath.Join(f.dataDir, filepath.Dir(prefix)), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(f.dataDir, path)
			if err != nil {
				return err
			}

			// Check if this file matches the prefix
			if strings.HasPrefix(relPath, prefix) || strings.HasPrefix(relPath, strings.ReplaceAll(prefix, "\\", "/")) {
				files = append(files, FileInfo{
					Key:  relPath,
					Size: info.Size(),
				})
			}
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// Global storage backend instance
var globalBackend StorageBackendWithInfo

// GetStorageBackend returns the configured storage backend
func GetStorageBackend() StorageBackendWithInfo {
	if globalBackend == nil {
		dataDir := config.Get("DATA_DIR", "/data")
		globalBackend = NewFilesystemBackend(dataDir)
	}
	return globalBackend
}
