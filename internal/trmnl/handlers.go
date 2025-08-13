package trmnl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
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

		response = gin.H{
			"status":       status,
			"image_url":    imageURL,
			"filename":     filename,
			"refresh_rate": fmt.Sprintf("%d", device.RefreshRate),
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

	// Update device's last playlist index for next rotation
	db := database.GetDB()
	deviceService := database.NewDeviceService(db)
	if err := deviceService.UpdateLastPlaylistIndex(device.ID, nextIndex); err != nil {
		// Log error but don't fail the request
		// The rotation will still work, just might repeat an item next time
	}

	// Get the user plugin details
	pluginService := database.NewPluginService(db)

	userPlugin, err := pluginService.GetUserPluginByID(item.UserPluginID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user plugin: %w", err)
	}

	// Handle different plugin types
	switch userPlugin.Plugin.Type {
	case "redirect":
		response, err := processRedirectPlugin(userPlugin)
		return response, &item, err
	case "alias":
		response, err := processAliasPlugin(userPlugin)
		return response, &item, err
	case "core_proxy":
		response, err := processCoreProxyPlugin(device, userPlugin)
		return response, &item, err
	default:
		// For other plugin types, return default response
		return gin.H{
			"image_url": getImageURLForDevice(device),
			"filename":  "display.png",
		}, &item, nil
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

	db := database.GetDB()
	firmwareService := database.NewFirmwareService(db)

	// 2. Get latest firmware version
	latestFirmware, err := firmwareService.GetLatestFirmwareVersion()
	if err != nil {
		return defaultResponse
	}

	// 3. Compare with device's current version
	if device.FirmwareVersion != "" && device.FirmwareVersion >= latestFirmware.Version {
		return defaultResponse
	}

	// 4. Check if firmware file is downloaded and available
	if !latestFirmware.IsDownloaded || latestFirmware.FilePath == "" {
		return defaultResponse
	}

	// 5. Generate firmware URL - try to use absolute URL if site_url is configured
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

	// Check if firmware file exists and is downloaded
	if !fwVersion.IsDownloaded || fwVersion.FilePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firmware file not available"})
		return
	}

	// Verify device is allowed to download firmware
	if !device.AllowFirmwareUpdates {
		c.JSON(http.StatusForbidden, gin.H{"error": "Firmware updates are disabled for this device"})
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
