package bootstrap

import (
	"fmt"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"gorm.io/gorm"
)

// BootstrapSystemPlugins syncs all registered system plugins from the registry into the database
func BootstrapSystemPlugins(db *gorm.DB) error {
	unifiedPluginService := database.NewUnifiedPluginService(db)
	
	// Get all registered system plugins from the registry
	allPlugins := plugins.GetAllInfo()
	
	for _, pluginInfo := range allPlugins {
		// Create or update the plugin definition in the database
		_, err := unifiedPluginService.CreateSystemPluginDefinition(
			pluginInfo.Type,                  // identifier
			pluginInfo.Name,                  // name
			pluginInfo.Description,           // description
			pluginInfo.ConfigSchema,          // config schema
			pluginInfo.Version,               // version
			pluginInfo.Author,                // author
			pluginInfo.RequiresProcessing,    // requires processing
		)
		
		if err != nil {
			return fmt.Errorf("failed to bootstrap system plugin %s: %w", pluginInfo.Type, err)
		}
	}
	
	return nil
}