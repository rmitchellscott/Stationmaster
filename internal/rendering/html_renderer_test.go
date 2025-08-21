package rendering

import (
	"os"
	"testing"
	"path/filepath"
)

func TestGetChromiumBinary(t *testing.T) {
	// Test with environment variable set
	t.Run("with CHROMIUM_BIN env var", func(t *testing.T) {
		// Create a temporary file to use as mock chromium binary
		tempDir := t.TempDir()
		chromiumPath := filepath.Join(tempDir, "chromium")
		if err := os.WriteFile(chromiumPath, []byte("fake chromium"), 0755); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		
		os.Setenv("CHROMIUM_BIN", chromiumPath)
		defer os.Unsetenv("CHROMIUM_BIN")
		
		result, err := getChromiumBinary()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != chromiumPath {
			t.Errorf("Expected %s, got %s", chromiumPath, result)
		}
	})
	
	t.Run("with CHROME_BIN env var", func(t *testing.T) {
		// Create a temporary file to use as mock chrome binary
		tempDir := t.TempDir()
		chromePath := filepath.Join(tempDir, "chrome")
		if err := os.WriteFile(chromePath, []byte("fake chrome"), 0755); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		
		os.Setenv("CHROME_BIN", chromePath)
		defer os.Unsetenv("CHROME_BIN")
		
		result, err := getChromiumBinary()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != chromePath {
			t.Errorf("Expected %s, got %s", chromePath, result)
		}
	})
	
	t.Run("with invalid CHROMIUM_BIN path", func(t *testing.T) {
		invalidPath := "/nonexistent/path/chromium"
		os.Setenv("CHROMIUM_BIN", invalidPath)
		defer os.Unsetenv("CHROMIUM_BIN")
		
		result, err := getChromiumBinary()
		if err == nil {
			t.Errorf("Expected error for invalid path, got result: %s", result)
		}
		if result != "" {
			t.Errorf("Expected empty result on error, got: %s", result)
		}
	})
	
	t.Run("without env vars", func(t *testing.T) {
		os.Unsetenv("CHROMIUM_BIN")
		os.Unsetenv("CHROME_BIN")
		
		result, err := getChromiumBinary()
		// Should return an error since no binary is available and env vars are unset
		// The exact error depends on whether a system binary is found
		if err != nil {
			t.Logf("Expected error when no binary found: %v", err)
		} else {
			t.Logf("Found system binary: %s", result)
		}
	})
}

func TestRenderOptionsDefault(t *testing.T) {
	opts := DefaultRenderOptions()
	
	if opts.Width <= 0 {
		t.Error("Width should be positive")
	}
	if opts.Height <= 0 {
		t.Error("Height should be positive")
	}
	if opts.Quality <= 0 || opts.Quality > 100 {
		t.Error("Quality should be between 1-100")
	}
}