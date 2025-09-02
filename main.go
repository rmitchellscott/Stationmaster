package main

import (
	// standard library
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	// third-party
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	// internal
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/bootstrap"
	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/handlers"
	"github.com/rmitchellscott/stationmaster/internal/locales"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/middleware"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	_ "github.com/rmitchellscott/stationmaster/internal/plugins/private" // Register private plugin factory
	_ "github.com/rmitchellscott/stationmaster/internal/plugins/mashup"  // Register mashup plugin factory
	"github.com/rmitchellscott/stationmaster/internal/pollers"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
	"github.com/rmitchellscott/stationmaster/internal/sse"
	"github.com/rmitchellscott/stationmaster/internal/trmnl"

	"github.com/rmitchellscott/stationmaster/internal/version"

	// Plugin imports for auto-registration
	_ "github.com/rmitchellscott/stationmaster/internal/plugins/alias"
	_ "github.com/rmitchellscott/stationmaster/internal/plugins/core_proxy"
	_ "github.com/rmitchellscott/stationmaster/internal/plugins/external"  // Register external plugin factory
	_ "github.com/rmitchellscott/stationmaster/internal/plugins/image_display"
	_ "github.com/rmitchellscott/stationmaster/internal/plugins/redirect"
	_ "github.com/rmitchellscott/stationmaster/internal/plugins/screenshot"
)

//go:generate npm --prefix ui install
//go:generate npm --prefix ui run build
//go:embed ui/dist
//go:embed ui/dist/assets
var embeddedUI embed.FS

//go:embed assets/trmnl/*
var embeddedTRNMLAssets embed.FS


// Global render poller for handlers to schedule renders
var globalRenderPoller *pollers.RenderPoller

