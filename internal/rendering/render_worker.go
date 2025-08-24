package rendering

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/png" // Register PNG decoder
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/imageprocessing"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// RenderWorker handles background rendering of plugin content
type RenderWorker struct {
	db          *gorm.DB
	staticDir   string
	renderedDir string
	factory     *plugins.UnifiedPluginFactory
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
		factory:     plugins.GetPluginFactory(),
	}, nil
}

// ProcessRenderQueue processes pending render jobs
func (w *RenderWorker) ProcessRenderQueue(ctx context.Context) error {
	// Get pending render jobs, ensuring only one job per plugin instance
	// by selecting the earliest scheduled job for each plugin_instance_id
	
	// First, find the earliest job ID for each plugin_instance_id using a subquery
	type JobID struct {
		ID uuid.UUID
	}
	
	var jobIDs []JobID
	err := w.db.WithContext(ctx).Raw(`
		SELECT id FROM render_queues rq1
		WHERE status = ? AND scheduled_for <= ?
		AND id = (
			SELECT id FROM render_queues rq2
			WHERE rq2.plugin_instance_id = rq1.plugin_instance_id
			AND rq2.status = ? AND rq2.scheduled_for <= ?
			ORDER BY priority DESC, scheduled_for ASC
			LIMIT 1
		)
		GROUP BY plugin_instance_id, id
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

	// Load plugin instance with associations
	var pluginInstance database.PluginInstance
	err = w.db.WithContext(ctx).
		Preload("User").
		Preload("PluginDefinition").
		First(&pluginInstance, job.PluginInstanceID).Error
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
	if !pluginInstance.IsActive {
		w.markJobCancelled(ctx, job, "user plugin instance is inactive")
		return nil
	}

	// Check if plugin definition was loaded correctly
	if pluginInstance.PluginDefinition.ID == uuid.Nil {
		w.markJobCancelled(ctx, job, "plugin definition not found or inactive")
		return nil
	}

	// Check if plugin definition is still active
	if !pluginInstance.PluginDefinition.IsActive {
		w.markJobCancelled(ctx, job, "plugin definition is inactive")
		return nil
	}

	// Get all device models for this user's devices
	var deviceModels []database.DeviceModel
	err = w.db.WithContext(ctx).
		Distinct("device_models.id", "device_models.screen_width", "device_models.screen_height", "device_models.bit_depth").
		Joins("JOIN devices ON devices.device_model_id = device_models.id").
		Where("devices.user_id = ? AND devices.is_active = ?", pluginInstance.UserID, true).
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

		err := w.renderForDeviceModel(ctx, pluginInstance, deviceModel)
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
	
	// For private plugins, persist any instance field updates (like LastImageHash)
	if pluginInstance.PluginDefinition.PluginType == "private" {
		err = w.db.WithContext(ctx).Save(&pluginInstance).Error
		if err != nil {
			logging.Warn("[RENDER_WORKER] Failed to save plugin instance updates", 
				"plugin", pluginInstance.Name, "error", err)
		}
	}

	// Clean up any other pending jobs for this plugin instance to prevent duplicates
	err = w.db.WithContext(ctx).Model(&database.RenderQueue{}).
		Where("plugin_instance_id = ? AND status = ? AND id != ?", pluginInstance.ID, "pending", job.ID).
		Update("status", "cancelled").Error
	if err != nil {
		logging.Error("[RENDER_WORKER] Failed to clean up duplicate pending jobs", "error", err)
	}

	// Schedule next render with selective force refresh logic
	shouldSkipReschedule := w.isForceRefreshWithDailyRate(job, pluginInstance.RefreshInterval)
	w.scheduleNextRenderWithOptions(ctx, pluginInstance, shouldSkipReschedule)

	return nil
}

// renderForDeviceModel renders a plugin for a specific device model resolution
func (w *RenderWorker) renderForDeviceModel(ctx context.Context, pluginInstance database.PluginInstance, deviceModel database.DeviceModel) error {
	var plugin plugins.Plugin
	var err error
	
	// Create plugin based on type
	if pluginInstance.PluginDefinition.PluginType == "private" {
		// Use private plugin factory
		plugin, err = w.factory.CreatePlugin(&pluginInstance.PluginDefinition, &pluginInstance)
		if err != nil {
			return fmt.Errorf("failed to create private plugin: %w", err)
		}
	} else if pluginInstance.PluginDefinition.PluginType == "system" {
		// System plugin - get from registry
		var exists bool
		plugin, exists = plugins.Get(pluginInstance.PluginDefinition.Identifier)
		if !exists {
			return fmt.Errorf("system plugin %s not found in registry", pluginInstance.PluginDefinition.Identifier)
		}
	} else {
		return fmt.Errorf("unknown plugin type: %s", pluginInstance.PluginDefinition.PluginType)
	}

	// Skip rendering for plugins that don't require processing
	if !plugin.RequiresProcessing() {
		logging.Debug("[RENDER_WORKER] Skipping render - plugin doesn't require processing", "plugin_type", pluginInstance.PluginDefinition.PluginType)
		return nil
	}

	// Create a mock device for plugin context - we need this for compatibility
	mockDevice := &database.Device{
		ID:          pluginInstance.UserID, // Use user ID as device ID for context
		UserID:      &pluginInstance.UserID,
		DeviceModel: &deviceModel,
	}

	// Create plugin context
	pluginCtx, err := plugins.NewPluginContext(mockDevice, &pluginInstance)
	if err != nil {
		return fmt.Errorf("failed to create plugin context: %w", err)
	}

	// Process plugin
	response, err := plugin.Process(pluginCtx)
	if err != nil {
		return fmt.Errorf("plugin processing failed: %w", err)
	}
	
	// Handle no-change responses - skip rendering
	if plugins.IsNoChangeResponse(response) {
		logging.Info("[RENDER_WORKER] Skipping render - no data changes", 
			"plugin", pluginInstance.Name, "device_model", fmt.Sprintf("%dx%d", deviceModel.ScreenWidth, deviceModel.ScreenHeight))
		return nil
	}

	// Clean up old rendered content and files for this resolution BEFORE creating new ones
	var oldContent []database.RenderedContent
	err = w.db.WithContext(ctx).
		Where("plugin_instance_id = ? AND width = ? AND height = ? AND bit_depth = ?",
			pluginInstance.ID, deviceModel.ScreenWidth, deviceModel.ScreenHeight, deviceModel.BitDepth).
		Find(&oldContent).Error
	if err != nil {
		logging.Error("[RENDER_WORKER] Failed to find old content", "error", err)
	} else {
		// Delete old image files first
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
			Where("plugin_instance_id = ? AND width = ? AND height = ? AND bit_depth = ?",
				pluginInstance.ID, deviceModel.ScreenWidth, deviceModel.ScreenHeight, deviceModel.BitDepth).
			Delete(&database.RenderedContent{}).Error
		if err != nil {
			logging.Error("[RENDER_WORKER] Failed to clean up old content records", "error", err)
		}
	}

	var imagePath string
	var fileSize int64

	if plugin.PluginType() == plugins.PluginTypeImage {
		// Check if plugin provided image data (new approach)
		if imageData, ok := plugins.GetImageData(response); ok {
			var processedImageData []byte

			// For private plugins, we need to process the image to correct bit depth
			if pluginInstance.PluginDefinition.PluginType == "private" {
				// Decode the raw PNG image from browserless
				img, _, err := image.Decode(bytes.NewReader(imageData))
				if err != nil {
					return fmt.Errorf("failed to decode private plugin image: %w", err)
				}

				// Convert to grayscale and quantize to target bit depth (no dithering)
				quantizedImg := imageprocessing.QuantizeToGrayscalePalette(img, deviceModel.BitDepth)
				if quantizedImg == nil {
					return fmt.Errorf("failed to quantize private plugin image")
				}

				// Encode as PNG with correct bit depth
				processedImageData, err = imageprocessing.EncodePalettedPNG(quantizedImg, deviceModel.BitDepth)
				if err != nil {
					return fmt.Errorf("failed to encode private plugin image: %w", err)
				}

				logging.Debug("[RENDER_WORKER] Processed private plugin image", 
					"original_size", len(imageData), 
					"processed_size", len(processedImageData),
					"bit_depth", deviceModel.BitDepth)
			} else {
				// For other image plugins, use raw data (they may already be processed)
				processedImageData = imageData
			}

			// Save processed image data to disk
			randomString := generateRandomString(10)
			filename := fmt.Sprintf("%s_%s_%dx%d_%d_%s.png",
				pluginInstance.ID, pluginInstance.PluginDefinition.PluginType,
				deviceModel.ScreenWidth, deviceModel.ScreenHeight, deviceModel.BitDepth, randomString)
			imagePath = filepath.Join(w.renderedDir, filename)

			// Write file with better error handling
			err = os.WriteFile(imagePath, processedImageData, 0644)
			if err != nil {
				return fmt.Errorf("failed to save image plugin image: %w", err)
			}

			// Verify file was actually written and is accessible
			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				return fmt.Errorf("image file was not created successfully: %s", imagePath)
			} else if err != nil {
				logging.Warn("[RENDER_WORKER] File verification failed", "path", imagePath, "error", err)
			}

			// Get final file size
			fileInfo, err := os.Stat(imagePath)
			if err != nil {
				logging.Warn("[RENDER_WORKER] Could not get file size", "path", imagePath, "error", err)
				fileSize = int64(len(processedImageData)) // Use expected size
			} else {
				fileSize = fileInfo.Size()
			}

			logging.Debug("[RENDER_WORKER] Successfully wrote image file", "path", imagePath, "size", fileSize)
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
		logging.Debug("[RENDER_WORKER] Skipping data plugin rendering - HTML rendering not available", "plugin_type", pluginInstance.PluginDefinition.PluginType)
		return nil // Skip this render, don't error
	}

	// Store rendered content record
	renderedContent := database.RenderedContent{
		ID:           uuid.New(),
		PluginInstanceID: pluginInstance.ID,  // Now nullable
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

	logging.Info("[RENDER_WORKER] Rendered plugin", 
		"type", pluginInstance.PluginDefinition.PluginType, 
		"plugin_name", pluginInstance.Name,
		"username", pluginInstance.User.Username,
		"width", deviceModel.ScreenWidth, 
		"height", deviceModel.ScreenHeight, 
		"bit_depth", deviceModel.BitDepth, 
		"path", imagePath)

	return nil
}

// scheduleNextRender schedules the next render for a plugin based on its refresh interval with timezone support
func (w *RenderWorker) scheduleNextRender(ctx context.Context, pluginInstance database.PluginInstance) {
	w.scheduleNextRenderWithOptions(ctx, pluginInstance, false)
}

// scheduleNextRenderWithOptions schedules the next render with options for force refresh handling
func (w *RenderWorker) scheduleNextRenderWithOptions(ctx context.Context, pluginInstance database.PluginInstance, skipReschedule bool) {
	if !pluginInstance.IsActive {
		return
	}

	// Check if plugin requires processing before scheduling
	var requiresProcessing bool
	
	if pluginInstance.PluginDefinition.PluginType == "private" {
		// Private plugins always require processing (they generate HTML)
		requiresProcessing = true
	} else if pluginInstance.PluginDefinition.PluginType == "system" {
		// System plugin - check registry
		plugin, exists := plugins.Get(pluginInstance.PluginDefinition.Identifier)
		if !exists {
			logging.Warn("[RENDER_WORKER] System plugin not found in registry", "plugin", pluginInstance.PluginDefinition.Identifier, "instance_name", pluginInstance.Name, "instance_id", pluginInstance.ID)
			return
		}
		requiresProcessing = plugin.RequiresProcessing()
	} else {
		// Unknown plugin type
		logging.Error("[RENDER_WORKER] Unknown plugin type", "plugin_type", pluginInstance.PluginDefinition.PluginType, "instance_name", pluginInstance.Name, "instance_id", pluginInstance.ID)
		return
	}
	
	if !requiresProcessing {
		logging.Info("[RENDER_WORKER] Skipping next render schedule", "plugin", pluginInstance.PluginDefinition.Identifier, "type", pluginInstance.PluginDefinition.PluginType, "reason", "doesn't require processing")
		return
	}

	// Skip rescheduling if requested (for force refresh on "x times daily" rates)
	if skipReschedule {
		logging.Info("[RENDER_WORKER] Skipping reschedule per request", "plugin", pluginInstance.Name, "instance_id", pluginInstance.ID)
		return
	}

	// Check if there's already a pending job for this plugin instance
	var existingCount int64
	err := w.db.WithContext(ctx).Model(&database.RenderQueue{}).
		Where("plugin_instance_id = ? AND status = ?", pluginInstance.ID, "pending").
		Count(&existingCount).Error
	if err != nil {
		logging.Info("[RENDER_WORKER] Failed to check existing jobs", "plugin_instance_id", pluginInstance.ID, "error", err)
		return
	}

	if existingCount > 0 {
		logging.Info("[RENDER_WORKER] Skipping next render schedule", "plugin", pluginInstance.Name, "instance_id", pluginInstance.ID, "reason", "already has pending job")
		return
	}

	// Calculate next render time based on refresh interval and user timezone
	nextRender, err := w.calculateNextRenderTime(ctx, pluginInstance)
	if err != nil {
		logging.Error("[RENDER_WORKER] Failed to calculate next render time", "error", err, "plugin", pluginInstance.Name)
		// Fallback to simple interval scheduling
		nextRender = time.Now().Add(time.Duration(pluginInstance.RefreshInterval) * time.Second)
	}

	renderJob := database.RenderQueue{
		ID:           uuid.New(),
		PluginInstanceID: pluginInstance.ID,
		Priority:     0,
		ScheduledFor: nextRender,
		Status:       "pending",
	}

	err = w.db.WithContext(ctx).Create(&renderJob).Error
	if err != nil {
		logging.Info("[RENDER_WORKER] Failed to schedule next render", "error", err)
	} else {
		logging.Info("[RENDER_WORKER] Scheduled next render", 
			"plugin", pluginInstance.Name, 
			"username", pluginInstance.User.Username,
			"plugin_type", pluginInstance.PluginDefinition.PluginType,
			"scheduled_for", nextRender.Format(time.RFC3339))
	}
}

// calculateNextRenderTime calculates the next render time based on refresh interval and user timezone
func (w *RenderWorker) calculateNextRenderTime(ctx context.Context, pluginInstance database.PluginInstance) (time.Time, error) {
	// Get user timezone
	userTimezone := "UTC" // Default fallback
	if pluginInstance.User.Timezone != "" {
		userTimezone = pluginInstance.User.Timezone
	}
	
	location, err := time.LoadLocation(userTimezone)
	if err != nil {
		logging.Warn("[RENDER_WORKER] Invalid timezone, using UTC", "timezone", userTimezone, "error", err)
		location = time.UTC
	}
	
	now := time.Now().In(location)
	refreshInterval := pluginInstance.RefreshInterval
	
	// Handle different refresh intervals with smart scheduling
	switch refreshInterval {
	case database.RefreshRateDaily: // Daily - 00:15 local time
		nextRender := time.Date(now.Year(), now.Month(), now.Day(), 0, 15, 0, 0, location)
		if nextRender.Before(now) {
			nextRender = nextRender.Add(24 * time.Hour) // Next day
		}
		return nextRender.UTC(), nil
		
	case database.RefreshRate2xDay: // Twice daily - 12:00, 00:00 local time
		schedules := []int{0, 12} // Hours: midnight, noon
		return w.findNextScheduledTime(now, location, schedules), nil
		
	case database.RefreshRate3xDay: // 3 times daily - 08:00, 16:00, 00:00 local time
		schedules := []int{0, 8, 16} // Hours: midnight, 8am, 4pm
		return w.findNextScheduledTime(now, location, schedules), nil
		
	case database.RefreshRate4xDay: // 4 times daily - 06:00, 12:00, 18:00, 00:00 local time
		schedules := []int{0, 6, 12, 18} // Hours: midnight, 6am, noon, 6pm
		return w.findNextScheduledTime(now, location, schedules), nil
		
	default:
		// For interval-based rates (15min, 30min, hourly, etc.), use simple addition
		return time.Now().Add(time.Duration(refreshInterval) * time.Second), nil
	}
}

// findNextScheduledTime finds the next scheduled time from a list of hours
func (w *RenderWorker) findNextScheduledTime(now time.Time, location *time.Location, scheduleHours []int) time.Time {
	currentHour := now.Hour()
	currentMinute := now.Minute()
	
	// Find the next scheduled hour today
	for _, hour := range scheduleHours {
		if hour > currentHour || (hour == currentHour && currentMinute < 15) {
			// Schedule at :15 minutes past the hour (like TRMNL's 00:15)
			nextRender := time.Date(now.Year(), now.Month(), now.Day(), hour, 15, 0, 0, location)
			return nextRender.UTC()
		}
	}
	
	// No more scheduled times today, use first schedule tomorrow
	nextRender := time.Date(now.Year(), now.Month(), now.Day()+1, scheduleHours[0], 15, 0, 0, location)
	return nextRender.UTC()
}

// isForceRefreshWithDailyRate determines if this is a force refresh job with a daily-type rate
// Returns true if we should skip rescheduling (preserve original schedule)
func (w *RenderWorker) isForceRefreshWithDailyRate(job database.RenderQueue, refreshInterval int) bool {
	// Check if this is a force refresh job (priority 999)
	if job.Priority != 999 {
		return false
	}
	
	// Check if this is a "x times daily" rate that should preserve schedule
	dailyRates := []int{
		database.RefreshRateDaily,   // Daily
		database.RefreshRate2xDay,   // Twice daily 
		database.RefreshRate3xDay,   // 3 times daily
		database.RefreshRate4xDay,   // 4 times daily
	}
	
	for _, dailyRate := range dailyRates {
		if refreshInterval == dailyRate {
			return true
		}
	}
	
	// For interval-based rates (15min, 30min, hourly, etc.), allow normal rescheduling
	return false
}


// markJobFailed marks a render job as failed with an error message
func (w *RenderWorker) markJobFailed(ctx context.Context, job database.RenderQueue, errorMsg string) {
	err := w.db.WithContext(ctx).Model(&job).Updates(database.RenderQueue{
		Status:       "failed",
		ErrorMessage: errorMsg,
	}).Error
	if err != nil {
		logging.Info("[RENDER_WORKER] Failed to mark job as failed", "error", err)
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
				logging.Info("[RENDER_WORKER] Failed to delete file", "path", content.ImagePath, "error", err)
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
		logging.Info("[RENDER_WORKER] Cleaned up old rendered content items", "count", len(oldContent))
	}

	return nil
}

// CleanupOldContentSmart removes old rendered content based on plugin refresh intervals
func (w *RenderWorker) CleanupOldContentSmart(ctx context.Context) error {
	// Find all unique PluginInstance IDs that have rendered content
	var pluginInstanceIDs []uuid.UUID
	err := w.db.WithContext(ctx).
		Model(&database.RenderedContent{}).
		Distinct("plugin_instance_id").
		Pluck("plugin_instance_id", &pluginInstanceIDs).Error
	if err != nil {
		return fmt.Errorf("failed to get user plugin IDs: %w", err)
	}

	totalCleaned := 0
	
	for _, pluginInstanceID := range pluginInstanceIDs {
		// Get the plugin instance to access its refresh interval
		var pluginInstance database.PluginInstance
		err := w.db.WithContext(ctx).First(&pluginInstance, pluginInstanceID).Error
		if err != nil {
			logging.Info("[RENDER_WORKER] Skipping cleanup for missing plugin", "plugin_instance_id", pluginInstanceID, "error", err)
			continue
		}

		// Calculate retention period based on refresh interval (keep 2-3 refresh cycles)
		// Minimum 1 hour, maximum 24 hours
		refreshInterval := time.Duration(pluginInstance.RefreshInterval) * time.Second
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
			Where("plugin_instance_id = ? AND rendered_at < ?", pluginInstanceID, cutoff).
			Find(&oldContent).Error
		if err != nil {
			logging.Info("[RENDER_WORKER] Failed to find old content for plugin", "plugin_instance_id", pluginInstanceID, "error", err)
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
			Where("plugin_instance_id = ? AND rendered_at < ?", pluginInstanceID, cutoff).
			Delete(&database.RenderedContent{}).Error
		if err != nil {
			logging.Info("[RENDER_WORKER] Failed to delete old content records for plugin", "plugin_instance_id", pluginInstanceID, "error", err)
			continue
		}
		
		totalCleaned += len(oldContent)
		if len(oldContent) > 0 {
			logging.Info("[RENDER_WORKER] Plugin cleanup completed", "plugin_name", pluginInstance.Name, "refresh_interval", refreshInterval, "items_cleaned", len(oldContent), "retention_period", retentionPeriod)
		}
	}

	if totalCleaned > 0 {
		logging.Info("[RENDER_WORKER] Smart cleanup completed", "total_items_removed", totalCleaned)
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
