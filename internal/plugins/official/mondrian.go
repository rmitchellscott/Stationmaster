package official

import (
	"fmt"
	"os"

	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// MondrianPlugin implements the Mondrian art generator plugin
type MondrianPlugin struct {
	*OfficialPluginAdapter
}

// NewMondrianPlugin creates a new Mondrian plugin instance
func NewMondrianPlugin() (*MondrianPlugin, error) {
	adapter, err := NewOfficialPluginAdapter(
		"mondrian",
		WithDescription("Generate abstract Mondrian-style art compositions"),
		WithConfigSchema(`{
			"type": "object",
			"properties": {},
			"additionalProperties": false
		}`),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Mondrian adapter: %w", err)
	}
	
	return &MondrianPlugin{
		OfficialPluginAdapter: adapter,
	}, nil
}

// Validate validates the plugin settings (no settings for Mondrian)
func (p *MondrianPlugin) Validate(settings map[string]interface{}) error {
	// Mondrian has no configuration options, so nothing to validate
	return nil
}

// Register the Mondrian plugin when this package is imported
func init() {
	// Skip registration during testing
	if len(os.Args) > 0 && (os.Args[0] == "/tmp/go-build" || 
		contains(os.Args, "test") || contains(os.Args, ".test")) {
		return
	}
	
	// We'll register this in a registry function that gets called from main
	registerMondrianPlugin()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func registerMondrianPlugin() {
	mondrianPlugin, err := NewMondrianPlugin()
	if err != nil {
		// Log error but don't panic on startup
		fmt.Printf("Failed to create Mondrian plugin: %v\n", err)
		return
	}
	
	err = plugins.Register(mondrianPlugin)
	if err != nil {
		fmt.Printf("Failed to register Mondrian plugin: %v\n", err)
	}
}