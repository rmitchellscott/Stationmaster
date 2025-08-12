package trmnl

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/database"
)

// SetupHandler handles device setup requests from TRMNL devices
// POST /api/setup with header 'ID': 'MAC_ADDRESS'
func SetupHandler(c *gin.Context) {
	macAddress := c.GetHeader("ID")
	if macAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device ID header"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Check if device already exists
	device, err := deviceService.GetDeviceByDeviceID(macAddress)
	if err == nil {
		// Device already exists, return existing API key and friendly ID
		c.JSON(http.StatusOK, gin.H{
			"api_key":     device.APIKey,
			"friendly_id": device.DeviceID,
			"image_url":   getImageURLForDevice(device),
			"filename":    "display.png",
		})
		return
	}

	// Device doesn't exist, need to register it
	// For now, we'll require manual device linking through the web interface
	// This prevents unauthorized devices from being automatically added
	c.JSON(http.StatusNotFound, gin.H{
		"error": "Device not found. Please link your device through the web interface first.",
	})
}

// DisplayHandler handles display requests from TRMNL devices
// GET /api/display with headers for device authentication and status
func DisplayHandler(c *gin.Context) {
	deviceID := c.GetHeader("ID")
	accessToken := c.GetHeader("Access-Token")
	refreshRateStr := c.GetHeader("Refresh-Rate")
	batteryVoltageStr := c.GetHeader("Battery-Voltage")
	firmwareVersion := c.GetHeader("FW-Version")
	rssiStr := c.GetHeader("RSSI")

	if deviceID == "" || accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing device ID or access token"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Get device by API key
	device, err := deviceService.GetDeviceByAPIKey(accessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token"})
		return
	}

	// Verify device ID matches
	if device.DeviceID != deviceID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Device ID mismatch"})
		return
	}

	// Parse and update device status
	var batteryVoltage float64
	var rssi int
	var refreshRate int

	if batteryVoltageStr != "" {
		if bv, err := strconv.ParseFloat(batteryVoltageStr, 64); err == nil {
			batteryVoltage = bv
		}
	}

	if rssiStr != "" {
		if r, err := strconv.Atoi(rssiStr); err == nil {
			rssi = r
		}
	}

	if refreshRateStr != "" {
		if rr, err := strconv.Atoi(refreshRateStr); err == nil {
			refreshRate = rr
		}
	}

	// Update device status
	err = deviceService.UpdateDeviceStatus(device.DeviceID, firmwareVersion, batteryVoltage, rssi)
	if err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	// Get current playlist items for this device
	playlistService := database.NewPlaylistService(db)
	_, err = playlistService.GetActivePlaylistItemsForTime(device.ID, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get playlist items"})
		return
	}

	// For now, return a basic response
	// In a full implementation, this would generate or serve the appropriate image
	response := gin.H{
		"image_url": getImageURLForDevice(device),
		"filename":  "display.png",
	}

	// Check if we need to update refresh rate
	if refreshRate > 0 && refreshRate != device.RefreshRate {
		device.RefreshRate = refreshRate
		deviceService.UpdateDevice(device)
	}

	// Check for firmware updates (placeholder)
	// response["update_firmware"] = false
	// response["firmware_url"] = ""
	// response["reset_firmware"] = false

	c.JSON(http.StatusOK, response)
}

// LogsHandler handles log submissions from TRMNL devices
// POST /api/logs
func LogsHandler(c *gin.Context) {
	deviceID := c.GetHeader("ID")
	accessToken := c.GetHeader("Access-Token")

	if deviceID == "" || accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing device ID or access token"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Verify device
	device, err := deviceService.GetDeviceByAPIKey(accessToken)
	if err != nil || device.DeviceID != deviceID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid device credentials"})
		return
	}

	// Parse log data
	var logData map[string]interface{}
	if err := c.ShouldBindJSON(&logData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log data"})
		return
	}

	// Store logs (placeholder - in a full implementation you'd store these properly)
	// TODO: Implement log storage and processing

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// getImageURLForDevice generates an image URL for the device based on its active playlist
func getImageURLForDevice(device *database.Device) string {
	// Placeholder implementation
	// In a full implementation, this would:
	// 1. Get the currently active playlist item based on schedule
	// 2. Generate content based on the plugin type and settings
	// 3. Return a URL to the generated image
	return "/api/trmnl/devices/" + device.ID.String() + "/image"
}

// DeviceImageHandler serves generated images for devices
func DeviceImageHandler(c *gin.Context) {
	deviceIDStr := c.Param("deviceId")
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

	// Get current playlist items
	playlistService := database.NewPlaylistService(db)
	_, err = playlistService.GetActivePlaylistItemsForTime(device.ID, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get playlist items"})
		return
	}

	// Generate image based on active plugins
	// For now, return a placeholder
	// TODO: Implement actual image generation
	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "no-cache")
	
	// Return a simple placeholder image (1x1 transparent PNG)
	placeholder := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	
	c.Data(http.StatusOK, "image/png", placeholder)
}