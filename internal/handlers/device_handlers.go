package handlers

import (
	"context"
	"fmt"
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

// ImportDeviceHandler imports a device with existing credentials
func ImportDeviceHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID

	var req struct {
		MacAddress    string `json:"mac_address" binding:"required"`
		APIKey        string `json:"api_key" binding:"required"`
		FriendlyID    string `json:"friendly_id" binding:"required"`
		Name          string `json:"name"`
		DeviceModelID *uint  `json:"device_model_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	var modelName string
	if req.DeviceModelID != nil && *req.DeviceModelID != 0 {
		deviceModel, err := deviceService.ValidateDeviceModelByID(*req.DeviceModelID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		modelName = deviceModel.ModelName
	}

	device, err := deviceService.ImportDevice(userUUID, req.MacAddress, req.APIKey, req.FriendlyID, req.Name, modelName)
	if err != nil {
		if err.Error() == "device with MAC address "+req.MacAddress+" already exists" ||
			err.Error() == "device with friendly ID "+req.FriendlyID+" already exists" ||
			err.Error() == "device with this API key already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import device"})
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

// To add a new device setting: add the field to Device in models.go, then add a line here.
var deviceSettingsFields = map[string]string{
	"name":                       "name",
	"refresh_rate":               "refresh_rate",
	"is_active":                  "is_active",
	"allow_firmware_updates":     "allow_firmware_updates",
	"is_shareable":               "is_shareable",
	"sleep_enabled":              "sleep_enabled",
	"sleep_start_time":           "sleep_start_time",
	"sleep_end_time":             "sleep_end_time",
	"sleep_show_screen":          "sleep_show_screen",
	"firmware_update_start_time": "firmware_update_start_time",
	"firmware_update_end_time":   "firmware_update_end_time",
	"target_firmware_version":    "target_firmware_version",
	"maximum_compatibility":      "maximum_compatibility",
	"touchbar_mode":              "touchbar_mode",
	"temperature_profile":        "temperature_profile",
}

var timeFields = map[string]string{
	"sleep_start_time":          "",
	"sleep_end_time":            "",
	"firmware_update_start_time": "00:00",
	"firmware_update_end_time":   "23:59",
}

func buildDeviceUpdates(raw map[string]interface{}) (map[string]interface{}, error) {
	updates := map[string]interface{}{}

	for jsonKey, dbCol := range deviceSettingsFields {
		val, exists := raw[jsonKey]
		if !exists {
			if defaultVal, isTime := timeFields[jsonKey]; isTime {
				updates[dbCol] = defaultVal
			}
			continue
		}

		if _, isTime := timeFields[jsonKey]; isTime {
			if s, ok := val.(string); ok && s != "" {
				if err := validateTimeFormat(s); err != nil {
					return nil, fmt.Errorf("invalid %s format: %w", jsonKey, err)
				}
				updates[dbCol] = s
			} else {
				updates[dbCol] = timeFields[jsonKey]
			}
			continue
		}

		updates[dbCol] = val
	}

	return updates, nil
}

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

	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
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

	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if val, ok := raw["is_shareable"]; ok {
		if shareable, ok := val.(bool); ok && !shareable && device.IsShareable {
			playlistService := database.NewPlaylistService(db)
			if err := playlistService.ClearMirroredPlaylistsForSourceDevice(device.ID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear mirrored devices"})
				return
			}
		}
	}

	if val, ok := raw["clear_model_override"]; ok {
		if clear, ok := val.(bool); ok && clear {
			device.ManualModelOverride = false
			device.DeviceModelID = nil
			deviceService.UpdateDevice(device)
		}
		delete(raw, "clear_model_override")
	} else if val, ok := raw["device_model_id"]; ok {
		if modelIDFloat, ok := val.(float64); ok {
			modelID := uint(modelIDFloat)
			if modelID != 0 {
				if _, err := deviceService.ValidateDeviceModelByID(modelID); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				device.DeviceModelID = &modelID
				device.ManualModelOverride = true
			} else {
				device.DeviceModelID = nil
				device.ManualModelOverride = false
			}
			deviceService.UpdateDevice(device)
		}
		delete(raw, "device_model_id")
	}

	updates, err := buildDeviceUpdates(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(updates) > 0 {
		if err := deviceService.UpdateDeviceFields(deviceID, updates); err != nil {
			logging.Error("[DEVICE UPDATE] Failed to update device", "device_id", device.ID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device"})
			return
		}
	}

	device, _ = deviceService.GetDeviceByID(deviceID)

	// Broadcast device settings update via SSE if sleep settings changed
	if _, hasSleep := raw["sleep_enabled"]; hasSleep {
		userTimezone := "UTC"
		if user.Timezone != "" {
			userTimezone = user.Timezone
		}
		currentlySleeping := trmnl.IsInSleepPeriod(device, userTimezone)
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

func UnclaimDeviceHandler(c *gin.Context) {
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

func AdminDeleteDeviceHandler(c *gin.Context) {
	deviceIDStr := c.Param("id")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	err = deviceService.AdminDeleteDevice(deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device deleted successfully"})
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

	// Send initial device status - just use what database says is current
	if device.LastPlaylistItemID != nil {
		playlistService := database.NewPlaylistService(db)
		
		// Get the current item directly by UUID (single source of truth)
		currentItem, err := playlistService.GetPlaylistItemByID(*device.LastPlaylistItemID)
		if err == nil && currentItem != nil {
			// Get active items for context and index calculation
			activeItems, _ := playlistService.GetActivePlaylistItemsForTime(deviceID, time.Now().UTC())
			
			// Calculate index in active items for frontend compatibility
			currentIndex := -1
			for i, item := range activeItems {
				if item.ID == currentItem.ID {
					currentIndex = i
					break
				}
			}
			
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
					"current_index": currentIndex, // -1 if not in active items (e.g., hidden)
					"current_item":  *currentItem,  // Always the actual current item
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
		queryTime = time.Now().UTC()
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

	if len(activeItems) > 0 && device.LastPlaylistItemID != nil {
		// Find the current item by UUID
		for i, item := range activeItems {
			if item.ID == *device.LastPlaylistItemID {
				currentIndex = i
				currentlyShowing = &activeItems[i]
				break
			}
		}
	}

	// Get user timezone for sleep calculations
	userTimezone := "UTC"
	if user.Timezone != "" {
		userTimezone = user.Timezone
	}

	// Check current sleep state and if sleep screen would be served
	currentlySleeping := trmnl.IsInSleepPeriod(device, userTimezone)
	sleepScreenServed := currentlySleeping && device.SleepShowScreen

	// Response
	response := gin.H{
		"device_id":           deviceID.String(),
		"query_time":          queryTime.UTC(),
		"current_index":       currentIndex,
		"currently_showing":   currentlyShowing,
		"active_items":        activeItems,
		"total_visible_items": len(visibleItems),
		"total_active_items":  len(activeItems),
		"sleep_config": map[string]interface{}{
			"enabled":               device.SleepEnabled,
			"start_time":            device.SleepStartTime,
			"end_time":              device.SleepEndTime,
			"show_screen":           device.SleepShowScreen,
			"currently_sleeping":    currentlySleeping,
			"sleep_screen_served":   sleepScreenServed,
		},
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
	now := time.Now().UTC()
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
	now := time.Now().UTC()
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
