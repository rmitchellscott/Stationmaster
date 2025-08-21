package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/sse"
	"github.com/rmitchellscott/stationmaster/internal/trmnl"
)

// GetDevicesHandler returns all devices for the current user
func GetDevicesHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	devices, err := deviceService.GetDevicesByUserID(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// ClaimDeviceHandler claims an unclaimed device for the current user
func ClaimDeviceHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID

	var req struct {
		FriendlyID string `json:"friendly_id" binding:"required"`
		Name       string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	device, err := deviceService.ClaimDeviceByIdentifier(userUUID, req.FriendlyID, req.Name)
	if err != nil {
		if err.Error() == "device already claimed" {
			c.JSON(http.StatusConflict, gin.H{"error": "Device already claimed by another user"})
		} else if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found. Please check the device ID or MAC address."})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim device"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"device": device})
}

// GetDeviceHandler returns a specific device
func GetDeviceHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	device, err := deviceService.GetDeviceByID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Verify ownership
	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"device": device})
}

// UpdateDeviceHandler updates a device
func UpdateDeviceHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	var req struct {
		Name                    string  `json:"name"`
		RefreshRate             int     `json:"refresh_rate"`
		IsActive                *bool   `json:"is_active"`
		AllowFirmwareUpdates    *bool   `json:"allow_firmware_updates"`
		DeviceModelID           *uint   `json:"device_model_id"`
		ClearModelOverride      *bool   `json:"clear_model_override"`
		IsShareable             *bool   `json:"is_shareable"`
		SleepEnabled            *bool   `json:"sleep_enabled"`
		SleepStartTime          string  `json:"sleep_start_time"`
		SleepEndTime            string  `json:"sleep_end_time"`
		SleepShowScreen         *bool   `json:"sleep_show_screen"`
		FirmwareUpdateStartTime string  `json:"firmware_update_start_time"`
		FirmwareUpdateEndTime   string  `json:"firmware_update_end_time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	device, err := deviceService.GetDeviceByID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Verify ownership
	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update fields
	if req.Name != "" {
		device.Name = req.Name
	}
	if req.RefreshRate > 0 {
		device.RefreshRate = req.RefreshRate
	}
	if req.IsActive != nil {
		device.IsActive = *req.IsActive
	}
	if req.AllowFirmwareUpdates != nil {
		device.AllowFirmwareUpdates = *req.AllowFirmwareUpdates
	}
	if req.IsShareable != nil {
		// If device is being set to not shareable, clear any mirrored devices
		if !*req.IsShareable && device.IsShareable {
			playlistService := database.NewPlaylistService(db)
			err := playlistService.ClearMirroredPlaylistsForSourceDevice(device.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear mirrored devices"})
				return
			}
		}
		device.IsShareable = *req.IsShareable
	}

	// Handle model updates
	if req.ClearModelOverride != nil && *req.ClearModelOverride {
		// Clear manual override - reset to device-reported model or empty
		device.ManualModelOverride = false
		device.DeviceModelID = nil
		// TODO: In future, set DeviceModelID based on ReportedModelName
	} else if req.DeviceModelID != nil {
		// Validate the model exists if not zero
		if *req.DeviceModelID != 0 {
			if _, err := deviceService.ValidateDeviceModelByID(*req.DeviceModelID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			device.DeviceModelID = req.DeviceModelID
			device.ManualModelOverride = true
		} else {
			// Zero means clear the model
			device.DeviceModelID = nil
			device.ManualModelOverride = false
		}
	}

	// Handle sleep mode configuration
	logging.Logf("[DEVICE UPDATE] Sleep mode request: enabled=%v, start=%s, end=%s, show_screen=%v", 
		req.SleepEnabled, req.SleepStartTime, req.SleepEndTime, req.SleepShowScreen)
	
	if req.SleepEnabled != nil {
		device.SleepEnabled = *req.SleepEnabled
		logging.Logf("[DEVICE UPDATE] Set sleep enabled to: %v", device.SleepEnabled)
	}
	// Always update sleep times if provided (frontend always sends these values)
	if req.SleepStartTime != "" {
		// Validate time format (HH:MM)
		if err := validateTimeFormat(req.SleepStartTime); err != nil {
			logging.Logf("[DEVICE UPDATE] Invalid sleep start time format: %s, error: %v", req.SleepStartTime, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sleep start time format. Use HH:MM"})
			return
		}
		device.SleepStartTime = req.SleepStartTime
		logging.Logf("[DEVICE UPDATE] Set sleep start time to: %s", device.SleepStartTime)
	} else {
		// If empty string is sent, clear the field
		device.SleepStartTime = ""
		logging.Logf("[DEVICE UPDATE] Cleared sleep start time")
	}
	
	if req.SleepEndTime != "" {
		// Validate time format (HH:MM)
		if err := validateTimeFormat(req.SleepEndTime); err != nil {
			logging.Logf("[DEVICE UPDATE] Invalid sleep end time format: %s, error: %v", req.SleepEndTime, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sleep end time format. Use HH:MM"})
			return
		}
		device.SleepEndTime = req.SleepEndTime
		logging.Logf("[DEVICE UPDATE] Set sleep end time to: %s", device.SleepEndTime)
	} else {
		// If empty string is sent, clear the field
		device.SleepEndTime = ""
		logging.Logf("[DEVICE UPDATE] Cleared sleep end time")
	}
	if req.SleepShowScreen != nil {
		device.SleepShowScreen = *req.SleepShowScreen
		logging.Logf("[DEVICE UPDATE] Set sleep show screen to: %v", device.SleepShowScreen)
	}

	// Handle firmware update schedule configuration
	logging.Logf("[DEVICE UPDATE] Firmware schedule request: start=%s, end=%s", 
		req.FirmwareUpdateStartTime, req.FirmwareUpdateEndTime)
	
	// Always update firmware update times if provided (frontend always sends these values)
	if req.FirmwareUpdateStartTime != "" {
		// Validate time format (HH:MM)
		if err := validateTimeFormat(req.FirmwareUpdateStartTime); err != nil {
			logging.Logf("[DEVICE UPDATE] Invalid firmware start time format: %s, error: %v", req.FirmwareUpdateStartTime, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid firmware start time format. Use HH:MM"})
			return
		}
		device.FirmwareUpdateStartTime = req.FirmwareUpdateStartTime
		logging.Logf("[DEVICE UPDATE] Set firmware start time to: %s", device.FirmwareUpdateStartTime)
	} else {
		// If empty string is sent, set to default
		device.FirmwareUpdateStartTime = "00:00"
		logging.Logf("[DEVICE UPDATE] Set firmware start time to default: 00:00")
	}
	
	if req.FirmwareUpdateEndTime != "" {
		// Validate time format (HH:MM)
		if err := validateTimeFormat(req.FirmwareUpdateEndTime); err != nil {
			logging.Logf("[DEVICE UPDATE] Invalid firmware end time format: %s, error: %v", req.FirmwareUpdateEndTime, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid firmware end time format. Use HH:MM"})
			return
		}
		device.FirmwareUpdateEndTime = req.FirmwareUpdateEndTime
		logging.Logf("[DEVICE UPDATE] Set firmware end time to: %s", device.FirmwareUpdateEndTime)
	} else {
		// If empty string is sent, set to default
		device.FirmwareUpdateEndTime = "23:59"
		logging.Logf("[DEVICE UPDATE] Set firmware end time to default: 23:59")
	}

	err = deviceService.UpdateDevice(device)
	if err != nil {
		logging.Logf("[DEVICE UPDATE] Failed to update device %s: %v", device.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device"})
		return
	}

	// Broadcast device settings update via SSE if sleep settings changed
	if req.SleepEnabled != nil || req.SleepStartTime != "" || req.SleepEndTime != "" || req.SleepShowScreen != nil {
		// Get user timezone for sleep calculation
		userTimezone := "UTC"
		if user.Timezone != "" {
			userTimezone = user.Timezone
		}
		
		// Check if device is currently in sleep period
		currentlySleeping := trmnl.IsInSleepPeriod(device, userTimezone)
		
		// Send updated sleep config via SSE
		sseService := sse.GetSSEService()
		sseService.BroadcastToDevice(device.ID, sse.Event{
			Type: "device_settings_updated",
			Data: map[string]interface{}{
				"device_id": device.ID.String(),
				"sleep_config": map[string]interface{}{
					"enabled":            device.SleepEnabled,
					"start_time":         device.SleepStartTime,
					"end_time":           device.SleepEndTime,
					"show_screen":        device.SleepShowScreen,
					"currently_sleeping": currentlySleeping,
				},
				"timestamp": time.Now().UTC(),
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"device": device})
}

// DeleteDeviceHandler deletes a device
func DeleteDeviceHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	device, err := deviceService.GetDeviceByID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Verify ownership
	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	err = deviceService.UnclaimDevice(deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlink device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device unlinked successfully"})
}

// Admin device handlers

// GetAllDevicesHandler returns all devices (admin only)
func GetAllDevicesHandler(c *gin.Context) {
	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	devices, err := deviceService.GetAllDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// UnlinkDeviceHandler unlinks a device from its user account (admin only)
func UnlinkDeviceHandler(c *gin.Context) {
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	err = deviceService.UnlinkDevice(deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlink device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device unlinked successfully"})
}

// GetDeviceStatsHandler returns device statistics (admin only)
func GetDeviceStatsHandler(c *gin.Context) {
	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	stats, err := deviceService.GetDeviceStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get device statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetDeviceLogsHandler returns logs for a specific device
func GetDeviceLogsHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	device, err := deviceService.GetDeviceByID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Verify ownership
	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Parse query parameters for pagination
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get logs for the device
	logs, err := deviceService.GetDeviceLogsByDeviceID(deviceID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch device logs"})
		return
	}

	// Get total count for pagination
	totalCount, err := deviceService.GetDeviceLogsCount(deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get logs count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":        logs,
		"total_count": totalCount,
		"limit":       limit,
		"offset":      offset,
	})
}

// DeviceEventsHandler handles SSE connections for device events
func DeviceEventsHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	device, err := deviceService.GetDeviceByID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Verify ownership
	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	sseService := sse.GetSSEService()
	client := sseService.AddClient(deviceID, userUUID, c.Writer)
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to establish SSE connection"})
		return
	}

	// Send initial device status
	playlistService := database.NewPlaylistService(db)
	activeItems, err := playlistService.GetActivePlaylistItemsForTime(deviceID, time.Now())
	if err == nil && len(activeItems) > 0 {
		currentIndex := device.LastPlaylistIndex
		if currentIndex >= 0 && currentIndex < len(activeItems) {
			// Get user timezone for sleep calculation
			userTimezone := "UTC"
			if user.Timezone != "" {
				userTimezone = user.Timezone
			}
			
			// Check if device is currently in sleep period
			currentlySleeping := trmnl.IsInSleepPeriod(device, userTimezone)
			
			sseService.BroadcastToDevice(deviceID, sse.Event{
				Type: "playlist_index_changed",
				Data: map[string]interface{}{
					"device_id":     deviceID.String(),
					"current_index": currentIndex,
					"current_item":  activeItems[currentIndex],
					"active_items":  activeItems,
					"timestamp":     time.Now().UTC(),
					"sleep_config": map[string]interface{}{
						"enabled":            device.SleepEnabled,
						"start_time":         device.SleepStartTime,
						"end_time":           device.SleepEndTime,
						"show_screen":        device.SleepShowScreen,
						"currently_sleeping": currentlySleeping,
					},
				},
			})
		}
	}

	// Keep connection alive until client disconnects
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	select {
	case <-client.Done:
	case <-ctx.Done():
	}

	sseService.RemoveClient(client.ID)
}

// DeviceActiveItemsHandler returns schedule-filtered active items for a device at a specific time
func DeviceActiveItemsHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	device, err := deviceService.GetDeviceByID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Verify ownership
	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Parse query time parameter (optional)
	queryTimeStr := c.Query("at")
	var queryTime time.Time
	if queryTimeStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, queryTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time format. Use RFC3339 format (e.g., 2024-01-15T10:30:00Z)"})
			return
		}
		queryTime = parsedTime
	} else {
		queryTime = time.Now()
	}

	// Get active playlist items for the specified time
	playlistService := database.NewPlaylistService(db)
	activeItems, err := playlistService.GetActivePlaylistItemsForTime(deviceID, queryTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active playlist items"})
		return
	}

	// Get total visible items for comparison
	playlist, err := playlistService.GetDefaultPlaylistForDevice(deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get default playlist"})
		return
	}

	allItems, err := playlistService.GetPlaylistItems(playlist.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get playlist items"})
		return
	}

	visibleItems := make([]database.PlaylistItem, 0)
	for _, item := range allItems {
		if item.IsVisible {
			visibleItems = append(visibleItems, item)
		}
	}

	// Determine current index and currently showing item
	var currentIndex int = -1
	var currentlyShowing *database.PlaylistItem = nil

	if len(activeItems) > 0 {
		// Use the device's last playlist index to determine currently showing
		if device.LastPlaylistIndex >= 0 && device.LastPlaylistIndex < len(activeItems) {
			currentIndex = device.LastPlaylistIndex
			currentlyShowing = &activeItems[currentIndex]
		}
	}

	// Response
	response := gin.H{
		"device_id":           deviceID.String(),
		"query_time":          queryTime.UTC(),
		"current_index":       currentIndex,
		"currently_showing":   currentlyShowing,
		"active_items":        activeItems,
		"total_visible_items": len(visibleItems),
		"total_active_items":  len(activeItems),
	}

	c.JSON(http.StatusOK, response)
}

// GetDeviceModelOptionsHandler returns all available device models for user selection
func GetDeviceModelOptionsHandler(c *gin.Context) {
	_, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	models, err := deviceService.GetAllDeviceModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch device models"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"models": models})
}

// MirrorDeviceHandler mirrors another device's playlist to the current device
func MirrorDeviceHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	var req struct {
		SourceFriendlyID string `json:"source_friendly_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Get and verify the target device (the one being set up to mirror)
	device, err := deviceService.GetDeviceByID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Verify ownership of target device
	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Find the source device by friendly ID
	sourceDevice, err := deviceService.GetDeviceByFriendlyID(req.SourceFriendlyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source device not found"})
		return
	}

	// Verify source device is shareable
	if !sourceDevice.IsShareable {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source device is not shareable"})
		return
	}

	// Verify source device is claimed
	if !sourceDevice.IsClaimed || sourceDevice.UserID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source device is not claimed"})
		return
	}

	// Copy playlists from source to target device
	playlistService := database.NewPlaylistService(db)
	err = playlistService.CopyPlaylistItems(sourceDevice.ID, device.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to copy playlist items"})
		return
	}

	// Update the target device to record the mirror relationship
	now := time.Now()
	device.MirrorSourceID = &sourceDevice.ID
	device.MirrorSyncedAt = &now

	err = deviceService.UpdateDevice(device)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Device successfully mirrored",
		"source_device": sourceDevice.Name,
		"synced_at":     now,
	})
}

