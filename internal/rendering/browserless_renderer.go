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

// ContentRequest represents the request payload for browserless /content endpoint (no options field)
type ContentRequest struct {
	HTML     string `json:"html"`
	Viewport struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"viewport"`
	GotoOptions struct {
		WaitUntil string `json:"waitUntil"`
		Timeout   int    `json:"timeout"`
	} `json:"gotoOptions"`
	WaitForSelector *WaitForSelector `json:"waitForSelector,omitempty"`
}

// RenderFlags contains flags detected during rendering
type RenderFlags struct {
	SkipScreenGeneration bool `json:"skip_screen_generation"`
	SkipDisplay          bool `json:"skip_display"`
}

// RenderHTMLResult contains the result of HTML rendering including any flags
type RenderHTMLResult struct {
	ImageData []byte      `json:"image_data"`
	Flags     RenderFlags `json:"flags"`
}

// RenderHTML renders HTML content to an image using browserless
func (r *BrowserlessRenderer) RenderHTML(ctx context.Context, html string, width, height int) ([]byte, error) {
	result, err := r.RenderHTMLWithResult(ctx, html, width, height)
	if err != nil {
		return nil, err
	}
	return result.ImageData, nil
}

// RenderHTMLWithResult renders HTML content and returns both image data and flags
func (r *BrowserlessRenderer) RenderHTMLWithResult(ctx context.Context, html string, width, height int) (*RenderHTMLResult, error) {
	
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
	
	// Flag detection is now handled automatically in WrapWithTRNMLAssets - no manual injection needed
	
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
		
		// Log the complete HTML content (JSON-escaped in structured logs)
		logging.Browserless("Complete HTML content",
			"content", html,
		)
		
		// TEMPORARY: Log raw HTML content without JSON escaping to verify it's correct
		fmt.Printf("=== RAW HTML START ===\n%s\n=== RAW HTML END ===\n", html)
		
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
	
	// Debug: Log ALL response headers to understand what browserless returns
	allHeaders := make(map[string][]string)
	for key, values := range resp.Header {
		allHeaders[key] = values
	}
	
	logging.Browserless("Complete browserless response headers",
		"status_code", resp.StatusCode,
		"all_headers", allHeaders,
	)
	
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
		
		// Log console output for debugging
		if consoleOutput, exists := debugHeaders["X-Response-Console"]; exists && len(consoleOutput) > 0 {
			logging.Browserless("Console output from browserless",
				"console_output", consoleOutput,
			)
		}
	} else {
		logging.Browserless("No debug headers found in browserless response",
			"available_headers", allHeaders,
		)
	}
	
	// Read response body (image data)
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read browserless response: %w", err)
	}
	
	// Parse TRMNL flags from DOM attributes using browserless /content endpoint
	flags, err := r.parseTRMNLFlagsFromDOM(ctx, html)
	if err != nil {
		logging.Browserless("Failed to parse TRMNL flags from DOM, continuing with no flags",
			"error", err.Error(),
		)
		flags = RenderFlags{} // Continue with no flags rather than failing
	}
	
	// If SKIP_SCREEN_GENERATION was detected, we should abort
	if flags.SkipScreenGeneration {
		return nil, fmt.Errorf("render skipped due at plugin's request")
	}
	
	// Return both image data and flags
	return &RenderHTMLResult{
		ImageData: imageData,
		Flags:     flags,
	}, nil
}

// parseTRMNLFlagsFromHeaders extracts TRMNL flags from browserless response headers
func (r *BrowserlessRenderer) parseTRMNLFlagsFromHeaders(headers http.Header) RenderFlags {
	flags := RenderFlags{}
	
	logging.Browserless("Starting TRMNL flag parsing from headers",
		"total_headers", len(headers),
	)
	
	// Check X-Response-Console header for our TRMNL flag messages
	consoleOutputs := headers["X-Response-Console"]
	logging.Browserless("Checking X-Response-Console header",
		"header_exists", len(consoleOutputs) > 0,
		"console_entries_count", len(consoleOutputs),
		"console_outputs", consoleOutputs,
	)
	
	for i, output := range consoleOutputs {
		logging.Browserless("Processing console output entry",
			"entry_index", i,
			"output_length", len(output),
			"output_content", output,
		)
		
		// Look for our specific console log messages
		if strings.Contains(output, "[TRMNL] SKIP_SCREEN_GENERATION flag detected") {
			flags.SkipScreenGeneration = true
			logging.Info("[BROWSERLESS] TRMNL_SKIP_SCREEN_GENERATION detected in console output")
		}
		if strings.Contains(output, "[TRMNL] SKIP_DISPLAY flag detected") {
			flags.SkipDisplay = true
			logging.Info("[BROWSERLESS] TRMNL_SKIP_DISPLAY detected in console output")
		}
	}
	
	// Fallback: Check other debug headers if console output not available
	if !flags.SkipScreenGeneration && !flags.SkipDisplay {
		logging.Browserless("No flags found in console output, trying fallback DOM check")
		flags = r.fallbackDOMCheck(headers)
	}
	
	logging.Browserless("Final TRMNL flag parsing result",
		"skip_screen_generation", flags.SkipScreenGeneration,
		"skip_display", flags.SkipDisplay,
	)
	
	return flags
}

// parseTRMNLFlagsFromDOM extracts TRMNL flags by checking DOM attributes via browserless /content endpoint
func (r *BrowserlessRenderer) parseTRMNLFlagsFromDOM(ctx context.Context, html string) (RenderFlags, error) {
	flags := RenderFlags{}
	
	logging.Browserless("Starting TRMNL flag detection via DOM content check",
		"method", "content_endpoint",
	)
	
	// Prepare browserless content request to get DOM after JavaScript execution
	contentReq := ContentRequest{
		HTML: html,
		Viewport: struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}{
			Width:  800,
			Height: 480,
		},
	}
	
	// Set wait options for complete JavaScript execution (same as screenshot request)
	contentReq.GotoOptions.WaitUntil = "networkidle0"
	contentReq.GotoOptions.Timeout = 60000
	
	// Wait for completion signal (same as screenshot request)  
	contentReq.WaitForSelector = &WaitForSelector{
		Selector: "body[data-render-complete='true']",
		Timeout:  20000,
		Visible:  false,
	}
	
	// Marshal request to JSON
	requestBody, err := json.Marshal(contentReq)
	if err != nil {
		return flags, fmt.Errorf("failed to marshal content request: %w", err)
	}
	
	// Make request to browserless /content endpoint
	contentURL := fmt.Sprintf("%s/content", r.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", contentURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return flags, fmt.Errorf("failed to create content HTTP request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := r.client.Do(httpReq)
	if err != nil {
		return flags, fmt.Errorf("failed to make content request to browserless: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return flags, fmt.Errorf("browserless content request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Read DOM content
	domContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return flags, fmt.Errorf("failed to read browserless content response: %w", err)
	}
	
	domHTML := string(domContent)
	logging.Browserless("Received DOM content from browserless",
		"content_length", len(domHTML),
		"has_body_tag", strings.Contains(domHTML, "<body"),
	)
	
	// Check for TRMNL flag attributes in the DOM
	if strings.Contains(domHTML, `data-trmnl-skip-screen-generation="true"`) ||
		strings.Contains(domHTML, `data-trmnl-skip-screen-generation='true'`) {
		flags.SkipScreenGeneration = true
		logging.Info("[BROWSERLESS] TRMNL_SKIP_SCREEN_GENERATION detected in DOM attributes")
	}
	
	if strings.Contains(domHTML, `data-trmnl-skip-display="true"`) ||
		strings.Contains(domHTML, `data-trmnl-skip-display='true'`) {
		flags.SkipDisplay = true
		logging.Info("[BROWSERLESS] TRMNL_SKIP_DISPLAY detected in DOM attributes")
	}
	
	logging.Browserless("DOM flag detection completed",
		"skip_screen_generation", flags.SkipScreenGeneration,
		"skip_display", flags.SkipDisplay,
	)
	
	return flags, nil
}

// fallbackDOMCheck provides a fallback method using /content endpoint to check DOM
func (r *BrowserlessRenderer) fallbackDOMCheck(headers http.Header) RenderFlags {
	// This is now obsolete since we moved to DOM-based detection
	// Keeping for backward compatibility but it will always return empty flags
	flags := RenderFlags{}
	
	logging.Debug("[BROWSERLESS] fallbackDOMCheck called but DOM detection now handled in parseTRMNLFlagsFromDOM")
	
	return flags
}

// DefaultBrowserlessRenderer creates a renderer with default options
func DefaultBrowserlessRenderer() (*BrowserlessRenderer, error) {
	return NewBrowserlessRenderer()
}