package rendering

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// QueueManager handles scheduling and managing render jobs
type QueueManager struct {
	db *gorm.DB
}

// NewQueueManager creates a new queue manager
func NewQueueManager(db *gorm.DB) *QueueManager {
	return &QueueManager{db: db}
}

// ScheduleRender schedules a render job for a user plugin
func (qm *QueueManager) ScheduleRender(ctx context.Context, userPluginID uuid.UUID, priority int, scheduledFor time.Time) error {
	renderJob := database.RenderQueue{
		ID:           uuid.New(),
		UserPluginID: userPluginID,
		Priority:     priority,
		ScheduledFor: scheduledFor,
		Status:       "pending",
	}

	err := qm.db.WithContext(ctx).Create(&renderJob).Error
	if err != nil {
		return fmt.Errorf("failed to create render job: %w", err)
	}

	logging.Logf("[QUEUE_MANAGER] Scheduled render job for plugin %s at %s",
		userPluginID, scheduledFor.Format(time.RFC3339))

	return nil
}

// ScheduleImmediateRender schedules a high-priority immediate render
func (qm *QueueManager) ScheduleImmediateRender(ctx context.Context, userPluginID uuid.UUID) error {
	return qm.ScheduleRender(ctx, userPluginID, 100, time.Now())
}

// ScheduleInitialRenders schedules initial render jobs for all active user plugins
func (qm *QueueManager) ScheduleInitialRenders(ctx context.Context) error {
	var userPlugins []database.UserPlugin
	err := qm.db.WithContext(ctx).
		Preload("Plugin").
		Where("is_active = ?", true).
		Find(&userPlugins).Error
	if err != nil {
		return fmt.Errorf("failed to load active user plugins: %w", err)
	}

	logging.Logf("[QUEUE_MANAGER] Scheduling initial renders for %d plugins", len(userPlugins))

	for _, userPlugin := range userPlugins {
		// Check if plugin requires processing before scheduling
		plugin, exists := plugins.Get(userPlugin.Plugin.Type)
		if !exists {
			logging.Logf("[QUEUE_MANAGER] Skipping %s - plugin type not found in registry", userPlugin.Plugin.Type)
			continue
		}

		if !plugin.RequiresProcessing() {
			logging.Logf("[QUEUE_MANAGER] Skipping %s - plugin doesn't require processing", userPlugin.Plugin.Type)
			continue
		}

		// Check if there's already a pending job for this plugin
		var existingCount int64
		err := qm.db.WithContext(ctx).Model(&database.RenderQueue{}).
			Where("user_plugin_id = ? AND status = ?", userPlugin.ID, "pending").
			Count(&existingCount).Error
		if err != nil {
			logging.Logf("[QUEUE_MANAGER] Failed to check existing jobs for %s: %v", userPlugin.ID, err)
			continue
		}

		if existingCount > 0 {
			logging.Logf("[QUEUE_MANAGER] Skipping %s - already has pending job", userPlugin.ID)
			continue
		}

		// Schedule immediate render for plugin activation
		if err := qm.ScheduleImmediateRender(ctx, userPlugin.ID); err != nil {
			logging.Logf("[QUEUE_MANAGER] Failed to schedule render for %s: %v", userPlugin.ID, err)
		}
	}

	return nil
}

// UpdateRefreshInterval updates the refresh interval for a user plugin and reschedules
func (qm *QueueManager) UpdateRefreshInterval(ctx context.Context, userPluginID uuid.UUID, newInterval int) error {
	// Update the user plugin refresh interval
	err := qm.db.WithContext(ctx).Model(&database.UserPlugin{}).
		Where("id = ?", userPluginID).
		Update("refresh_interval", newInterval).Error
	if err != nil {
		return fmt.Errorf("failed to update refresh interval: %w", err)
	}

	// Cancel any pending jobs for this plugin
	err = qm.db.WithContext(ctx).Model(&database.RenderQueue{}).
		Where("user_plugin_id = ? AND status = ?", userPluginID, "pending").
		Update("status", "cancelled").Error
	if err != nil {
		logging.Logf("[QUEUE_MANAGER] Failed to cancel pending jobs: %v", err)
	}

	// Schedule a new job with the updated interval
	nextRender := time.Now().Add(time.Duration(newInterval) * time.Second)
	return qm.ScheduleRender(ctx, userPluginID, 0, nextRender)
}

