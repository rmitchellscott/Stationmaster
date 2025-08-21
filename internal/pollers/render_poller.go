package pollers

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// RenderPoller handles background rendering of plugin content
type RenderPoller struct {
	*BasePoller
	renderWorker *rendering.RenderWorker
	queueManager *rendering.QueueManager
}

// NewRenderPoller creates a new render poller
func NewRenderPoller(db *gorm.DB, staticDir string, config PollerConfig) (*RenderPoller, error) {
	renderWorker, err := rendering.NewRenderWorker(db, staticDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create render worker: %w", err)
	}

	queueManager := rendering.NewQueueManager(db)

	poller := &RenderPoller{
		renderWorker: renderWorker,
		queueManager: queueManager,
	}

	// Create base poller with our poll function
	poller.BasePoller = NewBasePoller(config, poller.poll)

	return poller, nil
}

// poll processes the render queue and cleans up old content
func (p *RenderPoller) poll(ctx context.Context) error {
	// Process pending render jobs
	if err := p.renderWorker.ProcessRenderQueue(ctx); err != nil {
		return fmt.Errorf("failed to process render queue: %w", err)
	}

	// Smart cleanup based on plugin refresh intervals
	if err := p.renderWorker.CleanupOldContentSmart(ctx); err != nil {
		return fmt.Errorf("failed to cleanup old content: %w", err)
	}

	// Cleanup old completed/failed/cancelled render jobs (keep 24 hours)
	if err := p.queueManager.CleanupOldJobs(ctx, 24*time.Hour); err != nil {
		return fmt.Errorf("failed to cleanup old render jobs: %w", err)
	}

	return nil
}