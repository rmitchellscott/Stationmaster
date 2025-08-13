package pollers

import (
	"context"
	"sync"

	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// Manager manages multiple pollers
type Manager struct {
	pollers map[string]Poller
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

// NewManager creates a new poller manager
func NewManager() *Manager {
	return &Manager{
		pollers: make(map[string]Poller),
	}
}

// Register adds a poller to the manager
func (m *Manager) Register(poller Poller) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollers[poller.Name()] = poller
	logging.Logf("[POLLER MANAGER] Registered poller: %s", poller.Name())
}

// Unregister removes a poller from the manager
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if poller, exists := m.pollers[name]; exists {
		if poller.IsRunning() {
			poller.Stop()
		}
		delete(m.pollers, name)
		logging.Logf("[POLLER MANAGER] Unregistered poller: %s", name)
	}
}

// Start starts all registered pollers
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil // Already running
	}

	m.ctx, m.cancel = context.WithCancel(ctx)
	m.running = true

	logging.Logf("[POLLER MANAGER] Starting %d pollers", len(m.pollers))

	for name, poller := range m.pollers {
		if err := poller.Start(m.ctx); err != nil {
			logging.Logf("[POLLER MANAGER] Failed to start poller %s: %v", name, err)
			continue
		}
	}

	return nil
}

// Stop stops all pollers gracefully
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil // Already stopped
	}

	logging.Logf("[POLLER MANAGER] Stopping all pollers")

	var wg sync.WaitGroup
	for name, poller := range m.pollers {
		if poller.IsRunning() {
			wg.Add(1)
			go func(name string, p Poller) {
				defer wg.Done()
				if err := p.Stop(); err != nil {
					logging.Logf("[POLLER MANAGER] Error stopping poller %s: %v", name, err)
				}
			}(name, poller)
		}
	}

	wg.Wait()
	m.cancel()
	m.running = false

	logging.Logf("[POLLER MANAGER] All pollers stopped")
	return nil
}

// GetPoller returns a poller by name
func (m *Manager) GetPoller(name string) (Poller, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	poller, exists := m.pollers[name]
	return poller, exists
}

// ListPollers returns all registered poller names
func (m *Manager) ListPollers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.pollers))
	for name := range m.pollers {
		names = append(names, name)
	}
	return names
}

// IsRunning returns true if the manager is running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}