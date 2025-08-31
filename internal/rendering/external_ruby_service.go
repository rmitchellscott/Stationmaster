package rendering

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// ExternalRubyService handles all Ruby template rendering via external service
type ExternalRubyService struct {
	serviceURL string
	client     *http.Client
}

// NewExternalRubyService creates a new external Ruby service client
func NewExternalRubyService() *ExternalRubyService {
	serviceURL := os.Getenv("EXTERNAL_PLUGIN_SERVICES")
	if serviceURL == "" {
		serviceURL = "http://stationmaster-plugins:3000"
	}

	return &ExternalRubyService{
		serviceURL: serviceURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RenderTemplate renders a Liquid template using the external Ruby service
func (s *ExternalRubyService) RenderTemplate(ctx context.Context, template string, data map[string]interface{}) (string, error) {
	if template == "" {
		return "", fmt.Errorf("template cannot be empty")
	}

	// Create request payload
	renderRequest := map[string]interface{}{
		"template": template,
		"data":     data,
	}

	requestJSON, err := json.Marshal(renderRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal render request: %w", err)
	}

	// Build URL for render endpoint
	url := fmt.Sprintf("%s/api/render", s.serviceURL)

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create render request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("render request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("render service returned status %d", resp.StatusCode)
	}

	// Parse response
	var renderResponse struct {
		Success      bool   `json:"success"`
		RenderedHTML string `json:"rendered_html"`
		Error        string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&renderResponse); err != nil {
		return "", fmt.Errorf("failed to parse render response: %w", err)
	}

	if !renderResponse.Success {
		return "", fmt.Errorf("template rendering failed: %s", renderResponse.Error)
	}

	return renderResponse.RenderedHTML, nil
}

// ValidateTemplate validates a Liquid template using the external Ruby service
func (s *ExternalRubyService) ValidateTemplate(ctx context.Context, template string) error {
	// For validation, we can try to render with empty data
	_, err := s.RenderTemplate(ctx, template, map[string]interface{}{})
	return err
}

// IsServiceAvailable checks if the external Ruby service is reachable
func (s *ExternalRubyService) IsServiceAvailable(ctx context.Context) bool {
	url := fmt.Sprintf("%s/api/health", s.serviceURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}