package auth

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/backup"
	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/export"
	"github.com/rmitchellscott/stationmaster/internal/restore"
	"github.com/rmitchellscott/stationmaster/internal/smtp"
	"gorm.io/gorm"
)

// TestSMTPHandler tests SMTP configuration (admin only)
func TestSMTPHandler(c *gin.Context) {
	if !database.IsMultiUserMode() {
		c.JSON(http.StatusNotFound, gin.H{"error": "SMTP testing not available in single-user mode"})
		return
	}

	_, ok := RequireAdmin(c)
	if !ok {
		return
	}

	// Test SMTP connection
	if err := smtp.TestSMTPConnection(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "SMTP connection failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "SMTP connection successful",
	})
}

// GetSystemStatusHandler returns system status information (admin only)
func GetSystemStatusHandler(c *gin.Context) {
	if !database.IsMultiUserMode() {
		c.JSON(http.StatusNotFound, gin.H{"error": "System status not available in single-user mode"})
		return
	}

	_, ok := RequireAdmin(c)
	if !ok {
		return
	}

	// Get database stats
	dbStats, err := database.GetDatabaseStats(database.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database stats"})
		return
	}

	// Check SMTP configuration (without testing connection)
	smtpConfigured := smtp.IsSMTPConfigured()
	var smtpStatus string
	if smtpConfigured {
		smtpStatus = "configured"
	} else {
		smtpStatus = "not_configured"
	}

	// Get system settings
	registrationEnabled, registrationLocked := database.GetRegistrationSetting()
	maxAPIKeys, _ := database.GetSystemSetting("max_api_keys_per_user")
	siteURL, _ := database.GetSystemSetting("site_url")

	// Check if we're in dry run mode
	dryRunMode := config.Get("DRY_RUN", "") != ""

	// Check authentication methods
	oidcEnabled := IsOIDCEnabled()
	proxyAuthEnabled := IsProxyAuthEnabled()

	// Count active sessions for dashboard
	var activeSessions int64
	database.DB.Model(&database.UserSession{}).Where("expires_at > ?", time.Now()).Count(&activeSessions)

	c.JSON(http.StatusOK, gin.H{
		"database": gin.H{
			"total_users":  dbStats.TotalUsers,
			"active_users": dbStats.ActiveUsers,
			"api_keys": gin.H{
				"total":  dbStats.TotalAPIKeys,
				"active": dbStats.ActiveAPIKeys,
			},
			"documents":       0, // Stationmaster doesn't have documents
			"active_sessions": activeSessions,
		},
		"smtp": gin.H{
			"configured": smtpConfigured,
			"status":     smtpStatus,
		},
		"settings": gin.H{
			"registration_enabled":        registrationEnabled,
			"registration_enabled_locked": registrationLocked,
			"max_api_keys_per_user":       maxAPIKeys,
			"site_url":                    siteURL,
		},
		"auth": gin.H{
			"oidc_enabled":       oidcEnabled,
			"proxy_auth_enabled": proxyAuthEnabled,
		},
		"mode":    "multi_user",
		"dry_run": dryRunMode,
	})
}

// UpdateSystemSettingHandler updates a system setting (admin only)
func UpdateSystemSettingHandler(c *gin.Context) {
	if !database.IsMultiUserMode() {
		c.JSON(http.StatusNotFound, gin.H{"error": "System settings not available in single-user mode"})
		return
	}

	user, ok := RequireAdmin(c)
	if !ok {
		return
	}

	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_type": "invalid_request"})
		return
	}

	// Check if registration_enabled is locked by environment variable
	if req.Key == "registration_enabled" && database.IsRegistrationSettingLocked() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Registration setting is controlled by environment variable"})
		return
	}

	// Validate allowed settings
	allowedSettings := map[string]bool{
		"registration_enabled":         true,
		"max_api_keys_per_user":        true,
		"password_reset_timeout_hours": true,
		"site_url":                     true,
	}

	if !allowedSettings[req.Key] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setting not allowed to be updated"})
		return
	}

	// Update the setting
	if err := database.SetSystemSetting(req.Key, req.Value, &user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Setting updated successfully",
	})
}

