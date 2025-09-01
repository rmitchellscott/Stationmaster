package rendering

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
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
		SELECT rq1.id FROM render_queues rq1
		JOIN plugin_instances pi ON rq1.plugin_instance_id = pi.id
		WHERE rq1.status = ? AND rq1.scheduled_for <= ?
		AND pi.needs_config_update = false
		AND rq1.id = (
			SELECT rq2.id FROM render_queues rq2
			JOIN plugin_instances pi2 ON rq2.plugin_instance_id = pi2.id
			WHERE rq2.plugin_instance_id = rq1.plugin_instance_id
			AND rq2.status = ? AND rq2.scheduled_for <= ?
			AND pi2.needs_config_update = false
			ORDER BY rq2.priority DESC, rq2.scheduled_for ASC
			LIMIT 1
		)
		GROUP BY rq1.plugin_instance_id, rq1.id
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
	if pluginInstance.PluginDefinition.ID == "" {
		w.markJobCancelled(ctx, job, "plugin definition not found or inactive")
		return nil
	}

	// Check if plugin definition is still active
	if !pluginInstance.PluginDefinition.IsActive {
		w.markJobCancelled(ctx, job, "plugin definition is inactive")
		return nil
	}

	// Get only devices that have this plugin instance in their playlists
	playlistService := database.NewPlaylistService(w.db)
	devices, err := playlistService.GetDevicesUsingPluginInstance(pluginInstance.ID)
	if err != nil {
		w.markJobFailed(ctx, job, fmt.Sprintf("failed to load devices using plugin instance: %v", err))
		return err
	}

	if len(devices) == 0 {
		// No devices using this plugin instance, skip rendering
		logging.Info("[RENDER_WORKER] No devices using plugin instance, marking job as completed", "plugin_instance_id", pluginInstance.ID)
		err = w.db.WithContext(ctx).Model(&job).Update("status", "completed").Error
		if err != nil {
			logging.Error("[RENDER_WORKER] Failed to mark job as completed", "error", err)
		}
		return nil
	}

	// Process plugin and render for each individual device
	for _, device := range devices {
		if ctx.Err() != nil {
			break
		}

		// Skip devices without a device model
		if device.DeviceModel == nil {
			logging.Warn("[RENDER_WORKER] Skipping device without device model", "device_id", device.ID, "friendly_id", device.FriendlyID)
			continue
		}

		err := w.renderForDevice(ctx, pluginInstance, device)
		if err != nil {
			logging.Error("[RENDER_WORKER] Failed to render for device", "device_id", device.ID, "friendly_id", device.FriendlyID, "error", err)
			continue // Continue with other devices
		}
	}


	// Mark job as completed
	err = w.db.WithContext(ctx).Model(&job).Update("status", "completed").Error
	if err != nil {
		logging.Error("[RENDER_WORKER] Failed to mark job as completed", "error", err)
	}
	

	// Clean up any other pending jobs for this plugin instance to prevent duplicates
	err = w.db.WithContext(ctx).Model(&database.RenderQueue{}).
		Where("plugin_instance_id = ? AND status = ? AND id != ?", pluginInstance.ID, "pending", job.ID).
		Update("status", "cancelled").Error
	if err != nil {
		logging.Error("[RENDER_WORKER] Failed to clean up duplicate pending jobs", "error", err)
	}

	// Schedule next render based on explicit flag
	w.scheduleNextRenderWithOptions(ctx, pluginInstance, job.IndependentRender)

	return nil
}

