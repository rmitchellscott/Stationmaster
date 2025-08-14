package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"gorm.io/gorm"
)

// DeviceService handles device-related database operations
type DeviceService struct {
	db *gorm.DB
}

// NewDeviceService creates a new device service
func NewDeviceService(db *gorm.DB) *DeviceService {
	return &DeviceService{db: db}
}

// CreateUnclaimedDevice creates a new unclaimed device from a MAC address
func (ds *DeviceService) CreateUnclaimedDevice(macAddress string, modelName string) (*Device, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	friendlyID, err := ds.generateFriendlyID()
	if err != nil {
		return nil, err
	}

	device := &Device{
		MacAddress:  macAddress,
		FriendlyID:  friendlyID,
		APIKey:      apiKey,
		RefreshRate: 1800, // 30 minutes default
		IsActive:    true,
		IsClaimed:   false,
	}

	// Only set ModelName if it's not empty to avoid foreign key constraint issues
	if modelName != "" {
		device.ModelName = &modelName
	}

	if err := ds.db.Create(device).Error; err != nil {
		return nil, err
	}

	return device, nil
}

// ClaimDevice claims an unclaimed device for a user
func (ds *DeviceService) ClaimDevice(userID uuid.UUID, friendlyID, name string) (*Device, error) {
	device, err := ds.GetDeviceByFriendlyID(friendlyID)
	if err != nil {
		return nil, err
	}

	if device.IsClaimed {
		return nil, fmt.Errorf("device already claimed")
	}

	device.UserID = &userID
	device.Name = name
	device.IsClaimed = true

	if err := ds.db.Save(device).Error; err != nil {
		return nil, err
	}

	return device, nil
}

// GetDevicesByUserID returns all claimed devices for a specific user
func (ds *DeviceService) GetDevicesByUserID(userID uuid.UUID) ([]Device, error) {
	var devices []Device
	err := ds.db.Preload("DeviceModel").Where("user_id = ? AND is_claimed = ?", userID, true).Order("created_at DESC").Find(&devices).Error
	return devices, err
}

