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
			ID: "202501130000_add_bit_depth_to_device_models",
			Migrate: func(tx *gorm.DB) error {
				// Check if bit_depth column already exists
				if tx.Migrator().HasColumn(&DeviceModel{}, "bit_depth") {
					return nil // Column already exists, skip
				}

				// For SQLite, we need to handle foreign keys carefully
				var dbType string
				if err := tx.Raw("SELECT sqlite_version()").Scan(&dbType); err == nil {
					// SQLite - temporarily disable foreign keys
					if err := tx.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
						return fmt.Errorf("failed to disable foreign keys: %w", err)
					}
					defer tx.Exec("PRAGMA foreign_keys = ON")
				}

				// Add the bit_depth column with default value of 1
				if err := tx.Exec("ALTER TABLE device_models ADD COLUMN bit_depth INTEGER DEFAULT 1").Error; err != nil {
					return fmt.Errorf("failed to add bit_depth column: %w", err)
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// SQLite doesn't support dropping columns easily, so we'll leave it
				return nil
			},
		},
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