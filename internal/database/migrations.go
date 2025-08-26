package database

import (
	"fmt"
	"time"

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
				logging.Info("[MIGRATION] Skipping private_plugins table creation (legacy code removed)")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] No private_plugins table to drop (legacy code removed)")
				return nil
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
		{
			ID: "20250825_add_screen_options_to_plugin_definitions",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Adding remove_bleed_margin and enable_dark_mode fields to plugin_definitions")
				
				// Add remove_bleed_margin boolean field
				if err := tx.Exec("ALTER TABLE plugin_definitions ADD COLUMN remove_bleed_margin BOOLEAN DEFAULT FALSE").Error; err != nil {
					return fmt.Errorf("failed to add remove_bleed_margin column: %w", err)
				}
				
				// Add enable_dark_mode boolean field
				if err := tx.Exec("ALTER TABLE plugin_definitions ADD COLUMN enable_dark_mode BOOLEAN DEFAULT FALSE").Error; err != nil {
					return fmt.Errorf("failed to add enable_dark_mode column: %w", err)
				}
				
				logging.Info("[MIGRATION] Successfully added screen option fields to plugin_definitions")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Removing screen option fields from plugin_definitions")
				
				if err := tx.Exec("ALTER TABLE plugin_definitions DROP COLUMN IF EXISTS enable_dark_mode").Error; err != nil {
					return fmt.Errorf("failed to drop enable_dark_mode column: %w", err)
				}
				
				if err := tx.Exec("ALTER TABLE plugin_definitions DROP COLUMN IF EXISTS remove_bleed_margin").Error; err != nil {
					return fmt.Errorf("failed to drop remove_bleed_margin column: %w", err)
				}
				
				return nil
			},
		},
		{
			ID: "20250825_add_screen_options_to_private_plugins",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Adding remove_bleed_margin and enable_dark_mode fields to private_plugins")
				
				// Add remove_bleed_margin boolean field
				if err := tx.Exec("ALTER TABLE private_plugins ADD COLUMN remove_bleed_margin BOOLEAN DEFAULT FALSE").Error; err != nil {
					return fmt.Errorf("failed to add remove_bleed_margin column: %w", err)
				}
				
				// Add enable_dark_mode boolean field
				if err := tx.Exec("ALTER TABLE private_plugins ADD COLUMN enable_dark_mode BOOLEAN DEFAULT FALSE").Error; err != nil {
					return fmt.Errorf("failed to add enable_dark_mode column: %w", err)
				}
				
				logging.Info("[MIGRATION] Successfully added screen option fields to private_plugins")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Removing screen option fields from private_plugins")
				
				if err := tx.Exec("ALTER TABLE private_plugins DROP COLUMN IF EXISTS enable_dark_mode").Error; err != nil {
					return fmt.Errorf("failed to drop enable_dark_mode column: %w", err)
				}
				
				if err := tx.Exec("ALTER TABLE private_plugins DROP COLUMN IF EXISTS remove_bleed_margin").Error; err != nil {
					return fmt.Errorf("failed to drop remove_bleed_margin column: %w", err)
				}
				
				return nil
			},
		},
		{
			ID: "20250825_add_independent_render_to_render_queue",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Adding independent_render field to render_queues table")
				
				// Add the new column with default false
				if err := tx.Exec("ALTER TABLE render_queues ADD COLUMN IF NOT EXISTS independent_render BOOLEAN DEFAULT FALSE").Error; err != nil {
					return fmt.Errorf("failed to add independent_render column: %w", err)
				}
				
				logging.Info("[MIGRATION] Successfully added independent_render field")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Removing independent_render field from render_queues")
				
				if err := tx.Exec("ALTER TABLE render_queues DROP COLUMN IF EXISTS independent_render").Error; err != nil {
					return fmt.Errorf("failed to drop independent_render column: %w", err)
				}
				
				return nil
			},
		},
		{
			ID: "20250826_create_webhook_data_table_and_settings",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Creating webhook data table and adding webhook system settings")
				
				// Create webhook data table using the PrivatePluginWebhookData model
				// We need to define the webhook data model inline since it's currently only in handlers
				type PrivatePluginWebhookData struct {
					ID           string                 `json:"id" gorm:"primaryKey"`
					PluginID     string                 `json:"plugin_id" gorm:"index;not null"`
					MergedData   map[string]interface{} `json:"merged_data" gorm:"type:json"`
					RawData      map[string]interface{} `json:"raw_data" gorm:"type:json"`
					MergeStrategy string                `json:"merge_strategy" gorm:"size:20;default:'default'"`
					ReceivedAt   time.Time              `json:"received_at"`
					ContentType  string                 `json:"content_type"`
					ContentSize  int                    `json:"content_size"`
					SourceIP     string                 `json:"source_ip"`
				}
				
				if err := tx.AutoMigrate(&PrivatePluginWebhookData{}); err != nil {
					return fmt.Errorf("failed to create private_plugin_webhook_data table: %w", err)
				}
				
				// Add webhook system settings with defaults
				webhookSettings := []SystemSetting{
					{
						Key:         "webhook_rate_limit_per_hour",
						Value:       "30",
						Description: "Maximum number of webhook requests per user per hour",
						UpdatedAt:   time.Now(),
					},
					{
						Key:         "webhook_max_request_size_kb",
						Value:       "5",
						Description: "Maximum webhook request payload size in KB",
						UpdatedAt:   time.Now(),
					},
				}
				
				for _, setting := range webhookSettings {
					// Use ON CONFLICT DO NOTHING to avoid overwriting existing settings
					if err := tx.Exec(`
						INSERT INTO system_settings (key, value, description, updated_at)
						VALUES (?, ?, ?, ?)
						ON CONFLICT (key) DO NOTHING
					`, setting.Key, setting.Value, setting.Description, setting.UpdatedAt).Error; err != nil {
						return fmt.Errorf("failed to insert webhook setting %s: %w", setting.Key, err)
					}
				}
				
				logging.Info("[MIGRATION] Successfully created webhook data table and settings")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Rolling back webhook data table and settings")
				
				// Drop webhook data table
				if err := tx.Migrator().DropTable("private_plugin_webhook_data"); err != nil {
					return fmt.Errorf("failed to drop webhook data table: %w", err)
				}
				
				// Remove webhook system settings
				webhookSettingKeys := []string{
					"webhook_rate_limit_per_hour",
					"webhook_max_request_size_kb",
				}
				
				for _, key := range webhookSettingKeys {
					if err := tx.Exec("DELETE FROM system_settings WHERE key = ?", key).Error; err != nil {
						logging.Warn("[MIGRATION] Failed to remove webhook setting", "key", key, "error", err)
					}
				}
				
				return nil
			},
		},
		{
			ID: "20250826_migrate_private_plugins_to_instances",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Migrating private plugins to plugin instances")
				
				// Remove webhook_token column from plugin_instances if it exists (from earlier migration attempts)
				tx.Exec("ALTER TABLE plugin_instances DROP COLUMN IF EXISTS webhook_token")
				tx.Exec("DROP INDEX IF EXISTS idx_plugin_instances_webhook_token")
				
				// Find all private plugins (including those that may have had webhook tokens)
				var privatePlugins []struct {
					ID     string `gorm:"column:id"`
					Name   string `gorm:"column:name"`
					UserID string `gorm:"column:user_id"`
				}
				
				err := tx.Raw(`
					SELECT id, name, user_id 
					FROM private_plugins
				`).Scan(&privatePlugins).Error
				
				if err != nil {
					return fmt.Errorf("failed to query private plugins: %w", err)
				}
				
				logging.Info("[MIGRATION] Found private plugins", "count", len(privatePlugins))
				
				// For each private plugin, ensure there's a corresponding plugin definition and instance
				for _, pp := range privatePlugins {
					// Check if there's already a plugin definition for this private plugin
					var existingDefCount int64
					if err := tx.Raw(`
						SELECT COUNT(*) FROM plugin_definitions 
						WHERE plugin_type = 'private' AND identifier = ?
					`, pp.ID).Scan(&existingDefCount).Error; err != nil {
						logging.Warn("[MIGRATION] Failed to check existing plugin definition", "private_plugin_id", pp.ID, "error", err)
						continue
					}
					
					var definitionID string
					if existingDefCount == 0 {
						// Create plugin definition for this private plugin
						definitionID = pp.ID // Use same UUID
						err := tx.Exec(`
							INSERT INTO plugin_definitions (id, plugin_type, owner_id, identifier, name, description, version, author, requires_processing, created_at, updated_at)
							VALUES (?, 'private', ?, ?, ?, 'Migrated private plugin', '1.0.0', 'System Migration', true, NOW(), NOW())
						`, definitionID, pp.UserID, pp.ID, pp.Name).Error
						
						if err != nil {
							logging.Warn("[MIGRATION] Failed to create plugin definition", "private_plugin_id", pp.ID, "error", err)
							continue
						}
						
						logging.Info("[MIGRATION] Created plugin definition", "definition_id", definitionID, "private_plugin_id", pp.ID)
					} else {
						// Use existing definition ID
						if err := tx.Raw(`
							SELECT id FROM plugin_definitions 
							WHERE plugin_type = 'private' AND identifier = ?
						`, pp.ID).Scan(&definitionID).Error; err != nil {
							logging.Warn("[MIGRATION] Failed to get existing plugin definition ID", "private_plugin_id", pp.ID, "error", err)
							continue
						}
					}
					
					// Check if there's already a plugin instance for this definition
					var existingInstanceCount int64
					if err := tx.Raw(`
						SELECT COUNT(*) FROM plugin_instances 
						WHERE plugin_definition_id = ? AND user_id = ?
					`, definitionID, pp.UserID).Scan(&existingInstanceCount).Error; err != nil {
						logging.Warn("[MIGRATION] Failed to check existing plugin instance", "definition_id", definitionID, "error", err)
						continue
					}
					
					if existingInstanceCount == 0 {
						// Create plugin instance for this private plugin
						err := tx.Exec(`
							INSERT INTO plugin_instances (id, user_id, plugin_definition_id, name, settings, created_at, updated_at)
							VALUES (gen_random_uuid(), ?, ?, ?, '{}', NOW(), NOW())
						`, pp.UserID, definitionID, pp.Name).Error
						
						if err != nil {
							logging.Warn("[MIGRATION] Failed to create plugin instance", "definition_id", definitionID, "error", err)
							continue
						}
						
						logging.Info("[MIGRATION] Created plugin instance", "definition_id", definitionID, "name", pp.Name)
					}
				}
				
				logging.Info("[MIGRATION] Successfully migrated private plugins to plugin instances")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Rolling back private plugin migration")
				
				// This rollback removes plugin instances created by this migration
				// Be careful - this could remove user data
				logging.Warn("[MIGRATION] Rollback would remove plugin instances - skipping for data safety")
				
				return nil
			},
		},
		{
			ID: "20250826_fix_webhook_data_table_column_name",
			Migrate: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Fixing webhook data table column name from plugin_id to plugin_instance_id")
				
				// Check if the table exists first
				if !tx.Migrator().HasTable("private_plugin_webhook_data") {
					logging.Info("[MIGRATION] private_plugin_webhook_data table does not exist, skipping")
					return nil
				}
				
				// Check for both columns to handle different migration states
				hasPluginId := tx.Migrator().HasColumn("private_plugin_webhook_data", "plugin_id")
				hasPluginInstanceId := tx.Migrator().HasColumn("private_plugin_webhook_data", "plugin_instance_id")
				
				if hasPluginId && hasPluginInstanceId {
					// Both columns exist - drop the old plugin_id column
					logging.Info("[MIGRATION] Both plugin_id and plugin_instance_id columns exist, dropping plugin_id")
					if err := tx.Exec("ALTER TABLE private_plugin_webhook_data DROP COLUMN plugin_id").Error; err != nil {
						return fmt.Errorf("failed to drop plugin_id column: %w", err)
					}
					logging.Info("[MIGRATION] Successfully dropped plugin_id column")
				} else if hasPluginId && !hasPluginInstanceId {
					// Only plugin_id exists - rename it
					logging.Info("[MIGRATION] Only plugin_id column exists, renaming to plugin_instance_id")
					if err := tx.Exec("ALTER TABLE private_plugin_webhook_data RENAME COLUMN plugin_id TO plugin_instance_id").Error; err != nil {
						return fmt.Errorf("failed to rename plugin_id column to plugin_instance_id: %w", err)
					}
					logging.Info("[MIGRATION] Successfully renamed plugin_id column to plugin_instance_id")
				} else if !hasPluginId && hasPluginInstanceId {
					// Only plugin_instance_id exists - already migrated
					logging.Info("[MIGRATION] Only plugin_instance_id column exists, migration already completed")
				} else {
					// Neither column exists - unexpected state
					logging.Warn("[MIGRATION] Neither plugin_id nor plugin_instance_id column found in private_plugin_webhook_data table")
				}
				
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				logging.Info("[MIGRATION] Rolling back webhook data table column name change")
				
				// Check if table exists
				if !tx.Migrator().HasTable("private_plugin_webhook_data") {
					logging.Info("[MIGRATION] private_plugin_webhook_data table does not exist, skipping rollback")
					return nil
				}
				
				// Check current state
				hasPluginId := tx.Migrator().HasColumn("private_plugin_webhook_data", "plugin_id")
				hasPluginInstanceId := tx.Migrator().HasColumn("private_plugin_webhook_data", "plugin_instance_id")
				
				if !hasPluginId && hasPluginInstanceId {
					// Only plugin_instance_id exists - rename it back to plugin_id
					logging.Info("[MIGRATION] Renaming plugin_instance_id back to plugin_id")
					if err := tx.Exec("ALTER TABLE private_plugin_webhook_data RENAME COLUMN plugin_instance_id TO plugin_id").Error; err != nil {
						return fmt.Errorf("failed to rename plugin_instance_id column back to plugin_id: %w", err)
					}
				} else {
					logging.Info("[MIGRATION] Column state doesn't require rollback action")
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