// SyncMirrorHandler re-syncs the playlist from the source device
func SyncMirrorHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Get and verify the device
	device, err := deviceService.GetDeviceByID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Verify ownership
	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Verify device is currently mirroring
	if device.MirrorSourceID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device is not mirroring another device"})
		return
	}

	// Get the source device
	sourceDevice, err := deviceService.GetDeviceByID(*device.MirrorSourceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source device not found"})
		return
	}

	// Verify source device is still shareable
	if !sourceDevice.IsShareable {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source device is no longer shareable"})
		return
	}

	// Re-copy playlists from source to target device
	playlistService := database.NewPlaylistService(db)
	err = playlistService.CopyPlaylistItems(sourceDevice.ID, device.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync playlist items"})
		return
	}

	// Update the sync timestamp
	now := time.Now()
	device.MirrorSyncedAt = &now

	err = deviceService.UpdateDevice(device)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Device successfully synced",
		"source_device": sourceDevice.Name,
		"synced_at":     now,
	})
}

// UnmirrorDeviceHandler stops mirroring and makes the device independent
func UnmirrorDeviceHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Get and verify the device
	device, err := deviceService.GetDeviceByID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Verify ownership
	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Verify device is currently mirroring
	if device.MirrorSourceID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device is not mirroring another device"})
		return
	}

	// Clear mirrored playlist items
	playlistService := database.NewPlaylistService(db)
	err = playlistService.ClearMirroredPlaylists(device.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear mirrored playlists"})
		return
	}

	// Clear the mirror relationship
	device.MirrorSourceID = nil
	device.MirrorSyncedAt = nil

	err = deviceService.UpdateDevice(device)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Device successfully unmirrored",
	})
}

// validateTimeFormat validates that a time string is in HH:MM format
func validateTimeFormat(timeStr string) error {
	_, err := time.Parse("15:04", timeStr)
	return err
}