// GetSystemSettingsHandler returns all system settings (admin only)
func GetSystemSettingsHandler(c *gin.Context) {
	if !database.IsMultiUserMode() {
		c.JSON(http.StatusNotFound, gin.H{"error": "System settings not available in single-user mode"})
		return
	}

	_, ok := RequireAdmin(c)
	if !ok {
		return
	}

	// Get all system settings
	var settings []database.SystemSetting
	if err := database.DB.Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve settings"})
		return
	}

	// Convert to map for easier frontend consumption
	settingsMap := make(map[string]interface{})
	for _, setting := range settings {
		settingsMap[setting.Key] = gin.H{
			"value":       setting.Value,
			"description": setting.Description,
			"updated_at":  setting.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, settingsMap)
}

// CleanupDataHandler performs database cleanup (admin only)
func CleanupDataHandler(c *gin.Context) {
	if !database.IsMultiUserMode() {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data cleanup not available in single-user mode"})
		return
	}

	_, ok := RequireAdmin(c)
	if !ok {
		return
	}

	// Default cleanup: 30 days for sessions, 7 days for login attempts
	if err := database.CleanupOldData(database.DB, 30, 7); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Data cleanup completed successfully",
	})
}

// AnalyzeBackupHandler analyzes an uploaded backup file (admin only)
func AnalyzeBackupHandler(c *gin.Context) {
	_, ok := RequireAdmin(c)
	if !ok {
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		c.JSON(http.StatusBadRequest, gin.H{"error_type": "parse_form_failed"})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("backup_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_type": "no_backup_file"})
		return
	}
	defer file.Close()

	// Validate file type
	filename := header.Filename
	if !strings.HasSuffix(filename, ".tar.gz") && !strings.HasSuffix(filename, ".tgz") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file type. Expected .tar.gz or .tgz file",
			"valid": false,
		})
		return
	}

	// Create temporary file for upload
	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, "analyze_"+filename)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "create_temp_file_failed"})
		return
	}
	defer os.Remove(tempFilePath)
	defer tempFile.Close()

	// Copy uploaded file to temp location
	if _, err := io.Copy(tempFile, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "save_uploaded_file_failed"})
		return
	}

	// Analyze the backup
	analysis, err := export.AnalyzeBackup(tempFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "backup_analyze_failed"})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// CreateBackupJobHandler creates a new backup job (admin only)
