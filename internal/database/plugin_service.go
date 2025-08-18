package database

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PluginService handles plugin-related database operations
type PluginService struct {
	db *gorm.DB
}

// NewPluginService creates a new plugin service
func NewPluginService(db *gorm.DB) *PluginService {
	return &PluginService{db: db}
}

// CreatePlugin creates a new system plugin (admin only)
func (ps *PluginService) CreatePlugin(name, pluginType, description, configSchema, version, author string) (*Plugin, error) {
	plugin := &Plugin{
		Name:         name,
		Type:         pluginType,
		Description:  description,
		ConfigSchema: configSchema,
		Version:      version,
		Author:       author,
		IsActive:     true,
	}

	if err := ps.db.Create(plugin).Error; err != nil {
		return nil, err
	}

	return plugin, nil
}

// GetAllPlugins returns all system plugins
func (ps *PluginService) GetAllPlugins() ([]Plugin, error) {
	var plugins []Plugin
	err := ps.db.Where("is_active = ?", true).Order("name").Find(&plugins).Error
	return plugins, err
}

// GetPluginByID returns a plugin by its ID
func (ps *PluginService) GetPluginByID(pluginID uuid.UUID) (*Plugin, error) {
	var plugin Plugin
	err := ps.db.First(&plugin, "id = ? AND is_active = ?", pluginID, true).Error
	if err != nil {
		return nil, err
	}
	return &plugin, nil
}

// UpdatePlugin updates a system plugin
func (ps *PluginService) UpdatePlugin(plugin *Plugin) error {
	return ps.db.Save(plugin).Error
}

// DeletePlugin soft deletes a system plugin
func (ps *PluginService) DeletePlugin(pluginID uuid.UUID) error {
	return ps.db.Model(&Plugin{}).Where("id = ?", pluginID).Update("is_active", false).Error
}

// CreateUserPlugin creates a user instance of a plugin
func (ps *PluginService) CreateUserPlugin(userID, pluginID uuid.UUID, name string, settings map[string]interface{}) (*UserPlugin, error) {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	userPlugin := &UserPlugin{
		UserID:   userID,
		PluginID: pluginID,
		Name:     name,
		Settings: string(settingsJSON),
		IsActive: true,
	}

	if err := ps.db.Create(userPlugin).Error; err != nil {
		return nil, err
	}

	return userPlugin, nil
}

// GetUserPluginsByUserID returns all plugin instances for a user
func (ps *PluginService) GetUserPluginsByUserID(userID uuid.UUID) ([]UserPlugin, error) {
	var userPlugins []UserPlugin
	err := ps.db.Preload("Plugin").Preload("PlaylistItems").Where("user_id = ? AND is_active = ?", userID, true).Order("created_at DESC").Find(&userPlugins).Error
	return userPlugins, err
}

// GetUserPluginByID returns a user plugin instance by ID
func (ps *PluginService) GetUserPluginByID(userPluginID uuid.UUID) (*UserPlugin, error) {
	var userPlugin UserPlugin
	err := ps.db.Preload("Plugin").First(&userPlugin, "id = ? AND is_active = ?", userPluginID, true).Error
	if err != nil {
		return nil, err
	}
	return &userPlugin, nil
}

// UpdateUserPlugin updates a user plugin instance
func (ps *PluginService) UpdateUserPlugin(userPlugin *UserPlugin) error {
	return ps.db.Save(userPlugin).Error
}

// UpdateUserPluginSettings updates just the settings of a user plugin
func (ps *PluginService) UpdateUserPluginSettings(userPluginID uuid.UUID, settings map[string]interface{}) error {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	return ps.db.Model(&UserPlugin{}).Where("id = ?", userPluginID).Update("settings", string(settingsJSON)).Error
}

// DeleteUserPlugin soft deletes a user plugin instance
func (ps *PluginService) DeleteUserPlugin(userPluginID uuid.UUID) error {
	return ps.db.Transaction(func(tx *gorm.DB) error {
		// First soft delete the user plugin
		if err := tx.Model(&UserPlugin{}).Where("id = ?", userPluginID).Update("is_active", false).Error; err != nil {
			return err
		}

		// Then delete any playlist items that reference this user plugin
		return tx.Where("user_plugin_id = ?", userPluginID).Delete(&PlaylistItem{}).Error
	})
}

// GetUserPluginSettings returns the parsed settings for a user plugin
func (ps *PluginService) GetUserPluginSettings(userPluginID uuid.UUID) (map[string]interface{}, error) {
	userPlugin, err := ps.GetUserPluginByID(userPluginID)
	if err != nil {
		return nil, err
	}

	var settings map[string]interface{}
	if userPlugin.Settings != "" {
		if err := json.Unmarshal([]byte(userPlugin.Settings), &settings); err != nil {
			return nil, err
		}
	} else {
		settings = make(map[string]interface{})
	}

	return settings, nil
}

// GetPluginStats returns plugin usage statistics
func (ps *PluginService) GetPluginStats() (map[string]interface{}, error) {
	var totalPlugins int64
	var activePlugins int64
	var totalUserPlugins int64
	var activeUserPlugins int64

	if err := ps.db.Model(&Plugin{}).Count(&totalPlugins).Error; err != nil {
		return nil, err
	}

	if err := ps.db.Model(&Plugin{}).Where("is_active = ?", true).Count(&activePlugins).Error; err != nil {
		return nil, err
	}

	if err := ps.db.Model(&UserPlugin{}).Count(&totalUserPlugins).Error; err != nil {
		return nil, err
	}

	if err := ps.db.Model(&UserPlugin{}).Where("is_active = ?", true).Count(&activeUserPlugins).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_plugins":       totalPlugins,
		"active_plugins":      activePlugins,
		"total_user_plugins":  totalUserPlugins,
		"active_user_plugins": activeUserPlugins,
	}, nil
}
