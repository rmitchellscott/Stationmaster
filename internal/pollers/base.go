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
		logging.Info("[POLLER] Poller is disabled, skipping start", "name", p.config.Name)
		return nil
	}

	logging.Info("[POLLER] Starting poller", "name", p.config.Name, "interval", p.config.Interval)

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

	logging.Info("[POLLER] Stopping poller", "name", p.config.Name)

	p.cancel()
	p.wg.Wait()
	p.running = false

	logging.Info("[POLLER] Poller stopped", "name", p.config.Name)
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
	logging.Info("[POLLER] Updated poller interval", "name", p.config.Name, "interval", interval)
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

		logging.Warn("[POLLER] Poller attempt failed", "name", p.config.Name, "attempt", attempt+1, "max_retries", p.config.MaxRetries, "error", err)

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

	logging.Error("[POLLER] Poller failed after all attempts", "name", p.config.Name, "max_retries", p.config.MaxRetries)
}