func CreateBackupJobHandler(c *gin.Context) {
	user, ok := RequireAdmin(c)
	if !ok {
		return
	}

	var req struct {
		IncludeFiles   bool        `json:"include_files"`
		IncludeConfigs bool        `json:"include_configs"`
		UserIDs        []uuid.UUID `json:"user_ids,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Create the backup job
	job, err := backup.CreateBackupJob(database.DB, user.ID, req.IncludeFiles, req.IncludeConfigs, req.UserIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "create_backup_job_failed"})
		return
	}

	// Start the backup worker if not running
	backup.EnsureWorkerRunning(database.DB)

	c.JSON(http.StatusCreated, job)
}

// GetBackupJobsHandler returns backup jobs for the admin user
func GetBackupJobsHandler(c *gin.Context) {
	user, ok := RequireAdmin(c)
	if !ok {
		return
	}

	jobs, err := backup.GetBackupJobs(database.DB, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "get_backup_jobs_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"total": len(jobs),
	})
}

// GetBackupJobHandler returns a specific backup job
func GetBackupJobHandler(c *gin.Context) {
	user, ok := RequireAdmin(c)
	if !ok {
		return
	}

	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_type": "invalid_job_id"})
		return
	}

	job, err := backup.GetBackupJob(database.DB, jobID, user.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error_type": "backup_job_not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get backup job"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// DeleteBackupJobHandler deletes a backup job and its file
func DeleteBackupJobHandler(c *gin.Context) {
	user, ok := RequireAdmin(c)
	if !ok {
		return
	}

	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_type": "invalid_job_id"})
		return
	}

	if err := backup.DeleteBackupJob(database.DB, jobID, user.ID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error_type": "backup_job_not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "delete_backup_job_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UploadRestoreFileHandler handles restore file uploads (admin only)
func UploadRestoreFileHandler(c *gin.Context) {
	user, ok := RequireAdmin(c)
	if !ok {
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil { // 100MB max
		c.JSON(http.StatusBadRequest, gin.H{"error_type": "parse_form_failed"})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("restore_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No restore file provided"})
		return
	}
	defer file.Close()

	// Validate file type
	filename := header.Filename
	if !strings.HasSuffix(filename, ".tar.gz") && !strings.HasSuffix(filename, ".tgz") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      fmt.Sprintf("Invalid file type. Please select a backup archive (.tar.gz or .tgz). Selected file: %s", filename),
			"error_type": "backup_file_type",
		})
		return
	}

	// Create upload directory
	dataDir := config.Get("DATA_DIR", "/data")
	uploadDir := filepath.Join(dataDir, "restore_uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "create_temp_file_failed"})
		return
	}

	// Create unique filename
	uploadID := uuid.New()
	uploadPath := filepath.Join(uploadDir, uploadID.String()+".tar.gz")

	// Save uploaded file
	outFile, err := os.Create(uploadPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "create_temp_file_failed"})
		return
	}
	defer outFile.Close()

	size, err := io.Copy(outFile, file)
	if err != nil {
		os.Remove(uploadPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "save_uploaded_file_failed"})
		return
	}

	// Create restore upload record
	restoreUpload := database.RestoreUpload{
		AdminUserID: user.ID,
		Filename:    filename,
		FilePath:    uploadPath,
		FileSize:    size,
		Status:      "uploaded",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	if err := database.DB.Create(&restoreUpload).Error; err != nil {
		os.Remove(uploadPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "save_upload_record_failed"})
		return
	}

	c.JSON(http.StatusCreated, restoreUpload)
}

// GetRestoreUploadsHandler returns pending restore uploads
func GetRestoreUploadsHandler(c *gin.Context) {
	user, ok := RequireAdmin(c)
	if !ok {
		return
	}

	var uploads []database.RestoreUpload
	if err := database.DB.Where("admin_user_id = ?", user.ID).
		Order("created_at DESC").
		Find(&uploads).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get restore uploads"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uploads": uploads,
		"total":   len(uploads),
	})
}

// DeleteRestoreUploadHandler deletes a restore upload
func DeleteRestoreUploadHandler(c *gin.Context) {
	user, ok := RequireAdmin(c)
	if !ok {
		return
	}

	uploadIDStr := c.Param("id")
	uploadID, err := uuid.Parse(uploadIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_type": "invalid_upload_id"})
		return
	}

	var upload database.RestoreUpload
	if err := database.DB.Where("id = ? AND admin_user_id = ?", uploadID, user.ID).First(&upload).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error_type": "upload_not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find upload"})
		return
	}

	// Clean up any associated extraction jobs
	restore.CleanupExtractionJob(database.DB, uploadID, user.ID)

	// Remove the file
	if upload.FilePath != "" {
		os.Remove(upload.FilePath)
	}

	// Delete the record
	if err := database.DB.Delete(&upload).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error_type": "restore_delete_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RestoreDatabaseHandler performs the actual restore operation
func RestoreDatabaseHandler(c *gin.Context) {
	user, ok := RequireAdmin(c)
	if !ok {
		return
	}

	var req struct {
		RestoreUploadID uuid.UUID `json:"restore_upload_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get the restore upload
	var upload database.RestoreUpload
	if err := database.DB.Where("id = ? AND admin_user_id = ?", req.RestoreUploadID, user.ID).First(&upload).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error_type": "upload_not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find upload"})
		return
	}

	// Create importer and restore
	importer := export.NewImporter(database.DB, config.Get("DATA_DIR", "/data"))

	options := export.ImportOptions{
		OverwriteDatabase: true,
		OverwriteFiles:    true,
	}

	metadata, err := importer.Import(upload.FilePath, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error_type": "restore_failed",
			"error":      fmt.Sprintf("Failed to restore database: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  fmt.Sprintf("Backup restored successfully from %s", upload.Filename),
		"metadata": metadata,
	})
}
