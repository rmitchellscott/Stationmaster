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
		// Add any future migrations here
		// For now, we rely on AutoMigrate for the initial schema
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