// renderForDevice renders a plugin for a specific device
func (w *RenderWorker) renderForDevice(ctx context.Context, pluginInstance database.PluginInstance, device database.Device) error {
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
	} else if pluginInstance.PluginDefinition.PluginType == "mashup" {
		// Use mashup plugin factory
		plugin, err = w.factory.CreatePlugin(&pluginInstance.PluginDefinition, &pluginInstance)
		if err != nil {
			return fmt.Errorf("failed to create mashup plugin: %w", err)
		}
	} else if pluginInstance.PluginDefinition.PluginType == "external" {
		// Use external plugin factory
		plugin, err = w.factory.CreatePlugin(&pluginInstance.PluginDefinition, &pluginInstance)
		if err != nil {
			return fmt.Errorf("failed to create external plugin: %w", err)
		}
	} else {
		return fmt.Errorf("unknown plugin type: %s", pluginInstance.PluginDefinition.PluginType)
	}

	// Skip rendering for plugins that don't require processing
	if !plugin.RequiresProcessing() {
		logging.Debug("[RENDER_WORKER] Skipping render - plugin doesn't require processing", "plugin_type", pluginInstance.PluginDefinition.PluginType)
		return nil
	}

	// Fetch user data for plugin context
	user, err := database.NewUserService(w.db).GetUserByID(pluginInstance.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// Create plugin context with real device instance
	pluginCtx, err := plugins.NewPluginContext(&device, &pluginInstance, user)
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
			"plugin", pluginInstance.Name, "device", device.FriendlyID, "device_model", fmt.Sprintf("%dx%d", device.DeviceModel.ScreenWidth, device.DeviceModel.ScreenHeight))
		return nil
	}

	// Note: Old content cleanup now happens AFTER new content is saved to avoid
	// "record not found" errors during hash comparison

	var imagePath string
	var fileSize int64
	var contentHash *string
	var contentChanged bool = true // Default to true, set to false if content unchanged

	if plugin.PluginType() == plugins.PluginTypeImage {
		// Check if plugin provided image data (new approach)
		if imageData, ok := plugins.GetImageData(response); ok {
			var processedImageData []byte

			// For private plugins, external plugins, mashups, and system plugins that render via browserless, we need to process the image to correct bit depth
			if pluginInstance.PluginDefinition.PluginType == "private" || pluginInstance.PluginDefinition.PluginType == "external" || 
			   pluginInstance.PluginDefinition.PluginType == "mashup" || (pluginInstance.PluginDefinition.PluginType == "system" && plugin.RequiresProcessing()) {
				// Decode the raw PNG image from browserless
				img, _, err := image.Decode(bytes.NewReader(imageData))
				if err != nil {
					return fmt.Errorf("failed to decode browserless plugin image: %w", err)
				}

				// Convert to grayscale and quantize to target bit depth (no dithering)
				quantizedImg := imageprocessing.QuantizeToGrayscalePalette(img, device.DeviceModel.BitDepth)
				if quantizedImg == nil {
					return fmt.Errorf("failed to quantize browserless plugin image")
				}

				// Encode as PNG with correct bit depth
				processedImageData, err = imageprocessing.EncodePalettedPNG(quantizedImg, device.DeviceModel.BitDepth)
				if err != nil {
					return fmt.Errorf("failed to encode browserless plugin image: %w", err)
				}

				logging.Debug("[RENDER_WORKER] Processed browserless plugin image", 
					"plugin_type", pluginInstance.PluginDefinition.PluginType, 
					"device", device.FriendlyID,
					"original_size", len(imageData), 
					"processed_size", len(processedImageData),
					"bit_depth", device.DeviceModel.BitDepth)
			} else {
				// For other image plugins, use raw data (they may already be processed)
				processedImageData = imageData
			}

			// Check if processed image content has changed by comparing with existing rendered content
			newHash := w.calculateImageHash(processedImageData)
			contentHash = &newHash
			
			// Query for existing RenderedContent with same plugin_instance_id and device_id
			var existingContent database.RenderedContent
			err = w.db.WithContext(ctx).
				Where("plugin_instance_id = ? AND device_id = ?", pluginInstance.ID, device.ID).
				Order("rendered_at DESC").
				First(&existingContent).Error
			
			if err == nil && existingContent.ContentHash != nil && *existingContent.ContentHash == newHash {
				// Content unchanged - update last_checked_at and continue with job completion
				now := time.Now()
				existingContent.LastCheckedAt = &now
				existingContent.RenderAttempts = 0 // Reset attempts on successful check
				
				updateErr := w.db.WithContext(ctx).Save(&existingContent).Error
				if updateErr != nil {
					logging.Warn("[RENDER_WORKER] Failed to update last_checked_at", "error", updateErr)
				}
				
				logging.Info("[RENDER_WORKER] Content unchanged - updated last checked time",
					"plugin_instance_id", pluginInstance.ID,
					"plugin_name", pluginInstance.Name,
					"device", device.FriendlyID,
					"content_hash", newHash,
					"existing_path", existingContent.ImagePath,
					"last_checked_at", now)
				
				// Set variables for job completion logic (skip file save, use existing path)
				imagePath = existingContent.ImagePath
				fileSize = existingContent.FileSize
				contentChanged = false
				
				// Continue to job completion logic instead of returning early
			} else if err != nil && err != gorm.ErrRecordNotFound {
				// Log error but continue with render (don't fail the job)
				logging.Warn("[RENDER_WORKER] Failed to check existing content, continuing with render", 
					"error", err, "plugin_instance_id", pluginInstance.ID)
			}

			// Save processed image data to disk with simplified device-specific naming
			randomString := generateRandomString(10)
			filename := fmt.Sprintf("%s_%s_%s.png",
				pluginInstance.ID, device.ID, randomString)
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

	// Store rendered content record only if content changed
	if contentChanged {
		// Get previous hash for debugging if we have existing content
		var previousHash *string
		var existingForPreviousHash database.RenderedContent
		err = w.db.WithContext(ctx).
			Where("plugin_instance_id = ? AND device_id = ?", pluginInstance.ID, device.ID).
			Order("rendered_at DESC").
			First(&existingForPreviousHash).Error
		if err == nil && existingForPreviousHash.ContentHash != nil {
			previousHash = existingForPreviousHash.ContentHash
		}
		
		renderedContent := database.RenderedContent{
			ID:             uuid.New(),
			PluginInstanceID: pluginInstance.ID,
			DeviceID:       &device.ID, // Include specific device ID
			Width:          device.DeviceModel.ScreenWidth,
			Height:         device.DeviceModel.ScreenHeight,
			BitDepth:       device.DeviceModel.BitDepth,
			ImagePath:      imagePath,
			FileSize:       fileSize,
			ContentHash:    contentHash,
			RenderedAt:     time.Now(),
			LastCheckedAt:  nil, // Will be set on future hash checks
			PreviousHash:   previousHash,
			RenderAttempts: 0, // Reset attempts on successful render
		}

		err = w.db.WithContext(ctx).Create(&renderedContent).Error
		if err != nil {
			return fmt.Errorf("failed to store rendered content: %w", err)
		}
		
		// Cleanup old content for this plugin after successful save
		if err := w.CleanupOldContentForPlugin(ctx, pluginInstance.ID); err != nil {
			logging.Warn("[RENDER_WORKER] Failed to cleanup old content after render", "plugin_instance_id", pluginInstance.ID, "error", err)
		}
	}

	if contentChanged {
		logging.Info("[RENDER_WORKER] Rendered plugin with new content", 
			"type", pluginInstance.PluginDefinition.PluginType, 
			"plugin_name", pluginInstance.Name,
			"username", pluginInstance.User.Username,
			"device", device.FriendlyID,
			"width", device.DeviceModel.ScreenWidth, 
			"height", device.DeviceModel.ScreenHeight, 
			"bit_depth", device.DeviceModel.BitDepth, 
			"path", imagePath)
	} else {
		logging.Info("[RENDER_WORKER] Completed plugin render - content unchanged", 
			"type", pluginInstance.PluginDefinition.PluginType, 
			"plugin_name", pluginInstance.Name,
			"username", pluginInstance.User.Username,
			"device", device.FriendlyID,
			"existing_path", imagePath)
	}

	// Cleanup is now handled synchronously after content save (above)
	// This ensures proper timing and prevents race conditions

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
	} else if pluginInstance.PluginDefinition.PluginType == "mashup" {
		// Mashup plugins always require processing (they generate HTML from children)
		requiresProcessing = true
	} else if pluginInstance.PluginDefinition.PluginType == "external" {
		// External plugins always require processing (they generate HTML from external service)
		requiresProcessing = true
	} else {
		// Unknown plugin type
		logging.Error("[RENDER_WORKER] Unknown plugin type", "plugin_type", pluginInstance.PluginDefinition.PluginType, "instance_name", pluginInstance.Name, "instance_id", pluginInstance.ID)
		return
	}
	
	if !requiresProcessing {
		logging.Info("[RENDER_WORKER] Skipping next render schedule", "plugin", pluginInstance.PluginDefinition.Identifier, "type", pluginInstance.PluginDefinition.PluginType, "reason", "doesn't require processing")
		return
	}

	// Skip rescheduling for independent renders (plugin updates, playlist additions, etc.)
	if skipReschedule {
		logging.Info("[RENDER_WORKER] Skipping reschedule - independent render", "plugin", pluginInstance.Name, "instance_id", pluginInstance.ID)
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
		ID:               uuid.New(),
		PluginInstanceID: pluginInstance.ID,
		Priority:         0,
		ScheduledFor:     nextRender,
		Status:           "pending",
		IndependentRender: false, // Regular recurring jobs should continue rescheduling
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
	
	// For mashup plugins, use the minimum refresh rate of child plugins
	if pluginInstance.PluginDefinition.PluginType == "mashup" {
		mashupService := database.NewMashupService(w.db)
		childRefreshRate, err := mashupService.CalculateRefreshRate(pluginInstance.ID)
		if err != nil {
			logging.Warn("[RENDER_WORKER] Failed to calculate mashup refresh rate, using instance rate", "plugin", pluginInstance.Name, "error", err)
		} else {
			refreshInterval = childRefreshRate
			logging.Debug("[RENDER_WORKER] Using calculated mashup refresh rate", "plugin", pluginInstance.Name, "child_rate", childRefreshRate, "original_rate", pluginInstance.RefreshInterval)
		}
	}
	
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

		// Use simple "latest + 1 previous" retention policy instead of time-based
		// This ensures we always keep exactly 2 most recent versions per plugin+device combination
		// regardless of refresh interval or timing
		refreshInterval := time.Duration(pluginInstance.RefreshInterval) * time.Second
		
		// Find old content to cleanup: keep latest + 1 previous per device, delete the rest
		// Use a more sophisticated query to keep exactly 2 most recent records per plugin+device combination
		var oldContent []database.RenderedContent
		err = w.db.WithContext(ctx).Raw(`
			SELECT rc1.* FROM rendered_contents rc1
			WHERE rc1.plugin_instance_id = ?
			AND (
				SELECT COUNT(*) FROM rendered_contents rc2
				WHERE rc2.plugin_instance_id = rc1.plugin_instance_id
				AND rc2.device_id = rc1.device_id
				AND rc2.rendered_at > rc1.rendered_at
			) >= 2
		`, pluginInstanceID).Find(&oldContent).Error
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
		
		// Delete database records for this plugin using the same latest + 1 previous logic
		result := w.db.WithContext(ctx).Exec(`
			DELETE FROM rendered_contents rc1 
			WHERE rc1.plugin_instance_id = ?
			AND (
				SELECT COUNT(*) FROM rendered_contents rc2
				WHERE rc2.plugin_instance_id = rc1.plugin_instance_id
				AND rc2.device_id = rc1.device_id
				AND rc2.rendered_at > rc1.rendered_at
			) >= 2
		`, pluginInstanceID)
		err = result.Error
		if err != nil {
			logging.Info("[RENDER_WORKER] Failed to delete old content records for plugin", "plugin_instance_id", pluginInstanceID, "error", err)
			continue
		}
		
		totalCleaned += len(oldContent)
		if len(oldContent) > 0 {
			logging.Info("[RENDER_WORKER] Plugin cleanup completed", "plugin_name", pluginInstance.Name, "refresh_interval", refreshInterval, "items_cleaned", len(oldContent), "retention_policy", "latest_plus_one_previous")
		}
	}

	if totalCleaned > 0 {
		logging.Info("[RENDER_WORKER] Smart cleanup completed", "total_items_removed", totalCleaned)
	}

	return nil
}