// CancelPendingJobs cancels all pending jobs for a user plugin
func (qm *QueueManager) CancelPendingJobs(ctx context.Context, userPluginID uuid.UUID) error {
	err := qm.db.WithContext(ctx).Model(&database.RenderQueue{}).
		Where("user_plugin_id = ? AND status = ?", userPluginID, "pending").
		Update("status", "cancelled").Error
	if err != nil {
		return fmt.Errorf("failed to cancel pending jobs: %w", err)
	}

	logging.Logf("[QUEUE_MANAGER] Cancelled pending jobs for plugin %s", userPluginID)
	return nil
}

// GetQueueStats returns statistics about the render queue
func (qm *QueueManager) GetQueueStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count jobs by status
	var statusCounts []struct {
		Status string
		Count  int64
	}

	err := qm.db.WithContext(ctx).Model(&database.RenderQueue{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	statusMap := make(map[string]int64)
	for _, sc := range statusCounts {
		statusMap[sc.Status] = sc.Count
	}
	stats["status_counts"] = statusMap

	// Get oldest pending job
	var oldestPending database.RenderQueue
	err = qm.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("scheduled_for ASC").
		First(&oldestPending).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to get oldest pending job: %w", err)
	}
	if err != gorm.ErrRecordNotFound {
		stats["oldest_pending"] = oldestPending.ScheduledFor
	}

	// Get recent failed jobs count (last 24 hours)
	var recentFailedCount int64
	err = qm.db.WithContext(ctx).Model(&database.RenderQueue{}).
		Where("status = ? AND updated_at > ?", "failed", time.Now().Add(-24*time.Hour)).
		Count(&recentFailedCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get recent failed count: %w", err)
	}
	stats["recent_failed_count"] = recentFailedCount

	return stats, nil
}

// RetryFailedJobs reschedules failed jobs that haven't exceeded max attempts
func (qm *QueueManager) RetryFailedJobs(ctx context.Context, maxAttempts int) error {
	var failedJobs []database.RenderQueue
	err := qm.db.WithContext(ctx).
		Where("status = ? AND attempts < ?", "failed", maxAttempts).
		Find(&failedJobs).Error
	if err != nil {
		return fmt.Errorf("failed to find failed jobs: %w", err)
	}

	if len(failedJobs) == 0 {
		return nil
	}

	logging.Logf("[QUEUE_MANAGER] Retrying %d failed jobs", len(failedJobs))

	// Schedule retry in 5 minutes
	retryTime := time.Now().Add(5 * time.Minute)
	
	for _, job := range failedJobs {
		err = qm.db.WithContext(ctx).Model(&job).Updates(database.RenderQueue{
			Status:       "pending",
			ScheduledFor: retryTime,
			ErrorMessage: "", // Clear error message
		}).Error
		if err != nil {
			logging.Logf("[QUEUE_MANAGER] Failed to retry job %s: %v", job.ID, err)
		}
	}

	return nil
}

// CleanupOldJobs removes old completed and failed jobs
func (qm *QueueManager) CleanupOldJobs(ctx context.Context, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	
	result := qm.db.WithContext(ctx).
		Where("status IN ? AND updated_at < ?", []string{"completed", "failed", "cancelled"}, cutoff).
		Delete(&database.RenderQueue{})
	
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old jobs: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		logging.Logf("[QUEUE_MANAGER] Cleaned up %d old jobs", result.RowsAffected)
	}

	return nil
}