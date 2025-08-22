package rendering

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	staticDir   string
	renderedDir string
}

// generateRandomString creates a cryptographically secure random string
func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based randomness if crypto/rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())[:length]
	}
	return hex.EncodeToString(bytes)[:length]
}

// NewRenderWorker creates a new render worker instance
func NewRenderWorker(db *gorm.DB, staticDir string) (*RenderWorker, error) {
	renderedDir := filepath.Join(staticDir, "rendered")

	// Ensure rendered directory exists
	if err := os.MkdirAll(renderedDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rendered directory: %w", err)
	}

	return &RenderWorker{
		db:          db,
		staticDir:   staticDir,
		renderedDir: renderedDir,
	}, nil
}

// ProcessRenderQueue processes pending render jobs
func (w *RenderWorker) ProcessRenderQueue(ctx context.Context) error {
	// Get pending render jobs, ensuring only one job per plugin instance
	// by selecting the earliest scheduled job for each user_plugin_id
	
	// First, find the earliest job ID for each user_plugin_id using a subquery
	type JobID struct {
		ID uuid.UUID
	}
	
	var jobIDs []JobID
	err := w.db.WithContext(ctx).Raw(`
		SELECT id FROM render_queues rq1
		WHERE status = ? AND scheduled_for <= ?
		AND id = (
			SELECT id FROM render_queues rq2
			WHERE rq2.user_plugin_id = rq1.user_plugin_id
			AND rq2.status = ? AND rq2.scheduled_for <= ?
			ORDER BY priority DESC, scheduled_for ASC
			LIMIT 1
		)
		GROUP BY user_plugin_id, id
		LIMIT 10
	`, "pending", time.Now(), "pending", time.Now()).Scan(&jobIDs).Error

	if err != nil {
		return fmt.Errorf("failed to find job IDs: %w", err)
	}

	if len(jobIDs) == 0 {
		return nil
	}

	// Extract IDs for the main query
	ids := make([]uuid.UUID, len(jobIDs))
	for i, jobID := range jobIDs {
		ids[i] = jobID.ID
	}

	// Now fetch the actual jobs using GORM
	var jobs []database.RenderQueue
	err = w.db.WithContext(ctx).
		Where("id IN ?", ids).
		Order("priority DESC, scheduled_for ASC").
		Find(&jobs).Error

	if err != nil {
		return fmt.Errorf("failed to fetch render jobs: %w", err)
	}

	logging.Info("[RENDER_WORKER] Processing render jobs", "job_count", len(jobs))

	for _, job := range jobs {
		if ctx.Err() != nil {
			break // Context cancelled
		}

		if err := w.processRenderJob(ctx, job); err != nil {
			logging.Error("[RENDER_WORKER] Failed to process job", "job_id", job.ID, "error", err)
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
		if err == gorm.ErrRecordNotFound {
			// Plugin instance was deleted, cancel the job instead of marking as failed
			w.markJobCancelled(ctx, job, "user plugin instance no longer exists")
		} else {
			w.markJobFailed(ctx, job, fmt.Sprintf("failed to load user plugin: %v", err))
		}
		return err
	}

	// Check if user plugin is still active
	if !userPlugin.IsActive {
		w.markJobCancelled(ctx, job, "user plugin instance is inactive")
		return nil
	}

	// Get all device models for this user's devices
	var deviceModels []database.DeviceModel
	err = w.db.WithContext(ctx).
		Distinct("device_models.id", "device_models.screen_width", "device_models.screen_height", "device_models.bit_depth").
		Joins("JOIN devices ON devices.device_model_id = device_models.id").
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
			logging.Error("[RENDER_WORKER] Failed to mark job as completed", "error", err)
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
			logging.Error("[RENDER_WORKER] Failed to render for device model", "width", deviceModel.ScreenWidth, "height", deviceModel.ScreenHeight, "error", err)
			continue // Continue with other resolutions
		}
	}

	// Mark job as completed
	err = w.db.WithContext(ctx).Model(&job).Update("status", "completed").Error
	if err != nil {
		logging.Error("[RENDER_WORKER] Failed to mark job as completed", "error", err)
	}

	// Clean up any other pending jobs for this plugin instance to prevent duplicates
	err = w.db.WithContext(ctx).Model(&database.RenderQueue{}).
		Where("user_plugin_id = ? AND status = ? AND id != ?", userPlugin.ID, "pending", job.ID).
		Update("status", "cancelled").Error
	if err != nil {
		logging.Error("[RENDER_WORKER] Failed to clean up duplicate pending jobs", "error", err)
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
		logging.Debug("[RENDER_WORKER] Skipping render - plugin doesn't require processing", "plugin_type", userPlugin.Plugin.Type)
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
		// Check if plugin provided image data (new approach)
		if imageData, ok := plugins.GetImageData(response); ok {
			// Save image data to disk using same pattern as data plugins
			randomString := generateRandomString(10)
			filename := fmt.Sprintf("%s_%s_%dx%d_%d_%s.png",
				userPlugin.ID, userPlugin.Plugin.Type,
				deviceModel.ScreenWidth, deviceModel.ScreenHeight, deviceModel.BitDepth, randomString)
			imagePath = filepath.Join(w.renderedDir, filename)

			err = os.WriteFile(imagePath, imageData, 0644)
			if err != nil {
				return fmt.Errorf("failed to save image plugin image: %w", err)
			}

			fileSize = int64(len(imageData))
		} else {
			// Fallback to URL reference for backward compatibility
			imageURL, ok := plugins.GetImageURL(response)
			if !ok {
				return fmt.Errorf("image plugin response missing image URL and image data")
			}
			imagePath = imageURL
			fileSize = 0 // URL reference, no local file
		}
	} else if plugin.PluginType() == plugins.PluginTypeData {
		// Data plugins are not supported without HTML renderer
		logging.Debug("[RENDER_WORKER] Skipping data plugin rendering - HTML rendering not available", "plugin_type", userPlugin.Plugin.Type)
		return nil // Skip this render, don't error
	}

	// Clean up old rendered content and files for this resolution
	var oldContent []database.RenderedContent
	err = w.db.WithContext(ctx).
		Where("user_plugin_id = ? AND width = ? AND height = ? AND bit_depth = ?",
			userPlugin.ID, deviceModel.ScreenWidth, deviceModel.ScreenHeight, deviceModel.BitDepth).
		Find(&oldContent).Error
	if err != nil {
		logging.Error("[RENDER_WORKER] Failed to find old content", "error", err)
	} else {
		// Delete old image files
		for _, content := range oldContent {
			var fullPath string
			if filepath.IsAbs(content.ImagePath) {
				fullPath = content.ImagePath
			} else {
				// Convert relative path to absolute
				fullPath = filepath.Join(w.staticDir, content.ImagePath)
			}
			
			if filepath.HasPrefix(fullPath, w.renderedDir) {
				if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
					logging.Error("[RENDER_WORKER] Failed to delete old image", "path", fullPath, "error", err)
				} else if err == nil {
					logging.Debug("[RENDER_WORKER] Deleted old image", "path", fullPath)
				}
			}
		}
		
		// Delete database records
		err = w.db.WithContext(ctx).
			Where("user_plugin_id = ? AND width = ? AND height = ? AND bit_depth = ?",
				userPlugin.ID, deviceModel.ScreenWidth, deviceModel.ScreenHeight, deviceModel.BitDepth).
			Delete(&database.RenderedContent{}).Error
		if err != nil {
			logging.Error("[RENDER_WORKER] Failed to clean up old content records", "error", err)
		}
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

	logging.Info("[RENDER_WORKER] Rendered plugin", "type", userPlugin.Plugin.Type, "width", deviceModel.ScreenWidth, "height", deviceModel.ScreenHeight, "bit_depth", deviceModel.BitDepth, "path", imagePath)

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
		logging.Info("[RENDER_WORKER] Skipping next render schedule for %s - doesn't require processing", userPlugin.Plugin.Type)
		return
	}

	// Check if there's already a pending job for this plugin instance
	var existingCount int64
	err := w.db.WithContext(ctx).Model(&database.RenderQueue{}).
		Where("user_plugin_id = ? AND status = ?", userPlugin.ID, "pending").
		Count(&existingCount).Error
	if err != nil {
		logging.Info("[RENDER_WORKER] Failed to check existing jobs for %s: %v", userPlugin.ID, err)
		return
	}

	if existingCount > 0 {
		logging.Info("[RENDER_WORKER] Skipping next render schedule for %s - already has pending job", userPlugin.Name)
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

	err = w.db.WithContext(ctx).Create(&renderJob).Error
	if err != nil {
		logging.Info("[RENDER_WORKER] Failed to schedule next render: %v", err)
	} else {
		logging.Info("[RENDER_WORKER] Scheduled next render for %s at %s",
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
		logging.Info("[RENDER_WORKER] Failed to mark job as failed: %v", err)
	}
}

// markJobCancelled marks a render job as cancelled with a reason message
func (w *RenderWorker) markJobCancelled(ctx context.Context, job database.RenderQueue, reason string) {
	err := w.db.WithContext(ctx).Model(&job).Updates(database.RenderQueue{
		Status:       "cancelled",
		ErrorMessage: reason,
	}).Error
	if err != nil {
		logging.Error("[RENDER_WORKER] Failed to mark job as cancelled", "error", err)
	} else {
		logging.Info("[RENDER_WORKER] Cancelled job", "job_id", job.ID, "reason", reason)
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
				logging.Info("[RENDER_WORKER] Failed to delete file %s: %v", content.ImagePath, err)
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
		logging.Info("[RENDER_WORKER] Cleaned up %d old rendered content items", len(oldContent))
	}

	return nil
}

// CleanupOldContentSmart removes old rendered content based on plugin refresh intervals
func (w *RenderWorker) CleanupOldContentSmart(ctx context.Context) error {
	// Find all unique UserPlugin IDs that have rendered content
	var userPluginIDs []uuid.UUID
	err := w.db.WithContext(ctx).
		Model(&database.RenderedContent{}).
		Distinct("user_plugin_id").
		Pluck("user_plugin_id", &userPluginIDs).Error
	if err != nil {
		return fmt.Errorf("failed to get user plugin IDs: %w", err)
	}

	totalCleaned := 0
	
	for _, userPluginID := range userPluginIDs {
		// Get the user plugin to access its refresh interval
		var userPlugin database.UserPlugin
		err := w.db.WithContext(ctx).First(&userPlugin, userPluginID).Error
		if err != nil {
			logging.Info("[RENDER_WORKER] Skipping cleanup for missing plugin %s: %v", userPluginID, err)
			continue
		}

		// Calculate retention period based on refresh interval (keep 2-3 refresh cycles)
		// Minimum 1 hour, maximum 24 hours
		refreshInterval := time.Duration(userPlugin.RefreshInterval) * time.Second
		retentionPeriod := refreshInterval * 3 // Keep 3 refresh cycles
		
		// Apply bounds
		minRetention := 1 * time.Hour
		maxRetention := 24 * time.Hour
		if retentionPeriod < minRetention {
			retentionPeriod = minRetention
		} else if retentionPeriod > maxRetention {
			retentionPeriod = maxRetention
		}
		
		cutoff := time.Now().Add(-retentionPeriod)
		
		// Find old content for this specific user plugin
		var oldContent []database.RenderedContent
		err = w.db.WithContext(ctx).
			Where("user_plugin_id = ? AND rendered_at < ?", userPluginID, cutoff).
			Find(&oldContent).Error
		if err != nil {
			logging.Info("[RENDER_WORKER] Failed to find old content for plugin %s: %v", userPluginID, err)
			continue
		}
		
		if len(oldContent) == 0 {
			continue
		}

		// Delete files for this plugin
		filesDeleted := 0
		for _, content := range oldContent {
			var fullPath string
			if filepath.IsAbs(content.ImagePath) {
				fullPath = content.ImagePath
			} else {
				// Convert relative path to absolute
				fullPath = filepath.Join(w.staticDir, content.ImagePath)
			}
			
			if filepath.HasPrefix(fullPath, w.renderedDir) {
				if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
					logging.Error("[RENDER_WORKER] Failed to delete old image", "path", fullPath, "error", err)
				} else if err == nil {
					filesDeleted++
				}
			}
		}
		
		// Delete database records for this plugin
		err = w.db.WithContext(ctx).
			Where("user_plugin_id = ? AND rendered_at < ?", userPluginID, cutoff).
			Delete(&database.RenderedContent{}).Error
		if err != nil {
			logging.Info("[RENDER_WORKER] Failed to delete old content records for plugin %s: %v", userPluginID, err)
			continue
		}
		
		totalCleaned += len(oldContent)
		if len(oldContent) > 0 {
			logging.Info("[RENDER_WORKER] Plugin %s (refresh: %v): cleaned up %d items (retention: %v)",
				userPlugin.Name, refreshInterval, len(oldContent), retentionPeriod)
		}
	}

	if totalCleaned > 0 {
		logging.Info("[RENDER_WORKER] Smart cleanup completed: %d total items removed", totalCleaned)
	}

	return nil
}