// ScheduleRender schedules an immediate render for plugin instances using the global render poller
func ScheduleRender(pluginInstanceIDs []uuid.UUID) {
	if globalRenderPoller == nil {
		logging.Warn("[RENDER] Global render poller not available")
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	for _, instanceID := range pluginInstanceIDs {
		if err := globalRenderPoller.ScheduleImmediateRender(ctx, instanceID.String()); err != nil {
			logging.Error("[RENDER] Failed to schedule immediate render", "instance_id", instanceID, "error", err)
		} else {
			logging.Info("[RENDER] Scheduled immediate render", "instance_id", instanceID)
		}
	}
}

// Initialize the global render scheduler function for handlers
func init() {
	handlers.SetRenderScheduler(ScheduleRender)
}

func main() {
	_ = godotenv.Load()
	logging.InfoWithComponent(logging.ComponentStartup, "Starting Stationmaster", "version", version.String())

	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version.String())
		os.Exit(0)
	}

	// Initialize database (always in multi-user mode)
	if err := database.Initialize(); err != nil {
		logging.ErrorWithComponent(logging.ComponentStartup, "Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := database.MigrateToMultiUser(); err != nil {
		logging.ErrorWithComponent(logging.ComponentStartup, "Failed to setup initial user", "error", err)
		os.Exit(1)
	}

	// Bootstrap system plugins into the database
	db := database.GetDB()
	if err := bootstrap.BootstrapSystemPlugins(db); err != nil {
		logging.ErrorWithComponent(logging.ComponentStartup, "Failed to bootstrap system plugins", "error", err)
		os.Exit(1)
	}
	logging.InfoWithComponent(logging.ComponentStartup, "System plugins bootstrapped successfully")

	// Initialize OIDC if configured
	if err := auth.InitOIDC(); err != nil {
		logging.Error("[STARTUP] Failed to initialize OIDC", "error", err)
		os.Exit(1)
	}

	// Initialize proxy auth if configured
	auth.InitProxyAuth()

	// Initialize SSE service
	sse.InitializeSSEService()

	// Initialize and start background pollers
	pollerManager := pollers.NewManager()

	// Register pollers (reuse db from bootstrap)
	
	// Initialize plugin factory
	plugins.InitPluginFactory(db)
	
	// Initialize external plugin scanner
	pluginScanner := plugins.NewPluginScannerService(db)
	
	// Perform initial plugin scan
	scanCtx, scanCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if err := pluginScanner.ScanAndRegisterPlugins(scanCtx); err != nil {
		logging.WarnWithComponent(logging.ComponentStartup, "Initial external plugin scan failed", "error", err)
	}
	scanCancel()
	
	// Start periodic plugin scanning (every 5 minutes)
	pluginScanner.StartPeriodicScanning(5 * time.Minute)
	
	// Initialize plugin processor with database
	if err := trmnl.InitPluginProcessor(db); err != nil {
		logging.Error("[STARTUP] Failed to initialize plugin processor", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := trmnl.CleanupPluginProcessor(); err != nil {
			logging.Error("Failed to cleanup plugin processor", "error", err)
		}
	}()
	firmwarePoller := pollers.NewFirmwarePoller(db)
	modelPoller := pollers.NewModelPoller(db)
	
	// Create render poller for pre-rendering plugin content
	staticDir := config.Get("STATIC_DIR", "./static")
	renderPollerConfig := pollers.PollerConfig{
		Name:        "render_poller",
		Enabled:     true,
		Interval:    30 * time.Second, // Check for render jobs every 30 seconds
		Timeout:     5 * time.Minute,  // Allow up to 5 minutes for render processing
		MaxRetries:  3,
		RetryDelay:  10 * time.Second,
	}
	renderPoller, err := pollers.NewRenderPoller(db, staticDir, renderPollerConfig)
	if err != nil {
		logging.Error("[STARTUP] Failed to create render poller", "error", err)
		os.Exit(1)
	}
	
	// Set global reference for handlers to use
	globalRenderPoller = renderPoller

	pollerManager.Register(firmwarePoller)
	pollerManager.Register(modelPoller)
	pollerManager.Register(renderPoller)

	// Start pollers and SSE keep-alive
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := pollerManager.Start(ctx); err != nil {
		logging.Error("[STARTUP] Failed to start pollers", "error", err)
		os.Exit(1)
	}

	// Schedule initial renders for existing active plugins
	queueManager := rendering.NewQueueManager(db)
	if err := queueManager.ScheduleInitialRenders(ctx); err != nil {
		logging.Error("Failed to schedule initial renders", "error", err)
	}

	// Start SSE keep-alive service
	sseService := sse.GetSSEService()
	go sseService.KeepAlive(ctx)

	port := config.Get("PORT", "")
	if port == "" {
		port = "8000"
	}
	addr := ":" + port

	uiFS, err := fs.Sub(embeddedUI, "ui/dist")
	if err != nil {
		logging.Error("[STARTUP] Failed to create embedded UI filesystem", "error", err)
		os.Exit(1)
	}

	if mode := config.Get("GIN_MODE", ""); mode != "" {
		gin.SetMode(mode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Configure CORS for browser-based device simulators
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	corsConfig.AllowHeaders = []string{
		"Origin",
		"Content-Type",
		"Accept",
		"Authorization",
		// TRMNL device headers
		"ID",
		"Access-Token",
		"Refresh-Rate",
		"Battery-Voltage",
		"Fw-Version",
		"Rssi",
		"Model",
		"Width",
		"Height",
		"User-Agent",
	}
	router.Use(cors.New(corsConfig))

	// Initialize locale manager for TRMNL i18n compatibility
	localeManager, err := locales.NewLocaleManager()
	if err != nil {
		logging.ErrorWithComponent(logging.ComponentStartup, "Failed to initialize locale manager", "error", err)
		os.Exit(1)
	}
	logging.InfoWithComponent(logging.ComponentStartup, "Locale manager initialized", 
		"locales", len(localeManager.GetAvailableLocales()))

	// Register public locale API routes (needed by browserless for template rendering)
	handlers.RegisterLocaleRoutes(router, localeManager)

	// Public auth endpoints
	router.POST("/api/auth/login", auth.MultiUserLoginHandler)
	router.POST("/api/auth/logout", auth.LogoutHandler)
	router.GET("/api/auth/check", auth.MultiUserCheckAuthHandler)
	router.GET("/api/auth/registration-status", auth.GetRegistrationStatusHandler)

	// OIDC endpoints
	router.GET("/api/auth/oidc/login", auth.OIDCAuthHandler)
	router.GET("/api/auth/oidc/callback", auth.OIDCCallbackHandler)
	router.POST("/api/auth/oidc/logout", auth.OIDCLogoutHandler)
	router.GET("/api/auth/proxy/check", auth.ProxyAuthCheckHandler)

	// TRMNL device endpoints (public - device authentication handled internally)
	router.GET("/api/setup", trmnl.SetupHandler)
	router.GET("/api/setup/", trmnl.SetupHandler)
	router.GET("/api/display", trmnl.DisplayHandler)
	router.GET("/api/current_screen", trmnl.CurrentScreenHandler)
	router.POST("/api/logs", trmnl.LogsHandler)
	router.POST("/api/log", trmnl.LogsHandler)
	router.GET("/api/trmnl/devices/:deviceId/image", trmnl.DeviceImageHandler)
	router.GET("/api/trmnl/firmware/:version/download", trmnl.FirmwareDownloadHandler)
	router.POST("/api/trmnl/firmware/update-complete", trmnl.FirmwareUpdateCompleteHandler)

	// Private plugin instance webhook endpoints (public - instance ID-based authentication with rate limiting)
	rateLimiter := middleware.NewWebhookRateLimiter(database.GetDB())
	router.POST("/api/webhooks/instance/:id", 
		rateLimiter.RequestSizeLimit(),
		rateLimiter.RateLimit(),
		handlers.WebhookHandler,
	)

	// Public firmware downloads (no authentication required)
	// Custom handler to serve firmware files - supports both proxy and download modes
	router.GET("/files/firmware/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		// Remove leading slash from filepath
		if strings.HasPrefix(filepath, "/") {
			filepath = filepath[1:]
		}

		// Extract version from filename (e.g., "firmware_1.6.5.bin" -> "1.6.5")
		version := ""
		if strings.HasPrefix(filepath, "firmware_") && strings.HasSuffix(filepath, ".bin") {
			version = strings.TrimPrefix(filepath, "firmware_")
			version = strings.TrimSuffix(version, ".bin")
		}

		if version == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid firmware filename"})
			return
		}

		// Check firmware mode
		firmwareMode := config.Get("FIRMWARE_MODE", "proxy")
		
		if firmwareMode == "proxy" {
			// Proxy mode - forward to TRMNL API
			db := database.GetDB()
			firmwareService := database.NewFirmwareService(db)
			
			fwVersion, err := firmwareService.GetFirmwareVersionByVersion(version)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Firmware version not found"})
				return
			}

			if fwVersion.DownloadURL == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Firmware download URL not available"})
				return
			}

			// Create HTTP client for proxying
			client := &http.Client{
				Timeout: 5 * time.Minute, // Allow time for large firmware downloads
			}

			// Create request to TRMNL API
			req, err := http.NewRequest("GET", fwVersion.DownloadURL, nil)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to proxy firmware request"})
				return
			}

			// Make request to TRMNL
			resp, err := client.Do(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch firmware from upstream"})
				return
			}
			defer resp.Body.Close()

			// Check response status
			if resp.StatusCode != http.StatusOK {
				c.JSON(http.StatusBadGateway, gin.H{"error": "Upstream firmware server error"})
				return
			}

			// Set response headers
			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath))
			if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
				c.Header("Content-Length", contentLength)
			}

			// Stream the response from TRMNL to client
			c.Status(http.StatusOK)
			_, err = io.Copy(c.Writer, resp.Body)
			if err != nil {
				// Log error but can't return JSON at this point since we've started streaming
				logging.Error("[FIRMWARE PROXY] Failed to stream firmware", "version", version, "error", err)
				return
			}
		} else {
			// Download mode - serve local file
			storageDir := config.Get("FIRMWARE_STORAGE_DIR", "/data/firmware")
			filePath := storageDir + "/" + filepath
			c.File(filePath)
		}
	})

	// Registration and password reset
	router.POST("/api/auth/register", auth.MultiUserAuthMiddleware(), auth.RegisterHandler)
	router.POST("/api/auth/register/public", auth.PublicRegisterHandler)
	router.POST("/api/auth/password-reset", auth.PasswordResetHandler)
	router.POST("/api/auth/password-reset/confirm", auth.PasswordResetConfirmHandler)

	// Protected routes (always require authentication)
	protected := router.Group("/api")
	protected.Use(auth.MultiUserAuthMiddleware())

	// User management endpoints (admin only)
	users := protected.Group("/users")
	{
		users.GET("", auth.GetUsersHandler)                               // GET /api/users - list all users (admin)
		users.GET("/:id", auth.GetUserHandler)                            // GET /api/users/:id - get user (admin)
		users.PUT("/:id", auth.UpdateUserHandler)                         // PUT /api/users/:id - update user (admin)
		users.POST("/:id/password", auth.AdminUpdatePasswordHandler)      // POST /api/users/:id/password - update password (admin)
		users.POST("/:id/reset-password", auth.AdminResetPasswordHandler) // POST /api/users/:id/reset-password - reset password (admin)
		users.POST("/:id/deactivate", auth.DeactivateUserHandler)         // POST /api/users/:id/deactivate - deactivate user (admin)
		users.POST("/:id/activate", auth.ActivateUserHandler)             // POST /api/users/:id/activate - activate user (admin)
		users.POST("/:id/promote", auth.PromoteUserHandler)               // POST /api/users/:id/promote - promote user to admin (admin)
		users.POST("/:id/demote", auth.DemoteUserHandler)                 // POST /api/users/:id/demote - demote admin to user (admin)
		users.DELETE("/:id", auth.DeleteUserHandler)                      // DELETE /api/users/:id - delete user (admin)
		users.GET("/stats", auth.GetUserStatsHandler)                     // GET /api/users/stats - get user statistics (admin)
	}

	// Profile management endpoints
	profile := protected.Group("/profile")
	{
		profile.PUT("", auth.UpdateCurrentUserHandler)         // PUT /api/profile - update current user
		profile.POST("/password", auth.UpdatePasswordHandler)  // POST /api/profile/password - update password
		profile.GET("/stats", auth.GetCurrentUserStatsHandler) // GET /api/profile/stats - get current user stats
		profile.DELETE("", auth.DeleteCurrentUserHandler)      // DELETE /api/profile - delete current user account
	}

	// API key management endpoints
	apiKeys := protected.Group("/api-keys")
	{
		apiKeys.GET("", auth.GetAPIKeysHandler)                       // GET /api/api-keys - list user's API keys
		apiKeys.POST("", auth.CreateAPIKeyHandler)                    // POST /api/api-keys - create new API key
		apiKeys.GET("/:id", auth.GetAPIKeyHandler)                    // GET /api/api-keys/:id - get specific API key
		apiKeys.PUT("/:id", auth.UpdateAPIKeyHandler)                 // PUT /api/api-keys/:id - update API key name
		apiKeys.DELETE("/:id", auth.DeleteAPIKeyHandler)              // DELETE /api/api-keys/:id - delete API key
		apiKeys.POST("/:id/deactivate", auth.DeactivateAPIKeyHandler) // POST /api/api-keys/:id/deactivate - deactivate API key
	}

	// Admin API key management
	adminApiKeys := protected.Group("/admin/api-keys")
	adminApiKeys.Use(auth.AdminRequiredMiddleware())
	{
		adminApiKeys.GET("", auth.GetAllAPIKeysHandler)                  // GET /api/admin/api-keys - list all API keys
		adminApiKeys.GET("/stats", auth.GetAPIKeyStatsHandler)           // GET /api/admin/api-keys/stats - get API key stats
		adminApiKeys.POST("/cleanup", auth.CleanupExpiredAPIKeysHandler) // POST /api/admin/api-keys/cleanup - cleanup expired keys
	}

	// Admin endpoints
	admin := protected.Group("/admin")
	admin.Use(auth.AdminRequiredMiddleware())
	{
		admin.GET("/status", auth.GetSystemStatusHandler)       // GET /api/admin/status - get system status
		admin.GET("/settings", auth.GetSystemSettingsHandler)   // GET /api/admin/settings - get system settings
		admin.PUT("/settings", auth.UpdateSystemSettingHandler) // PUT /api/admin/settings - update system setting
		admin.POST("/test-smtp", auth.TestSMTPHandler)          // POST /api/admin/test-smtp - test SMTP config
		admin.POST("/cleanup", auth.CleanupDataHandler)         // POST /api/admin/cleanup - cleanup old data

		// Backup & Restore endpoints
		admin.POST("/backup/analyze", auth.AnalyzeBackupHandler)              // POST /api/admin/backup/analyze - analyze backup file
		admin.POST("/backup-job", auth.CreateBackupJobHandler)                // POST /api/admin/backup-job - create background backup job
		admin.GET("/backup-jobs", auth.GetBackupJobsHandler)                  // GET /api/admin/backup-jobs - get backup jobs
		admin.GET("/backup-job/:id", auth.GetBackupJobHandler)                // GET /api/admin/backup-job/:id - get backup job
		admin.DELETE("/backup-job/:id", auth.DeleteBackupJobHandler)          // DELETE /api/admin/backup-job/:id - delete backup job
		admin.POST("/restore/upload", auth.UploadRestoreFileHandler)          // POST /api/admin/restore/upload - upload restore file
		admin.GET("/restore/uploads", auth.GetRestoreUploadsHandler)          // GET /api/admin/restore/uploads - get pending uploads
		admin.DELETE("/restore/uploads/:id", auth.DeleteRestoreUploadHandler) // DELETE /api/admin/restore/uploads/:id - delete restore upload
		admin.POST("/restore", auth.RestoreDatabaseHandler)                   // POST /api/admin/restore - restore from backup

		// Admin device management
		admin.GET("/devices", handlers.GetAllDevicesHandler)              // GET /api/admin/devices - list all devices
		admin.GET("/devices/stats", handlers.GetDeviceStatsHandler)       // GET /api/admin/devices/stats - get device statistics
		admin.DELETE("/devices/:id/unlink", handlers.UnlinkDeviceHandler) // DELETE /api/admin/devices/:id/unlink - unlink device


		// Firmware management endpoints
		admin.GET("/firmware/versions", handlers.GetFirmwareVersionsHandler)              // GET /api/admin/firmware/versions - list firmware versions
		admin.GET("/firmware/latest", handlers.GetLatestFirmwareVersionHandler)           // GET /api/admin/firmware/latest - get latest firmware version
		admin.GET("/firmware/stats", handlers.GetFirmwareStatsHandler)                    // GET /api/admin/firmware/stats - get firmware statistics
		admin.GET("/firmware/status", handlers.GetFirmwareStatusHandler)                  // GET /api/admin/firmware/status - get real-time download status
		admin.GET("/firmware/mode", handlers.GetFirmwareModeHandler)                      // GET /api/admin/firmware/mode - get current firmware mode
		admin.POST("/firmware/versions/:id/retry", handlers.RetryFirmwareDownloadHandler) // POST /api/admin/firmware/versions/:id/retry - retry firmware download
		admin.DELETE("/firmware/versions/:id", handlers.DeleteFirmwareVersionHandler)     // DELETE /api/admin/firmware/versions/:id - delete firmware version

		// Device model management endpoints
		admin.GET("/device-models", handlers.GetDeviceModelsHandler) // GET /api/admin/device-models - list device models

		// Manual polling endpoints
		admin.POST("/firmware/poll", handlers.TriggerFirmwarePollHandler) // POST /api/admin/firmware/poll - trigger manual firmware poll
		admin.POST("/models/poll", handlers.TriggerModelPollHandler)      // POST /api/admin/models/poll - trigger manual model poll
		
		// External plugin management endpoints
		admin.GET("/external-plugins", handlers.AdminGetExternalPluginsHandler)       // GET /api/admin/external-plugins - list external plugins for admin
		admin.DELETE("/external-plugins/:id", handlers.AdminDeleteExternalPluginHandler) // DELETE /api/admin/external-plugins/:id - delete external plugin
	}

	// Device management endpoints
	devices := protected.Group("/devices")
	{
		devices.GET("", handlers.GetDevicesHandler)                         // GET /api/devices - list user's devices
		devices.GET("/models", handlers.GetDeviceModelOptionsHandler)       // GET /api/devices/models - list available device models
		devices.POST("/claim", handlers.ClaimDeviceHandler)                 // POST /api/devices/claim - claim unclaimed device
		devices.GET("/:id", handlers.GetDeviceHandler)                      // GET /api/devices/:id - get specific device
		devices.PUT("/:id", handlers.UpdateDeviceHandler)                   // PUT /api/devices/:id - update device
		devices.DELETE("/:id", handlers.DeleteDeviceHandler)                // DELETE /api/devices/:id - delete device
		devices.GET("/:id/logs", handlers.GetDeviceLogsHandler)             // GET /api/devices/:id/logs - get device logs
		devices.GET("/:id/events", handlers.DeviceEventsHandler)            // GET /api/devices/:id/events - SSE for device events
		devices.GET("/:id/active-items", handlers.DeviceActiveItemsHandler) // GET /api/devices/:id/active-items - get schedule-filtered active items
		devices.POST("/:id/mirror", handlers.MirrorDeviceHandler)           // POST /api/devices/:id/mirror - mirror another device
		devices.POST("/:id/sync-mirror", handlers.SyncMirrorHandler)        // POST /api/devices/:id/sync-mirror - sync from mirrored device
		devices.DELETE("/:id/unmirror", handlers.UnmirrorDeviceHandler)     // DELETE /api/devices/:id/unmirror - stop mirroring
	}


	// Unified plugin system endpoints
	pluginDefs := protected.Group("/plugin-definitions")
	{
		pluginDefs.GET("", handlers.GetAvailablePluginDefinitionsHandler) // GET /api/plugin-definitions - list all available plugin definitions (system + private)
		pluginDefs.POST("", handlers.CreatePluginDefinitionHandler) // POST /api/plugin-definitions - create new plugin definition (private only)
		pluginDefs.GET("/:id", handlers.GetPluginDefinitionHandler) // GET /api/plugin-definitions/:id - get single plugin definition
		pluginDefs.PUT("/:id", handlers.UpdatePluginDefinitionHandler) // PUT /api/plugin-definitions/:id - update plugin definition
		pluginDefs.DELETE("/:id", handlers.DeletePluginDefinitionHandler) // DELETE /api/plugin-definitions/:id - delete plugin definition
		pluginDefs.POST("/validate", handlers.ValidatePluginDefinitionHandler) // POST /api/plugin-definitions/validate - validate plugin templates
		pluginDefs.POST("/test", handlers.TestPluginDefinitionHandler) // POST /api/plugin-definitions/test - test plugin template rendering
		pluginDefs.GET("/refresh-rate-options", handlers.GetRefreshRateOptionsHandler) // GET /api/plugin-definitions/refresh-rate-options - get available refresh rates
		pluginDefs.POST("/validate-settings", handlers.ValidatePluginSettingsHandler) // POST /api/plugin-definitions/validate-settings - validate plugin settings
		pluginDefs.POST("/import", handlers.ImportPluginDefinitionHandler) // POST /api/plugin-definitions/import - import TRMNL-compatible ZIP file
		pluginDefs.GET("/:id/export", handlers.ExportPluginDefinitionHandler) // GET /api/plugin-definitions/:id/export - export plugin as TRMNL-compatible ZIP file
		pluginDefs.GET("/types", handlers.GetAvailablePluginTypesHandler) // GET /api/plugin-definitions/types - get available plugin types
		pluginDefs.POST("/debug/validate-yaml", handlers.ValidateTRMNLYAMLHandler) // POST /api/plugin-definitions/debug/validate-yaml - validate TRMNL YAML format
		pluginDefs.POST("/debug/test-conversion", handlers.TestTRMNLConversionHandler) // POST /api/plugin-definitions/debug/test-conversion - test bidirectional TRMNL conversion
		
		// Mashup endpoints
		pluginDefs.POST("/mashup", handlers.CreateMashupHandler) // POST /api/plugin-definitions/mashup - create mashup plugin
		pluginDefs.GET("/mashup/layouts", handlers.GetAvailableMashupLayoutsHandler) // GET /api/plugin-definitions/mashup/layouts - get available layouts
		pluginDefs.GET("/mashup/layouts/:layout/slots", handlers.GetMashupSlotsHandler) // GET /api/plugin-definitions/mashup/layouts/:layout/slots - get slots for layout
	}

	protected.GET("/plugin-instances", handlers.GetPluginInstancesHandler) // GET /api/plugin-instances - list user's plugin instances
	protected.POST("/plugin-instances", handlers.CreatePluginInstanceFromDefinitionHandler) // POST /api/plugin-instances - create plugin instance from definition
	
	// Static routes must come before parameterized routes
	protected.GET("/plugin-instances/private", handlers.GetUserPrivatePluginInstancesHandler) // GET /api/plugin-instances/private - get user's private plugin instances for mashup children
	
	// Parameterized routes (all using :id parameter)
	protected.PUT("/plugin-instances/:id", handlers.UpdatePluginInstanceHandler) // PUT /api/plugin-instances/:id - update plugin instance
	protected.DELETE("/plugin-instances/:id", handlers.DeletePluginInstanceHandler) // DELETE /api/plugin-instances/:id - delete plugin instance
	protected.POST("/plugin-instances/:id/force-refresh", handlers.ForceRefreshPluginInstanceHandler) // POST /api/plugin-instances/:id/force-refresh - force refresh plugin instance
	protected.GET("/plugin-instances/:id/schema-diff", handlers.GetPluginInstanceSchemaDiffHandler) // GET /api/plugin-instances/:id/schema-diff - get schema differences for instance
	
	// Mashup instance endpoints (using consistent :id parameter)
	protected.POST("/plugin-instances/:id/mashup/children", handlers.AssignMashupChildrenHandler) // POST /api/plugin-instances/:id/mashup/children - assign children to mashup slots
	protected.GET("/plugin-instances/:id/mashup/children", handlers.GetMashupChildrenHandler) // GET /api/plugin-instances/:id/mashup/children - get current mashup children



	// Playlist management endpoints
	playlists := protected.Group("/playlists")
	{
		playlists.GET("", handlers.GetPlaylistsHandler)                                // GET /api/playlists - list user's playlists
		playlists.POST("", handlers.CreatePlaylistHandler)                             // POST /api/playlists - create playlist
		playlists.GET("/:id", handlers.GetPlaylistHandler)                             // GET /api/playlists/:id - get playlist with items
		playlists.PUT("/:id", handlers.UpdatePlaylistHandler)                          // PUT /api/playlists/:id - update playlist
		playlists.DELETE("/:id", handlers.DeletePlaylistHandler)                       // DELETE /api/playlists/:id - delete playlist
		playlists.POST("/:id/items", handlers.AddPlaylistItemHandler)                  // POST /api/playlists/:id/items - add item to playlist
		playlists.PUT("/:id/reorder", handlers.ReorderPlaylistItemsHandler)            // PUT /api/playlists/:id/reorder - reorder items (legacy)
		playlists.PUT("/:id/reorder-array", handlers.ReorderPlaylistItemsArrayHandler) // PUT /api/playlists/:id/reorder-array - reorder items by array
		playlists.PUT("/items/:itemId", handlers.UpdatePlaylistItemHandler)            // PUT /api/playlists/items/:itemId - update playlist item
		playlists.DELETE("/items/:itemId", handlers.DeletePlaylistItemHandler)         // DELETE /api/playlists/items/:itemId - delete playlist item
		playlists.POST("/items/:itemId/schedules", handlers.AddScheduleHandler)        // POST /api/playlists/items/:itemId/schedules - add schedule
		playlists.PUT("/schedules/:scheduleId", handlers.UpdateScheduleHandler)        // PUT /api/playlists/schedules/:scheduleId - update schedule
		playlists.DELETE("/schedules/:scheduleId", handlers.DeleteScheduleHandler)     // DELETE /api/playlists/schedules/:scheduleId - delete schedule
	}

	// Dashboard endpoint (simple placeholder for now)
	protected.GET("/dashboard", handlers.DashboardHandler)

	// User endpoints
	protected.POST("/user/complete-onboarding", handlers.CompleteOnboardingHandler)

	// Version endpoint
	protected.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, version.Get())
	})

	// Config endpoint
	router.GET("/api/config", handlers.ConfigHandler)

	// Static images (no authentication required)
	// Use explicit handlers to serve static files
	router.GET("/images/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		// Remove leading slash from filepath
		if strings.HasPrefix(filepath, "/") {
			filepath = filepath[1:]
		}
		
		// First try to serve from embedded TRMNL assets
		embedPath := "assets/trmnl/images/" + filepath
		if data, err := embeddedTRNMLAssets.ReadFile(embedPath); err == nil {
			// Determine content type from file extension
			contentType := "application/octet-stream"
			if strings.HasSuffix(strings.ToLower(filepath), ".png") {
				contentType = "image/png"
			} else if strings.HasSuffix(strings.ToLower(filepath), ".jpg") || strings.HasSuffix(strings.ToLower(filepath), ".jpeg") {
				contentType = "image/jpeg"
			} else if strings.HasSuffix(strings.ToLower(filepath), ".gif") {
				contentType = "image/gif"
			} else if strings.HasSuffix(strings.ToLower(filepath), ".svg") {
				contentType = "image/svg+xml"
			}
			
			c.Header("Content-Type", contentType)
			c.Header("Cache-Control", "public, max-age=31536000") // Cache for 1 year
			c.Data(http.StatusOK, contentType, data)
			return
		}
		
		// Fallback to filesystem images
		c.File("./images/" + filepath)
	})
	router.GET("/static/rendered/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		// Remove leading slash from filepath
		if strings.HasPrefix(filepath, "/") {
			filepath = filepath[1:]
		}
		c.File("./static/rendered/" + filepath)
	})

	// TRMNL assets (no authentication required - used by browserless)
	router.GET("/assets/trmnl/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		// Remove leading slash from filepath
		if strings.HasPrefix(filepath, "/") {
			filepath = filepath[1:]
		}
		
		// Serve from embedded assets
		assetPath := "assets/trmnl/" + filepath
		data, err := embeddedTRNMLAssets.ReadFile(assetPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		
		// Set appropriate content type based on file extension
		if strings.HasSuffix(filepath, ".css") {
			c.Header("Content-Type", "text/css")
		} else if strings.HasSuffix(filepath, ".js") {
			c.Header("Content-Type", "application/javascript")
		} else if strings.HasSuffix(filepath, ".ttf") {
			c.Header("Content-Type", "font/ttf")
		} else if strings.HasSuffix(filepath, ".woff") {
			c.Header("Content-Type", "font/woff")
		} else if strings.HasSuffix(filepath, ".woff2") {
			c.Header("Content-Type", "font/woff2")
		}
		
		// Set cache headers for assets
		c.Header("Cache-Control", "public, max-age=31536000") // 1 year
		
		c.Data(http.StatusOK, c.GetHeader("Content-Type"), data)
	})

	// TRMNL fonts at expected /fonts/ path (no authentication required)
	router.GET("/fonts/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		// Remove leading slash from filepath
		if strings.HasPrefix(filepath, "/") {
			filepath = filepath[1:]
		}
		
		// Serve TRMNL fonts from embedded assets
		assetPath := "assets/trmnl/fonts/" + filepath
		data, err := embeddedTRNMLAssets.ReadFile(assetPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		
		// Set appropriate content type for fonts
		if strings.HasSuffix(filepath, ".ttf") {
			c.Header("Content-Type", "font/ttf")
		} else if strings.HasSuffix(filepath, ".woff") {
			c.Header("Content-Type", "font/woff")
		} else if strings.HasSuffix(filepath, ".woff2") {
			c.Header("Content-Type", "font/woff2")
		}
		
		// Set cache headers for fonts
		c.Header("Cache-Control", "public, max-age=31536000") // 1 year
		
		c.Data(http.StatusOK, c.GetHeader("Content-Type"), data)
	})

	// Serve UI
	router.NoRoute(func(c *gin.Context) {
			p := strings.TrimPrefix(c.Request.URL.Path, "/")
			if p == "" {
				p = "index.html"
			}

			if p == "index.html" {
				envUsername := config.Get("AUTH_USERNAME", "")
				envPassword := config.Get("AUTH_PASSWORD", "")
				webAuthDisabled := envUsername == "" || envPassword == ""

				if webAuthDisabled {
					auth.ServeIndexWithSecret(c, uiFS, auth.GetUISecret())
					return
				}
			}

			if stat, err := fs.Stat(uiFS, p); err != nil || stat.IsDir() {
				p = "index.html"
				if p == "index.html" {
					envUsername := config.Get("AUTH_USERNAME", "")
					envPassword := config.Get("AUTH_PASSWORD", "")
					webAuthDisabled := envUsername == "" || envPassword == ""

					if webAuthDisabled {
						auth.ServeIndexWithSecret(c, uiFS, auth.GetUISecret())
						return
					}
				}
			}

			if strings.HasSuffix(p, ".js") {
				c.Header("Content-Type", "application/javascript")
			}
			if p == "index.html" {
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
				c.Header("Pragma", "no-cache")
				c.Header("Expires", "0")
			}
			http.ServeFileFS(c.Writer, c.Request, uiFS, p)
		})

	// Create HTTP server
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logging.Info("[STARTUP] Listening", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Error("[STARTUP] Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logging.Info("[SHUTDOWN] Shutting down server and pollers")

	// Stop pollers first
	if err := pollerManager.Stop(); err != nil {
		logging.Error("[SHUTDOWN] Error stopping pollers", "error", err)
	}

	// Give a timeout context for shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logging.Error("[SHUTDOWN] Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logging.Info("[SHUTDOWN] Server and pollers stopped")
}