// CleanupOldContentForPlugin removes old content for a specific plugin using latest + 1 previous retention
func (w *RenderWorker) CleanupOldContentForPlugin(ctx context.Context, pluginInstanceID uuid.UUID) error {
	
	// Find old content to cleanup: keep latest + 1 previous per device, delete the rest
	var oldContent []database.RenderedContent
	err := w.db.WithContext(ctx).Raw(`
		SELECT rc1.* FROM rendered_contents rc1
		WHERE rc1.plugin_instance_id = ?
		AND (
			SELECT COUNT(*) FROM rendered_contents rc2
			WHERE rc2.plugin_instance_id = rc1.plugin_instance_id
			AND rc2.device_id = rc1.device_id
			AND rc2.rendered_at > rc1.rendered_at
		) >= 2
	`, pluginInstanceID).Find(&oldContent).Error
	if err != nil {
		return fmt.Errorf("failed to find old content for plugin cleanup: %w", err)
	}
	
	if len(oldContent) == 0 {
		return nil // Nothing to clean up
	}

	// Delete files first
	filesDeleted := 0
	for _, content := range oldContent {
		var fullPath string
		if filepath.IsAbs(content.ImagePath) {
			fullPath = content.ImagePath
		} else {
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
	
	// Delete database records
	result := w.db.WithContext(ctx).Exec(`
		DELETE FROM rendered_contents rc1 
		WHERE rc1.plugin_instance_id = ?
		AND (
			SELECT COUNT(*) FROM rendered_contents rc2
			WHERE rc2.plugin_instance_id = rc1.plugin_instance_id
			AND rc2.device_id = rc1.device_id
			AND rc2.rendered_at > rc1.rendered_at
		) >= 2
	`, pluginInstanceID)
	
	if result.Error != nil {
		return fmt.Errorf("failed to delete old content records: %w", result.Error)
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

// calculateImageHash creates a SHA256 hash of image bytes
func (w *RenderWorker) calculateImageHash(imageBytes []byte) string {
	hash := sha256.Sum256(imageBytes)
	return fmt.Sprintf("%x", hash)
}
