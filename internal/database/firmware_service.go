package database

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FirmwareService struct {
	db *gorm.DB
}

func NewFirmwareService(db *gorm.DB) *FirmwareService {
	return &FirmwareService{db: db}
}

func (s *FirmwareService) GetAllFirmwareVersions() ([]FirmwareVersion, error) {
	var versions []FirmwareVersion
	err := s.db.Order("model_family ASC, released_at DESC").Find(&versions).Error
	return versions, err
}

func (s *FirmwareService) GetFirmwareVersionByID(id uuid.UUID) (*FirmwareVersion, error) {
	var version FirmwareVersion
	err := s.db.Where("id = ?", id).First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func (s *FirmwareService) GetFirmwareVersionByVersion(version string) (*FirmwareVersion, error) {
	var fwVersion FirmwareVersion
	err := s.db.Where("version = ?", version).First(&fwVersion).Error
	if err != nil {
		return nil, err
	}
	return &fwVersion, nil
}

func (s *FirmwareService) GetLatestFirmwareVersion() (*FirmwareVersion, error) {
	var version FirmwareVersion
	err := s.db.Where("is_latest = ? AND model_family = ?", true, "trmnl").First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func (s *FirmwareService) GetLatestFirmwareVersionForFamily(family string) (*FirmwareVersion, error) {
	var version FirmwareVersion
	err := s.db.Where("is_latest = ? AND model_family = ?", true, family).First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func (s *FirmwareService) GetFirmwareFamilies() ([]string, error) {
	var families []string
	err := s.db.Model(&FirmwareVersion{}).Distinct("model_family").Pluck("model_family", &families).Error
	return families, err
}

func (s *FirmwareService) CreateFirmwareVersion(version *FirmwareVersion) error {
	return s.db.Create(version).Error
}

func (s *FirmwareService) UpdateFirmwareVersion(version *FirmwareVersion) error {
	return s.db.Save(version).Error
}

func (s *FirmwareService) DeleteFirmwareVersion(id uuid.UUID) error {
	return s.db.Delete(&FirmwareVersion{}, id).Error
}

func (s *FirmwareService) GetDeviceModel(modelName string) (*DeviceModel, error) {
	var model DeviceModel
	err := s.db.Where("model_name = ?", modelName).First(&model).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (s *FirmwareService) GetAllDeviceModels() ([]DeviceModel, error) {
	var models []DeviceModel
	err := s.db.Where("is_active = ?", true).Order("display_name ASC").Find(&models).Error
	return models, err
}

func (s *FirmwareService) CreateDeviceModel(model *DeviceModel) error {
	return s.db.Create(model).Error
}

func (s *FirmwareService) UpdateDeviceModel(model *DeviceModel) error {
	return s.db.Save(model).Error
}

func (s *FirmwareService) CanUpdateFirmware(device *Device, firmwareVersion *FirmwareVersion) error {
	if device.DeviceModelID == nil {
		return errors.New("device model not specified")
	}

	var model DeviceModel
	err := s.db.Where("id = ?", *device.DeviceModelID).First(&model).Error
	if err != nil {
		return errors.New("device model not found")
	}

	if model.MinFirmware != "" && firmwareVersion.Version < model.MinFirmware {
		return errors.New("firmware version is below minimum required for device model")
	}

	return nil
}

// GetFirmwareFamily maps a device model name to its firmware family key.
// Returns the family key and whether the device is firmware-updatable.
func GetFirmwareFamily(modelName string) (string, bool) {
	families := map[string]string{
		"og_plus":               "trmnl",
		"og_png":                "trmnl",
		"v2":                    "trmnl_x",
		"og_bwry":               "trmnl_4clr",
		"xteink_x4":             "xteink_x4",
		"seeed_e1001":           "seeed_E1001",
		"seeed_e1002":           "seeed_E1002",
		"seeed_xiao_esp32c3":   "seeed_xiao_esp32c3",
		"waveshare_4_26":        "trmnl",
		"waveshare_7_5_bw":      "trmnl",
		"waveshare_7_5_bwr":     "trmnl",
		"waveshare_7_5_bwry":    "trmnl",
	}

	if family, ok := families[modelName]; ok {
		return family, true
	}
	return "", false
}
