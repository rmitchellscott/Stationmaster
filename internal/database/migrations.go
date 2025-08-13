package database

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"gorm.io/gorm"
)

// RunMigrations runs any pending database migrations using gormigrate
func RunMigrations(logPrefix string) error {
	logging.Logf("[%s] Running database migrations...", logPrefix)

	// Create migrator with our migrations
	m := gormigrate.New(DB, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "202501130001_add_core_proxy_plugin",
			Migrate: func(tx *gorm.DB) error {
				// Check if plugin already exists
				var existingPlugin Plugin
				err := tx.Where("name = ? AND type = ?", "Core Proxy", "core_proxy").First(&existingPlugin).Error
				if err == nil {
					// Plugin already exists, skip
					return nil
				}

				// Create Core Proxy plugin
				plugin := Plugin{
					Name:        "Core Proxy",
					Type:        "core_proxy",
					Description: "Proxies requests to TRMNL official server to display core screens in your playlist",
					ConfigSchema: `{
						"type": "object",
						"properties": {
							"device_mac": {
								"type": "string",
								"title": "TRMNL Device MAC Address",
								"description": "The MAC address of your TRMNL device registered on usetrmnl.com"
							},
							"access_token": {
								"type": "string",
								"title": "TRMNL Access Token",
								"description": "Your device access token from usetrmnl.com"
							},
							"timeout_seconds": {
								"type": "number",
								"title": "Timeout (seconds)",
								"description": "Request timeout in seconds (default: 5, max: 15)",
								"default": 5,
								"minimum": 1,
								"maximum": 15
							}
						},
						"required": ["device_mac", "access_token"]
					}`,
					Version:  "1.0.0",
					Author:   "Stationmaster",
					IsActive: true,
				}

				return tx.Create(&plugin).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Where("name = ? AND type = ?", "Core Proxy", "core_proxy").Delete(&Plugin{}).Error
			},
		},
	})

	// Set initial schema if this is a fresh database
	m.InitSchema(func(tx *gorm.DB) error {
		// AutoMigrate all models to set up initial schema
		models := GetAllModels()
		for _, model := range models {
			if err := tx.AutoMigrate(model); err != nil {
				return fmt.Errorf("failed to migrate %T: %w", model, err)
			}
		}
		return nil
	})

	// Run migrations
	if err := m.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logging.Logf("[%s] Migrations completed successfully", logPrefix)
	return nil
}