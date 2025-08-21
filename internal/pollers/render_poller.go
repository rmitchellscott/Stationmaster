package pollers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// RenderPoller handles background rendering of plugin content using a worker pool
type RenderPoller struct {
	*BasePoller
	workerPool   *rendering.RenderWorkerPool
	queueManager *rendering.QueueManager
	db           *gorm.DB
}

// NewRenderPoller creates a new render poller with worker pool
func NewRenderPoller(db *gorm.DB, staticDir string, config PollerConfig) (*RenderPoller, error) {
	// Use default worker pool configuration
	// TODO: Add environment variable support later if needed
	workerCount := 3   // Default worker count
	bufferSize := 100  // Default buffer size

	workerPool, err := rendering.NewRenderWorkerPool(db, staticDir, workerCount, bufferSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create render worker pool: %w", err)
	}

	queueManager := rendering.NewQueueManager(db)

	poller := &RenderPoller{
		workerPool:   workerPool,
		queueManager: queueManager,
		db:           db,
	}

	// Create base poller with reduced polling interval since worker pool handles jobs
	// We only need periodic cleanup now
	cleanupConfig := config
	cleanupConfig.Interval = 5 * time.Minute // Cleanup every 5 minutes
	poller.BasePoller = NewBasePoller(cleanupConfig, poller.poll)

	return poller, nil
}

// Start starts the render poller and its worker pool
func (p *RenderPoller) Start(ctx context.Context) error {
	// Start the worker pool first
	if err := p.workerPool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}
	
	// Start the base poller for cleanup tasks
	return p.BasePoller.Start(ctx)
}

// Stop stops the render poller and its worker pool
func (p *RenderPoller) Stop() error {
	// Stop the base poller first
	if err := p.BasePoller.Stop(); err != nil {
		logging.Error("[RENDER_POLLER] Failed to stop base poller", "error", err)
	}
	
	// Stop the worker pool
	return p.workerPool.Stop()
}

// GetMetrics returns worker pool metrics
func (p *RenderPoller) GetMetrics() rendering.WorkerMetrics {
	return p.workerPool.GetMetrics()
}

// GetQueueStats returns queue statistics
func (p *RenderPoller) GetQueueStats(ctx context.Context) (map[string]interface{}, error) {
	return p.queueManager.GetQueueStats(ctx)
}

// ScheduleImmediateRender schedules an immediate high-priority render
func (p *RenderPoller) ScheduleImmediateRender(ctx context.Context, userPluginID string) error {
	// Convert string to UUID if needed
	pluginID, err := parseUUID(userPluginID)
	if err != nil {
		return fmt.Errorf("invalid plugin ID: %w", err)
	}
	
	return p.queueManager.ScheduleImmediateRender(ctx, pluginID)
}

// poll now only handles cleanup tasks since workers process jobs continuously
func (p *RenderPoller) poll(ctx context.Context) error {
	logging.Debug("[RENDER_POLLER] Running cleanup tasks")
	
	// The worker pool handles job processing continuously,
	// so we only need to handle cleanup here occasionally
	
	metrics := p.workerPool.GetMetrics()
	logging.Debug("[RENDER_POLLER] Worker pool metrics", 
		"total_jobs", metrics.TotalJobs,
		"success_jobs", metrics.SuccessJobs, 
		"failed_jobs", metrics.FailedJobs,
		"active_workers", metrics.ActiveWorkers,
		"queue_length", metrics.QueueLength)
	
	return nil
}

// Helper functions
func parseInt(s string) int {
	// Simple integer parsing - in production you'd want proper error handling
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}