package logging

// Component constants for structured logging
// These replace the hardcoded bracketed prefixes like [STARTUP], [MODEL POLLER], etc.
const (
	ComponentStartup        = "startup"
	ComponentModelPoller    = "model-poller"
	ComponentFirmwarePoller = "firmware-poller"
	ComponentAPISetup       = "api-setup"
	ComponentAPIDisplay     = "api-display"
	ComponentAPILogs        = "api-logs"
	ComponentAPIScreen      = "api-current-screen"
	ComponentDatabase       = "database"
	ComponentAuth           = "auth"
	ComponentOIDC           = "oidc"
	ComponentSync           = "sync"
	ComponentRenderer       = "renderer"
	ComponentPlugins        = "plugins"
	ComponentRestore        = "restore"
	ComponentBackup         = "backup"
	ComponentExport         = "export"
	ComponentImport         = "import"
	ComponentPlaylist       = "playlist"
)