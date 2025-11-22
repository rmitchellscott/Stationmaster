package rendering

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const (
	socketPath       = "/tmp/liquid-renderer.sock"
	socketTimeout    = 30 * time.Second
	maxRetries       = 3
	retryDelay       = 100 * time.Millisecond
)

// EmbeddedLiquidRenderer renders Liquid templates using the embedded Ruby process via Unix socket
type EmbeddedLiquidRenderer struct {
	// Could add connection pooling here if needed
}

// NewEmbeddedLiquidRenderer creates a new embedded Liquid renderer
func NewEmbeddedLiquidRenderer() *EmbeddedLiquidRenderer {
	return &EmbeddedLiquidRenderer{}
}

// RenderTemplate renders a Liquid template with the given data
func (r *EmbeddedLiquidRenderer) RenderTemplate(
	ctx context.Context,
	template string,
	data map[string]interface{},
) (string, error) {
	var lastErr error

	// Retry logic in case of transient failures
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay * time.Duration(attempt))
		}

		html, err := r.renderWithRetry(ctx, template, data)
		if err == nil {
			return html, nil
		}

		lastErr = err
	}

	return "", fmt.Errorf("failed to render template after %d attempts: %w", maxRetries, lastErr)
}

func (r *EmbeddedLiquidRenderer) renderWithRetry(
	ctx context.Context,
	template string,
	data map[string]interface{},
) (string, error) {
	// Connect to Unix socket with timeout
	dialer := net.Dialer{
		Timeout: socketTimeout,
	}

	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return "", fmt.Errorf("failed to connect to liquid renderer socket: %w", err)
	}
	defer conn.Close()

	// Set deadline for the entire operation
	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	} else {
		conn.SetDeadline(time.Now().UTC().Add(socketTimeout))
	}

	// Prepare request
	request := map[string]interface{}{
		"template": template,
		"data":     data,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request
	_, err = conn.Write(requestJSON)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	// Signal we're done writing
	if tcpConn, ok := conn.(*net.UnixConn); ok {
		tcpConn.CloseWrite()
	}

	// Read response
	decoder := json.NewDecoder(conn)
	var response struct {
		Success   bool     `json:"success"`
		HTML      string   `json:"html"`
		Error     string   `json:"error"`
		Backtrace []string `json:"backtrace"`
	}

	err = decoder.Decode(&response)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		if len(response.Backtrace) > 0 {
			return "", fmt.Errorf("liquid rendering failed: %s\n%v", response.Error, response.Backtrace)
		}
		return "", fmt.Errorf("liquid rendering failed: %s", response.Error)
	}

	return response.HTML, nil
}

// IsAvailable checks if the embedded Ruby renderer is available
func (r *EmbeddedLiquidRenderer) IsAvailable() bool {
	conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
