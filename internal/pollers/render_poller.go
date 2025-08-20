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
}

// NewRenderPoller creates a new render poller
func NewRenderPoller(db *gorm.DB, staticDir string, config PollerConfig) (*RenderPoller, error) {
	renderWorker, err := rendering.NewRenderWorker(db, staticDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create render worker: %w", err)
	}

	poller := &RenderPoller{
		renderWorker: renderWorker,
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

	// Clean up old content (older than 7 days)
	if err := p.renderWorker.CleanupOldContent(ctx, 7*24*time.Hour); err != nil {
		return fmt.Errorf("failed to cleanup old content: %w", err)
	}

	return nil
}