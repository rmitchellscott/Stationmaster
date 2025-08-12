package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
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

	device, err := deviceService.ClaimDevice(userUUID, req.FriendlyID, req.Name)
	if err != nil {
		if err.Error() == "device already claimed" {
			c.JSON(http.StatusConflict, gin.H{"error": "Device already claimed by another user"})
		} else if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found. Please check the friendly ID."})
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
		Name        string `json:"name"`
		RefreshRate int    `json:"refresh_rate"`
		IsActive    *bool  `json:"is_active"`
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

	err = deviceService.UpdateDevice(device)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device"})
		return
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

	err = deviceService.DeleteDevice(deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device deleted successfully"})
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