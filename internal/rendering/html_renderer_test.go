package rendering

import (
	"os"
	"testing"
)

func TestGetChromiumBinary(t *testing.T) {
	// Test with environment variable set
	t.Run("with CHROMIUM_BIN env var", func(t *testing.T) {
		expected := "/test/chromium"
		os.Setenv("CHROMIUM_BIN", expected)
		defer os.Unsetenv("CHROMIUM_BIN")
		
		result := getChromiumBinary()
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})
	
	t.Run("with CHROME_BIN env var", func(t *testing.T) {
		expected := "/test/chrome"
		os.Setenv("CHROME_BIN", expected)
		defer os.Unsetenv("CHROME_BIN")
		
		result := getChromiumBinary()
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})
	
	t.Run("without env vars", func(t *testing.T) {
		os.Unsetenv("CHROMIUM_BIN")
		os.Unsetenv("CHROME_BIN")
		
		result := getChromiumBinary()
		// Should either find a system binary or return empty string
		// We can't predict what's available on the test system
		t.Logf("Binary detection result: %s", result)
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