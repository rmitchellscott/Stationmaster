package database

import (
	"fmt"
	"strings"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"gorm.io/gorm"
)

// RunMigrations runs any pending database migrations using gormigrate
func RunMigrations(logPrefix string) error {
	logging.Info("Running database migrations", "component", logPrefix)

	// Define migrations
	migrations := []*gormigrate.Migration{
		{
			ID: "20250821_drop_model_name_column",
			Migrate: func(tx *gorm.DB) error {
				// Check if column exists before trying to drop it
				if tx.Migrator().HasColumn(&Device{}, "model_name") {
					logging.Info("[MIGRATION] Dropping model_name column from devices table")
					return tx.Migrator().DropColumn(&Device{}, "model_name")
				}
				logging.Info("[MIGRATION] model_name column already removed")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// Add the column back if rolling back
				return tx.Migrator().AddColumn(&Device{}, "model_name")
			},
		},
		{
			ID: "20250822_cleanup_legacy_plugins",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Cleaning up legacy plugin entries from old migration system")

				// Find all plugins that might be legacy from old migration
				var plugins []Plugin
				if err := tx.Where("is_active = ?", true).Find(&plugins).Error; err != nil {
					return fmt.Errorf("failed to fetch plugins: %w", err)
				}

				var deletedCount int
				for _, plugin := range plugins {
					// Check if this is a legacy plugin by examining config schema
					islegacy := false

					// Look for old Core Proxy plugin with plugin_uuid schema
					if plugin.Type == "core_proxy" && plugin.Name == "Core Proxy" {
						// Check if config schema contains plugin_uuid (legacy) vs device_mac (real)
						if strings.Contains(plugin.ConfigSchema, "plugin_uuid") &&
							!strings.Contains(plugin.ConfigSchema, "device_mac") {
							islegacy = true
						}
					}

					if islegacy {
						// Check if any user plugins are using this plugin
						var userPluginCount int64
						if err := tx.Model(&UserPlugin{}).Where("plugin_id = ?", plugin.ID).Count(&userPluginCount).Error; err != nil {
							logging.Warn("[MIGRATION] Failed to count user plugins for legacy plugin",
								"plugin_id", plugin.ID, "plugin_name", plugin.Name, "error", err)
							continue
						}

						if userPluginCount > 0 {
							logging.Warn("[MIGRATION] Found legacy plugin with user instances - manual cleanup required",
								"plugin_id", plugin.ID, "plugin_name", plugin.Name, "user_plugin_count", userPluginCount)
							continue
						}

						// Safe to delete - no user plugins attached
						if err := tx.Delete(&plugin).Error; err != nil {
							logging.Warn("[MIGRATION] Failed to delete legacy plugin",
								"plugin_id", plugin.ID, "plugin_name", plugin.Name, "error", err)
							continue
						}

						logging.Info("[MIGRATION] Deleted legacy plugin",
							"plugin_id", plugin.ID, "plugin_name", plugin.Name, "plugin_type", plugin.Type)
						deletedCount++
					}
				}

				if deletedCount > 0 {
					logging.Info("[MIGRATION] legacy plugin cleanup completed", "deleted_count", deletedCount)
				} else {
					logging.Info("[MIGRATION] No legacy plugins found to clean up")
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// Cannot rollback deleted plugins without backup data
				logging.Warn("[MIGRATION] Cannot rollback legacy plugin cleanup - plugins were permanently deleted")
				return nil
			},
		},
	}

	// Create migrator with our migrations
	m := gormigrate.New(DB, gormigrate.DefaultOptions, migrations)

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

	logging.Info("Migrations completed successfully", "component", logPrefix)
	return nil
}
