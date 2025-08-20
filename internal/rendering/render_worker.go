package rendering

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// RenderWorker handles background rendering of plugin content
type RenderWorker struct {
	db          *gorm.DB
	renderer    *HTMLRenderer
	staticDir   string
	renderedDir string
}

// NewRenderWorker creates a new render worker instance
func NewRenderWorker(db *gorm.DB, staticDir string) (*RenderWorker, error) {
	// Create renderer with default options
	defaultOpts := RenderOptions{
		Width:   800,
		Height:  470,
		Quality: 90,
		DPI:     125,
	}
	renderer, err := NewHTMLRenderer(defaultOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	renderedDir := filepath.Join(staticDir, "rendered")

	// Ensure rendered directory exists
	if err := os.MkdirAll(renderedDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rendered directory: %w", err)
	}

	return &RenderWorker{
		db:          db,
		renderer:    renderer,
		staticDir:   staticDir,
		renderedDir: renderedDir,
	}, nil
}

// ProcessRenderQueue processes pending render jobs
func (w *RenderWorker) ProcessRenderQueue(ctx context.Context) error {
	// Get pending render jobs ordered by priority and scheduled time
	var jobs []database.RenderQueue
	err := w.db.WithContext(ctx).
		Where("status = ? AND scheduled_for <= ?", "pending", time.Now()).
		Order("priority DESC, scheduled_for ASC").
		Limit(10). // Process up to 10 jobs at once
		Find(&jobs).Error

	if err != nil {
		return fmt.Errorf("failed to fetch render jobs: %w", err)
	}

	if len(jobs) == 0 {
		logging.Logf("[RENDER_WORKER] No pending render jobs")
		return nil
	}

	logging.Logf("[RENDER_WORKER] Processing %d render jobs", len(jobs))

	for _, job := range jobs {
		if ctx.Err() != nil {
			break // Context cancelled
		}

		if err := w.processRenderJob(ctx, job); err != nil {
			logging.Logf("[RENDER_WORKER] Failed to process job %s: %v", job.ID, err)
		}
	}

	return nil
}

// processRenderJob processes a single render job
func (w *RenderWorker) processRenderJob(ctx context.Context, job database.RenderQueue) error {
	// Mark job as processing
	now := time.Now()
	err := w.db.WithContext(ctx).Model(&job).Updates(database.RenderQueue{
		Status:      "processing",
		LastAttempt: &now,
		Attempts:    job.Attempts + 1,
	}).Error
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Load user plugin with associations
	var userPlugin database.UserPlugin
	err = w.db.WithContext(ctx).
		Preload("User").
		Preload("Plugin").
		First(&userPlugin, job.UserPluginID).Error
	if err != nil {
		w.markJobFailed(ctx, job, fmt.Sprintf("failed to load user plugin: %v", err))
		return err
	}

	// Get all device models for this user's devices
	var deviceModels []database.DeviceModel
	err = w.db.WithContext(ctx).
		Distinct("device_models.id", "device_models.screen_width", "device_models.screen_height", "device_models.bit_depth").
		Joins("JOIN devices ON devices.model_name = device_models.model_name").
		Where("devices.user_id = ? AND devices.is_active = ?", userPlugin.UserID, true).
		Find(&deviceModels).Error
	if err != nil {
		w.markJobFailed(ctx, job, fmt.Sprintf("failed to load user device models: %v", err))
		return err
	}

	if len(deviceModels) == 0 {
		// No active devices, skip rendering
		err = w.db.WithContext(ctx).Model(&job).Update("status", "completed").Error
		if err != nil {
			logging.Logf("[RENDER_WORKER] Failed to mark job as completed: %v", err)
		}
		return nil
	}

	// Process plugin and render for each device resolution
	for _, deviceModel := range deviceModels {
		if ctx.Err() != nil {
			break
		}

		err := w.renderForDeviceModel(ctx, userPlugin, deviceModel)
		if err != nil {
			logging.Logf("[RENDER_WORKER] Failed to render for device model %dx%d: %v",
				deviceModel.ScreenWidth, deviceModel.ScreenHeight, err)
			continue // Continue with other resolutions
		}
	}

	// Mark job as completed
	err = w.db.WithContext(ctx).Model(&job).Update("status", "completed").Error
	if err != nil {
		logging.Logf("[RENDER_WORKER] Failed to mark job as completed: %v", err)
	}

	// Schedule next render
	w.scheduleNextRender(ctx, userPlugin)

	return nil
}

// renderForDeviceModel renders a plugin for a specific device model resolution
func (w *RenderWorker) renderForDeviceModel(ctx context.Context, userPlugin database.UserPlugin, deviceModel database.DeviceModel) error {
	// Get plugin
	plugin, exists := plugins.Get(userPlugin.Plugin.Type)
	if !exists {
		return fmt.Errorf("plugin type %s not found", userPlugin.Plugin.Type)
	}

	// Skip rendering for plugins that don't require processing
	if !plugin.RequiresProcessing() {
		logging.Logf("[RENDER_WORKER] Skipping render for %s - plugin doesn't require processing", userPlugin.Plugin.Type)
		return nil
	}

	// Create a mock device for plugin context - we need this for compatibility
	mockDevice := &database.Device{
		ID:          userPlugin.UserID, // Use user ID as device ID for context
		UserID:      &userPlugin.UserID,
		DeviceModel: &deviceModel,
	}

	// Create plugin context
	pluginCtx, err := plugins.NewPluginContext(mockDevice, &userPlugin)
	if err != nil {
		return fmt.Errorf("failed to create plugin context: %w", err)
	}

	// Process plugin
	response, err := plugin.Process(pluginCtx)
	if err != nil {
		return fmt.Errorf("plugin processing failed: %w", err)
	}

	var imagePath string
	var fileSize int64

	if plugin.PluginType() == plugins.PluginTypeImage {
		// For image plugins, we just store the URL reference
		imageURL, ok := plugins.GetImageURL(response)
		if !ok {
			return fmt.Errorf("image plugin response missing image URL")
		}
		imagePath = imageURL
		fileSize = 0 // URL reference, no local file
	} else if plugin.PluginType() == plugins.PluginTypeData {
		// For data plugins, render to image
		dataPlugin, ok := plugin.(plugins.DataPlugin)
		if !ok {
			return fmt.Errorf("plugin claims to be data type but doesn't implement DataPlugin interface")
		}

		// Get template from data plugin
		template := dataPlugin.RenderTemplate()

		// Get data from response
		data, ok := plugins.GetData(response)
		if !ok {
			return fmt.Errorf("data plugin response missing data")
		}

		// Set DPI based on bit depth
		dpi := 200
		if deviceModel.BitDepth == 1 {
			dpi = 150 // Lower DPI for 1-bit displays
		}

		renderOpts := RenderOptions{
			Width:   deviceModel.ScreenWidth,
			Height:  deviceModel.ScreenHeight,
			Quality: 90,
			DPI:     dpi,
		}

		imageData, err := w.renderer.RenderTemplateToImage(ctx, template, data, renderOpts)
		if err != nil {
			return fmt.Errorf("failed to render image: %w", err)
		}

		// Save image to disk
		filename := fmt.Sprintf("%s_%s_%dx%d_%d.png",
			userPlugin.ID, userPlugin.Plugin.Type,
			deviceModel.ScreenWidth, deviceModel.ScreenHeight, deviceModel.BitDepth)
		imagePath = filepath.Join(w.renderedDir, filename)

		err = os.WriteFile(imagePath, imageData, 0644)
		if err != nil {
			return fmt.Errorf("failed to save rendered image: %w", err)
		}

		fileSize = int64(len(imageData))
	}

	// Clean up old rendered content for this resolution
	err = w.db.WithContext(ctx).
		Where("user_plugin_id = ? AND width = ? AND height = ? AND bit_depth = ?",
			userPlugin.ID, deviceModel.ScreenWidth, deviceModel.ScreenHeight, deviceModel.BitDepth).
		Delete(&database.RenderedContent{}).Error
	if err != nil {
		logging.Logf("[RENDER_WORKER] Failed to clean up old content: %v", err)
	}

	// Store rendered content record
	renderedContent := database.RenderedContent{
		ID:           uuid.New(),
		UserPluginID: userPlugin.ID,
		Width:        deviceModel.ScreenWidth,
		Height:       deviceModel.ScreenHeight,
		BitDepth:     deviceModel.BitDepth,
		ImagePath:    imagePath,
		FileSize:     fileSize,
		RenderedAt:   time.Now(),
	}

	err = w.db.WithContext(ctx).Create(&renderedContent).Error
	if err != nil {
		return fmt.Errorf("failed to store rendered content: %w", err)
	}

	logging.Logf("[RENDER_WORKER] Rendered %s for %dx%d (%d-bit): %s",
		userPlugin.Plugin.Type, deviceModel.ScreenWidth, deviceModel.ScreenHeight, deviceModel.BitDepth, imagePath)

	return nil
}

// scheduleNextRender schedules the next render for a plugin based on its refresh interval
func (w *RenderWorker) scheduleNextRender(ctx context.Context, userPlugin database.UserPlugin) {
	if !userPlugin.IsActive {
		return
	}

	// Check if plugin requires processing before scheduling
	plugin, exists := plugins.Get(userPlugin.Plugin.Type)
	if !exists || !plugin.RequiresProcessing() {
		logging.Logf("[RENDER_WORKER] Skipping next render schedule for %s - doesn't require processing", userPlugin.Plugin.Type)
		return
	}

	nextRender := time.Now().Add(time.Duration(userPlugin.RefreshInterval) * time.Second)

	renderJob := database.RenderQueue{
		ID:           uuid.New(),
		UserPluginID: userPlugin.ID,
		Priority:     0,
		ScheduledFor: nextRender,
		Status:       "pending",
	}

	err := w.db.WithContext(ctx).Create(&renderJob).Error
	if err != nil {
		logging.Logf("[RENDER_WORKER] Failed to schedule next render: %v", err)
	} else {
		logging.Logf("[RENDER_WORKER] Scheduled next render for %s at %s",
			userPlugin.Name, nextRender.Format(time.RFC3339))
	}
}

// markJobFailed marks a render job as failed with an error message
func (w *RenderWorker) markJobFailed(ctx context.Context, job database.RenderQueue, errorMsg string) {
	err := w.db.WithContext(ctx).Model(&job).Updates(database.RenderQueue{
		Status:       "failed",
		ErrorMessage: errorMsg,
	}).Error
	if err != nil {
		logging.Logf("[RENDER_WORKER] Failed to mark job as failed: %v", err)
	}
}

// CleanupOldContent removes old rendered content and files
func (w *RenderWorker) CleanupOldContent(ctx context.Context, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)

	// Find old rendered content
	var oldContent []database.RenderedContent
	err := w.db.WithContext(ctx).
		Where("rendered_at < ?", cutoff).
		Find(&oldContent).Error
	if err != nil {
		return fmt.Errorf("failed to find old content: %w", err)
	}

	for _, content := range oldContent {
		// Delete file if it's a local file (not a URL)
		if filepath.IsAbs(content.ImagePath) && filepath.HasPrefix(content.ImagePath, w.renderedDir) {
			if err := os.Remove(content.ImagePath); err != nil && !os.IsNotExist(err) {
				logging.Logf("[RENDER_WORKER] Failed to delete file %s: %v", content.ImagePath, err)
			}
		}
	}

	// Delete database records
	err = w.db.WithContext(ctx).
		Where("rendered_at < ?", cutoff).
		Delete(&database.RenderedContent{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete old content records: %w", err)
	}

	if len(oldContent) > 0 {
		logging.Logf("[RENDER_WORKER] Cleaned up %d old rendered content items", len(oldContent))
	}

	return nil
}
