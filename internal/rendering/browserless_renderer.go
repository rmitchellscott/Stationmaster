package rendering

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/config"
)

// BrowserlessRenderer captures screenshots using an external browserless service
type BrowserlessRenderer struct {
	client  *http.Client
	baseURL string
}

// NewBrowserlessRenderer creates a new browserless renderer
func NewBrowserlessRenderer() (*BrowserlessRenderer, error) {
	baseURL := config.Get("BROWSERLESS_URL", "http://localhost:3000")
	if baseURL == "" {
		return nil, fmt.Errorf("BROWSERLESS_URL environment variable is required")
	}

	// Remove trailing slash if present
	if baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}

	return &BrowserlessRenderer{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: baseURL,
	}, nil
}

// ScreenshotRequest represents the request payload for browserless screenshot API
type ScreenshotRequest struct {
	URL      string `json:"url"`
	Viewport struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"viewport"`
	Options struct {
		Type           string `json:"type"`
		Quality        *int   `json:"quality,omitempty"`
		FullPage       bool   `json:"fullPage"`
		OmitBackground bool   `json:"omitBackground"`
	} `json:"options"`
	// Removed gotoOptions and waitForEvent to test minimal structure
}

// CaptureScreenshot captures a screenshot of the given URL using browserless
func (r *BrowserlessRenderer) CaptureScreenshot(ctx context.Context, url string, width, height int, fullPage bool, waitTimeSeconds int) ([]byte, error) {
	// Prepare minimal browserless request to test schema
	req := ScreenshotRequest{
		URL: url,
		Viewport: struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}{
			Width:  width,
			Height: height,
		},
	}
	
	req.Options.Type = "png"
	req.Options.FullPage = fullPage
	req.Options.OmitBackground = false
	
	// Removed gotoOptions and waitForEvent to isolate schema issue
	
	// Marshal request to JSON
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal screenshot request: %w", err)
	}
	
	// Make request to browserless
	screenshotURL := fmt.Sprintf("%s/screenshot", r.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", screenshotURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to browserless: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("browserless screenshot request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Read response body (image data)
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read browserless response: %w", err)
	}
	
	return imageData, nil
}

// Close cleans up the renderer (no-op for browserless)
func (r *BrowserlessRenderer) Close() error {
	return nil
}

// DefaultBrowserlessRenderer creates a renderer with default options
func DefaultBrowserlessRenderer() (*BrowserlessRenderer, error) {
	return NewBrowserlessRenderer()
}