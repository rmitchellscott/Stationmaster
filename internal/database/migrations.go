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
		/*{
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
		},*/
		{
			ID: "20250822_create_private_plugins",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Creating private_plugins table")

				// Create the PrivatePlugin table
				if err := tx.AutoMigrate(&PrivatePlugin{}); err != nil {
					return fmt.Errorf("failed to create private_plugins table: %w", err)
				}

				logging.Info("[MIGRATION] private_plugins table created successfully")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Dropping private_plugins table")
				return tx.Migrator().DropTable(&PrivatePlugin{})
			},
		},
		{
			ID: "20250823_cleanup_legacy_plugin_system",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Cleaning up legacy plugin system")

				// Drop legacy tables if they exist
				legacyTables := []string{"user_plugins", "plugins"}
				for _, tableName := range legacyTables {
					if tx.Migrator().HasTable(tableName) {
						logging.Info("[MIGRATION] Dropping legacy table", "table", tableName)
						if err := tx.Migrator().DropTable(tableName); err != nil {
							logging.Warn("[MIGRATION] Failed to drop legacy table", "table", tableName, "error", err)
							// Don't fail the migration if we can't drop legacy tables
							// They might have constraints we need to handle manually
						}
					}
				}

				// Clean up any orphaned render queue entries
				var orphanedCount int64
				result := tx.Exec(`DELETE FROM render_queues 
					WHERE plugin_instance_id NOT IN (
						SELECT id FROM plugin_instances WHERE is_active = true
					)`)
				
				if result.Error != nil {
					logging.Warn("[MIGRATION] Failed to clean orphaned render queue entries", "error", result.Error)
				} else {
					orphanedCount = result.RowsAffected
					if orphanedCount > 0 {
						logging.Info("[MIGRATION] Cleaned orphaned render queue entries", "count", orphanedCount)
					}
				}

				// Clean up any orphaned rendered content
				result = tx.Exec(`DELETE FROM rendered_contents 
					WHERE plugin_instance_id NOT IN (
						SELECT id FROM plugin_instances WHERE is_active = true
					)`)
				
				if result.Error != nil {
					logging.Warn("[MIGRATION] Failed to clean orphaned rendered content", "error", result.Error)
				} else {
					orphanedCount = result.RowsAffected
					if orphanedCount > 0 {
						logging.Info("[MIGRATION] Cleaned orphaned rendered content", "count", orphanedCount)
					}
				}

				// Clean up any orphaned playlist items
				result = tx.Exec(`DELETE FROM playlist_items 
					WHERE plugin_instance_id NOT IN (
						SELECT id FROM plugin_instances WHERE is_active = true
					)`)
				
				if result.Error != nil {
					logging.Warn("[MIGRATION] Failed to clean orphaned playlist items", "error", result.Error)
				} else {
					orphanedCount = result.RowsAffected
					if orphanedCount > 0 {
						logging.Info("[MIGRATION] Cleaned orphaned playlist items", "count", orphanedCount)
					}
				}

				logging.Info("[MIGRATION] Legacy plugin system cleanup completed")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Warn("[MIGRATION] Cannot rollback legacy plugin system cleanup - data was permanently deleted")
				return nil
			},
		},
		{
			ID: "20250823_add_foreign_key_constraints",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Adding foreign key constraints to prevent orphaned data")

				// Add NOT NULL constraint to plugin_instance_id where it should be required
				constraints := []struct {
					table  string
					column string
				}{
					{"render_queues", "plugin_instance_id"},
					{"rendered_contents", "plugin_instance_id"},
				}

				for _, constraint := range constraints {
					// Check if column allows NULL values
					var columnInfo struct {
						IsNullable string `gorm:"column:is_nullable"`
					}
					err := tx.Raw(`
						SELECT is_nullable 
						FROM information_schema.columns 
						WHERE table_name = ? AND column_name = ? AND table_schema = 'public'
					`, constraint.table, constraint.column).Scan(&columnInfo).Error
					
					if err != nil {
						logging.Warn("[MIGRATION] Failed to check column nullability", "table", constraint.table, "column", constraint.column, "error", err)
						continue
					}

					if columnInfo.IsNullable == "YES" {
						logging.Info("[MIGRATION] Adding NOT NULL constraint", "table", constraint.table, "column", constraint.column)
						err = tx.Exec(fmt.Sprintf(`
							ALTER TABLE %s 
							ALTER COLUMN %s SET NOT NULL
						`, constraint.table, constraint.column)).Error
						
						if err != nil {
							logging.Warn("[MIGRATION] Failed to add NOT NULL constraint", "table", constraint.table, "column", constraint.column, "error", err)
							// Don't fail the migration, just log the warning
						}
					}
				}

				logging.Info("[MIGRATION] Foreign key constraints migration completed")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Removing NOT NULL constraints")
				
				constraints := []struct {
					table  string
					column string
				}{
					{"render_queues", "plugin_instance_id"},
					{"rendered_contents", "plugin_instance_id"},
				}

				for _, constraint := range constraints {
					err := tx.Exec(fmt.Sprintf(`
						ALTER TABLE %s 
						ALTER COLUMN %s DROP NOT NULL
					`, constraint.table, constraint.column)).Error
					
					if err != nil {
						logging.Warn("[MIGRATION] Failed to drop NOT NULL constraint", "table", constraint.table, "column", constraint.column, "error", err)
					}
				}
				
				return nil
			},
		},
		{
			ID: "20250824_add_playlist_items_composite_index",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Adding composite index to playlist_items for better ORDER BY performance")
				
				// Create composite index for playlist_id and order_index
				indexSQL := "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_playlist_items_playlist_id_order_index ON playlist_items(playlist_id, order_index)"
				if err := tx.Exec(indexSQL).Error; err != nil {
					return fmt.Errorf("failed to create composite index: %w", err)
				}
				
				logging.Info("[MIGRATION] Composite index idx_playlist_items_playlist_id_order_index created successfully")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Dropping composite index from playlist_items")
				dropSQL := "DROP INDEX CONCURRENTLY IF EXISTS idx_playlist_items_playlist_id_order_index"
				return tx.Exec(dropSQL).Error
			},
		},
		{
			ID: "20250824_replace_playlist_index_with_item_id",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Replacing last_playlist_index with last_playlist_item_id for stable playlist tracking")
				
				// Add new UUID column
				if err := tx.Exec("ALTER TABLE devices ADD COLUMN last_playlist_item_id UUID REFERENCES playlist_items(id)").Error; err != nil {
					return fmt.Errorf("failed to add last_playlist_item_id column: %w", err)
				}
				
				// Convert existing indices to item UUIDs for devices that have playlists
				convertSQL := `
					UPDATE devices 
					SET last_playlist_item_id = (
						SELECT pi.id
						FROM playlist_items pi
						JOIN playlists p ON pi.playlist_id = p.id
						WHERE p.device_id = devices.id 
						  AND p.is_default = true 
						  AND pi.is_visible = true
						ORDER BY pi.order_index
						LIMIT 1 OFFSET devices.last_playlist_index
					)
					WHERE devices.last_playlist_index >= 0
					  AND EXISTS (
						  SELECT 1 FROM playlists 
						  WHERE device_id = devices.id AND is_default = true
					  )
				`
				
				if err := tx.Exec(convertSQL).Error; err != nil {
					logging.Warn("[MIGRATION] Failed to convert some playlist indices to UUIDs", "error", err)
					// Don't fail migration - some devices might not have playlists yet
				}
				
				// For devices without valid conversion, set to first available playlist item
				fallbackSQL := `
					UPDATE devices 
					SET last_playlist_item_id = (
						SELECT pi.id
						FROM playlist_items pi
						JOIN playlists p ON pi.playlist_id = p.id
						WHERE p.device_id = devices.id 
						  AND p.is_default = true 
						  AND pi.is_visible = true
						ORDER BY pi.order_index
						LIMIT 1
					)
					WHERE devices.last_playlist_item_id IS NULL
					  AND EXISTS (
						  SELECT 1 FROM playlists 
						  WHERE device_id = devices.id AND is_default = true
					  )
				`
				
				if err := tx.Exec(fallbackSQL).Error; err != nil {
					logging.Warn("[MIGRATION] Failed to set fallback playlist item IDs", "error", err)
				}
				
				// Drop the old index column
				if err := tx.Exec("ALTER TABLE devices DROP COLUMN last_playlist_index").Error; err != nil {
					return fmt.Errorf("failed to drop last_playlist_index column: %w", err)
				}
				
				logging.Info("[MIGRATION] Successfully replaced last_playlist_index with last_playlist_item_id")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Rolling back playlist index change")
				
				// Add back the old column
				if err := tx.Exec("ALTER TABLE devices ADD COLUMN last_playlist_index INT DEFAULT 0").Error; err != nil {
					return fmt.Errorf("failed to add back last_playlist_index column: %w", err)
				}
				
				// Convert UUIDs back to indices (best effort)
				convertBackSQL := `
					UPDATE devices 
					SET last_playlist_index = COALESCE((
						SELECT pi.order_index - 1
						FROM playlist_items pi
						WHERE pi.id = devices.last_playlist_item_id
					), 0)
					WHERE devices.last_playlist_item_id IS NOT NULL
				`
				
				if err := tx.Exec(convertBackSQL).Error; err != nil {
					logging.Warn("[MIGRATION] Failed to convert UUIDs back to indices", "error", err)
				}
				
				// Drop the UUID column
				if err := tx.Exec("ALTER TABLE devices DROP COLUMN last_playlist_item_id").Error; err != nil {
					return fmt.Errorf("failed to drop last_playlist_item_id column: %w", err)
				}
				
				return nil
			},
		},
		{
			ID: "20250824_rename_last_html_hash_to_last_image_hash",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Renaming last_html_hash to last_image_hash in plugin_instances table")
				
				// Check if the old column exists
				if tx.Migrator().HasColumn(&PluginInstance{}, "last_html_hash") {
					// Add the new column
					if err := tx.Exec("ALTER TABLE plugin_instances ADD COLUMN last_image_hash VARCHAR(64)").Error; err != nil {
						return fmt.Errorf("failed to add last_image_hash column: %w", err)
					}
					
					// Copy data from old column to new column
					if err := tx.Exec("UPDATE plugin_instances SET last_image_hash = last_html_hash WHERE last_html_hash IS NOT NULL").Error; err != nil {
						return fmt.Errorf("failed to copy data from last_html_hash to last_image_hash: %w", err)
					}
					
					// Drop the old column
					if err := tx.Exec("ALTER TABLE plugin_instances DROP COLUMN last_html_hash").Error; err != nil {
						return fmt.Errorf("failed to drop last_html_hash column: %w", err)
					}
					
					logging.Info("[MIGRATION] Successfully renamed last_html_hash to last_image_hash")
				} else {
					logging.Info("[MIGRATION] last_html_hash column not found, assuming already migrated")
				}
				
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Rolling back last_html_hash to last_image_hash rename")
				
				// Check if the new column exists
				if tx.Migrator().HasColumn(&PluginInstance{}, "last_image_hash") {
					// Add the old column back
					if err := tx.Exec("ALTER TABLE plugin_instances ADD COLUMN last_html_hash VARCHAR(64)").Error; err != nil {
						return fmt.Errorf("failed to add back last_html_hash column: %w", err)
					}
					
					// Copy data back
					if err := tx.Exec("UPDATE plugin_instances SET last_html_hash = last_image_hash WHERE last_image_hash IS NOT NULL").Error; err != nil {
						return fmt.Errorf("failed to copy data back from last_image_hash to last_html_hash: %w", err)
					}
					
					// Drop the new column
					if err := tx.Exec("ALTER TABLE plugin_instances DROP COLUMN last_image_hash").Error; err != nil {
						return fmt.Errorf("failed to drop last_image_hash column: %w", err)
					}
				}
				
				return nil
			},
		},
		{
			ID: "20250825_add_device_id_to_rendered_contents",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Adding device_id column to rendered_contents for per-device rendering")
				
				// Check if column already exists
				if tx.Migrator().HasColumn(&RenderedContent{}, "device_id") {
					logging.Info("[MIGRATION] device_id column already exists in rendered_contents")
					return nil
				}
				
				// Add the nullable device_id column with foreign key constraint
				if err := tx.Exec("ALTER TABLE rendered_contents ADD COLUMN device_id UUID REFERENCES devices(id)").Error; err != nil {
					return fmt.Errorf("failed to add device_id column to rendered_contents: %w", err)
				}
				
				// Add index for device_id lookups
				if err := tx.Exec("CREATE INDEX idx_rendered_contents_device_id ON rendered_contents(device_id)").Error; err != nil {
					return fmt.Errorf("failed to create index on device_id: %w", err)
				}
				
				// Add composite index for device_id + plugin_instance_id lookups (most common query)
				if err := tx.Exec("CREATE INDEX idx_rendered_contents_device_plugin ON rendered_contents(device_id, plugin_instance_id)").Error; err != nil {
					return fmt.Errorf("failed to create composite index on device_id and plugin_instance_id: %w", err)
				}
				
				logging.Info("[MIGRATION] Successfully added device_id column and indexes to rendered_contents")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Rolling back device_id column addition from rendered_contents")
				
				// Drop indexes first
				if err := tx.Exec("DROP INDEX IF EXISTS idx_rendered_contents_device_plugin").Error; err != nil {
					logging.Warn("[MIGRATION] Failed to drop composite index", "error", err)
				}
				
				if err := tx.Exec("DROP INDEX IF EXISTS idx_rendered_contents_device_id").Error; err != nil {
					logging.Warn("[MIGRATION] Failed to drop device_id index", "error", err)
				}
				
				// Drop the column
				if err := tx.Exec("ALTER TABLE rendered_contents DROP COLUMN device_id").Error; err != nil {
					return fmt.Errorf("failed to drop device_id column: %w", err)
				}
				
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
