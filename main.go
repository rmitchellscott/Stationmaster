package main

import (
	// standard library
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	// third-party
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	// internal
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/handlers"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/pollers"
	"github.com/rmitchellscott/stationmaster/internal/trmnl"
	"github.com/rmitchellscott/stationmaster/internal/version"
)

//go:generate npm --prefix ui install
//go:generate npm --prefix ui run build
//go:embed ui/dist
//go:embed ui/dist/assets
var embeddedUI embed.FS

func main() {
	_ = godotenv.Load()
	logging.Logf("[STARTUP] Starting Stationmaster %s", version.String())

	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version.String())
		os.Exit(0)
	}

	// Initialize database (always in multi-user mode)
	if err := database.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	if err := database.MigrateToMultiUser(); err != nil {
		log.Fatalf("Failed to setup initial user: %v", err)
	}

	// Initialize OIDC if configured
	if err := auth.InitOIDC(); err != nil {
		log.Fatalf("Failed to initialize OIDC: %v", err)
	}

	// Initialize proxy auth if configured
	auth.InitProxyAuth()

	// Initialize and start background pollers
	pollerManager := pollers.NewManager()
	
	// Register pollers
	db := database.GetDB()
	firmwarePoller := pollers.NewFirmwarePoller(db)
	modelPoller := pollers.NewModelPoller(db)
	
	pollerManager.Register(firmwarePoller)
	pollerManager.Register(modelPoller)
	
	// Start pollers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if err := pollerManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start pollers: %v", err)
	}

	port := config.Get("PORT", "")
	if port == "" {
		port = "8000"
	}
	addr := ":" + port

	uiFS, err := fs.Sub(embeddedUI, "ui/dist")
	if err != nil {
		log.Fatalf("embed error: %v", err)
	}

	if mode := config.Get("GIN_MODE", ""); mode != "" {
		gin.SetMode(mode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

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
	router.POST("/api/logs", trmnl.LogsHandler)
	router.POST("/api/log", trmnl.LogsHandler)
	router.GET("/api/trmnl/devices/:deviceId/image", trmnl.DeviceImageHandler)
	router.GET("/api/trmnl/firmware/:version/download", trmnl.FirmwareDownloadHandler)
	router.POST("/api/trmnl/firmware/update-complete", trmnl.FirmwareUpdateCompleteHandler)
	
	// Public firmware downloads (no authentication required)
	// Custom handler to serve firmware files
	router.GET("/files/firmware/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		// Remove leading slash from filepath
		if strings.HasPrefix(filepath, "/") {
			filepath = filepath[1:]
		}
		// Use the same storage directory configuration as the firmware poller
		storageDir := config.Get("FIRMWARE_STORAGE_DIR", "/data/firmware")
		filePath := storageDir + "/" + filepath
		c.File(filePath)
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
		admin.POST("/backup/analyze", auth.AnalyzeBackupHandler)             // POST /api/admin/backup/analyze - analyze backup file
		admin.POST("/backup-job", auth.CreateBackupJobHandler)               // POST /api/admin/backup-job - create background backup job
		admin.GET("/backup-jobs", auth.GetBackupJobsHandler)                 // GET /api/admin/backup-jobs - get backup jobs
		admin.GET("/backup-job/:id", auth.GetBackupJobHandler)               // GET /api/admin/backup-job/:id - get backup job
		admin.DELETE("/backup-job/:id", auth.DeleteBackupJobHandler)         // DELETE /api/admin/backup-job/:id - delete backup job
		admin.POST("/restore/upload", auth.UploadRestoreFileHandler)         // POST /api/admin/restore/upload - upload restore file
		admin.GET("/restore/uploads", auth.GetRestoreUploadsHandler)         // GET /api/admin/restore/uploads - get pending uploads
		admin.DELETE("/restore/uploads/:id", auth.DeleteRestoreUploadHandler) // DELETE /api/admin/restore/uploads/:id - delete restore upload
		admin.POST("/restore", auth.RestoreDatabaseHandler)                  // POST /api/admin/restore - restore from backup

		// Admin device management
		admin.GET("/devices", handlers.GetAllDevicesHandler)                 // GET /api/admin/devices - list all devices
		admin.GET("/devices/stats", handlers.GetDeviceStatsHandler)          // GET /api/admin/devices/stats - get device statistics
		admin.DELETE("/devices/:id/unlink", handlers.UnlinkDeviceHandler)    // DELETE /api/admin/devices/:id/unlink - unlink device

		// Admin plugin management
		admin.POST("/plugins", handlers.CreatePluginHandler)                 // POST /api/admin/plugins - create system plugin
		admin.PUT("/plugins/:id", handlers.UpdatePluginHandler)              // PUT /api/admin/plugins/:id - update system plugin
		admin.DELETE("/plugins/:id", handlers.DeletePluginHandler)           // DELETE /api/admin/plugins/:id - delete system plugin
		admin.GET("/plugins/stats", handlers.GetPluginStatsHandler)          // GET /api/admin/plugins/stats - get plugin statistics

		// Firmware management endpoints
		admin.GET("/firmware/versions", handlers.GetFirmwareVersionsHandler)           // GET /api/admin/firmware/versions - list firmware versions
		admin.GET("/firmware/latest", handlers.GetLatestFirmwareVersionHandler)       // GET /api/admin/firmware/latest - get latest firmware version
		admin.GET("/firmware/stats", handlers.GetFirmwareStatsHandler)                // GET /api/admin/firmware/stats - get firmware statistics
		admin.GET("/firmware/status", handlers.GetFirmwareStatusHandler)              // GET /api/admin/firmware/status - get real-time download status
		admin.POST("/firmware/versions/:id/retry", handlers.RetryFirmwareDownloadHandler) // POST /api/admin/firmware/versions/:id/retry - retry firmware download
		admin.DELETE("/firmware/versions/:id", handlers.DeleteFirmwareVersionHandler) // DELETE /api/admin/firmware/versions/:id - delete firmware version

		// Device model management endpoints
		admin.GET("/device-models", handlers.GetDeviceModelsHandler)                  // GET /api/admin/device-models - list device models

		// Manual polling endpoints  
		admin.POST("/firmware/poll", handlers.TriggerFirmwarePollHandler)             // POST /api/admin/firmware/poll - trigger manual firmware poll
		admin.POST("/models/poll", handlers.TriggerModelPollHandler)                  // POST /api/admin/models/poll - trigger manual model poll
	}

	// Device management endpoints
	devices := protected.Group("/devices")
	{
		devices.GET("", handlers.GetDevicesHandler)                         // GET /api/devices - list user's devices
		devices.POST("/claim", handlers.ClaimDeviceHandler)                 // POST /api/devices/claim - claim unclaimed device
		devices.GET("/:id", handlers.GetDeviceHandler)                      // GET /api/devices/:id - get specific device
		devices.PUT("/:id", handlers.UpdateDeviceHandler)                   // PUT /api/devices/:id - update device
		devices.DELETE("/:id", handlers.DeleteDeviceHandler)                // DELETE /api/devices/:id - delete device
		devices.GET("/:id/logs", handlers.GetDeviceLogsHandler)             // GET /api/devices/:id/logs - get device logs
	}

	// Plugin management endpoints
	plugins := protected.Group("/plugins")
	{
		plugins.GET("", handlers.GetPluginsHandler)                         // GET /api/plugins - list available plugins
	}

	// User plugin management endpoints
	userPlugins := protected.Group("/user-plugins")
	{
		userPlugins.GET("", handlers.GetUserPluginsHandler)                 // GET /api/user-plugins - list user's plugin instances
		userPlugins.POST("", handlers.CreateUserPluginHandler)              // POST /api/user-plugins - create plugin instance
		userPlugins.GET("/:id", handlers.GetUserPluginHandler)              // GET /api/user-plugins/:id - get plugin instance
		userPlugins.PUT("/:id", handlers.UpdateUserPluginHandler)           // PUT /api/user-plugins/:id - update plugin instance
		userPlugins.DELETE("/:id", handlers.DeleteUserPluginHandler)        // DELETE /api/user-plugins/:id - delete plugin instance
	}

	// Playlist management endpoints
	playlists := protected.Group("/playlists")
	{
		playlists.GET("", handlers.GetPlaylistsHandler)                     // GET /api/playlists - list user's playlists
		playlists.POST("", handlers.CreatePlaylistHandler)                  // POST /api/playlists - create playlist
		playlists.GET("/:id", handlers.GetPlaylistHandler)                  // GET /api/playlists/:id - get playlist with items
		playlists.PUT("/:id", handlers.UpdatePlaylistHandler)               // PUT /api/playlists/:id - update playlist
		playlists.DELETE("/:id", handlers.DeletePlaylistHandler)            // DELETE /api/playlists/:id - delete playlist
		playlists.POST("/:id/items", handlers.AddPlaylistItemHandler)       // POST /api/playlists/:id/items - add item to playlist
		playlists.PUT("/:id/reorder", handlers.ReorderPlaylistItemsHandler) // PUT /api/playlists/:id/reorder - reorder items
		playlists.PUT("/items/:itemId", handlers.UpdatePlaylistItemHandler) // PUT /api/playlists/items/:itemId - update playlist item
		playlists.DELETE("/items/:itemId", handlers.DeletePlaylistItemHandler) // DELETE /api/playlists/items/:itemId - delete playlist item
		playlists.POST("/items/:itemId/schedules", handlers.AddScheduleHandler)      // POST /api/playlists/items/:itemId/schedules - add schedule
		playlists.PUT("/schedules/:scheduleId", handlers.UpdateScheduleHandler)      // PUT /api/playlists/schedules/:scheduleId - update schedule
		playlists.DELETE("/schedules/:scheduleId", handlers.DeleteScheduleHandler)   // DELETE /api/playlists/schedules/:scheduleId - delete schedule
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

	// Serve UI
	if config.Get("DISABLE_UI", "") == "" {
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
				if strings.HasPrefix(p, "assets/") {
					c.AbortWithStatus(http.StatusNotFound)
					return
				}
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
	} else {
		logging.Logf("[STARTUP] DISABLE_UI is set → running in API-only mode (no UI).")
		router.NoRoute(func(c *gin.Context) {
			c.AbortWithStatus(http.StatusNotFound)
		})
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logging.Logf("[STARTUP] Listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	logging.Logf("[SHUTDOWN] Shutting down server and pollers...")

	// Stop pollers first
	if err := pollerManager.Stop(); err != nil {
		logging.Logf("[SHUTDOWN] Error stopping pollers: %v", err)
	}

	// Give a timeout context for shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logging.Logf("[SHUTDOWN] Server and pollers stopped")
}