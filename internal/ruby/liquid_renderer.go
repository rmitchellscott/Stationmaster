package ruby

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// LiquidRenderer handles Ruby-based liquid template rendering using trmnl-liquid
type LiquidRenderer struct {
	scriptPath string
	timeout    time.Duration
}

// RenderRequest represents the input to the Ruby liquid renderer
type RenderRequest struct {
	Template string                 `json:"template"`
	Data     map[string]interface{} `json:"data"`
}

// RenderResponse represents the output from the Ruby liquid renderer
type RenderResponse struct {
	Success bool   `json:"success"`
	Result  string `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NewLiquidRenderer creates a new Ruby-based liquid renderer
func NewLiquidRenderer(appDir string) (*LiquidRenderer, error) {
	scriptPath := filepath.Join(appDir, "scripts", "liquid_renderer.rb")
	
	return &LiquidRenderer{
		scriptPath: scriptPath,
		timeout:    30 * time.Second, // 30 second timeout for template rendering
	}, nil
}

// RenderTemplate processes a liquid template using the Ruby trmnl-liquid implementation
func (r *LiquidRenderer) RenderTemplate(ctx context.Context, template string, data map[string]interface{}) (string, error) {
	// Prepare request
	request := RenderRequest{
		Template: template,
		Data:     data,
	}
	
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal render request: %w", err)
	}
	
	// Debug: log the data being sent to Ruby
	logging.Debug("[RUBY_RENDERER] Sending data to Ruby script", "data_keys", getKeys(data), "template_length", len(template))
	
	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	
	// Execute Ruby script
	cmd := exec.CommandContext(ctxWithTimeout, "bundle", "exec", "ruby", r.scriptPath)
	cmd.Stdin = bytes.NewReader(requestJSON)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	logging.Debug("[RUBY_RENDERER] Executing liquid template", "template_length", len(template), "data_keys", len(data))
	
	err = cmd.Run()
	if err != nil {
		stderrStr := stderr.String()
		if ctxWithTimeout.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("Ruby liquid rendering timeout after %v", r.timeout)
		}
		return "", fmt.Errorf("Ruby script execution failed: %w, stderr: %s", err, stderrStr)
	}
	
	// Parse response
	var response RenderResponse
	err = json.Unmarshal(stdout.Bytes(), &response)
	if err != nil {
		return "", fmt.Errorf("failed to parse Ruby script response: %w, stdout: %s", err, stdout.String())
	}
	
	// Check if rendering was successful
	if !response.Success {
		return "", fmt.Errorf("Ruby liquid rendering failed: %s", response.Error)
	}
	
	logging.Debug("[RUBY_RENDERER] Template rendered successfully", "output_length", len(response.Result))
	
	return response.Result, nil
}

// RenderTemplateWithTimeout renders a template with a custom timeout
func (r *LiquidRenderer) RenderTemplateWithTimeout(ctx context.Context, template string, data map[string]interface{}, timeout time.Duration) (string, error) {
	originalTimeout := r.timeout
	r.timeout = timeout
	defer func() {
		r.timeout = originalTimeout
	}()
	
	return r.RenderTemplate(ctx, template, data)
}

// ValidateTemplate checks if a template is valid by attempting to parse it
func (r *LiquidRenderer) ValidateTemplate(ctx context.Context, template string) error {
	// Use empty data for validation
	_, err := r.RenderTemplate(ctx, template, map[string]interface{}{})
	return err
}

// GetScriptPath returns the path to the Ruby script
func (r *LiquidRenderer) GetScriptPath() string {
	return r.scriptPath
}

// getKeys returns the keys of a map for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}