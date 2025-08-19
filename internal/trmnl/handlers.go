package trmnl

import (
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
	debugMode := os.Getenv("DEBUG") != ""

	if debugMode {
		logging.Logf("[/api/setup] Request from %s %s %s", c.ClientIP(), c.Request.Method, c.Request.URL.Path)
	}

	macAddress := c.GetHeader("ID")
	modelHeader := c.GetHeader("Model") // Device model identifier (e.g., "og")

	if macAddress == "" {
		if debugMode {
			logging.Logf("[/api/setup] Error: Missing device ID header")
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device ID header"})
		return
	}

	if debugMode {
		logging.Logf("[/api/setup] Device MAC address: %s, Model: %s", macAddress, modelHeader)

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
	device, err = deviceService.CreateUnclaimedDevice(macAddress, modelHeader)
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
	firmwareVersion := c.GetHeader("Fw-Version") // Device sends "Fw-Version" not "FW-Version"
	rssiStr := c.GetHeader("Rssi")               // Device sends "Rssi" not "RSSI"
	modelHeader := c.GetHeader("Model")          // Device model identifier (e.g., "og")
	widthStr := c.GetHeader("Width")             // Screen width
	heightStr := c.GetHeader("Height")           // Screen height

	if debugMode {
		// Log all headers sent by device
		logging.Logf("[/api/display] Device headers - ID: %s, Access-Token: %s, Refresh-Rate: %s, Battery-Voltage: %s, Fw-Version: %s, Rssi: %s, Model: %s, Width: %s, Height: %s",
			deviceID, accessToken, refreshRateStr, batteryVoltageStr, firmwareVersion, rssiStr, modelHeader, widthStr, heightStr)

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

	// Update device status in database FIRST, then check firmware
	err = deviceService.UpdateDeviceStatus(device.MacAddress, firmwareVersion, batteryVoltage, rssi, modelHeader)
	if err != nil {
		logging.Logf("[/api/display] Failed to update device status for %s: %v", device.MacAddress, err)
	}

	// Refresh device data from database after status update
	device, err = deviceService.GetDeviceByAPIKey(accessToken)
	if err != nil {
		logging.Logf("[/api/display] Failed to refresh device data for %s: %v", device.MacAddress, err)
	} else {
		// Broadcast device status update to connected SSE clients
		sseService := sse.GetSSEService()
		sseService.BroadcastToDevice(device.ID, sse.Event{
			Type: "device_status_updated",
			Data: map[string]interface{}{
				"device_id":        device.ID.String(),
				"battery_voltage":  device.BatteryVoltage,
				"rssi":             device.RSSI,
				"firmware_version": device.FirmwareVersion,
				"last_seen":        device.LastSeen,
				"is_active":        device.IsActive,
				"timestamp":        time.Now().UTC(),
			},
		})
		logging.Logf("[SSE] Broadcasted device status update for device %s: battery=%.2f, rssi=%d", device.MacAddress, device.BatteryVoltage, device.RSSI)
	}

	// Get current playlist items for this device
	playlistService := database.NewPlaylistService(db)
	
	logging.Logf("[/api/display] Querying playlist items for device %s (friendly_id: %s, user_id: %s, claimed: %t)", 
		device.MacAddress, device.FriendlyID, 
		func() string { if device.UserID != nil { return device.UserID.String() } else { return "nil" } }(), 
		device.IsClaimed)
	
	activeItems, err := playlistService.GetActivePlaylistItemsForTime(device.ID, time.Now())
	if err != nil {
		if debugMode {
			logging.Logf("[/api/display] No playlist items found for device %s (this is normal for unclaimed devices): %v", device.MacAddress, err)
		}
		// For unclaimed devices or devices without playlists, use empty activeItems slice
		activeItems = []database.PlaylistItem{}
	} else {
		logging.Logf("[/api/display] Successfully retrieved %d active items for device %s", len(activeItems), device.MacAddress)
	}

	// Note: We no longer update the device's refresh rate in the database
	// based on headers from the device. The refresh rate determination is now:
	// 1. Plugin-provided refresh rate (if any)
	// 2. Playlist item duration override (if any)
	// 3. Device's stored refresh rate (fallback)

	// Determine device status
	status := 0
	if !device.IsClaimed {
		status = 202
	}

	// Check for low battery condition FIRST - takes precedence over everything
	if device.BatteryVoltage > 0 && device.BatteryVoltage < 3.2 {
		if debugMode {
			logging.Logf("[/api/display] Device %s has low battery (%.2fV), returning low battery image", device.MacAddress, device.BatteryVoltage)
		}

		// Get site URL for absolute URL construction
		imageURL := "/assets/low_battery.png"
		if siteURL, err := database.GetSystemSetting("site_url"); err == nil && siteURL != "" {
			siteURL = strings.TrimSuffix(siteURL, "/")
			imageURL = fmt.Sprintf("%s/assets/low_battery.png", siteURL)
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

		if debugMode {
			responseBytes, _ := json.Marshal(response)
			logging.Logf("[/api/display] Low battery response to device %s: %s", device.MacAddress, string(responseBytes))

			duration := time.Since(startTime)
			logging.Logf("[/api/display] Request processing time: %v", duration)
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// Check for firmware update AFTER device status is updated
	firmwareUpdate := checkFirmwareUpdate(device)

	// Process active plugins and generate response
	response, currentItem, err := processActivePlugins(device, activeItems)
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

		refreshRate := device.RefreshRate
		
		// Handle sleep mode for fallback response
		inSleepPeriod := isInSleepPeriod(device, userTimezone)
		if debugMode || device.SleepEnabled {
			logging.Logf("[SLEEP] Device %s sleep check (fallback): enabled=%v, timezone=%s, in_sleep=%v", 
				device.MacAddress, device.SleepEnabled, userTimezone, inSleepPeriod)
		}
		
		if inSleepPeriod {
			refreshRate = calculateSecondsUntilSleepEnd(device, userTimezone)
			
			// If sleep screen is enabled, override the image URL
			if device.SleepShowScreen {
				imageURL = "https://usetrmnl.com/images/setup/sleep.png"
				filename = "sleep"
			}
			
			logging.Logf("[SLEEP] Device %s is in sleep period (fallback), refresh rate set to %d seconds, show_screen=%v", 
				device.MacAddress, refreshRate, device.SleepShowScreen)
		}

		response = gin.H{
			"status":       status,
			"image_url":    imageURL,
			"filename":     filename,
			"refresh_rate": fmt.Sprintf("%d", refreshRate),
		}
	} else {
		// Ensure required fields are set when plugins succeed
		response["status"] = status

		// Implement refresh rate priority: plugin > playlist item override > device default
		if _, exists := response["refresh_rate"]; !exists {
			// Plugin didn't provide refresh rate, check playlist item override
			if currentItem != nil && currentItem.DurationOverride != nil {
				response["refresh_rate"] = fmt.Sprintf("%d", *currentItem.DurationOverride)
			} else {
				// Fallback to device's stored refresh rate
				response["refresh_rate"] = fmt.Sprintf("%d", device.RefreshRate)
			}
		}
		// If plugin provided refresh_rate, we use it as-is (highest priority)
	}

	// Handle sleep mode - override refresh rate and image if in sleep period
	inSleepPeriod := isInSleepPeriod(device, userTimezone)
	if debugMode || device.SleepEnabled {
		logging.Logf("[SLEEP] Device %s sleep check: enabled=%v, timezone=%s, in_sleep=%v", 
			device.MacAddress, device.SleepEnabled, userTimezone, inSleepPeriod)
	}
	
	if inSleepPeriod {
		sleepRefreshRate := calculateSecondsUntilSleepEnd(device, userTimezone)
		response["refresh_rate"] = fmt.Sprintf("%d", sleepRefreshRate)
		
		// If sleep screen is enabled, override the image URL
		if device.SleepShowScreen {
			response["image_url"] = "https://usetrmnl.com/images/setup/sleep.png"
			response["filename"] = "sleep"
		}
		
		logging.Logf("[SLEEP] Device %s is in sleep period, refresh rate set to %d seconds, show_screen=%v", 
			device.MacAddress, sleepRefreshRate, device.SleepShowScreen)
	}

	// Always add firmware update info to response
	response["update_firmware"] = firmwareUpdate.UpdateFirmware
	response["firmware_url"] = firmwareUpdate.FirmwareURL
	response["reset_firmware"] = firmwareUpdate.ResetFirmware

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
	debugMode := os.Getenv("DEBUG") != ""

	if debugMode {
		logging.Logf("[/api/logs] Request from %s %s %s", c.ClientIP(), c.Request.Method, c.Request.URL.Path)
	}

	deviceID := c.GetHeader("ID")
	accessToken := c.GetHeader("Access-Token")

	if debugMode {
		logging.Logf("[/api/logs] Device headers - ID: %s, Access-Token: %s", deviceID, accessToken)

		// Log all headers for debugging
		logging.Logf("[/api/logs] All request headers:")
		for name, values := range c.Request.Header {
			for _, value := range values {
				logging.Logf("[/api/logs] Header %s: %s", name, value)
			}
		}
	}

	if deviceID == "" || accessToken == "" {
		if debugMode {
			if deviceID == "" {
				logging.Logf("[/api/logs] Authentication failed: Missing device ID")
			}
			if accessToken == "" {
				logging.Logf("[/api/logs] Authentication failed: Missing access token")
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing device ID or access token"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Verify device
	device, err := deviceService.GetDeviceByAPIKey(accessToken)
	if err != nil || device.MacAddress != deviceID {
		if debugMode {
			if err != nil {
				logging.Logf("[/api/logs] Authentication failed: Invalid access token '%s' for device ID '%s' - %v", accessToken, deviceID, err)
			} else {
				logging.Logf("[/api/logs] Authentication failed: Device ID mismatch - expected '%s', got '%s'", device.MacAddress, deviceID)
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid device credentials"})
		return
	}

	if debugMode {
		logging.Logf("[/api/logs] Authentication successful for device %s (friendly_id: %s)", device.MacAddress, device.FriendlyID)
	}

	// Parse log data
	var logData map[string]interface{}
	if err := c.ShouldBindJSON(&logData); err != nil {
		if debugMode {
			logging.Logf("[/api/logs] Failed to parse JSON data from device %s: %v", deviceID, err)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log data"})
		return
	}

	if debugMode {
		logDataBytes, _ := json.Marshal(logData)
		logging.Logf("[/api/logs] Received log data from device %s: %s", deviceID, string(logDataBytes))
	}

	// Extract log level if provided
	level := "info"
	if levelStr, ok := logData["level"].(string); ok && levelStr != "" {
		level = levelStr
	}

	if debugMode {
		logging.Logf("[/api/logs] Log level for device %s: %s", deviceID, level)
	}

	// Convert log data back to JSON string for storage
	logDataBytes, err := json.Marshal(logData)
	if err != nil {
		if debugMode {
			logging.Logf("[/api/logs] Failed to marshal log data for device %s: %v", deviceID, err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process log data"})
		return
	}

	// Store the log entry
	deviceLog, err := deviceService.CreateDeviceLog(device.ID, string(logDataBytes), level)
	if err != nil {
		logging.Logf("[/api/logs] Failed to store log data for device %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store log data"})
		return
	}

	if debugMode {
		logging.Logf("[/api/logs] Successfully stored log entry %s for device %s (level: %s)", deviceLog.ID, deviceID, level)
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
func processActivePlugins(device *database.Device, activeItems []database.PlaylistItem) (gin.H, *database.PlaylistItem, error) {
	if len(activeItems) == 0 {
		return nil, nil, fmt.Errorf("no active playlist items")
	}

	// Calculate next item index for rotation
	nextIndex := 0
	if len(activeItems) > 1 {
		// Use modulo to wrap around when reaching the end
		nextIndex = (device.LastPlaylistIndex + 1) % len(activeItems)
	}

	// Get the next item in rotation
	item := activeItems[nextIndex]

	// Get the user plugin details
	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	userPlugin, err := pluginService.GetUserPluginByID(item.UserPluginID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user plugin: %w", err)
	}

	var response gin.H
	var pluginErr error

	// Handle different plugin types
	switch userPlugin.Plugin.Type {
	case "redirect":
		response, pluginErr = processRedirectPlugin(userPlugin)
	case "alias":
		response, pluginErr = processAliasPlugin(userPlugin)
	case "core_proxy":
		response, pluginErr = processCoreProxyPlugin(device, userPlugin)
	default:
		// For other plugin types, return default response
		response = gin.H{
			"image_url": getImageURLForDevice(device),
			"filename":  "display.png",
		}
	}

	// Only update the playlist index if plugin processing was successful
	if pluginErr == nil {
		deviceService := database.NewDeviceService(db)
		if err := deviceService.UpdateLastPlaylistIndex(device.ID, nextIndex); err != nil {
			// Log error but don't fail the request
			// The rotation will still work, just might repeat an item next time
			logging.Logf("[PLAYLIST] Failed to update last playlist index for device %s: %v", device.MacAddress, err)
		} else {
			// Get user timezone for sleep calculations
			userTimezone := "UTC" // Default fallback
			if device.UserID != nil {
				userService := database.NewUserService(db)
				user, err := userService.GetUserByID(*device.UserID)
				if err == nil && user.Timezone != "" {
					userTimezone = user.Timezone
				}
			}
			
			// Check if device is currently in sleep period for SSE event
			currentlySleeping := isInSleepPeriod(device, userTimezone)
			
			// Broadcast playlist index change to connected SSE clients
			sseService := sse.GetSSEService()
			sseService.BroadcastToDevice(device.ID, sse.Event{
				Type: "playlist_index_changed",
				Data: map[string]interface{}{
					"device_id":     device.ID.String(),
					"current_index": nextIndex,
					"current_item":  item,
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
			logging.Logf("[SSE] Broadcasted playlist index change for device %s: index %d, sleep_enabled=%v, currently_sleeping=%v", 
				device.MacAddress, nextIndex, device.SleepEnabled, currentlySleeping)
		}
	}

	return response, &item, pluginErr
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

// processAliasPlugin handles alias plugin type by returning the configured image URL directly
func processAliasPlugin(userPlugin *database.UserPlugin) (gin.H, error) {
	// Parse plugin settings
	var settings map[string]interface{}
	if userPlugin.Settings != "" {
		if err := json.Unmarshal([]byte(userPlugin.Settings), &settings); err != nil {
			return nil, fmt.Errorf("failed to parse plugin settings: %w", err)
		}
	}

	// Get image URL from settings
	imageURL, ok := settings["image_url"].(string)
	if !ok || imageURL == "" {
		return nil, fmt.Errorf("image_url not configured in plugin settings")
	}

	// Return response with the image URL
	response := gin.H{
		"image_url": imageURL,
		"filename":  time.Now().Format("2006-01-02T15:04:05"),
	}

	return response, nil
}

// processCoreProxyPlugin handles core_proxy plugin type by forwarding requests to TRMNL's official server
func processCoreProxyPlugin(device *database.Device, userPlugin *database.UserPlugin) (gin.H, error) {
	// Parse plugin settings
	var settings map[string]interface{}
	if userPlugin.Settings != "" {
		if err := json.Unmarshal([]byte(userPlugin.Settings), &settings); err != nil {
			return nil, fmt.Errorf("failed to parse plugin settings: %w", err)
		}
	}

	// Get TRMNL device MAC and access token from settings
	deviceMac, ok := settings["device_mac"].(string)
	if !ok || deviceMac == "" {
		return nil, fmt.Errorf("device_mac not configured in plugin settings")
	}

	accessToken, ok := settings["access_token"].(string)
	if !ok || accessToken == "" {
		return nil, fmt.Errorf("access_token not configured in plugin settings")
	}

	// Get timeout (default to 5 seconds)
	timeoutSeconds := 5
	if timeout, ok := settings["timeout_seconds"].(float64); ok && timeout > 0 {
		timeoutSeconds = int(timeout)
		if timeoutSeconds > 15 {
			timeoutSeconds = 15 // Cap at 15 seconds
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	// Create request to TRMNL's API
	req, err := http.NewRequest("GET", "https://usetrmnl.com/api/display", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers that TRMNL expects
	req.Header.Set("ID", deviceMac)
	req.Header.Set("Access-Token", accessToken)

	// Forward device status headers if available from our local device
	if device.FirmwareVersion != "" {
		req.Header.Set("Fw-Version", device.FirmwareVersion)
	}
	if device.BatteryVoltage > 0 {
		req.Header.Set("Battery-Voltage", fmt.Sprintf("%.2f", device.BatteryVoltage))
	}
	if device.RSSI != 0 {
		req.Header.Set("Rssi", fmt.Sprintf("%d", device.RSSI))
	}
	if device.RefreshRate > 0 {
		req.Header.Set("Refresh-Rate", fmt.Sprintf("%d", device.RefreshRate))
	}

	// Make request to TRMNL
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from TRMNL API: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TRMNL API returned status %d", resp.StatusCode)
	}

	// Parse JSON response from TRMNL
	var trmnlResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&trmnlResponse); err != nil {
		return nil, fmt.Errorf("failed to parse TRMNL response: %w", err)
	}

	// Build response compatible with our API
	response := gin.H{}

	// Copy filename if provided
	if filename, ok := trmnlResponse["filename"]; ok {
		response["filename"] = filename
	} else {
		response["filename"] = time.Now().Format("2006-01-02T15:04:05")
	}

	// Copy image_url if provided
	if imageURL, ok := trmnlResponse["image_url"]; ok {
		response["image_url"] = imageURL
	}

	// Copy refresh_rate if provided
	if refreshRate, ok := trmnlResponse["refresh_rate"]; ok {
		response["refresh_rate"] = fmt.Sprintf("%v", refreshRate)
	}

	// Copy any other fields that might be useful
	if url, ok := trmnlResponse["url"]; ok {
		response["image_url"] = url
	}

	return response, nil
}

// FirmwareUpdateResponse represents firmware update information for a device
type FirmwareUpdateResponse struct {
	UpdateFirmware bool   `json:"update_firmware"`
	FirmwareURL    string `json:"firmware_url,omitempty"`
	ResetFirmware  bool   `json:"reset_firmware"`
}

// checkFirmwareUpdate checks if device needs a firmware update and can receive one
func checkFirmwareUpdate(device *database.Device) FirmwareUpdateResponse {
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
	// Get user timezone for schedule calculations
	userTimezone := "UTC" // Default fallback
	if device.UserID != nil {
		db := database.GetDB()
		userService := database.NewUserService(db)
		user, err := userService.GetUserByID(*device.UserID)
		if err == nil && user.Timezone != "" {
			userTimezone = user.Timezone
		}
	}
	
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

	// 6. Generate firmware URL - try to use absolute URL if site_url is configured
	firmwareURL := fmt.Sprintf("/files/firmware/firmware_%s.bin", latestFirmware.Version)

	// Get site URL from settings to create absolute URL
	if siteURL, err := database.GetSystemSetting("site_url"); err == nil && siteURL != "" {
		siteURL = strings.TrimSuffix(siteURL, "/") // Remove trailing slash
		firmwareURL = fmt.Sprintf("%s/files/firmware/firmware_%s.bin", siteURL, latestFirmware.Version)
		logging.Logf("[FIRMWARE UPDATE] Using absolute URL: %s", firmwareURL)
	} else {
		logging.Logf("[FIRMWARE UPDATE] Using relative URL (no site_url configured): %s", firmwareURL)
	}

	logging.Logf("[FIRMWARE UPDATE] Device %s (v%s) will be updated to v%s",
		device.MacAddress, device.FirmwareVersion, latestFirmware.Version)

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
		logging.Logf("[FIRMWARE PROXY] Device %s requesting firmware %s, proxying to %s", device.MacAddress, firmwareVersion, fwVersion.DownloadURL)

		// Create HTTP client for proxying
		client := &http.Client{
			Timeout: 5 * time.Minute, // Allow time for large firmware downloads
		}

		// Create request to TRMNL API
		req, err := http.NewRequest("GET", fwVersion.DownloadURL, nil)
		if err != nil {
			logging.Logf("[FIRMWARE PROXY] Failed to create proxy request: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to proxy firmware request"})
			return
		}

		// Make request to TRMNL
		resp, err := client.Do(req)
		if err != nil {
			logging.Logf("[FIRMWARE PROXY] Failed to fetch from TRMNL: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch firmware from upstream"})
			return
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode != http.StatusOK {
			logging.Logf("[FIRMWARE PROXY] TRMNL returned status %d for firmware %s", resp.StatusCode, firmwareVersion)
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
			logging.Logf("[FIRMWARE PROXY] Failed to stream firmware %s to device %s: %v", firmwareVersion, device.MacAddress, err)
			return
		}

		logging.Logf("[FIRMWARE PROXY] Successfully proxied firmware %s to device %s", firmwareVersion, device.MacAddress)
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
		logging.Logf("[FIRMWARE] Device %s downloading firmware %s", device.MacAddress, firmwareVersion)

		c.File(fwVersion.FilePath)

		logging.Logf("[FIRMWARE DOWNLOAD] Device %s successfully downloaded firmware %s", device.MacAddress, firmwareVersion)
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
				logging.Logf("[FIRMWARE UPDATE] Failed to update device %s firmware version: %v", device.MacAddress, err)
			}
		}

		logging.Logf("[FIRMWARE UPDATE] Device %s successfully updated to firmware v%s",
			device.MacAddress, req.NewVersion)

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Firmware update completion recorded",
		})
	} else if req.Status == "failed" {
		logging.Logf("[FIRMWARE UPDATE] Device %s firmware update failed: %s",
			device.MacAddress, req.Message)

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
	debugMode := os.Getenv("DEBUG") != ""

	if debugMode {
		logging.Logf("[/api/current_screen] Request from %s %s %s", c.ClientIP(), c.Request.Method, c.Request.URL.Path)
	}

	// Extract Access-Token header only (simpler auth than /api/display)
	accessToken := c.GetHeader("Access-Token")

	if debugMode {
		logging.Logf("[/api/current_screen] Access-Token: %s", accessToken)
	}

	if accessToken == "" {
		if debugMode {
			logging.Logf("[/api/current_screen] Authentication failed: Missing access token")
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing access token"})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)

	// Get device by API key
	device, err := deviceService.GetDeviceByAPIKey(accessToken)
	if err != nil {
		if debugMode {
			logging.Logf("[/api/current_screen] Authentication failed: Invalid access token '%s' - %v", accessToken, err)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token"})
		return
	}

	if debugMode {
		logging.Logf("[/api/current_screen] Authentication successful for device %s (friendly_id: %s)", device.MacAddress, device.FriendlyID)
	}

	// Get current playlist items for this device
	playlistService := database.NewPlaylistService(db)
	activeItems, err := playlistService.GetActivePlaylistItemsForTime(device.ID, time.Now())
	if err != nil {
		if debugMode {
			logging.Logf("[/api/current_screen] No playlist items found for device %s: %v", device.MacAddress, err)
		}
		activeItems = []database.PlaylistItem{}
	}

	// Determine device status
	status := 200
	if !device.IsClaimed {
		status = 202
	}

	// Process current plugin without advancing playlist
	response, err := processCurrentPlugin(device, activeItems)
	if err != nil {
		// Fall back to default response if plugin processing fails
		if debugMode {
			logging.Logf("[/api/current_screen] No active plugins for device %s, using default response (status: %d)", device.MacAddress, status)
		}

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

	if debugMode {
		responseBytes, _ := json.Marshal(response)
		logging.Logf("[/api/current_screen] Response to device %s: %s", device.MacAddress, string(responseBytes))

		duration := time.Since(startTime)
		logging.Logf("[/api/current_screen] Request processing time: %v", duration)
	}

	c.JSON(http.StatusOK, response)
}

// processCurrentPlugin processes the current playlist item without advancing the index
func processCurrentPlugin(device *database.Device, activeItems []database.PlaylistItem) (gin.H, error) {
	if len(activeItems) == 0 {
		return nil, fmt.Errorf("no active playlist items")
	}

	// Get the current item based on existing LastPlaylistIndex (don't advance)
	currentIndex := device.LastPlaylistIndex
	if currentIndex < 0 || currentIndex >= len(activeItems) {
		currentIndex = 0 // Default to first item if index is invalid
	}

	item := activeItems[currentIndex]

	// Get the user plugin details
	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	userPlugin, err := pluginService.GetUserPluginByID(item.UserPluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user plugin: %w", err)
	}

	var response gin.H
	var pluginErr error

	// Handle different plugin types (same logic as processActivePlugins but without index update)
	switch userPlugin.Plugin.Type {
	case "redirect":
		response, pluginErr = processRedirectPlugin(userPlugin)
	case "alias":
		response, pluginErr = processAliasPlugin(userPlugin)
	case "core_proxy":
		response, pluginErr = processCoreProxyPlugin(device, userPlugin)
	default:
		response = gin.H{
			"image_url": getImageURLForDevice(device),
			"filename":  "display.png",
		}
	}

	// Apply duration override if no refresh_rate was provided by plugin
	if pluginErr == nil {
		if _, exists := response["refresh_rate"]; !exists && item.DurationOverride != nil {
			response["refresh_rate"] = *item.DurationOverride
		}
	}

	return response, pluginErr
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
		logging.Logf("[SLEEP] Invalid timezone %s for device %s, using UTC", userTimezone, device.MacAddress)
		loc = time.UTC
	}

	// Get current time in device's timezone
	now := time.Now().In(loc)
	
	// Parse sleep start and end times
	startTime, err := parseSleepTime(device.SleepStartTime, now)
	if err != nil {
		logging.Logf("[SLEEP] Invalid start time %s for device %s: %v", device.SleepStartTime, device.MacAddress, err)
		return false
	}
	
	endTime, err := parseSleepTime(device.SleepEndTime, now)
	if err != nil {
		logging.Logf("[SLEEP] Invalid end time %s for device %s: %v", device.SleepEndTime, device.MacAddress, err)
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
		logging.Logf("[SLEEP] Invalid timezone %s for device %s, using UTC", userTimezone, device.MacAddress)
		loc = time.UTC
	}

	// Get current time in device's timezone
	now := time.Now().In(loc)
	
	// Parse sleep end time
	endTime, err := parseSleepTime(device.SleepEndTime, now)
	if err != nil {
		logging.Logf("[SLEEP] Invalid end time %s for device %s: %v", device.SleepEndTime, device.MacAddress, err)
		return device.RefreshRate
	}

	// Parse sleep start time to handle periods that cross midnight
	startTime, err := parseSleepTime(device.SleepStartTime, now)
	if err != nil {
		logging.Logf("[SLEEP] Invalid start time %s for device %s: %v", device.SleepStartTime, device.MacAddress, err)
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
		logging.Logf("[FIRMWARE UPDATE] Invalid timezone %s for device %s, using UTC", userTimezone, device.MacAddress)
		loc = time.UTC
	}

	// Get current time in device's timezone
	now := time.Now().In(loc)
	
	// Parse firmware update start and end times
	startTime, err := parseSleepTime(device.FirmwareUpdateStartTime, now)
	if err != nil {
		logging.Logf("[FIRMWARE UPDATE] Invalid start time %s for device %s: %v", device.FirmwareUpdateStartTime, device.MacAddress, err)
		return true // Default to allow if invalid time
	}
	
	endTime, err := parseSleepTime(device.FirmwareUpdateEndTime, now)
	if err != nil {
		logging.Logf("[FIRMWARE UPDATE] Invalid end time %s for device %s: %v", device.FirmwareUpdateEndTime, device.MacAddress, err)
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