// CleanupOrphanedFiles removes image files that exist but have no corresponding database records
func (w *RenderWorker) CleanupOrphanedFiles(ctx context.Context) error {
	// Get all files in the rendered directory
	files, err := filepath.Glob(filepath.Join(w.renderedDir, "*.png"))
	if err != nil {
		return fmt.Errorf("failed to list rendered files: %w", err)
	}

	if len(files) == 0 {
		return nil
	}

	// Get all image paths from database
	var dbPaths []string
	err = w.db.WithContext(ctx).Model(&database.RenderedContent{}).
		Pluck("image_path", &dbPaths).Error
	if err != nil {
		return fmt.Errorf("failed to get database image paths: %w", err)
	}

	// Convert database paths to absolute paths for comparison
	dbAbsPaths := make(map[string]bool)
	for _, dbPath := range dbPaths {
		var fullPath string
		if filepath.IsAbs(dbPath) {
			fullPath = dbPath
		} else {
			// Database paths already include the static directory (e.g., "static/rendered/file.png")
			// so we don't need to add w.staticDir prefix
			fullPath = dbPath
		}
		dbAbsPaths[fullPath] = true
	}

	// Find orphaned files and delete them
	orphanedCount := 0
	for _, file := range files {
		if !dbAbsPaths[file] {
			if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
				logging.Error("[RENDER_WORKER] Failed to delete orphaned file", "path", file, "error", err)
			} else if err == nil {
				orphanedCount++
				logging.Debug("[RENDER_WORKER] Deleted orphaned file", "path", file)
			}
		}
	}

	if orphanedCount > 0 {
		logging.Info("[RENDER_WORKER] Cleaned up orphaned files", "count", orphanedCount)
	}

	return nil
}
