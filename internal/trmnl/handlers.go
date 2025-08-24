package trmnl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/sse"
)

// SetupHandler handles device setup requests from TRMNL devices
// GET /api/setup with header 'ID': 'MAC_ADDRESS'
func SetupHandler(c *gin.Context) {
	logging.Debug("[/api/setup] Request received", "client_ip", c.ClientIP(), "method", c.Request.Method, "path", c.Request.URL.Path)

	macAddress := c.GetHeader("ID")
	modelHeader := c.GetHeader("Model") // Device model identifier (e.g., "og")

	if macAddress == "" {
		logging.Debug("[/api/setup] Error: Missing device ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device ID header"})
		return
	}

	logging.Debug("[/api/setup] Device details", "mac_address", macAddress, "model", modelHeader)

	for name, values := range c.Request.Header {
		for _, value := range values {
			logging.Debug("[/api/setup] Request header", "name", name, "value", value)
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

		if logging.IsDebugEnabled() {
			responseBytes, _ := json.Marshal(response)
			logging.Debug("[/api/setup] Response for existing device", "mac_address", macAddress, "response", string(responseBytes))
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// Device doesn't exist, auto-register it as unclaimed
	device, err = deviceService.CreateUnclaimedDevice(macAddress, modelHeader)
	if err != nil {
		logging.Error("[/api/setup] Error creating device", "mac_address", macAddress, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device"})
		return
	}

	logging.Debug("[/api/setup] Created new device", "mac_address", macAddress, "friendly_id", device.FriendlyID)

	// Return the new device information
	response := gin.H{
		"status":      200,
		"api_key":     device.APIKey,
		"friendly_id": device.FriendlyID,
		"image_url":   "https://usetrmnl.com/images/setup/setup-logo.bmp",
		"filename":    "empty_state",
	}

	if logging.IsDebugEnabled() {
		responseBytes, _ := json.Marshal(response)
		logging.Debug("[/api/setup] Response for new device", "mac_address", macAddress, "response", string(responseBytes))
	}

	c.JSON(http.StatusOK, response)
}

// DisplayHandler handles display requests from TRMNL devices
// GET /api/display with headers for device authentication and status
func DisplayHandler(c *gin.Context) {
	startTime := time.Now()

	logging.DebugWithComponent(logging.ComponentAPIDisplay, "Request received", "client_ip", c.ClientIP(), "method", c.Request.Method, "path", c.Request.URL.Path)
	
	// Variables to capture for background operations (must be declared early for defer)
	var backgroundData struct {
		deviceID             uuid.UUID
		accessToken          string
		statusValues         interface{}
		shouldUpdatePlaylist bool
		currentItem          *database.PlaylistItem
		activeItems          []database.PlaylistItem
		device               *database.Device
	}
	
	// Defer background operations to ensure they run even if API fails
	defer func() {
		if backgroundData.accessToken != "" {
			go func() {
				// Update device status in database
				deviceService := database.NewDeviceService(database.GetDB())
				if statusValues, ok := backgroundData.statusValues.(struct {
					macAddress       string
					firmwareVersion  string
					batteryVoltage   float64
					rssi             int
					modelHeader      string
					deviceID         uuid.UUID
				}); ok {
					err := deviceService.UpdateDeviceStatus(statusValues.macAddress, statusValues.firmwareVersion, statusValues.batteryVoltage, statusValues.rssi, statusValues.modelHeader)
					if err != nil {
						logging.Error("[BACKGROUND] Failed to update device status", "mac_address", statusValues.macAddress, "error", err)
					}
				}
				
				// Update playlist item ID if needed
				if backgroundData.shouldUpdatePlaylist && backgroundData.currentItem != nil {
					if err := deviceService.UpdateLastPlaylistItemID(backgroundData.deviceID, backgroundData.currentItem.ID); err != nil {
						logging.Error("[BACKGROUND] Failed to update last playlist item ID", "device_id", backgroundData.deviceID, "item_id", backgroundData.currentItem.ID, "error", err)
					} else {
						// Broadcast playlist change via SSE
						processor := GetPluginProcessor()
						if processor != nil {
							processor.broadcastPlaylistChange(backgroundData.device, *backgroundData.currentItem, backgroundData.activeItems)
						}
					}
				}
				
				// Refresh device data for SSE broadcast
				refreshedDevice, err := deviceService.GetDeviceByAPIKey(backgroundData.accessToken)
				if err != nil {
					logging.Error("[BACKGROUND] Failed to refresh device data", "device_id", backgroundData.deviceID, "error", err)
				} else {
					// Broadcast device status update to connected SSE clients
					sseService := sse.GetSSEService()
					sseService.BroadcastToDevice(backgroundData.deviceID, sse.Event{
						Type: "device_status_updated",
						Data: map[string]interface{}{
							"device_id":        refreshedDevice.ID.String(),
							"battery_voltage":  refreshedDevice.BatteryVoltage,
							"rssi":             refreshedDevice.RSSI,
							"firmware_version": refreshedDevice.FirmwareVersion,
							"last_seen":        refreshedDevice.LastSeen,
							"is_active":        refreshedDevice.IsActive,
							"timestamp":        time.Now().UTC(),
						},
					})
				}
			}()
		}
	}()
	

	// Extract headers
	deviceID := c.GetHeader("ID")
	accessToken := c.GetHeader("Access-Token")
	refreshRateStr := c.GetHeader("Refresh-Rate")
	batteryVoltageStr := c.GetHeader("Battery-Voltage")
	firmwareVersion := c.GetHeader("Fw-Version") // Device sends "Fw-Version" not "FW-Version"
	rssiStr := c.GetHeader("Rssi")               // Device sends "Rssi" not "RSSI"
	modelHeader := c.GetHeader("Model")          // Device model identifier (e.g., "og")
	widthStr := c.GetHeader("Width")             // Screen width
	heightStr := c.GetHeader("Height")           // Screen height

	logging.Debug("[/api/display] Device headers", 
		"device_id", deviceID, "access_token", accessToken, "refresh_rate", refreshRateStr,
		"battery_voltage", batteryVoltageStr, "firmware_version", firmwareVersion, 
		"rssi", rssiStr, "model", modelHeader, "width", widthStr, "height", heightStr)

	if userAgent := c.GetHeader("User-Agent"); userAgent != "" {
		logging.Debug("[/api/display] User-Agent", "user_agent", userAgent)
	}

	for name, values := range c.Request.Header {
		for _, value := range values {
			logging.Debug("[/api/display] Request header", "name", name, "value", value)
		}
	}

	if deviceID == "" || accessToken == "" {
		if deviceID == "" {
			logging.Debug("[/api/display] Authentication failed: Missing device ID")
		}
		if accessToken == "" {
			logging.Debug("[/api/display] Authentication failed: Missing or empty access token - device may not have stored API key properly")
		}
		logging.Debug("[/api/display] Rejecting request with 401 Unauthorized")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing device ID or access token"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Get device by API key
	device, err := deviceService.GetDeviceByAPIKey(accessToken)
	if err != nil {
		logging.Debug("[/api/display] Authentication failed: Invalid access token", "access_token", accessToken, "device_id", deviceID, "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token"})
		return
	}

	// Verify device ID matches (deviceID header should contain the MAC address)
	if device.MacAddress != deviceID {
		logging.Debug("[/api/display] Authentication failed: Device ID mismatch", "expected", device.MacAddress, "got", deviceID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Device ID mismatch"})
		return
	}

	logging.Debug("[/api/display] Authentication successful", "mac_address", device.MacAddress, "friendly_id", device.FriendlyID)

	// Get user timezone for sleep mode calculations
	userTimezone := "UTC" // Default fallback
	if device.UserID != nil {
		userService := database.NewUserService(db)
		user, err := userService.GetUserByID(*device.UserID)
		if err == nil && user.Timezone != "" {
			userTimezone = user.Timezone
		}
	}

	// Parse and update device status
	var batteryVoltage float64
	var rssi int

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

	// Note: We still read the refresh rate header for completeness but don't use it
	// to update the database as refresh rate is now determined by the priority logic

	// Store device status values for background update (defer until after response)
	statusValues := struct {
		macAddress       string
		firmwareVersion  string
		batteryVoltage   float64
		rssi             int
		modelHeader      string
		deviceID         uuid.UUID
	}{
		macAddress:      device.MacAddress,
		firmwareVersion: firmwareVersion,
		batteryVoltage:  batteryVoltage,
		rssi:            rssi,
		modelHeader:     modelHeader,
		deviceID:        device.ID,
	}
	
	// Capture data for background operations
	backgroundData.deviceID = device.ID
	backgroundData.accessToken = accessToken
	backgroundData.statusValues = statusValues
	

	// Get current playlist items for this device
	playlistService := database.NewPlaylistService(db)
	
	logging.Debug("[/api/display] Querying playlist items", 
		"mac_address", device.MacAddress, "friendly_id", device.FriendlyID, 
		"user_id", func() string { if device.UserID != nil { return device.UserID.String() } else { return "nil" } }(), 
		"claimed", device.IsClaimed)
	
	activeItems, err := playlistService.GetActivePlaylistItemsForTime(device.ID, time.Now())
	if err != nil {
		logging.Debug("[/api/display] No playlist items found for device (this is normal for unclaimed devices)", "mac_address", device.MacAddress, "error", err)
		// For unclaimed devices or devices without playlists, use empty activeItems slice
		activeItems = []database.PlaylistItem{}
	} else {
		logging.Info("[/api/display] Successfully retrieved active items", "count", len(activeItems), "mac_address", device.MacAddress)
	}

	// Note: We no longer update the device's refresh rate in the database
	// based on headers from the device. The refresh rate determination is now:
	// 1. Plugin-provided refresh rate (if any)
	// 2. Playlist item duration override (if any)
	// 3. Device's stored refresh rate (fallback)

	// Build base URL for image responses
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

	// Determine device status
	status := 0
	if !device.IsClaimed {
		status = 202
	}

	// Check for low battery condition FIRST - takes precedence over everything
	if device.BatteryVoltage > 0 && device.BatteryVoltage < 3.2 {
		logging.Warn("[/api/display] Device has low battery, returning low battery image", "mac_address", device.MacAddress, "voltage", device.BatteryVoltage)

		// Use relative path for low battery image URL, then convert to absolute
		imageURL := "/images/low_battery.png"
		if strings.HasPrefix(imageURL, "/images/") {
			imageURL = baseURL + imageURL
		}

		response := gin.H{
			"status":          0,
			"image_url":       imageURL,
			"filename":        "low_battery",
			"refresh_rate":    fmt.Sprintf("%d", device.RefreshRate),
			"update_firmware": false,
			"firmware_url":    "",
			"reset_firmware":  false,
		}

		if logging.IsDebugEnabled() {
			responseBytes, _ := json.Marshal(response)
			logging.Debug("[/api/display] Low battery response", "mac_address", device.MacAddress, "response", string(responseBytes))
		}
		logging.Debug("[/api/display] Request processing time", "duration", time.Since(startTime))

		c.JSON(http.StatusOK, response)
		return
	}

	// Check for firmware update AFTER device status is updated
	firmwareUpdate := checkFirmwareUpdate(c, device, userTimezone)

	// Process active plugins and generate response with 2-second timeout
	processor := GetPluginProcessor()
	var response gin.H
	var currentItem *database.PlaylistItem
	var pluginErr error
	var timedOut bool
	
	// Create timeout context for plugin processing
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// Channel to receive plugin processing results
	type pluginResult struct {
		response    gin.H
		currentItem *database.PlaylistItem
		pluginErr   error
	}
	resultChan := make(chan pluginResult, 1)
	
	// Run plugin processing in goroutine
	go func() {
		var res pluginResult
		if processor != nil {
			res.response, res.currentItem, res.pluginErr = processor.processActivePlugins(device, activeItems)
		} else {
			// No processor available - return error
			res.pluginErr = fmt.Errorf("unified plugin processor not available")
		}
		resultChan <- res
	}()
	
	// Wait for either plugin processing to complete or timeout
	select {
	case result := <-resultChan:
		response = result.response
		currentItem = result.currentItem
		pluginErr = result.pluginErr
	case <-ctx.Done():
		// Timeout occurred
		timedOut = true
		logging.Warn("[/api/display] Plugin processing timed out after 2 seconds", "mac_address", device.MacAddress)
		
		// Create timeout error response with smart refresh rate
		timeoutRefreshRate := device.RefreshRate
		
		// Try to get current item for duration override if we have active items
		if len(activeItems) > 0 {
			// Find current item by UUID or use first item
			var currentItem *database.PlaylistItem
			if device.LastPlaylistItemID != nil {
				for _, item := range activeItems {
					if item.ID == *device.LastPlaylistItemID {
						currentItem = &item
						break
					}
				}
			}
			if currentItem == nil {
				currentItem = &activeItems[0]
			}
			if currentItem.DurationOverride != nil {
				timeoutRefreshRate = *currentItem.DurationOverride
			}
		}
		
		response = gin.H{
			"image_url":  "/images/timeout_error.png",
			"filename":   "timeout_error",
			"refresh_rate": fmt.Sprintf("%d", timeoutRefreshRate),
		}
		pluginErr = fmt.Errorf("plugin processing timeout")
	}
	
	// Set background data for playlist update if plugin processing was successful and not timed out
	if pluginErr == nil && !timedOut && len(activeItems) > 0 {
		backgroundData.shouldUpdatePlaylist = true
		backgroundData.currentItem = currentItem
		backgroundData.activeItems = activeItems
		backgroundData.device = device
	}
	if pluginErr != nil && !timedOut {
		// Plugin error (not timeout) - use generic error response with smart refresh rate
		logging.Debug("[/api/display] Plugin processing failed, using error response", "mac_address", device.MacAddress, "error", pluginErr)
		
		errorRefreshRate := device.RefreshRate
		if len(activeItems) > 0 {
			// Find current item by UUID or use first item
			var currentItem *database.PlaylistItem
			if device.LastPlaylistItemID != nil {
				for _, item := range activeItems {
					if item.ID == *device.LastPlaylistItemID {
						currentItem = &item
						break
					}
				}
			}
			if currentItem == nil {
				currentItem = &activeItems[0]
			}
			if currentItem.DurationOverride != nil {
				errorRefreshRate = *currentItem.DurationOverride
			}
		}
		
		response = gin.H{
			"image_url": "/images/generic_error.png",
			"filename": "generic_error",
			"refresh_rate": fmt.Sprintf("%d", errorRefreshRate),
		}
	} else {
		// Ensure required fields are set when plugins succeed
		response["status"] = status

		// Implement refresh rate priority: playlist item override > plugin > device default
		if currentItem != nil && currentItem.DurationOverride != nil {
			// Playlist override takes highest priority
			response["refresh_rate"] = fmt.Sprintf("%d", *currentItem.DurationOverride)
		} else if _, exists := response["refresh_rate"]; !exists {
			// No playlist override and plugin didn't provide refresh rate, use device default
			response["refresh_rate"] = fmt.Sprintf("%d", device.RefreshRate)
		}
		// If no playlist override but plugin provided refresh_rate, keep plugin rate
	}

	// Handle sleep mode - override refresh rate and image if in sleep period
	inSleepPeriod := isInSleepPeriod(device, userTimezone)
	
	if inSleepPeriod {
		sleepRefreshRate := calculateSecondsUntilSleepEnd(device, userTimezone)
		response["refresh_rate"] = fmt.Sprintf("%d", sleepRefreshRate)
		
		// If sleep screen is enabled, override the image URL
		if device.SleepShowScreen {
			response["image_url"] = "/images/sleep.png"
			response["filename"] = "sleep"
		}
		
	}

	// Always add firmware update info to response
	response["update_firmware"] = firmwareUpdate.UpdateFirmware
	response["firmware_url"] = firmwareUpdate.FirmwareURL
	response["reset_firmware"] = firmwareUpdate.ResetFirmware

	// Convert relative image URLs to absolute URLs (final step before response)
	if response != nil {
		if imageURL, exists := response["image_url"]; exists {
			if imageURLStr, ok := imageURL.(string); ok {
				// Handle various relative URL patterns
				if strings.HasPrefix(imageURLStr, "/static/rendered/") {
					response["image_url"] = baseURL + imageURLStr
				} else if strings.HasPrefix(imageURLStr, "static/rendered/") {
					response["image_url"] = baseURL + "/" + imageURLStr
				} else if strings.HasPrefix(imageURLStr, "/images/") {
					response["image_url"] = baseURL + imageURLStr
				} else if strings.HasPrefix(imageURLStr, "/api/trmnl/devices/") {
					response["image_url"] = baseURL + imageURLStr
				} else if strings.HasPrefix(imageURLStr, "/") && !strings.HasPrefix(imageURLStr, "http://") && !strings.HasPrefix(imageURLStr, "https://") {
					// Any other relative URL starting with /
					response["image_url"] = baseURL + imageURLStr
				}
			}
		}
	}

	if logging.IsDebugEnabled() {
		responseBytes, _ := json.Marshal(response)
		logging.Debug("[/api/display] Response to device", "device_id", deviceID, "response", string(responseBytes))
	}
	logging.Debug("[/api/display] Request processing time", "duration", time.Since(startTime))

	c.JSON(http.StatusOK, response)
	
}

// LogsHandler handles log submissions from TRMNL devices
// POST /api/logs
func LogsHandler(c *gin.Context) {
	logging.Debug("[/api/logs] Request received", "client_ip", c.ClientIP(), "method", c.Request.Method, "path", c.Request.URL.Path)

	deviceID := c.GetHeader("ID")
	accessToken := c.GetHeader("Access-Token")

	logging.Debug("[/api/logs] Device headers", "device_id", deviceID, "access_token", accessToken)

	for name, values := range c.Request.Header {
		for _, value := range values {
			logging.Debug("[/api/logs] Request header", "name", name, "value", value)
		}
	}

	if deviceID == "" || accessToken == "" {
		if deviceID == "" {
			logging.Debug("[/api/logs] Authentication failed: Missing device ID")
		}
		if accessToken == "" {
			logging.Debug("[/api/logs] Authentication failed: Missing access token")
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing device ID or access token"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Verify device
	device, err := deviceService.GetDeviceByAPIKey(accessToken)
	if err != nil || device.MacAddress != deviceID {
		if err != nil {
			logging.Debug("[/api/logs] Authentication failed: Invalid access token", "access_token", accessToken, "device_id", deviceID, "error", err)
		} else {
			logging.Debug("[/api/logs] Authentication failed: Device ID mismatch", "expected", device.MacAddress, "got", deviceID)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid device credentials"})
		return
	}

	logging.Debug("[/api/logs] Authentication successful", "mac_address", device.MacAddress, "friendly_id", device.FriendlyID)

	// Parse log data
	var logData map[string]interface{}
	if err := c.ShouldBindJSON(&logData); err != nil {
		logging.Debug("[/api/logs] Failed to parse JSON data", "device_id", deviceID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log data"})
		return
	}

	if logging.IsDebugEnabled() {
		logDataBytes, _ := json.Marshal(logData)
		logging.Debug("[/api/logs] Received log data", "device_id", deviceID, "data", string(logDataBytes))
	}

	// Extract log level if provided
	level := "info"
	if levelStr, ok := logData["level"].(string); ok && levelStr != "" {
		level = levelStr
	}

	logging.Debug("[/api/logs] Log level determined", "device_id", deviceID, "level", level)

	// Convert log data back to JSON string for storage
	logDataBytes, err := json.Marshal(logData)
	if err != nil {
		logging.Error("[/api/logs] Failed to marshal log data", "device_id", deviceID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process log data"})
		return
	}

	// Store the log entry
	deviceLog, err := deviceService.CreateDeviceLog(device.ID, string(logDataBytes), level)
	if err != nil {
		logging.Error("[/api/logs] Failed to store log data", "device_id", deviceID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store log data"})
		return
	}

	logging.Debug("[/api/logs] Successfully stored log entry", "log_id", deviceLog.ID, "device_id", deviceID, "level", level)

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


// parsePluginSettings parses plugin settings from JSON string
func parsePluginSettings(settingsJSON string) (map[string]interface{}, error) {
	var settings map[string]interface{}
	if settingsJSON != "" {
		if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
			return nil, fmt.Errorf("failed to parse plugin settings: %w", err)
		}
	}
	return settings, nil
}

// FirmwareUpdateResponse represents firmware update information for a device
type FirmwareUpdateResponse struct {
	UpdateFirmware bool   `json:"update_firmware"`
	FirmwareURL    string `json:"firmware_url,omitempty"`
	ResetFirmware  bool   `json:"reset_firmware"`
}

// checkFirmwareUpdate checks if device needs a firmware update and can receive one
func checkFirmwareUpdate(c *gin.Context, device *database.Device, userTimezone string) FirmwareUpdateResponse {
	// Default response - no firmware update
	defaultResponse := FirmwareUpdateResponse{
		UpdateFirmware: false,
		FirmwareURL:    "",
		ResetFirmware:  false,
	}

	// 1. Check if updates are allowed for this device
	if !device.AllowFirmwareUpdates {
		return defaultResponse
	}

	// 2. Check if we're in the firmware update schedule window
	// Use user timezone passed from caller (eliminates duplicate lookup)
	
	if !isInFirmwareUpdatePeriod(device, userTimezone) {
		return defaultResponse
	}

	db := database.GetDB()
	firmwareService := database.NewFirmwareService(db)

	// 3. Get latest firmware version
	latestFirmware, err := firmwareService.GetLatestFirmwareVersion()
	if err != nil {
		return defaultResponse
	}

	// 4. Compare with device's current version
	if device.FirmwareVersion != "" && device.FirmwareVersion >= latestFirmware.Version {
		return defaultResponse
	}

	// 5. Check if firmware is available based on current mode
	firmwareMode := os.Getenv("FIRMWARE_MODE")
	if firmwareMode == "" {
		firmwareMode = "proxy" // Default to proxy mode
	}

	if firmwareMode == "proxy" {
		// In proxy mode, firmware is available if we have a download URL
		if latestFirmware.DownloadURL == "" {
			return defaultResponse
		}
	} else {
		// In download mode, firmware must be downloaded locally
		if !latestFirmware.IsDownloaded || latestFirmware.FilePath == "" {
			return defaultResponse
		}
	}

	// 6. Generate firmware URL using request-based host return
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	firmwareURL := fmt.Sprintf("%s/files/firmware/firmware_%s.bin", baseURL, latestFirmware.Version)
	
	logging.Debug("[FIRMWARE UPDATE] Using request-based URL", "url", firmwareURL)

	logging.Info("[FIRMWARE UPDATE] Device will be updated", "mac_address", device.MacAddress, "current_version", device.FirmwareVersion, "new_version", latestFirmware.Version)

	return FirmwareUpdateResponse{
		UpdateFirmware: true,
		FirmwareURL:    firmwareURL,
		ResetFirmware:  false, // Usually false unless doing a factory reset
	}
}

// FirmwareDownloadHandler serves firmware files for device downloads
func FirmwareDownloadHandler(c *gin.Context) {
	firmwareVersion := c.Param("version")
	deviceID := c.GetHeader("ID")
	accessToken := c.GetHeader("Access-Token")

	// Authenticate device
	if deviceID == "" || accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing device credentials"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)
	firmwareService := database.NewFirmwareService(db)

	// Verify device
	device, err := deviceService.GetDeviceByAPIKey(accessToken)
	if err != nil || device.MacAddress != deviceID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid device credentials"})
		return
	}

	// Get firmware version
	fwVersion, err := firmwareService.GetFirmwareVersionByVersion(firmwareVersion)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firmware version not found"})
		return
	}

	// Verify device is allowed to download firmware
	if !device.AllowFirmwareUpdates {
		c.JSON(http.StatusForbidden, gin.H{"error": "Firmware updates are disabled for this device"})
		return
	}

	// Check firmware mode - proxy or download
	firmwareMode := os.Getenv("FIRMWARE_MODE")
	if firmwareMode == "" {
		firmwareMode = "proxy" // Default to proxy mode
	}

	if firmwareMode == "proxy" {
		// Proxy mode - forward request to TRMNL API
		if fwVersion.DownloadURL == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Firmware download URL not available"})
			return
		}

		// Log the proxy request
		logging.Info("[FIRMWARE PROXY] Device requesting firmware, proxying", "mac_address", device.MacAddress, "version", firmwareVersion, "url", fwVersion.DownloadURL)

		// Create HTTP client for proxying
		client := &http.Client{
			Timeout: 5 * time.Minute, // Allow time for large firmware downloads
		}

		// Create request to TRMNL API
		req, err := http.NewRequest("GET", fwVersion.DownloadURL, nil)
		if err != nil {
			logging.Error("[FIRMWARE PROXY] Failed to create proxy request", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to proxy firmware request"})
			return
		}

		// Make request to TRMNL
		resp, err := client.Do(req)
		if err != nil {
			logging.Error("[FIRMWARE PROXY] Failed to fetch from TRMNL", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch firmware from upstream"})
			return
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode != http.StatusOK {
			logging.Error("[FIRMWARE PROXY] TRMNL returned error status", "status_code", resp.StatusCode, "version", firmwareVersion)
			c.JSON(http.StatusBadGateway, gin.H{"error": "Upstream firmware server error"})
			return
		}

		// Set response headers
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"firmware_%s.bin\"", firmwareVersion))
		if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
			c.Header("Content-Length", contentLength)
		}

		// Stream the response from TRMNL to device
		c.Status(http.StatusOK)
		_, err = io.Copy(c.Writer, resp.Body)
		if err != nil {
			logging.Error("[FIRMWARE PROXY] Failed to stream firmware", "version", firmwareVersion, "mac_address", device.MacAddress, "error", err)
			return
		}

		logging.Info("[FIRMWARE PROXY] Successfully proxied firmware", "version", firmwareVersion, "mac_address", device.MacAddress)
	} else {
		// Download mode - serve local file
		// Check if firmware file exists and is downloaded
		if !fwVersion.IsDownloaded || fwVersion.FilePath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Firmware file not available"})
			return
		}

		// Serve the firmware file
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"firmware_%s.bin\"", firmwareVersion))

		if fwVersion.FileSize > 0 {
			c.Header("Content-Length", fmt.Sprintf("%d", fwVersion.FileSize))
		}

		// Log the download
		logging.Info("[FIRMWARE] Device downloading firmware", "mac_address", device.MacAddress, "version", firmwareVersion)

		c.File(fwVersion.FilePath)

		logging.Info("[FIRMWARE DOWNLOAD] Device successfully downloaded firmware", "mac_address", device.MacAddress, "version", firmwareVersion)
	}
}

// FirmwareUpdateCompleteHandler handles device reporting firmware update completion
func FirmwareUpdateCompleteHandler(c *gin.Context) {
	deviceID := c.GetHeader("ID")
	accessToken := c.GetHeader("Access-Token")

	// Authenticate device
	if deviceID == "" || accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing device credentials"})
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

	// Parse request body
	var req struct {
		Status     string `json:"status" binding:"required"` // "success" or "failed"
		NewVersion string `json:"new_version,omitempty"`     // For successful updates
		Message    string `json:"message,omitempty"`         // Optional status message
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if req.Status == "success" {
		// Update device firmware version
		if req.NewVersion != "" {
			device.FirmwareVersion = req.NewVersion
			if err := deviceService.UpdateDevice(device); err != nil {
				logging.Error("[FIRMWARE UPDATE] Failed to update device firmware version", "mac_address", device.MacAddress, "error", err)
			}
		}

		logging.Info("[FIRMWARE UPDATE] Device successfully updated", "mac_address", device.MacAddress, "new_version", req.NewVersion)

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Firmware update completion recorded",
		})
	} else if req.Status == "failed" {
		logging.Error("[FIRMWARE UPDATE] Device firmware update failed", "mac_address", device.MacAddress, "message", req.Message)

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Firmware update failure recorded",
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status, must be 'success' or 'failed'"})
	}
}

// CurrentScreenHandler handles current screen requests without advancing playlist
// GET /api/current_screen with Access-Token header only
func CurrentScreenHandler(c *gin.Context) {
	startTime := time.Now()

	logging.Debug("[/api/current_screen] Request received", "client_ip", c.ClientIP(), "method", c.Request.Method, "path", c.Request.URL.Path)

	// Extract Access-Token header only (simpler auth than /api/display)
	accessToken := c.GetHeader("Access-Token")

	logging.Debug("[/api/current_screen] Access token received", "access_token", accessToken)

	if accessToken == "" {
		logging.Debug("[/api/current_screen] Authentication failed: Missing access token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing access token"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Get device by API key
	device, err := deviceService.GetDeviceByAPIKey(accessToken)
	if err != nil {
		logging.Debug("[/api/current_screen] Authentication failed: Invalid access token", "access_token", accessToken, "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token"})
		return
	}

	logging.Debug("[/api/current_screen] Authentication successful", "mac_address", device.MacAddress, "friendly_id", device.FriendlyID)

	// Get current playlist items for this device
	playlistService := database.NewPlaylistService(db)
	activeItems, err := playlistService.GetActivePlaylistItemsForTime(device.ID, time.Now())
	if err != nil {
		logging.Debug("[/api/current_screen] No playlist items found", "mac_address", device.MacAddress, "error", err)
		activeItems = []database.PlaylistItem{}
	}

	// Determine device status
	status := 200
	if !device.IsClaimed {
		status = 202
	}

	// Build base URL for image responses
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

	// Process current plugin without advancing playlist
	processor := GetPluginProcessor()
	var response gin.H
	var pluginErr error
	
	if processor != nil {
		response, pluginErr = processor.processCurrentPlugin(device, activeItems)
	} else {
		// No processor available - return error
		pluginErr = fmt.Errorf("unified plugin processor not available")
	}
	
	// Convert relative image URLs to absolute URLs
	if response != nil {
		if imageURL, exists := response["image_url"]; exists {
			if imageURLStr, ok := imageURL.(string); ok {
				// Handle various relative URL patterns
				if strings.HasPrefix(imageURLStr, "/static/rendered/") {
					response["image_url"] = baseURL + imageURLStr
				} else if strings.HasPrefix(imageURLStr, "static/rendered/") {
					response["image_url"] = baseURL + "/" + imageURLStr
				} else if strings.HasPrefix(imageURLStr, "/images/") {
					response["image_url"] = baseURL + imageURLStr
				} else if strings.HasPrefix(imageURLStr, "/api/trmnl/devices/") {
					response["image_url"] = baseURL + imageURLStr
				} else if strings.HasPrefix(imageURLStr, "/") && !strings.HasPrefix(imageURLStr, "http://") && !strings.HasPrefix(imageURLStr, "https://") {
					// Any other relative URL starting with /
					response["image_url"] = baseURL + imageURLStr
				}
			}
		}
	}
	if pluginErr != nil {
		// Fall back to default response if plugin processing fails
		logging.Debug("[/api/current_screen] No active plugins, using default response", "mac_address", device.MacAddress, "status", status)

		// For unclaimed devices (status 202), provide setup image
		imageURL := getImageURLForDevice(device)
		filename := time.Now().Format("2006-01-02T15:04:05")

		if status == 202 {
			imageURL = "https://usetrmnl.com/images/setup/setup-logo.bmp"
			filename = "empty_state"
		}

		response = gin.H{
			"status":       status,
			"image_url":    imageURL,
			"filename":     filename,
			"refresh_rate": device.RefreshRate,
			"rendered_at":  nil,
		}
	} else {
		// Ensure required fields are set when plugins succeed
		response["status"] = status
		response["rendered_at"] = nil

		// Set refresh rate if not provided by plugin
		if _, exists := response["refresh_rate"]; !exists {
			response["refresh_rate"] = device.RefreshRate
		}
	}

	if logging.IsDebugEnabled() {
		responseBytes, _ := json.Marshal(response)
		logging.Debug("[/api/current_screen] Response to device", "mac_address", device.MacAddress, "response", string(responseBytes))
	}
	logging.Debug("[/api/current_screen] Request processing time", "duration", time.Since(startTime))

	c.JSON(http.StatusOK, response)
}

// isInSleepPeriod checks if the current time falls within the device's sleep schedule
// IsInSleepPeriod checks if a device is currently in its sleep period (exported version)
func IsInSleepPeriod(device *database.Device, userTimezone string) bool {
	return isInSleepPeriod(device, userTimezone)
}

func isInSleepPeriod(device *database.Device, userTimezone string) bool {
	if !device.SleepEnabled || device.SleepStartTime == "" || device.SleepEndTime == "" {
		return false
	}

	// Parse timezone
	loc, err := time.LoadLocation(userTimezone)
	if err != nil {
		loc = time.UTC
	}

	// Get current time in device's timezone
	now := time.Now().In(loc)
	
	// Parse sleep start and end times
	startTime, err := parseSleepTime(device.SleepStartTime, now)
	if err != nil {
		return false
	}
	
	endTime, err := parseSleepTime(device.SleepEndTime, now)
	if err != nil {
		return false
	}

	// Handle sleep periods that cross midnight
	if startTime.After(endTime) {
		// Sleep period crosses midnight (e.g., 22:00 to 06:00)
		return now.After(startTime) || now.Before(endTime)
	} else {
		// Sleep period is within the same day (e.g., 01:00 to 05:00)
		return now.After(startTime) && now.Before(endTime)
	}
}

// calculateSecondsUntilSleepEnd calculates seconds until the end of the current sleep period
func calculateSecondsUntilSleepEnd(device *database.Device, userTimezone string) int {
	if !device.SleepEnabled || device.SleepStartTime == "" || device.SleepEndTime == "" {
		return device.RefreshRate
	}

	// Parse timezone
	loc, err := time.LoadLocation(userTimezone)
	if err != nil {
		loc = time.UTC
	}

	// Get current time in device's timezone
	now := time.Now().In(loc)
	
	// Parse sleep end time
	endTime, err := parseSleepTime(device.SleepEndTime, now)
	if err != nil {
		return device.RefreshRate
	}

	// Parse sleep start time to handle periods that cross midnight
	startTime, err := parseSleepTime(device.SleepStartTime, now)
	if err != nil {
		return device.RefreshRate
	}

	// Handle sleep periods that cross midnight
	if startTime.After(endTime) {
		// Sleep period crosses midnight
		if now.After(startTime) {
			// We're after start time, so end time is tomorrow
			endTime = endTime.Add(24 * time.Hour)
		}
		// If we're before end time and start time is after end time, 
		// then we're in the early morning part of the sleep period
	}

	// Calculate seconds until end time
	duration := endTime.Sub(now)
	seconds := int(duration.Seconds())
	
	// Ensure we return a positive value and cap at max refresh rate
	if seconds <= 0 {
		return device.RefreshRate
	}
	
	// Cap at 24 hours to prevent extremely long refresh rates
	maxSeconds := 24 * 60 * 60
	if seconds > maxSeconds {
		return maxSeconds
	}
	
	return seconds
}

// parseSleepTime parses a time string (HH:MM) and returns a time.Time for the given date
func parseSleepTime(timeStr string, referenceTime time.Time) (time.Time, error) {
	// Parse time in HH:MM format
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format %s: %w", timeStr, err)
	}
	
	// Create time for the same date as reference time
	return time.Date(
		referenceTime.Year(),
		referenceTime.Month(),
		referenceTime.Day(),
		t.Hour(),
		t.Minute(),
		0, 0,
		referenceTime.Location(),
	), nil
}

// isInFirmwareUpdatePeriod checks if the current time falls within the device's firmware update schedule
func isInFirmwareUpdatePeriod(device *database.Device, userTimezone string) bool {
	if device.FirmwareUpdateStartTime == "" || device.FirmwareUpdateEndTime == "" {
		return true // Default to always allow if no schedule is set
	}

	// Parse timezone
	loc, err := time.LoadLocation(userTimezone)
	if err != nil {
		logging.Warn("[FIRMWARE UPDATE] Invalid timezone, using UTC", "timezone", userTimezone, "mac_address", device.MacAddress)
		loc = time.UTC
	}

	// Get current time in device's timezone
	now := time.Now().In(loc)
	
	// Parse firmware update start and end times
	startTime, err := parseSleepTime(device.FirmwareUpdateStartTime, now)
	if err != nil {
		logging.Warn("[FIRMWARE UPDATE] Invalid start time", "start_time", device.FirmwareUpdateStartTime, "mac_address", device.MacAddress, "error", err)
		return true // Default to allow if invalid time
	}
	
	endTime, err := parseSleepTime(device.FirmwareUpdateEndTime, now)
	if err != nil {
		logging.Warn("[FIRMWARE UPDATE] Invalid end time", "end_time", device.FirmwareUpdateEndTime, "mac_address", device.MacAddress, "error", err)
		return true // Default to allow if invalid time
	}

	// Handle firmware update periods that cross midnight
	if startTime.After(endTime) {
		// Update period crosses midnight (e.g., 22:00 to 06:00)
		return now.After(startTime) || now.Before(endTime)
	} else {
		// Update period is within the same day (e.g., 01:00 to 05:00)
		return now.After(startTime) && now.Before(endTime)
	}
}
