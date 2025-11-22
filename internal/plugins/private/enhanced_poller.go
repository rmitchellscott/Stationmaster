package private

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// EnhancedPollingConfig represents the configuration for polling external data
type EnhancedPollingConfig struct {
	URLs        []EnhancedURLConfig `json:"urls"`
	Interval    int                 `json:"interval"`    // Polling interval in seconds (not used in on-demand polling)
	Timeout     int                 `json:"timeout"`     // Request timeout in seconds
	MaxSize     int                 `json:"max_size"`    // Maximum response size in bytes
	UserAgent   string              `json:"user_agent"`  // Custom User-Agent header
	RetryCount  int                 `json:"retry_count"` // Number of retries on failure
}

// EnhancedURLConfig represents configuration for a single URL to poll
type EnhancedURLConfig struct {
	URL     string      `json:"url"`
	Headers interface{} `json:"headers"` // Can be string (TRMNL format) or map for legacy
	Method  string      `json:"method"`  // GET, POST, etc.
	Body    string      `json:"body"`    // Request body for POST requests
}

// EnhancedPolledData represents the result of polling external URLs
type EnhancedPolledData struct {
	Data     map[string]interface{} `json:"data"`
	Success  bool                   `json:"success"`
	Errors   []string               `json:"errors,omitempty"`
	Duration time.Duration          `json:"duration"`
}

// EnhancedDataPoller handles polling external URLs for private plugin data with robust error handling
type EnhancedDataPoller struct {
	client *http.Client
}

// NewEnhancedDataPoller creates a new enhanced data poller
func NewEnhancedDataPoller() *EnhancedDataPoller {
	return &EnhancedDataPoller{
		client: &http.Client{
			Timeout: 30 * time.Second, // Default timeout, will be overridden per request
		},
	}
}

// PollData polls all configured URLs for a private plugin with enhanced error handling and retries
func (p *EnhancedDataPoller) PollData(ctx context.Context, plugin *database.PluginDefinition, templateData map[string]interface{}) (*EnhancedPolledData, error) {
	if plugin.DataStrategy == nil || *plugin.DataStrategy != "polling" {
		return nil, fmt.Errorf("plugin is not configured for polling")
	}

	if plugin.PollingConfig == nil {
		return nil, fmt.Errorf("no polling configuration found")
	}

	startTime := time.Now().UTC()
	result := &EnhancedPolledData{
		Data:     make(map[string]interface{}),
		Success:  true,
		Errors:   []string{},
		Duration: 0,
	}

	// Parse polling configuration
	var config EnhancedPollingConfig
	if err := json.Unmarshal(plugin.PollingConfig, &config); err != nil {
		return nil, fmt.Errorf("invalid polling configuration: %w", err)
	}

	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = 10
	}
	if config.MaxSize == 0 {
		config.MaxSize = 1024 * 1024 // 1MB default
	}
	if config.RetryCount == 0 {
		config.RetryCount = 2
	}
	if config.UserAgent == "" {
		config.UserAgent = "TRMNL-Private-Plugin/1.0"
	}

	// Update client timeout
	p.client.Timeout = time.Duration(config.Timeout) * time.Second

	// Poll each configured URL
	for i, urlConfig := range config.URLs {
		urlData, err := p.pollSingleURL(ctx, urlConfig, &config, templateData)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to poll %s: %v", urlConfig.URL, err))
			result.Success = false
			continue
		}

		// Store data using index-based key for multiple URLs or direct merge for single URL
		if len(config.URLs) == 1 {
			// Single URL: merge data directly into root context
			if dataMap, ok := urlData.(map[string]interface{}); ok {
				// If it's a JSON object, merge all keys directly
				for k, v := range dataMap {
					result.Data[k] = v
				}
			} else {
				// If it's not a JSON object, store as 'data'
				result.Data["data"] = urlData
			}
		} else {
			// Multiple URLs: use indexed keys (IDX_0, IDX_1, etc.)
			result.Data[fmt.Sprintf("IDX_%d", i)] = urlData
		}
	}

	result.Duration = time.Since(startTime)

	logging.Info("[ENHANCED_POLLER] Polling completed",
		"plugin_id", plugin.ID,
		"plugin_name", plugin.Name,
		"urls_count", len(config.URLs),
		"success", result.Success,
		"errors_count", len(result.Errors),
		"duration_ms", result.Duration.Milliseconds())

	return result, nil
}

