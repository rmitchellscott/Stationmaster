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
		{
			ID: "202501180000_add_manual_model_fields",
			Migrate: func(tx *gorm.DB) error {
				// Check if manual_model_override column already exists
				if tx.Migrator().HasColumn(&Device{}, "manual_model_override") {
					return nil // Columns already exist, skip
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

				// Add the manual_model_override column with default value of false
				if err := tx.Exec("ALTER TABLE devices ADD COLUMN manual_model_override BOOLEAN DEFAULT FALSE").Error; err != nil {
					return fmt.Errorf("failed to add manual_model_override column: %w", err)
				}

				// Add the reported_model_name column
				if err := tx.Exec("ALTER TABLE devices ADD COLUMN reported_model_name VARCHAR(100)").Error; err != nil {
					return fmt.Errorf("failed to add reported_model_name column: %w", err)
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// SQLite doesn't support dropping columns easily, so we'll leave them
				return nil
			},
		},
		{
			ID: "202501180001_add_device_models_table_and_samples",
			Migrate: func(tx *gorm.DB) error {
				// Check if device_models table already exists
				if tx.Migrator().HasTable("device_models") {
					// Table exists, check if we need to add sample data
					var count int64
					if err := tx.Table("device_models").Count(&count).Error; err != nil {
						return fmt.Errorf("failed to count device models: %w", err)
					}
					
					if count > 0 {
						return nil // Table exists with data, skip
					}
				} else {
					// Create the device_models table manually to avoid foreign key constraints
					createTableSQL := `
						CREATE TABLE device_models (
							id TEXT PRIMARY KEY,
							model_name TEXT NOT NULL UNIQUE,
							display_name TEXT NOT NULL,
							description TEXT,
							screen_width INTEGER NOT NULL,
							screen_height INTEGER NOT NULL,
							color_depth INTEGER DEFAULT 1,
							bit_depth INTEGER DEFAULT 1,
							has_wi_fi BOOLEAN DEFAULT TRUE,
							has_battery BOOLEAN DEFAULT TRUE,
							has_buttons INTEGER DEFAULT 0,
							capabilities TEXT,
							min_firmware TEXT,
							is_active BOOLEAN DEFAULT TRUE,
							created_at DATETIME,
							updated_at DATETIME
						)
					`
					
					if err := tx.Exec(createTableSQL).Error; err != nil {
						return fmt.Errorf("failed to create device_models table: %w", err)
					}
				}

				// Insert sample device models
				sampleModels := []map[string]interface{}{
					{
						"id":           "01234567-89ab-cdef-0123-456789abcdef",
						"model_name":   "og_plus",
						"display_name": "TRMNL Original",
						"description":  "The original TRMNL e-ink display device",
						"screen_width": 800,
						"screen_height": 480,
						"color_depth":  1,
						"bit_depth":    1,
						"has_wi_fi":    true,
						"has_battery":  true,
						"has_buttons":  0,
						"capabilities": `["display", "wifi", "battery"]`,
						"is_active":    true,
						"created_at":   "2025-01-18T00:00:00Z",
						"updated_at":   "2025-01-18T00:00:00Z",
					},
					{
						"id":           "01234567-89ab-cdef-0123-456789abcde0",
						"model_name":   "v2",
						"display_name": "TRMNL v2",
						"description":  "Second generation TRMNL device with improved display",
						"screen_width": 800,
						"screen_height": 480,
						"color_depth":  8,
						"bit_depth":    4,
						"has_wi_fi":    true,
						"has_battery":  true,
						"has_buttons":  0,
						"capabilities": `["display", "wifi", "battery", "grayscale"]`,
						"is_active":    true,
						"created_at":   "2025-01-18T00:00:00Z",
						"updated_at":   "2025-01-18T00:00:00Z",
					},
				}

				for _, model := range sampleModels {
					if err := tx.Table("device_models").Create(model).Error; err != nil {
						return fmt.Errorf("failed to create device model %s: %w", model["model_name"], err)
					}
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Exec("DELETE FROM device_models WHERE model_name IN ('og_plus', 'v2')").Error
			},
		},
		{
			ID: "202501190000_add_device_mirroring_fields",
			Migrate: func(tx *gorm.DB) error {
				// Check if is_sharable column already exists
				if tx.Migrator().HasColumn(&Device{}, "is_sharable") {
					return nil // Columns already exist, skip
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

				// Add the is_sharable column with default value of false
				if err := tx.Exec("ALTER TABLE devices ADD COLUMN is_sharable BOOLEAN DEFAULT FALSE").Error; err != nil {
					return fmt.Errorf("failed to add is_sharable column: %w", err)
				}

				// Add the mirror_source_id column
				if err := tx.Exec("ALTER TABLE devices ADD COLUMN mirror_source_id TEXT").Error; err != nil {
					return fmt.Errorf("failed to add mirror_source_id column: %w", err)
				}

				// Add the mirror_synced_at column
				if err := tx.Exec("ALTER TABLE devices ADD COLUMN mirror_synced_at DATETIME").Error; err != nil {
					return fmt.Errorf("failed to add mirror_synced_at column: %w", err)
				}

				// Add index on mirror_source_id
				if err := tx.Exec("CREATE INDEX idx_devices_mirror_source_id ON devices(mirror_source_id)").Error; err != nil {
					return fmt.Errorf("failed to create index on mirror_source_id: %w", err)
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// SQLite doesn't support dropping columns easily, so we'll leave them
				return nil
			},
		},
		{
			ID: "202508190000_consolidate_device_playlists",
			Migrate: func(tx *gorm.DB) error {
				logging.Logf("[MIGRATION] Consolidating device playlists to ensure one per device")
				
				// Create a playlist service to use the consolidation function
				playlistService := NewPlaylistService(tx)
				
				// Run the consolidation
				if err := playlistService.ConsolidateDevicePlaylists(); err != nil {
					return fmt.Errorf("failed to consolidate device playlists: %w", err)
				}
				
				// Add a partial unique index for SQLite to prevent multiple default playlists per device
				// This creates a unique constraint only for records where is_default = true
				if err := tx.Exec(`
					CREATE UNIQUE INDEX IF NOT EXISTS idx_device_default_playlist 
					ON playlists(device_id) 
					WHERE is_default = 1
				`).Error; err != nil {
					return fmt.Errorf("failed to create unique index for default playlists: %w", err)
				}
				
				logging.Logf("[MIGRATION] Successfully consolidated playlists and added unique constraint")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// Drop the unique index
				return tx.Exec("DROP INDEX IF EXISTS idx_device_default_playlist").Error
			},
		},
		{
			ID: "202508190001_rename_is_sharable_to_is_shareable",
			Migrate: func(tx *gorm.DB) error {
				logging.Logf("[MIGRATION] Renaming is_sharable column to is_shareable")
				
				// Check if the old column exists and new column doesn't
				hasOldColumn := tx.Migrator().HasColumn(&Device{}, "is_sharable")
				hasNewColumn := tx.Migrator().HasColumn(&Device{}, "is_shareable")
				
				if !hasOldColumn && hasNewColumn {
					// Migration already completed
					logging.Logf("[MIGRATION] Column already renamed, skipping")
					return nil
				}
				
				if !hasOldColumn {
					// Neither column exists, create the new one
					if err := tx.Exec("ALTER TABLE devices ADD COLUMN is_shareable BOOLEAN DEFAULT FALSE").Error; err != nil {
						return fmt.Errorf("failed to add is_shareable column: %w", err)
					}
					logging.Logf("[MIGRATION] Created new is_shareable column")
					return nil
				}
				
				// For SQLite, we need to handle this carefully since it doesn't support column renames directly
				var dbType string
				if err := tx.Raw("SELECT sqlite_version()").Scan(&dbType); err == nil {
					// SQLite - we need to:
					// 1. Add new column
					// 2. Copy data
					// 3. Drop old column (SQLite doesn't support this easily, so we'll leave it)
					
					// Add new column
					if err := tx.Exec("ALTER TABLE devices ADD COLUMN is_shareable BOOLEAN DEFAULT FALSE").Error; err != nil {
						return fmt.Errorf("failed to add is_shareable column: %w", err)
					}
					
					// Copy data from old column to new column
					if err := tx.Exec("UPDATE devices SET is_shareable = is_sharable").Error; err != nil {
						return fmt.Errorf("failed to copy data to is_shareable column: %w", err)
					}
					
					logging.Logf("[MIGRATION] Successfully added is_shareable column and copied data")
				} else {
					// For other databases that support column rename
					if err := tx.Exec("ALTER TABLE devices RENAME COLUMN is_sharable TO is_shareable").Error; err != nil {
						return fmt.Errorf("failed to rename column: %w", err)
					}
					logging.Logf("[MIGRATION] Successfully renamed column is_sharable to is_shareable")
				}
				
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// For rollback, we would need to rename back, but this is complex with SQLite
				// Since this is a cosmetic change (correcting spelling), we'll leave the new column
				logging.Logf("[MIGRATION ROLLBACK] Leaving is_shareable column (SQLite limitations)")
				return nil
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
