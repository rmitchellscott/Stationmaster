package pollers

import (
	"context"
	"time"
)

// Poller represents a background polling service
type Poller interface {
	// Name returns the name of the poller for identification
	Name() string

	// Start begins the polling loop in a goroutine
	Start(ctx context.Context) error

	// Stop gracefully stops the poller
	Stop() error

	// IsRunning returns true if the poller is currently running
	IsRunning() bool

	// GetInterval returns the polling interval
	GetInterval() time.Duration

	// SetInterval updates the polling interval
	SetInterval(interval time.Duration)
}

// PollerConfig holds configuration for a poller
type PollerConfig struct {
	Name       string
	Interval   time.Duration
	Enabled    bool
	MaxRetries int
	RetryDelay time.Duration
	Timeout    time.Duration
}

// DefaultConfig returns a default poller configuration
func DefaultConfig(name string, interval time.Duration) PollerConfig {
	return PollerConfig{
		Name:       name,
		Interval:   interval,
		Enabled:    true,
		MaxRetries: 3,
		RetryDelay: 30 * time.Second,
		Timeout:    60 * time.Second,
	}
}
