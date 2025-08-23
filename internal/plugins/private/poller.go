package private

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"gorm.io/gorm"
)

// PollingConfig represents the configuration for polling external data
type PollingConfig struct {
	URLs        []URLConfig `json:"urls"`
	Interval    int         `json:"interval"`    // Polling interval in seconds
	Timeout     int         `json:"timeout"`     // Request timeout in seconds
	MaxSize     int         `json:"max_size"`    // Maximum response size in bytes
	UserAgent   string      `json:"user_agent"`  // Custom User-Agent header
	RetryCount  int         `json:"retry_count"` // Number of retries on failure
}

// URLConfig represents configuration for a single URL to poll
type URLConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Method  string            `json:"method"` // GET, POST, etc.
	Body    string            `json:"body"`   // Request body for POST requests
	Key     string            `json:"key"`    // Key to store this data under in the response
}

// PolledData represents the result of polling external URLs
type PolledData struct {
	PluginID    string                 `json:"plugin_id"`
	Data        map[string]interface{} `json:"data"`
	PolledAt    time.Time              `json:"polled_at"`
	Success     bool                   `json:"success"`
	Errors      []string               `json:"errors,omitempty"`
	Duration    time.Duration          `json:"duration"`
}

// DataPoller handles polling external URLs for private plugin data
type DataPoller struct {
	client *http.Client
}

// NewDataPoller creates a new data poller
func NewDataPoller() *DataPoller {
	return &DataPoller{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PollData polls all configured URLs for a private plugin
func (p *DataPoller) PollData(ctx context.Context, plugin *database.PrivatePlugin) (*PolledData, error) {
	if plugin.DataStrategy != "polling" {
		return nil, fmt.Errorf("plugin is not configured for polling")
	}

	startTime := time.Now()
	result := &PolledData{
		PluginID: plugin.ID.String(),
		Data:     make(map[string]interface{}),
		PolledAt: startTime,
		Success:  true,
		Errors:   []string{},
	}

	// Parse polling configuration
	var config PollingConfig
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
	for _, urlConfig := range config.URLs {
		urlData, err := p.pollSingleURL(ctx, urlConfig, &config)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to poll %s: %v", urlConfig.URL, err))
			result.Success = false
			continue
		}

		// Store data under the specified key or use URL as key
		key := urlConfig.Key
		if key == "" {
			key = urlConfig.URL
		}
		result.Data[key] = urlData
	}

	result.Duration = time.Since(startTime)

	logging.Info("[POLLER] Polling completed", 
		"plugin_id", plugin.ID,
		"plugin_name", plugin.Name,
		"urls_count", len(config.URLs),
		"success", result.Success,
		"errors_count", len(result.Errors),
		"duration_ms", result.Duration.Milliseconds())

	return result, nil
}

// pollSingleURL polls a single URL and returns the response data
func (p *DataPoller) pollSingleURL(ctx context.Context, urlConfig URLConfig, config *PollingConfig) (interface{}, error) {
	var lastErr error

	// Retry logic
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

		data, err := p.fetchURL(ctx, urlConfig, config)
		if err == nil {
			return data, nil
		}

		lastErr = err
		logging.Warn("[POLLER] URL fetch failed, retrying",
			"url", urlConfig.URL,
			"attempt", attempt+1,
			"max_attempts", config.RetryCount+1,
			"error", err)
	}

	return nil, fmt.Errorf("all attempts failed: %w", lastErr)
}

// fetchURL fetches data from a single URL
func (p *DataPoller) fetchURL(ctx context.Context, urlConfig URLConfig, config *PollingConfig) (interface{}, error) {
	// Create request
	method := urlConfig.Method
	if method == "" {
		method = "GET"
	}

	var bodyReader io.Reader
	if urlConfig.Body != "" {
		bodyReader = bytes.NewReader([]byte(urlConfig.Body))
	}

	req, err := http.NewRequestWithContext(ctx, method, urlConfig.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", config.UserAgent)
	
	for key, value := range urlConfig.Headers {
		req.Header.Set(key, value)
	}

	// Set content type for POST requests with body
	if method == "POST" && urlConfig.Body != "" && req.Header.Get("Content-Type") == "" {
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

// SchedulePolling sets up periodic polling for a private plugin
func (p *DataPoller) SchedulePolling(ctx context.Context, plugin *database.PrivatePlugin, callback func(*PolledData)) error {
	if plugin.DataStrategy != "polling" {
		return fmt.Errorf("plugin is not configured for polling")
	}

	// Parse polling configuration to get interval
	var config PollingConfig
	if err := json.Unmarshal(plugin.PollingConfig, &config); err != nil {
		return fmt.Errorf("invalid polling configuration: %w", err)
	}

	// Default interval
	if config.Interval == 0 {
		config.Interval = 300 // 5 minutes
	}

	// Minimum interval enforcement (prevent abuse)
	if config.Interval < 60 {
		config.Interval = 60 // Minimum 1 minute
	}

	interval := time.Duration(config.Interval) * time.Second

	// Start polling goroutine
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		logging.Info("[POLLER] Started polling for plugin",
			"plugin_id", plugin.ID,
			"plugin_name", plugin.Name,
			"interval", interval)

		for {
			select {
			case <-ctx.Done():
				logging.Info("[POLLER] Stopped polling for plugin",
					"plugin_id", plugin.ID,
					"plugin_name", plugin.Name)
				return
			case <-ticker.C:
				data, err := p.PollData(ctx, plugin)
				if err != nil {
					logging.Error("[POLLER] Polling failed",
						"plugin_id", plugin.ID,
						"plugin_name", plugin.Name,
						"error", err)
					continue
				}

				if callback != nil {
					callback(data)
				}
			}
		}
	}()

	return nil
}

// StorePolledData stores polled data for later retrieval by the plugin
func StorePolledData(db *gorm.DB, data *PolledData) error {
	// TODO: Implement storage of polled data
	// This could be stored in a separate table or cache
	// For now, we'll use a simple in-memory approach

	logging.Debug("[POLLER] Storing polled data",
		"plugin_id", data.PluginID,
		"data_keys", getMapKeys(data.Data),
		"success", data.Success)

	return nil
}

// RetrievePolledData retrieves the latest polled data for a plugin
func RetrievePolledData(db *gorm.DB, pluginID string) (*PolledData, error) {
	// TODO: Implement retrieval of polled data
	// For now, return empty data

	return &PolledData{
		PluginID: pluginID,
		Data:     make(map[string]interface{}),
		PolledAt: time.Now(),
		Success:  true,
	}, nil
}

// getMapKeys returns the keys of a map (helper function)
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}