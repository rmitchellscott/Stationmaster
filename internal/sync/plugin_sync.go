package sync

import (
	"log"

	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// SyncPluginRegistry synchronizes the plugin registry with the database
// This ensures all plugins registered in code are properly reflected in the database
func SyncPluginRegistry(db *gorm.DB) error {
	pluginService := database.NewPluginService(db)
	
	// Get all plugins from the registry
	registryPlugins := plugins.GetAllInfo()
	
	log.Printf("[PLUGIN_SYNC] Syncing %d plugins from registry to database", len(registryPlugins))
	
	for _, pluginInfo := range registryPlugins {
		err := syncSinglePlugin(pluginService, pluginInfo)
		if err != nil {
			log.Printf("[PLUGIN_SYNC] Failed to sync plugin %s: %v", pluginInfo.Type, err)
			// Continue with other plugins instead of failing entirely
			continue
		}
		log.Printf("[PLUGIN_SYNC] Successfully synced plugin: %s (%s)", pluginInfo.Name, pluginInfo.Type)
	}
	
	log.Printf("[PLUGIN_SYNC] Plugin registry sync completed")
	return nil
}

// syncSinglePlugin synchronizes a single plugin from registry to database
func syncSinglePlugin(pluginService *database.PluginService, pluginInfo plugins.PluginInfo) error {
	// First, try to find existing plugin by type
	existingPlugin, err := pluginService.GetPluginByType(pluginInfo.Type)
	if err != nil {
		// Plugin doesn't exist by type, check if one exists with the same name
		var nameConflictPlugin database.Plugin
		db := database.GetDB()
		nameErr := db.Where("name = ? AND is_active = ?", pluginInfo.Name, true).First(&nameConflictPlugin).Error
		
		if nameErr == nil {
			// Found a plugin with the same name but different type
			log.Printf("[PLUGIN_SYNC] Updating existing plugin '%s' (was type '%s', now type '%s')", 
				pluginInfo.Name, nameConflictPlugin.Type, pluginInfo.Type)
			
			// Update the existing plugin to match the registry (except name)
			nameConflictPlugin.Type = pluginInfo.Type
			nameConflictPlugin.Description = pluginInfo.Description
			nameConflictPlugin.ConfigSchema = pluginInfo.ConfigSchema
			nameConflictPlugin.Version = "1.0.0"
			nameConflictPlugin.Author = "Stationmaster"
			nameConflictPlugin.RequiresProcessing = pluginInfo.RequiresProcessing
			nameConflictPlugin.IsActive = true
			// Keep existing name to avoid conflicts
			
			return pluginService.UpdatePlugin(&nameConflictPlugin)
		}
		
		// No existing plugin found, create new one
		log.Printf("[PLUGIN_SYNC] Creating new plugin: %s (%s)", pluginInfo.Name, pluginInfo.Type)
		_, err := pluginService.CreatePluginWithProcessing(
			pluginInfo.Name,
			pluginInfo.Type,
			pluginInfo.Description,
			pluginInfo.ConfigSchema,
			"1.0.0",
			"Stationmaster",
			pluginInfo.RequiresProcessing,
		)
		return err
	}
	
	// Plugin exists by type, update it to ensure it's current (except name)
	log.Printf("[PLUGIN_SYNC] Updating existing plugin: %s (%s)", pluginInfo.Name, pluginInfo.Type)
	existingPlugin.Description = pluginInfo.Description
	existingPlugin.ConfigSchema = pluginInfo.ConfigSchema
	existingPlugin.RequiresProcessing = pluginInfo.RequiresProcessing
	existingPlugin.IsActive = true
	// Keep existing name to avoid conflicts
	
	return pluginService.UpdatePlugin(existingPlugin)
}

// CleanupOrphanedPlugins removes plugins from database that no longer exist in registry
// This is optional and can be called separately if needed
func CleanupOrphanedPlugins(db *gorm.DB) error {
	pluginService := database.NewPluginService(db)
	
	// Get all active plugins from database
	dbPlugins, err := pluginService.GetAllPlugins()
	if err != nil {
		return err
	}
	
	// Get all plugin types from registry
	registryTypes := plugins.GetAllTypes()
	registryTypeMap := make(map[string]bool)
	for _, pType := range registryTypes {
		registryTypeMap[pType] = true
	}
	
	// Find orphaned plugins
	var orphanedCount int
	for _, dbPlugin := range dbPlugins {
		if !registryTypeMap[dbPlugin.Type] {
			log.Printf("[PLUGIN_SYNC] Found orphaned plugin: %s (%s) - deactivating", dbPlugin.Name, dbPlugin.Type)
			dbPlugin.IsActive = false
			if err := pluginService.UpdatePlugin(&dbPlugin); err != nil {
				log.Printf("[PLUGIN_SYNC] Failed to deactivate orphaned plugin %s: %v", dbPlugin.Type, err)
			} else {
				orphanedCount++
			}
		}
	}
	
	if orphanedCount > 0 {
		log.Printf("[PLUGIN_SYNC] Deactivated %d orphaned plugins", orphanedCount)
	}
	
	return nil
}