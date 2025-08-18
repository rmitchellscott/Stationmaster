package handlers

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/pollers"
)

// GetFirmwareVersionsHandler returns all firmware versions
func GetFirmwareVersionsHandler(c *gin.Context) {
	db := database.GetDB()
	firmwareService := database.NewFirmwareService(db)

	versions, err := firmwareService.GetAllFirmwareVersions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get firmware versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"firmware_versions": versions})
}

// GetLatestFirmwareVersionHandler returns the latest firmware version
func GetLatestFirmwareVersionHandler(c *gin.Context) {
	db := database.GetDB()
	firmwareService := database.NewFirmwareService(db)

	version, err := firmwareService.GetLatestFirmwareVersion()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No firmware versions available"})
		return
	}

	c.JSON(http.StatusOK, version)
}

// DeleteFirmwareVersionHandler deletes a firmware version
func DeleteFirmwareVersionHandler(c *gin.Context) {
	versionIDStr := c.Param("id")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid firmware version ID"})
		return
	}

	db := database.GetDB()
	firmwareService := database.NewFirmwareService(db)

	// Get firmware version to check if it exists and get file path
	firmwareVersion, err := firmwareService.GetFirmwareVersionByID(versionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firmware version not found"})
		return
	}

	// Note: Allowing deletion of latest firmware version as requested by user

	// Delete the firmware file from disk if it exists
	if firmwareVersion.FilePath != "" {
		if err := os.Remove(firmwareVersion.FilePath); err != nil && !os.IsNotExist(err) {
			logging.Logf("[FIRMWARE DELETE] Failed to delete file %s: %v", firmwareVersion.FilePath, err)
			// Continue with database deletion even if file deletion fails
		}
	}

	// Delete from database
	if err := firmwareService.DeleteFirmwareVersion(versionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete firmware version from database"})
		return
	}

	logging.Logf("[FIRMWARE DELETE] Deleted firmware version %s", firmwareVersion.Version)
	c.JSON(http.StatusOK, gin.H{"message": "Firmware version deleted successfully"})
}

// GetDeviceModelsHandler returns all device models
func GetDeviceModelsHandler(c *gin.Context) {
	db := database.GetDB()
	firmwareService := database.NewFirmwareService(db)

	models, err := firmwareService.GetAllDeviceModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get device models"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"device_models": models})
}

// GetFirmwareStatsHandler returns firmware-related statistics
func GetFirmwareStatsHandler(c *gin.Context) {
	db := database.GetDB()
	firmwareService := database.NewFirmwareService(db)

	// Get firmware version count
	firmwareVersions, err := firmwareService.GetAllFirmwareVersions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get firmware statistics"})
		return
	}

	// Get device model count
	deviceModels, err := firmwareService.GetAllDeviceModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get device model statistics"})
		return
	}

	// Count devices by firmware update settings
	var devicesWithUpdatesEnabled, devicesWithUpdatesDisabled int64

	db.Model(&database.Device{}).Where("allow_firmware_updates = ?", true).Count(&devicesWithUpdatesEnabled)
	db.Model(&database.Device{}).Where("allow_firmware_updates = ?", false).Count(&devicesWithUpdatesDisabled)

	// Get device firmware version distribution
	type FirmwareDistribution struct {
		Version string `json:"version"`
		Count   int64  `json:"count"`
	}

	var distribution []FirmwareDistribution
	db.Model(&database.Device{}).
		Select("firmware_version as version, count(*) as count").
		Where("firmware_version != '' AND firmware_version IS NOT NULL").
		Group("firmware_version").
		Order("count DESC").
		Scan(&distribution)

	stats := gin.H{
		"firmware_versions": gin.H{
			"total":      len(firmwareVersions),
			"downloaded": countDownloadedVersions(firmwareVersions),
		},
		"device_models": gin.H{
			"total": len(deviceModels),
		},
		"update_settings": gin.H{
			"enabled":  devicesWithUpdatesEnabled,
			"disabled": devicesWithUpdatesDisabled,
		},
		"firmware_distribution": distribution,
	}

	c.JSON(http.StatusOK, stats)
}

// countDownloadedVersions counts how many firmware versions are downloaded
func countDownloadedVersions(versions []database.FirmwareVersion) int {
	count := 0
	for _, version := range versions {
		if version.IsDownloaded {
			count++
		}
	}
	return count
}

