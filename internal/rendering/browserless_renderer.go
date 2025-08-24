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

// WaitForSelector represents browserless waitForSelector options
type WaitForSelector struct {
	Selector string `json:"selector"`
	Timeout  int    `json:"timeout"`
	Visible  bool   `json:"visible"`
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
	GotoOptions struct {
		WaitUntil string `json:"waitUntil"`
		Timeout   int    `json:"timeout"`
	} `json:"gotoOptions"`
	Headers         map[string]string `json:"headers,omitempty"`
	WaitForSelector *WaitForSelector  `json:"waitForSelector,omitempty"`
}

// CaptureScreenshot captures a screenshot of the given URL using browserless
func (r *BrowserlessRenderer) CaptureScreenshot(ctx context.Context, url string, width, height int, waitTimeSeconds int, headers map[string]string) ([]byte, error) {
	// Prepare browserless request with proper wait time handling
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
	req.Options.FullPage = false
	req.Options.OmitBackground = false
	
	// Set wait options based on provided wait time
	req.GotoOptions.WaitUntil = "networkidle2"
	req.GotoOptions.Timeout = (waitTimeSeconds + 30) * 1000 // Convert to milliseconds and add buffer
	
	// Set custom headers if provided
	if headers != nil && len(headers) > 0 {
		req.Headers = headers
	}
	
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

// HTMLScreenshotRequest represents the request payload for browserless HTML screenshot API
type HTMLScreenshotRequest struct {
	HTML     string `json:"html"`
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
	GotoOptions struct {
		WaitUntil string `json:"waitUntil"`
		Timeout   int    `json:"timeout"`
	} `json:"gotoOptions"`
	WaitForSelector *WaitForSelector `json:"waitForSelector,omitempty"`
}

// RenderHTML renders HTML content to an image using browserless
func (r *BrowserlessRenderer) RenderHTML(ctx context.Context, html string, width, height int) ([]byte, error) {
	// Prepare browserless request for HTML content
	req := HTMLScreenshotRequest{
		HTML: html,
		Viewport: struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}{
			Width:  width,
			Height: height,
		},
	}
	
	req.Options.Type = "png"
	req.Options.FullPage = false
	req.Options.OmitBackground = false
	
	// Set wait options for complete asset loading
	req.GotoOptions.WaitUntil = "networkidle0" // Wait for all network requests to complete
	req.GotoOptions.Timeout = 60000 // 60 seconds timeout (increased for complete loading)
	
	// Wait for completion signal (more reliable than style-based detection)
	req.WaitForSelector = &WaitForSelector{
		Selector: "body[data-render-complete='true']", // Wait for render completion attribute
		Timeout:  8000,                                // 8 second timeout (reduced from 15)
		Visible:  false,                               // Don't require visibility, just presence
	}
	
	// Marshal request to JSON
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal HTML screenshot request: %w", err)
	}
	
	// Make request to browserless screenshot endpoint
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
		return nil, fmt.Errorf("browserless HTML screenshot request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Read response body (image data)
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read browserless response: %w", err)
	}
	
	return imageData, nil
}

// DefaultBrowserlessRenderer creates a renderer with default options
func DefaultBrowserlessRenderer() (*BrowserlessRenderer, error) {
	return NewBrowserlessRenderer()
}