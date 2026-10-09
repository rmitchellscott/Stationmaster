package pluginruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Same socket the Liquid renderer uses. Duplicated rather than imported from
// internal/rendering, which would close a cycle: plugins -> rendering -> plugins.
const (
	socketPath    = "/tmp/liquid-renderer.sock"
	socketTimeout = 30 * time.Second
	maxRetries    = 3
	retryDelay    = 100 * time.Millisecond
)

// Runtime shares the Liquid renderer's socket and constants, and replaces
// the HTTP hop to the separate stationmaster-plugins container.
type Runtime struct{}

func New() *Runtime {
	return &Runtime{}
}

func (r *Runtime) Execute(
	ctx context.Context,
	plugin string,
	layout string,
	settings map[string]interface{},
	trmnl map[string]interface{},
) (string, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay * time.Duration(attempt))
		}

		html, err := r.request(ctx, map[string]interface{}{
			"type":     "execute",
			"plugin":   plugin,
			"layout":   layout,
			"settings": settings,
			"trmnl":    trmnl,
		})
		if err == nil {
			return html, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("failed to execute plugin %s after %d attempts: %w", plugin, maxRetries, lastErr)
}

func (r *Runtime) Discover(ctx context.Context) ([]string, error) {
	conn, err := r.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := r.write(conn, map[string]interface{}{"type": "discover"}); err != nil {
		return nil, err
	}

	var response struct {
		Success bool     `json:"success"`
		Plugins []string `json:"plugins"`
		Error   string   `json:"error"`
	}
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode discover response: %w", err)
	}
	if !response.Success {
		return nil, fmt.Errorf("plugin discovery failed: %s", response.Error)
	}
	return response.Plugins, nil
}

// DiscoverMetadata returns the full plugin manifest, the same shape the Rails service
// served over HTTP, so the scanner's registration path is unchanged.
// WaitReady blocks until the Ruby process has created the socket. s6 orders service
// startup but not readiness, so without this the boot-time scan races the renderer
// and every plugin is registered unavailable.
func (r *Runtime) WaitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().UTC().Add(timeout)

	for {
		conn, err := net.DialTimeout("unix", socketPath, time.Second)
		if err == nil {
			conn.Close()
			return nil
		}

		if time.Now().UTC().After(deadline) {
			return fmt.Errorf("plugin runtime socket not ready after %s: %w", timeout, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func (r *Runtime) DiscoverMetadata(ctx context.Context) (json.RawMessage, error) {
	conn, err := r.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := r.write(conn, map[string]interface{}{"type": "discover"}); err != nil {
		return nil, err
	}

	var response struct {
		Success bool            `json:"success"`
		Plugins json.RawMessage `json:"plugins"`
		Error   string          `json:"error"`
	}
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode discover response: %w", err)
	}
	if !response.Success {
		return nil, fmt.Errorf("plugin discovery failed: %s", response.Error)
	}

	return response.Plugins, nil
}

type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func (r *Runtime) DynamicOptions(ctx context.Context, plugin, field, accessToken string) ([]Option, error) {
	conn, err := r.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := r.write(conn, map[string]interface{}{
		"type":         "dynamic_options",
		"plugin":       plugin,
		"field":        field,
		"access_token": accessToken,
	}); err != nil {
		return nil, err
	}

	var response struct {
		Success bool     `json:"success"`
		Options []Option `json:"options"`
		Error   string   `json:"error"`
	}
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode options response: %w", err)
	}
	if !response.Success {
		return nil, fmt.Errorf("option fetch failed: %s", response.Error)
	}

	return response.Options, nil
}

func (r *Runtime) request(ctx context.Context, payload map[string]interface{}) (string, error) {
	conn, err := r.dial(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if err := r.write(conn, payload); err != nil {
		return "", err
	}

	var response struct {
		Success   bool     `json:"success"`
		HTML      string   `json:"html"`
		Error     string   `json:"error"`
		Backtrace []string `json:"backtrace"`
	}
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		if len(response.Backtrace) > 0 {
			return "", fmt.Errorf("plugin execution failed: %s\n%v", response.Error, response.Backtrace)
		}
		return "", fmt.Errorf("plugin execution failed: %s", response.Error)
	}

	return response.HTML, nil
}

func (r *Runtime) dial(ctx context.Context) (net.Conn, error) {
	dialer := net.Dialer{Timeout: socketTimeout}
	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to plugin runtime socket: %w", err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	} else {
		conn.SetDeadline(time.Now().UTC().Add(socketTimeout))
	}
	return conn, nil
}

// The server reads to EOF, so the write half must close before it will answer.
func (r *Runtime) write(conn net.Conn, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	if _, err := conn.Write(body); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	if unixConn, ok := conn.(*net.UnixConn); ok {
		unixConn.CloseWrite()
	}
	return nil
}
