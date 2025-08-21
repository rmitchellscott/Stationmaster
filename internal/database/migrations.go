package database

import (
	"fmt"

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