// GetDeviceByID returns a device by its ID
func (ds *DeviceService) GetDeviceByID(deviceID uuid.UUID) (*Device, error) {
	var device Device
	err := ds.db.Preload("DeviceModel").First(&device, "id = ?", deviceID).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDeviceByMacAddress returns a device by its MAC address
func (ds *DeviceService) GetDeviceByMacAddress(macAddress string) (*Device, error) {
	var device Device
	err := ds.db.First(&device, "mac_address = ?", macAddress).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDeviceByFriendlyID returns a device by its friendly ID
func (ds *DeviceService) GetDeviceByFriendlyID(friendlyID string) (*Device, error) {
	var device Device
	err := ds.db.First(&device, "friendly_id = ?", friendlyID).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDeviceByAPIKey returns a device by its API key
func (ds *DeviceService) GetDeviceByAPIKey(apiKey string) (*Device, error) {
	var device Device
	err := ds.db.First(&device, "api_key = ? AND is_active = ?", apiKey, true).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// UpdateDevice updates a device
func (ds *DeviceService) UpdateDevice(device *Device) error {
	// Use Updates to avoid modifying associations
	updates := map[string]interface{}{
		"name":                   device.Name,
		"refresh_rate":           device.RefreshRate,
		"is_active":              device.IsActive,
		"allow_firmware_updates": device.AllowFirmwareUpdates,
	}
	return ds.db.Model(device).Updates(updates).Error
}

// UpdateRefreshRate updates only the refresh rate for a device (GORM-safe)
func (ds *DeviceService) UpdateRefreshRate(deviceID uuid.UUID, refreshRate int) error {
	return ds.db.Model(&Device{}).Where("id = ?", deviceID).Update("refresh_rate", refreshRate).Error
}

// mapDeviceModelName maps device-reported model names to database model names
func mapDeviceModelName(deviceModel string) string {
	modelMap := map[string]string{
		"og": "og_plus", // Original TRMNL model maps to og_plus
		// Add more mappings as needed
	}

	if mappedModel, exists := modelMap[deviceModel]; exists {
		return mappedModel
	}

	// Return original if no mapping exists
	return deviceModel
}

// UpdateLastPlaylistIndex updates the last shown playlist item index for rotation
func (ds *DeviceService) UpdateLastPlaylistIndex(deviceID uuid.UUID, index int) error {
	return ds.db.Model(&Device{}).Where("id = ?", deviceID).Update("last_playlist_index", index).Error
}

// UpdateDeviceStatus updates device status information from TRMNL requests
func (ds *DeviceService) UpdateDeviceStatus(macAddress string, firmwareVersion string, batteryVoltage float64, rssi int, modelName string) error {
	now := time.Now()

	logging.Logf("[DEVICE STATUS] Updating device %s: fw=%s, battery=%.2f, rssi=%d, model=%s",
		macAddress, firmwareVersion, batteryVoltage, rssi, modelName)

	// Use explicit transaction to ensure commit
	return ds.db.Transaction(func(tx *gorm.DB) error {
		// Always update these core fields
		updateFields := map[string]interface{}{
			"firmware_version": firmwareVersion,
			"battery_voltage":  batteryVoltage,
			"rssi":             rssi,
			"last_seen":        &now,
		}
		selectFields := []string{"firmware_version", "battery_voltage", "rssi", "last_seen"}

		// Handle model_name separately to avoid foreign key issues
		if modelName != "" {
			// Map device model name to database model name
			mappedModelName := mapDeviceModelName(modelName)
			logging.Logf("[DEVICE STATUS] Model mapping: %s -> %s", modelName, mappedModelName)

			// Check if device currently has a model_name
			var device Device
			if err := tx.Select("model_name").Where("mac_address = ?", macAddress).First(&device).Error; err == nil {
				if device.ModelName == nil || *device.ModelName == "" {
					// Check if the mapped model exists in device_models table
					var deviceModel DeviceModel
					if err := tx.Where("model_name = ?", mappedModelName).First(&deviceModel).Error; err == nil {
						// Model exists, safe to update
						updateFields["model_name"] = mappedModelName
						selectFields = append(selectFields, "model_name")
						logging.Logf("[DEVICE STATUS] Will update model_name to: %s", mappedModelName)
					} else {
						logging.Logf("[DEVICE STATUS] Model %s not found in device_models table, skipping model update", mappedModelName)
					}
				} else {
					logging.Logf("[DEVICE STATUS] Device already has model_name: %s, skipping", *device.ModelName)
				}
			}
		}

		// Use Select to force update all specified fields, even if they're empty strings
		result := tx.Model(&Device{}).Where("mac_address = ?", macAddress).
			Select(selectFields).
			Updates(updateFields)

		if result.Error != nil {
			logging.Logf("[DEVICE STATUS] Update failed for %s: %v", macAddress, result.Error)
			return result.Error
		}

		if result.RowsAffected == 0 {
			logging.Logf("[DEVICE STATUS] No rows affected for %s - device not found", macAddress)
			return fmt.Errorf("no rows affected - device not found")
		}

		logging.Logf("[DEVICE STATUS] Successfully updated %d rows for device %s", result.RowsAffected, macAddress)
		return nil
	})
}

// GetUnclaimedDevices returns all unclaimed devices (optional, for admin use)
func (ds *DeviceService) GetUnclaimedDevices() ([]Device, error) {
	var devices []Device
	err := ds.db.Where("is_claimed = ?", false).Order("created_at DESC").Find(&devices).Error
	return devices, err
}

// DeleteDevice deletes a device and all associated data
func (ds *DeviceService) DeleteDevice(deviceID uuid.UUID) error {
	return ds.db.Transaction(func(tx *gorm.DB) error {
		// Delete device will cascade to playlists, playlist items, and schedules
		return tx.Delete(&Device{}, "id = ?", deviceID).Error
	})
}

// UnlinkDevice removes a device from a user account (admin operation)
func (ds *DeviceService) UnlinkDevice(deviceID uuid.UUID) error {
	return ds.db.Transaction(func(tx *gorm.DB) error {
		// Delete device will cascade to playlists, playlist items, and schedules
		return tx.Delete(&Device{}, "id = ?", deviceID).Error
	})
}

// GetAllDevices returns all devices in the system (admin only)
func (ds *DeviceService) GetAllDevices() ([]Device, error) {
	var devices []Device
	err := ds.db.Preload("User").Preload("DeviceModel").Order("created_at DESC").Find(&devices).Error
	return devices, err
}

// GetDeviceStats returns device statistics
func (ds *DeviceService) GetDeviceStats() (map[string]interface{}, error) {
	var totalDevices int64
	var activeDevices int64
	var onlineDevices int64

	if err := ds.db.Model(&Device{}).Count(&totalDevices).Error; err != nil {
		return nil, err
	}

	if err := ds.db.Model(&Device{}).Where("is_active = ?", true).Count(&activeDevices).Error; err != nil {
		return nil, err
	}

	// Consider devices online if they've been seen in the last hour
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	if err := ds.db.Model(&Device{}).Where("last_seen > ? AND is_active = ?", oneHourAgo, true).Count(&onlineDevices).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_devices":  totalDevices,
		"active_devices": activeDevices,
		"online_devices": onlineDevices,
	}, nil
}

// generateAPIKey generates a random API key for a device
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 64 character hex string
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateFriendlyID generates a unique 6-character friendly ID for a device
func (ds *DeviceService) generateFriendlyID() (string, error) {
	for attempts := 0; attempts < 100; attempts++ {
		// Generate 3 random bytes for a 6-character hex string
		bytes := make([]byte, 3)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}

		friendlyID := hex.EncodeToString(bytes)
		friendlyID = friendlyID[:6]              // Ensure exactly 6 characters
		friendlyID = strings.ToUpper(friendlyID) // Convert to uppercase like "917F0B"

		// Check if this ID already exists
		var existingDevice Device
		err := ds.db.Where("friendly_id = ?", friendlyID).First(&existingDevice).Error
		if err != nil {
			if err.Error() == "record not found" {
				// ID is unique, we can use it
				return friendlyID, nil
			}
			// Some other database error
			return "", err
		}
		// ID already exists, try again
	}

	return "", fmt.Errorf("failed to generate unique friendly ID after 100 attempts")
}

// CreateDeviceLog stores a new log entry for a device
func (ds *DeviceService) CreateDeviceLog(deviceID uuid.UUID, logData string, level string) (*DeviceLog, error) {
	if level == "" {
		level = "info"
	}

	deviceLog := &DeviceLog{
		DeviceID:  deviceID,
		LogData:   logData,
		Level:     level,
		Timestamp: time.Now(),
	}

	if err := ds.db.Create(deviceLog).Error; err != nil {
		return nil, fmt.Errorf("failed to create device log in database: %w", err)
	}

	return deviceLog, nil
}

// GetDeviceLogsByDeviceID retrieves logs for a specific device with pagination
func (ds *DeviceService) GetDeviceLogsByDeviceID(deviceID uuid.UUID, limit int, offset int) ([]DeviceLog, error) {
	var logs []DeviceLog

	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}

	err := ds.db.Where("device_id = ?", deviceID).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, err
}

// GetDeviceLogsCount returns the total count of logs for a device
func (ds *DeviceService) GetDeviceLogsCount(deviceID uuid.UUID) (int64, error) {
	var count int64
	err := ds.db.Model(&DeviceLog{}).Where("device_id = ?", deviceID).Count(&count).Error
	return count, err
}

// CleanupOldDeviceLogs removes logs older than the specified duration
func (ds *DeviceService) CleanupOldDeviceLogs(olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	result := ds.db.Where("timestamp < ?", cutoffTime).Delete(&DeviceLog{})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
