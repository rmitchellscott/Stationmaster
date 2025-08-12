package trmnl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// SetupHandler handles device setup requests from TRMNL devices
// GET /api/setup with header 'ID': 'MAC_ADDRESS'
func SetupHandler(c *gin.Context) {
	debugMode := os.Getenv("DEBUG") != ""
	
	if debugMode {
		logging.Logf("[/api/setup] Request from %s %s %s", c.ClientIP(), c.Request.Method, c.Request.URL.Path)
	}
	
	macAddress := c.GetHeader("ID")
	if macAddress == "" {
		if debugMode {
			logging.Logf("[/api/setup] Error: Missing device ID header")
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device ID header"})
		return
	}
	
	if debugMode {
		logging.Logf("[/api/setup] Device MAC address: %s", macAddress)
		
		// Log all headers for debugging
		logging.Logf("[/api/setup] All request headers:")
		for name, values := range c.Request.Header {
			for _, value := range values {
				logging.Logf("[/api/setup] Header %s: %s", name, value)
			}
		}
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Check if device already exists
	device, err := deviceService.GetDeviceByMacAddress(macAddress)
	if err == nil {
		// Device already exists, return existing API key and friendly ID
		response := gin.H{
			"status":      200,
			"api_key":     device.APIKey,
			"friendly_id": device.FriendlyID,
			"image_url":   "https://usetrmnl.com/images/setup/setup-logo.bmp",
			"filename":    "empty_state",
		}
		
		if debugMode {
			responseBytes, _ := json.Marshal(response)
			logging.Logf("[/api/setup] Response for existing device %s: %s", macAddress, string(responseBytes))
		}
		
		c.JSON(http.StatusOK, response)
		return
	}

	// Device doesn't exist, auto-register it as unclaimed
	device, err = deviceService.CreateUnclaimedDevice(macAddress)
	if err != nil {
		if debugMode {
			logging.Logf("[/api/setup] Error creating device for MAC %s: %v", macAddress, err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device"})
		return
	}

	if debugMode {
		logging.Logf("[/api/setup] Created new device for MAC %s with friendly_id %s", macAddress, device.FriendlyID)
	}

	// Return the new device information
	response := gin.H{
		"status":      200,
		"api_key":     device.APIKey,
		"friendly_id": device.FriendlyID,
		"image_url":   "https://usetrmnl.com/images/setup/setup-logo.bmp",
		"filename":    "empty_state",
	}
	
	if debugMode {
		responseBytes, _ := json.Marshal(response)
		logging.Logf("[/api/setup] Response for new device %s: %s", macAddress, string(responseBytes))
	}
	
	c.JSON(http.StatusOK, response)
}

// DisplayHandler handles display requests from TRMNL devices
// GET /api/display with headers for device authentication and status
func DisplayHandler(c *gin.Context) {
	startTime := time.Now()
	debugMode := os.Getenv("DEBUG") != ""
	
	if debugMode {
		// Log request details
		logging.Logf("[/api/display] Request from %s %s %s", c.ClientIP(), c.Request.Method, c.Request.URL.Path)
	}
	
	// Extract headers
	deviceID := c.GetHeader("ID")
	accessToken := c.GetHeader("Access-Token")
	refreshRateStr := c.GetHeader("Refresh-Rate")
	batteryVoltageStr := c.GetHeader("Battery-Voltage")
	firmwareVersion := c.GetHeader("Fw-Version")  // Device sends "Fw-Version" not "FW-Version"
	rssiStr := c.GetHeader("Rssi")                // Device sends "Rssi" not "RSSI"
	
	if debugMode {
		// Log all headers sent by device
		logging.Logf("[/api/display] Device headers - ID: %s, Access-Token: %s, Refresh-Rate: %s, Battery-Voltage: %s, Fw-Version: %s, Rssi: %s", 
			deviceID, accessToken, refreshRateStr, batteryVoltageStr, firmwareVersion, rssiStr)
		
		// Log User-Agent if present
		if userAgent := c.GetHeader("User-Agent"); userAgent != "" {
			logging.Logf("[/api/display] User-Agent: %s", userAgent)
		}
		
		// Log all other headers for debugging
		logging.Logf("[/api/display] All request headers:")
		for name, values := range c.Request.Header {
			for _, value := range values {
				logging.Logf("[/api/display] Header %s: %s", name, value)
			}
		}
	}

	if deviceID == "" || accessToken == "" {
		if debugMode {
			if deviceID == "" {
				logging.Logf("[/api/display] Authentication failed: Missing device ID")
			}
			if accessToken == "" {
				logging.Logf("[/api/display] Authentication failed: Missing or empty access token - device may not have stored API key properly")
			}
			logging.Logf("[/api/display] Rejecting request with 401 Unauthorized")
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing device ID or access token"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Get device by API key
	device, err := deviceService.GetDeviceByAPIKey(accessToken)
	if err != nil {
		if debugMode {
			logging.Logf("[/api/display] Authentication failed: Invalid access token '%s' for device ID '%s' - %v", accessToken, deviceID, err)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token"})
		return
	}

	// Verify device ID matches (deviceID header should contain the MAC address)
	if device.MacAddress != deviceID {
		if debugMode {
			logging.Logf("[/api/display] Authentication failed: Device ID mismatch - expected '%s', got '%s'", device.MacAddress, deviceID)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Device ID mismatch"})
		return
	}
	
	if debugMode {
		logging.Logf("[/api/display] Authentication successful for device %s (friendly_id: %s)", device.MacAddress, device.FriendlyID)
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

	// Update device status in database
	err = deviceService.UpdateDeviceStatus(device.MacAddress, firmwareVersion, batteryVoltage, rssi)
	if err != nil {
		// Log error but don't fail the request
	}

	// Get current playlist items for this device
	playlistService := database.NewPlaylistService(db)
	activeItems, err := playlistService.GetActivePlaylistItemsForTime(device.ID, time.Now())
	if err != nil {
		if debugMode {
			logging.Logf("[/api/display] No playlist items found for device %s (this is normal for unclaimed devices): %v", device.MacAddress, err)
		}
		// For unclaimed devices or devices without playlists, use empty activeItems slice
		activeItems = []database.PlaylistItem{}
	}

	// Check if we need to update refresh rate using GORM-safe method
	if refreshRate > 0 && refreshRate != device.RefreshRate {
		err = deviceService.UpdateRefreshRate(device.ID, refreshRate)
		if err != nil {
			// Log error but don't fail the request
		}
	}

	// Determine device status
	status := 0
	if !device.IsClaimed {
		status = 202
	}

	// Process active plugins and generate response
	response, err := processActivePlugins(device, activeItems)
	if err != nil {
		// Fall back to default response if plugin processing fails
		if debugMode {
			logging.Logf("[/api/display] No active plugins for device %s, using default response (status: %d)", device.MacAddress, status)
		}
		
		// For unclaimed devices (status 202), provide setup image
		imageURL := getImageURLForDevice(device)
		filename := time.Now().Format("2006-01-02T15:04:05")
		
		if status == 202 {
			imageURL = "https://usetrmnl.com/images/setup/setup-logo.bmp"
			filename = "empty_state"
		}
		
		response = gin.H{
			"status":          status,
			"image_url":       imageURL,
			"filename":        filename,
			"refresh_rate":    fmt.Sprintf("%d", device.RefreshRate),
			"update_firmware": false,
			"firmware_url":    nil,
			"reset_firmware":  false,
		}
	} else {
		// Ensure required fields are set
		response["status"] = status
		response["update_firmware"] = false
		response["firmware_url"] = nil
		response["reset_firmware"] = false
		
		// Use device refresh rate if not overridden by plugin
		if _, exists := response["refresh_rate"]; !exists {
			response["refresh_rate"] = fmt.Sprintf("%d", device.RefreshRate)
		}
	}

	if debugMode {
		// Log response being sent back to device
		responseBytes, _ := json.Marshal(response)
		logging.Logf("[/api/display] Response to device %s: %s", deviceID, string(responseBytes))
		
		// Log request processing time
		duration := time.Since(startTime)
		logging.Logf("[/api/display] Request processing time: %v", duration)
	}

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
	if err != nil || device.MacAddress != deviceID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid device credentials"})
		return
	}

	// Parse log data
	var logData map[string]interface{}
	if err := c.ShouldBindJSON(&logData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log data"})
		return
	}

	// Extract log level if provided
	level := "info"
	if levelStr, ok := logData["level"].(string); ok && levelStr != "" {
		level = levelStr
	}

	// Convert log data back to JSON string for storage
	logDataBytes, err := json.Marshal(logData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process log data"})
		return
	}

	// Store the log entry
	_, err = deviceService.CreateDeviceLog(device.ID, string(logDataBytes), level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store log data"})
		return
	}

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

// processActivePlugins processes the active playlist items and generates appropriate response
func processActivePlugins(device *database.Device, activeItems []database.PlaylistItem) (gin.H, error) {
	if len(activeItems) == 0 {
		return nil, fmt.Errorf("no active playlist items")
	}

	// Get the first active item (in a full implementation, you might cycle through items)
	item := activeItems[0]
	
	// Get the user plugin details
	db := database.GetDB()
	pluginService := database.NewPluginService(db)
	
	userPlugin, err := pluginService.GetUserPluginByID(item.UserPluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user plugin: %w", err)
	}

	// Handle different plugin types
	switch userPlugin.Plugin.Type {
	case "redirect":
		return processRedirectPlugin(userPlugin)
	default:
		// For other plugin types, return default response
		return gin.H{
			"image_url": getImageURLForDevice(device),
			"filename":  "display.png",
		}, nil
	}
}

// processRedirectPlugin handles redirect plugin type by fetching JSON from external endpoint
func processRedirectPlugin(userPlugin *database.UserPlugin) (gin.H, error) {
	// Parse plugin settings
	var settings map[string]interface{}
	if userPlugin.Settings != "" {
		if err := json.Unmarshal([]byte(userPlugin.Settings), &settings); err != nil {
			return nil, fmt.Errorf("failed to parse plugin settings: %w", err)
		}
	}

	// Get endpoint URL from settings
	endpointURL, ok := settings["endpoint_url"].(string)
	if !ok || endpointURL == "" {
		return nil, fmt.Errorf("endpoint_url not configured in plugin settings")
	}

	// Get timeout (default to 2 seconds)
	timeoutSeconds := 2
	if timeout, ok := settings["timeout_seconds"].(float64); ok && timeout > 0 {
		timeoutSeconds = int(timeout)
		if timeoutSeconds > 10 {
			timeoutSeconds = 10 // Cap at 10 seconds
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	// Fetch JSON from endpoint
	resp, err := client.Get(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from endpoint: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("endpoint returned status %d", resp.StatusCode)
	}

	// Parse JSON response
	var pluginResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&pluginResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Extract required fields and build response
	response := gin.H{}
	
	// Copy filename if provided
	if filename, ok := pluginResponse["filename"]; ok {
		response["filename"] = filename
	} else {
		response["filename"] = "display.png" // Default filename
	}
	
	// Copy url as image_url if provided
	if url, ok := pluginResponse["url"]; ok {
		response["image_url"] = url
	} else if imageURL, ok := pluginResponse["image_url"]; ok {
		response["image_url"] = imageURL
	}
	
	// Copy refresh_rate if provided
	if refreshRate, ok := pluginResponse["refresh_rate"]; ok {
		response["refresh_rate"] = fmt.Sprintf("%v", refreshRate)
	}

	return response, nil
}