// pollSingleURL polls a single URL and returns the response data with retry logic
func (p *EnhancedDataPoller) pollSingleURL(ctx context.Context, urlConfig EnhancedURLConfig, config *EnhancedPollingConfig, templateData map[string]interface{}) (interface{}, error) {
	var lastErr error

	// Retry logic with exponential backoff
	for attempt := 0; attempt <= config.RetryCount; attempt++ {
		if attempt > 0 {
			// Wait before retrying (exponential backoff)
			waitTime := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
			}
		}

		data, err := p.fetchURL(ctx, urlConfig, config, templateData)
		if err == nil {
			return data, nil
		}

		lastErr = err
		logging.Warn("[ENHANCED_POLLER] URL fetch failed, retrying",
			"url", urlConfig.URL,
			"attempt", attempt+1,
			"max_attempts", config.RetryCount+1,
			"error", err)
	}

	return nil, fmt.Errorf("all attempts failed: %w", lastErr)
}

// fetchURL fetches data from a single URL with template variable substitution
func (p *EnhancedDataPoller) fetchURL(ctx context.Context, urlConfig EnhancedURLConfig, config *EnhancedPollingConfig, templateData map[string]interface{}) (interface{}, error) {
	// Replace template variables in URL and body
	processedURL := p.replaceMergeVariables(urlConfig.URL, templateData)
	processedBody := p.replaceMergeVariables(urlConfig.Body, templateData)

	// Create request
	method := urlConfig.Method
	if method == "" {
		method = "GET"
	}

	var bodyReader io.Reader
	if processedBody != "" {
		bodyReader = bytes.NewReader([]byte(processedBody))
	}

	req, err := http.NewRequestWithContext(ctx, method, processedURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent
	req.Header.Set("User-Agent", config.UserAgent)

	// Set headers from configuration (handle both string and map formats)
	headerMap := p.parseHeaders(urlConfig.Headers)
	for key, value := range headerMap {
		// Apply template variable substitution to header values
		processedValue := p.replaceMergeVariables(value, templateData)
		req.Header.Set(key, processedValue)
	}

	// Set content type for POST requests with body if not already set
	if method == "POST" && processedBody != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Make request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, int64(config.MaxSize))
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check if response was truncated
	if len(bodyBytes) == config.MaxSize {
		return nil, fmt.Errorf("response too large (limit: %d bytes)", config.MaxSize)
	}

	// Try to parse as JSON first
	var jsonData interface{}
	if json.Unmarshal(bodyBytes, &jsonData) == nil {
		return jsonData, nil
	}

	// If not JSON, return as string
	return string(bodyBytes), nil
}

// parseHeaders converts headers from either string (TRMNL format) or map format to map[string]string
func (p *EnhancedDataPoller) parseHeaders(headers interface{}) map[string]string {
	headerMap := make(map[string]string)

	switch h := headers.(type) {
	case string:
		// TRMNL format: key=value&key2=value2
		if strings.TrimSpace(h) == "" {
			return headerMap
		}
		pairs := strings.Split(h, "&")
		for _, pair := range pairs {
			if pair = strings.TrimSpace(pair); pair != "" {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) == 2 {
					// Handle URL decoding for special characters like %3D for =
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					// Basic URL decoding for common cases
					value = strings.ReplaceAll(value, "%3D", "=")
					value = strings.ReplaceAll(value, "%20", " ")
					headerMap[key] = value
				}
			}
		}
	case map[string]interface{}:
		// Legacy object format
		for key, value := range h {
			headerMap[key] = fmt.Sprintf("%v", value)
		}
	case map[string]string:
		// Direct map format
		headerMap = h
	}

	return headerMap
}

// replaceMergeVariables replaces {{ variable }} placeholders with values from template data
func (p *EnhancedDataPoller) replaceMergeVariables(template string, data map[string]interface{}) string {
	result := template
	for key, value := range data {
		placeholder := fmt.Sprintf("{{ %s }}", key)
		placeholderNoSpaces := fmt.Sprintf("{{%s}}", key)
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
		result = strings.ReplaceAll(result, placeholderNoSpaces, valueStr)
	}
	return result
}

