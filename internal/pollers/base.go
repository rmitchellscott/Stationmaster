package pollers

import (
	"context"
	"sync"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// BasePoller provides common functionality for all pollers
type BasePoller struct {
	config   PollerConfig
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	pollFunc func(ctx context.Context) error
}

// NewBasePoller creates a new base poller instance
func NewBasePoller(config PollerConfig, pollFunc func(ctx context.Context) error) *BasePoller {
	return &BasePoller{
		config:   config,
		pollFunc: pollFunc,
	}
}

// Name returns the name of the poller
func (p *BasePoller) Name() string {
	return p.config.Name
}

// Start begins the polling loop
func (p *BasePoller) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil // Already running
	}

	if !p.config.Enabled {
		logging.Logf("[POLLER] %s is disabled, skipping start", p.config.Name)
		return nil
	}

	logging.Logf("[POLLER] Starting %s with interval %v", p.config.Name, p.config.Interval)

	p.ctx, p.cancel = context.WithCancel(ctx)
	p.running = true

	p.wg.Add(1)
	go p.pollLoop()

	return nil
}

// Stop gracefully stops the poller
func (p *BasePoller) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil // Already stopped
	}

	logging.Logf("[POLLER] Stopping %s", p.config.Name)

	p.cancel()
	p.wg.Wait()
	p.running = false

	logging.Logf("[POLLER] %s stopped", p.config.Name)
	return nil
}

// IsRunning returns true if the poller is currently running
func (p *BasePoller) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// GetInterval returns the polling interval
func (p *BasePoller) GetInterval() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config.Interval
}

// SetInterval updates the polling interval
func (p *BasePoller) SetInterval(interval time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config.Interval = interval
	logging.Logf("[POLLER] Updated %s interval to %v", p.config.Name, interval)
}

// pollLoop runs the main polling loop
func (p *BasePoller) pollLoop() {
	defer p.wg.Done()

	// Run once immediately
	p.executeWithRetry()

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.executeWithRetry()
		}
	}
}

// executeWithRetry executes the poll function with retry logic
func (p *BasePoller) executeWithRetry() {
	for attempt := 0; attempt < p.config.MaxRetries; attempt++ {
		if p.ctx.Err() != nil {
			return // Context cancelled
		}

		ctx, cancel := context.WithTimeout(p.ctx, p.config.Timeout)
		err := p.pollFunc(ctx)
		cancel()

		if err == nil {
			return // Success
		}

		logging.Logf("[POLLER] %s attempt %d/%d failed: %v",
			p.config.Name, attempt+1, p.config.MaxRetries, err)

		if attempt < p.config.MaxRetries-1 {
			// Wait before retrying, but check for context cancellation
			select {
			case <-p.ctx.Done():
				return
			case <-time.After(p.config.RetryDelay):
				continue
			}
		}
	}

	logging.Logf("[POLLER] %s failed after %d attempts", p.config.Name, p.config.MaxRetries)
}
