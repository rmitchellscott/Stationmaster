package database

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
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

// CreateDevice creates a new device for a user
func (ds *DeviceService) CreateDevice(userID uuid.UUID, deviceID, friendlyName string) (*Device, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	device := &Device{
		UserID:       userID,
		DeviceID:     deviceID,
		FriendlyName: friendlyName,
		APIKey:       apiKey,
		RefreshRate:  1800, // 30 minutes default
		IsActive:     true,
	}

	if err := ds.db.Create(device).Error; err != nil {
		return nil, err
	}

	// Create default playlist for the device
	playlistService := NewPlaylistService(ds.db)
	_, err = playlistService.CreatePlaylist(userID, device.ID, "Default", true)
	if err != nil {
		// Rollback device creation if playlist creation fails
		ds.db.Delete(device)
		return nil, err
	}

	return device, nil
}

// GetDevicesByUserID returns all devices for a specific user
func (ds *DeviceService) GetDevicesByUserID(userID uuid.UUID) ([]Device, error) {
	var devices []Device
	err := ds.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&devices).Error
	return devices, err
}

// GetDeviceByID returns a device by its ID
func (ds *DeviceService) GetDeviceByID(deviceID uuid.UUID) (*Device, error) {
	var device Device
	err := ds.db.First(&device, "id = ?", deviceID).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDeviceByDeviceID returns a device by its device_id (MAC/friendly ID)
func (ds *DeviceService) GetDeviceByDeviceID(deviceID string) (*Device, error) {
	var device Device
	err := ds.db.First(&device, "device_id = ?", deviceID).Error
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
	return ds.db.Save(device).Error
}

// UpdateDeviceStatus updates device status information from TRMNL requests
func (ds *DeviceService) UpdateDeviceStatus(deviceID string, firmwareVersion string, batteryVoltage float64, rssi int) error {
	now := time.Now()
	return ds.db.Model(&Device{}).Where("device_id = ?", deviceID).Updates(map[string]interface{}{
		"firmware_version": firmwareVersion,
		"battery_voltage":  batteryVoltage,
		"rssi":            rssi,
		"last_seen":       &now,
	}).Error
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
	err := ds.db.Preload("User").Order("created_at DESC").Find(&devices).Error
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