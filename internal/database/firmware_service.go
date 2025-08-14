package database

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FirmwareService provides database operations for firmware management
type FirmwareService struct {
	db *gorm.DB
}

// NewFirmwareService creates a new firmware service
func NewFirmwareService(db *gorm.DB) *FirmwareService {
	return &FirmwareService{db: db}
}

// GetAllFirmwareVersions returns all firmware versions
func (s *FirmwareService) GetAllFirmwareVersions() ([]FirmwareVersion, error) {
	var versions []FirmwareVersion
	err := s.db.Order("released_at DESC").Find(&versions).Error
	return versions, err
}

// GetFirmwareVersionByID returns a firmware version by ID
func (s *FirmwareService) GetFirmwareVersionByID(id uuid.UUID) (*FirmwareVersion, error) {
	var version FirmwareVersion
	err := s.db.Where("id = ?", id).First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// GetFirmwareVersionByVersion returns a firmware version by version string
func (s *FirmwareService) GetFirmwareVersionByVersion(version string) (*FirmwareVersion, error) {
	var fwVersion FirmwareVersion
	err := s.db.Where("version = ?", version).First(&fwVersion).Error
	if err != nil {
		return nil, err
	}
	return &fwVersion, nil
}

// GetLatestFirmwareVersion returns the latest firmware version
func (s *FirmwareService) GetLatestFirmwareVersion() (*FirmwareVersion, error) {
	var version FirmwareVersion
	err := s.db.Where("is_latest = ?", true).First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// CreateFirmwareVersion creates a new firmware version
func (s *FirmwareService) CreateFirmwareVersion(version *FirmwareVersion) error {
	return s.db.Create(version).Error
}

// UpdateFirmwareVersion updates a firmware version
func (s *FirmwareService) UpdateFirmwareVersion(version *FirmwareVersion) error {
	return s.db.Save(version).Error
}

// DeleteFirmwareVersion deletes a firmware version
func (s *FirmwareService) DeleteFirmwareVersion(id uuid.UUID) error {
	return s.db.Delete(&FirmwareVersion{}, id).Error
}

// Job-related methods removed - using automatic firmware updates now

// GetDeviceModel returns a device model by name
func (s *FirmwareService) GetDeviceModel(modelName string) (*DeviceModel, error) {
	var model DeviceModel
	err := s.db.Where("model_name = ?", modelName).First(&model).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// GetAllDeviceModels returns all device models
func (s *FirmwareService) GetAllDeviceModels() ([]DeviceModel, error) {
	var models []DeviceModel
	err := s.db.Where("is_active = ?", true).Order("display_name ASC").Find(&models).Error
	return models, err
}

// CreateDeviceModel creates a new device model
func (s *FirmwareService) CreateDeviceModel(model *DeviceModel) error {
	return s.db.Create(model).Error
}

// UpdateDeviceModel updates a device model
func (s *FirmwareService) UpdateDeviceModel(model *DeviceModel) error {
	return s.db.Save(model).Error
}

// CanUpdateFirmware checks if a device can be updated to a specific firmware version
func (s *FirmwareService) CanUpdateFirmware(device *Device, firmwareVersion *FirmwareVersion) error {
	// Check if device has a model
	if device.ModelName == nil || *device.ModelName == "" {
		return errors.New("device model not specified")
	}

	// Get device model
	model, err := s.GetDeviceModel(*device.ModelName)
	if err != nil {
		return errors.New("device model not found")
	}

	// Check minimum firmware requirement
	if model.MinFirmware != "" && firmwareVersion.Version < model.MinFirmware {
		return errors.New("firmware version is below minimum required for device model")
	}

	// Check if firmware is downloaded
	if !firmwareVersion.IsDownloaded {
		return errors.New("firmware file not downloaded")
	}

	return nil
}