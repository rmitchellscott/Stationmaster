package rendering

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/logging"
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
	
	// Debug options to capture console output and errors
	AddScriptTag []map[string]interface{} `json:"addScriptTag,omitempty"`
	RejectRequestPattern []string `json:"rejectRequestPattern,omitempty"`
	RejectResourceTypes []string `json:"rejectResourceTypes,omitempty"`
	SetExtraHTTPHeaders map[string]string `json:"setExtraHTTPHeaders,omitempty"`
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
		Timeout:  20000,                               // 20 second timeout (increased for server-side rendering)
		Visible:  false,                               // Don't require visibility, just presence
	}
	
	// Add debugging script to capture console output and errors
	req.AddScriptTag = []map[string]interface{}{
		{
			"content": `
				console.log('[BROWSERLESS DEBUG] Debug script injected - starting monitoring');
				
				// Capture all console output
				const originalLog = console.log;
				const originalError = console.error;
				const originalWarn = console.warn;
				
				let logBuffer = [];
				
				console.log = function(...args) {
					logBuffer.push(['LOG', args.join(' ')]);
					originalLog.apply(console, args);
				};
				
				console.error = function(...args) {
					logBuffer.push(['ERROR', args.join(' ')]);
					originalError.apply(console, args);
				};
				
				console.warn = function(...args) {
					logBuffer.push(['WARN', args.join(' ')]);
					originalWarn.apply(console, args);
				};
				
				// Track script loading
				let scriptsLoaded = 0;
				let scriptsTotal = document.querySelectorAll('script[src]').length;
				console.log('[BROWSERLESS DEBUG] Found ' + scriptsTotal + ' external scripts to load');
				
				// Monitor for errors
				window.addEventListener('error', function(e) {
					console.error('[BROWSERLESS DEBUG] JavaScript error:', e.error, 'at', e.filename + ':' + e.lineno);
				});
				
				window.addEventListener('unhandledrejection', function(e) {
					console.error('[BROWSERLESS DEBUG] Unhandled promise rejection:', e.reason);
				});
				
				// Monitor document ready state
				console.log('[BROWSERLESS DEBUG] Document ready state:', document.readyState);
				
				// Check for render completion signal every 500ms
				let checkCount = 0;
				const checkInterval = setInterval(function() {
					checkCount++;
					const hasSignal = document.body && document.body.hasAttribute('data-render-complete');
					console.log('[BROWSERLESS DEBUG] Check #' + checkCount + ': render-complete=' + hasSignal);
					
					if (hasSignal) {
						console.log('[BROWSERLESS DEBUG] Render completion signal found! Clearing interval.');
						clearInterval(checkInterval);
					}
					
					if (checkCount >= 40) { // 20 seconds max
						console.error('[BROWSERLESS DEBUG] Timeout waiting for render completion signal');
						clearInterval(checkInterval);
					}
				}, 500);
				
				console.log('[BROWSERLESS DEBUG] Debug monitoring initialized');
			`,
		},
	}
	
	// Add extra headers for debugging
	req.SetExtraHTTPHeaders = map[string]string{
		"User-Agent": "TRMNL-Debug/1.0",
	}
	
	// DEBUG: Log HTML content being sent to browserless
	logging.Browserless("HTML content analysis",
		"html_size_chars", len(html),
		"html_size_bytes", len([]byte(html)),
		"viewport_width", width,
		"viewport_height", height,
	)
	
	// Show HTML structure analysis and full content
	if len(html) > 0 {
		// Count some basic HTML elements for complexity analysis
		scriptCount := strings.Count(html, "<script")
		styleCount := strings.Count(html, "<style")
		divCount := strings.Count(html, "<div")
		imgCount := strings.Count(html, "<img")
		
		logging.Browserless("HTML complexity analysis",
			"script_tags", scriptCount,
			"style_tags", styleCount, 
			"div_tags", divCount,
			"img_tags", imgCount,
		)
		
		// Log the complete HTML content
		logging.Browserless("Complete HTML content",
			"content", html,
		)
		
		// Check for potentially problematic patterns
		if strings.Contains(html, "data:image/") {
			dataImageCount := strings.Count(html, "data:image/")
			logging.Browserless("WARNING: Found data:image URLs",
				"data_image_count", dataImageCount,
				"warning", "data:image URLs can cause large payloads",
			)
		}
		
		if len(html) > 100000 {
			logging.Browserless("WARNING: Large HTML size",
				"html_size_chars", len(html),
				"warning", "HTML size may be too large for browserless",
			)
		}
	}
	
	// Note: browserless doesn't support debug parameter in this API version
	
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
		
		// Log debug information from browserless on failure
		headers := make(map[string][]string)
		for key, values := range resp.Header {
			headers[key] = values
		}
		
		logging.Browserless("Browserless request failed",
			"status_code", resp.StatusCode,
			"response_headers", headers,
			"response_body", string(body),
		)
		
		return nil, fmt.Errorf("browserless HTML screenshot request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Log debug information from successful responses too
	debugHeaders := make(map[string][]string)
	for key, values := range resp.Header {
		if key == "X-Response-Console" || key == "X-Response-Network" || key == "X-Debug" {
			debugHeaders[key] = values
		}
	}
	
	if len(debugHeaders) > 0 {
		logging.Browserless("Browserless request successful",
			"status_code", resp.StatusCode,
			"debug_headers", debugHeaders,
		)
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