// TriggerFirmwarePollHandler triggers a manual firmware poll with immediate feedback
func TriggerFirmwarePollHandler(c *gin.Context) {
	db := database.GetDB()

	logging.Logf("[MANUAL FIRMWARE POLL] Starting manual firmware poll")

	// Create firmware poller for discovery
	firmwarePoller := pollers.NewFirmwarePoller(db)

	// Create a context with timeout for the initial discovery
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Do the firmware discovery synchronously to create database entries immediately
	if err := firmwarePoller.DiscoverFirmware(ctx); err != nil {
		logging.Logf("[MANUAL FIRMWARE POLL] Discovery failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to discover firmware versions"})
		return
	}

	// Start downloads in a background goroutine
	go func() {
		// Create a background context for downloads (not tied to HTTP request)
		downloadCtx, downloadCancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer downloadCancel()

		logging.Logf("[MANUAL FIRMWARE POLL] Starting background downloads")

		// Start downloads for any pending firmware
		if err := firmwarePoller.StartPendingDownloads(downloadCtx); err != nil {
			logging.Logf("[MANUAL FIRMWARE POLL] Download failed: %v", err)
			return
		}

		logging.Logf("[MANUAL FIRMWARE POLL] Background downloads completed")
	}()

	// Return immediately with firmware entries now available
	c.JSON(http.StatusOK, gin.H{"message": "Firmware discovery completed, downloads started"})
}

// GetFirmwareStatusHandler returns real-time firmware download status
func GetFirmwareStatusHandler(c *gin.Context) {
	db := database.GetDB()
	firmwareService := database.NewFirmwareService(db)

	versions, err := firmwareService.GetAllFirmwareVersions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get firmware status"})
		return
	}

	// Return only the status information needed for UI updates
	type StatusInfo struct {
		ID               string `json:"id"`
		Version          string `json:"version"`
		DownloadStatus   string `json:"download_status"`
		DownloadProgress int    `json:"download_progress"`
		DownloadError    string `json:"download_error,omitempty"`
		IsDownloaded     bool   `json:"is_downloaded"`
	}

	statusList := make([]StatusInfo, len(versions))
	for i, version := range versions {
		statusList[i] = StatusInfo{
			ID:               version.ID.String(),
			Version:          version.Version,
			DownloadStatus:   version.DownloadStatus,
			DownloadProgress: version.DownloadProgress,
			DownloadError:    version.DownloadError,
			IsDownloaded:     version.IsDownloaded,
		}
	}

	c.JSON(http.StatusOK, gin.H{"firmware_status": statusList})
}

// RetryFirmwareDownloadHandler retries downloading a specific firmware version
func RetryFirmwareDownloadHandler(c *gin.Context) {
	versionIDStr := c.Param("id")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid firmware version ID"})
		return
	}

	db := database.GetDB()
	firmwareService := database.NewFirmwareService(db)

	// Get the firmware version
	firmwareVersion, err := firmwareService.GetFirmwareVersionByID(versionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firmware version not found"})
		return
	}

	// Start the download in background
	go func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()

		logging.Logf("[RETRY DOWNLOAD] Starting download for firmware %s", firmwareVersion.Version)

		// Create a firmware poller to use its download functionality
		firmwarePoller := pollers.NewFirmwarePoller(db)

		// Reset status to downloading
		firmwareVersion.DownloadStatus = "pending"
		firmwareVersion.DownloadProgress = 0
		firmwareVersion.DownloadError = ""
		db.Save(firmwareVersion)

		// Execute the download
		if err := firmwarePoller.DownloadFirmware(ctx, firmwareVersion); err != nil {
			logging.Logf("[RETRY DOWNLOAD] Failed to download firmware %s: %v", firmwareVersion.Version, err)
		} else {
			logging.Logf("[RETRY DOWNLOAD] Successfully downloaded firmware %s", firmwareVersion.Version)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Firmware download started"})
}

// TriggerModelPollHandler triggers a manual model poll
func TriggerModelPollHandler(c *gin.Context) {
	db := database.GetDB()

	// Create a temporary model poller for manual polling
	modelPoller := pollers.NewModelPoller(db)

	// Create a context with timeout for the manual poll
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Execute the poll directly (not through the background service)
	if err := modelPoller.Start(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start model poll"})
		return
	}

	// Wait a moment for the poll to start, then stop it
	time.Sleep(100 * time.Millisecond)
	if err := modelPoller.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete model poll"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model poll completed successfully"})
}
