package pollers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// FirmwarePoller polls for firmware updates
type FirmwarePoller struct {
	*BasePoller
	db           *gorm.DB
	apiURL       string
	storageDir   string
	firmwareMode string
}

// FirmwareVersionInfo represents firmware version information from the API
type FirmwareVersionInfo struct {
	Version    string `json:"version"`
	URL        string `json:"url"`
	MergedPath string `json:"merged_path"`
}

// NewFirmwarePoller creates a new firmware poller
func NewFirmwarePoller(db *gorm.DB) *FirmwarePoller {
	// Get configuration from environment variables
	interval := 6 * time.Hour // Default 6 hours
	if envInterval := config.Get("FIRMWARE_POLLER_INTERVAL", ""); envInterval != "" {
		if d, err := time.ParseDuration(envInterval); err == nil {
			interval = d
		}
	}

	enabled := config.Get("FIRMWARE_POLLER", "true") != "false"
	apiURL := config.Get("TRMNL_FIRMWARE_API_URL", "https://usetrmnl.com/api/firmware/latest")
	storageDir := config.Get("FIRMWARE_STORAGE_DIR", "/data/firmware")
	firmwareMode := config.Get("FIRMWARE_MODE", "proxy")

	config := PollerConfig{
		Name:       "firmware",
		Interval:   interval,
		Enabled:    enabled,
		MaxRetries: 3,
		RetryDelay: 30 * time.Second,
		Timeout:    2 * time.Minute,
	}

	poller := &FirmwarePoller{
		db:           db,
		apiURL:       apiURL,
		storageDir:   storageDir,
		firmwareMode: firmwareMode,
	}

	poller.BasePoller = NewBasePoller(config, poller.poll)
	return poller
}

// ExecutePoll executes a single poll operation (for manual triggering)
func (p *FirmwarePoller) ExecutePoll(ctx context.Context) error {
	return p.poll(ctx)
}

// DiscoverFirmware discovers firmware versions and creates database entries without downloading
func (p *FirmwarePoller) DiscoverFirmware(ctx context.Context) error {
	logging.Logf("[FIRMWARE DISCOVERY] Starting firmware discovery")

	// Ensure storage directory exists
	if err := os.MkdirAll(p.storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Fetch latest firmware information from API
	versionInfo, err := p.fetchLatestFirmwareVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch latest firmware version: %w", err)
	}

	logging.Logf("[FIRMWARE DISCOVERY] Found firmware version %s", versionInfo.Version)

	// Process the version - create database entry but don't download yet
	if err := p.discoverFirmwareVersion(ctx, *versionInfo); err != nil {
		return fmt.Errorf("error discovering version %s: %w", versionInfo.Version, err)
	}

	logging.Logf("[FIRMWARE DISCOVERY] Firmware discovery completed")
	return nil
}

// StartPendingDownloads starts downloads for firmware versions with pending status
func (p *FirmwarePoller) StartPendingDownloads(ctx context.Context) error {
	logging.Logf("[FIRMWARE DOWNLOADS] Starting pending downloads")
	
	// Skip downloads in proxy mode
	if p.firmwareMode != "download" {
		logging.Logf("[FIRMWARE DOWNLOADS] Proxy mode enabled, skipping file downloads")
		return nil
	}

	// Get all pending firmware versions
	var pendingVersions []database.FirmwareVersion
	if err := p.db.Where("download_status = ? OR download_status = ?", "pending", "failed").Find(&pendingVersions).Error; err != nil {
		return fmt.Errorf("failed to fetch pending firmware versions: %w", err)
	}

	if len(pendingVersions) == 0 {
		logging.Logf("[FIRMWARE DOWNLOADS] No pending downloads found")
		return nil
	}

	// Check if auto-download is enabled
	autoDownload := config.Get("FIRMWARE_AUTO_DOWNLOAD", "true") == "true"
	if !autoDownload {
		logging.Logf("[FIRMWARE DOWNLOADS] Auto-download disabled, skipping downloads")
		return nil
	}

	// Download each pending version
	for _, version := range pendingVersions {
		logging.Logf("[FIRMWARE DOWNLOADS] Starting download for version %s", version.Version)
		if err := p.downloadFirmwareFile(ctx, &version); err != nil {
			logging.Logf("[FIRMWARE DOWNLOADS] Failed to download version %s: %v", version.Version, err)
			// Continue with other downloads even if one fails
		}
	}

	logging.Logf("[FIRMWARE DOWNLOADS] Completed pending downloads")
	return nil
}

// DownloadFirmware directly downloads a specific firmware version (for retry functionality)
func (p *FirmwarePoller) DownloadFirmware(ctx context.Context, firmware *database.FirmwareVersion) error {
	return p.downloadFirmwareFile(ctx, firmware)
}

// poll performs the firmware polling operation
func (p *FirmwarePoller) poll(ctx context.Context) error {
	logging.Logf("[FIRMWARE POLLER] Starting firmware update check")

	// Ensure storage directory exists
	if err := os.MkdirAll(p.storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Fetch latest firmware information from API
	versionInfo, err := p.fetchLatestFirmwareVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch latest firmware version: %w", err)
	}

	logging.Logf("[FIRMWARE POLLER] Found firmware version %s", versionInfo.Version)

	// Process the latest version
	if err := p.processFirmwareVersion(ctx, *versionInfo); err != nil {
		return fmt.Errorf("error processing version %s: %w", versionInfo.Version, err)
	}

	logging.Logf("[FIRMWARE POLLER] Firmware update check completed")
	return nil
}

// fetchLatestFirmwareVersion fetches the latest firmware version information from the API
func (p *FirmwarePoller) fetchLatestFirmwareVersion(ctx context.Context) (*FirmwareVersionInfo, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var versionInfo FirmwareVersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	return &versionInfo, nil
}

// discoverFirmwareVersion processes a firmware version for discovery only (no downloads)
func (p *FirmwarePoller) discoverFirmwareVersion(ctx context.Context, versionInfo FirmwareVersionInfo) error {
	// Check if this version already exists in database
	var existingVersion database.FirmwareVersion
	err := p.db.Where("version = ?", versionInfo.Version).First(&existingVersion).Error

	if err == nil {
		// Version exists - just make sure it's marked as latest (and others aren't)
		if !existingVersion.IsLatest {
			return p.updateLatestVersion(versionInfo.Version)
		}
		return nil // Version exists and is properly marked
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error: %w", err)
	}

	// New version, create database record with pending status
	firmwareVersion := database.FirmwareVersion{
		Version:          versionInfo.Version,
		DownloadURL:      versionInfo.URL,
		IsLatest:         true,
		IsDownloaded:     false,
		DownloadStatus:   "pending",
		DownloadProgress: 0,
		ReleasedAt:       time.Now(), // Set to now since we don't get this from API
	}

	if err := p.db.Create(&firmwareVersion).Error; err != nil {
		return fmt.Errorf("failed to create firmware version: %w", err)
	}

	logging.Logf("[FIRMWARE DISCOVERY] Added new firmware version: %s (status: pending)", versionInfo.Version)

	// Update latest flags (mark others as not latest)
	if err := p.updateLatestVersion(versionInfo.Version); err != nil {
		logging.Logf("[FIRMWARE DISCOVERY] Error updating latest version flag: %v", err)
	}

	return nil
}

// processFirmwareVersion processes a single firmware version
func (p *FirmwarePoller) processFirmwareVersion(ctx context.Context, versionInfo FirmwareVersionInfo) error {
	// Check if this version already exists in database
	var existingVersion database.FirmwareVersion
	err := p.db.Where("version = ?", versionInfo.Version).First(&existingVersion).Error

	if err == nil {
		// Version exists - check if it's actually downloaded (only in download mode)
		if !existingVersion.IsDownloaded && p.firmwareMode == "download" {
			// Version exists in DB but file not downloaded - download it now
			logging.Logf("[FIRMWARE POLLER] Version %s exists but not downloaded, downloading now", versionInfo.Version)
			autoDownload := config.Get("FIRMWARE_AUTO_DOWNLOAD", "true") == "true"
			if autoDownload {
				if err := p.downloadFirmwareFile(ctx, &existingVersion); err != nil {
					logging.Logf("[FIRMWARE POLLER] Failed to download existing firmware %s: %v", versionInfo.Version, err)
				}
			}
		}

		// Make sure it's marked as latest (and others aren't)
		if !existingVersion.IsLatest {
			return p.updateLatestVersion(versionInfo.Version)
		}
		return nil // Version exists and is properly handled
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error: %w", err)
	}

	// New version, create database record
	firmwareVersion := database.FirmwareVersion{
		Version:          versionInfo.Version,
		DownloadURL:      versionInfo.URL,
		IsLatest:         true,
		IsDownloaded:     false,
		DownloadStatus:   "pending",
		DownloadProgress: 0,
		ReleasedAt:       time.Now(), // Set to now since we don't get this from API
	}

	if err := p.db.Create(&firmwareVersion).Error; err != nil {
		return fmt.Errorf("failed to create firmware version: %w", err)
	}

	logging.Logf("[FIRMWARE POLLER] Added new firmware version: %s", versionInfo.Version)

	// Update latest flags (mark others as not latest)
	if err := p.updateLatestVersion(versionInfo.Version); err != nil {
		logging.Logf("[FIRMWARE POLLER] Error updating latest version flag: %v", err)
	}

	// Optionally download firmware file (only in download mode)
	autoDownload := config.Get("FIRMWARE_AUTO_DOWNLOAD", "true") == "true"
	if autoDownload && p.firmwareMode == "download" {
		if err := p.downloadFirmwareFile(ctx, &firmwareVersion); err != nil {
			logging.Logf("[FIRMWARE POLLER] Failed to download firmware %s: %v", versionInfo.Version, err)
		}
	}

	return nil
}

// updateLatestVersion ensures only one version is marked as latest
func (p *FirmwarePoller) updateLatestVersion(latestVersion string) error {
	tx := p.db.Begin()

	// Clear all latest flags
	if err := tx.Model(&database.FirmwareVersion{}).Where("1 = 1").Update("is_latest", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Set the new latest version
	if err := tx.Model(&database.FirmwareVersion{}).Where("version = ?", latestVersion).Update("is_latest", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// downloadFirmwareFile downloads a firmware file to local storage with progress tracking
func (p *FirmwarePoller) downloadFirmwareFile(ctx context.Context, firmware *database.FirmwareVersion) error {
	filename := fmt.Sprintf("firmware_%s.bin", firmware.Version)
	filePath := filepath.Join(p.storageDir, filename)

	// Skip if already downloaded
	if firmware.IsDownloaded && firmware.FilePath != "" {
		if _, err := os.Stat(firmware.FilePath); err == nil {
			// Make sure status is set correctly
			firmware.DownloadStatus = "downloaded"
			firmware.DownloadProgress = 100
			p.db.Save(firmware)
			return nil
		}
	}

	// Set status to downloading
	firmware.DownloadStatus = "downloading"
	firmware.DownloadProgress = 0
	firmware.DownloadError = ""
	if err := p.db.Save(firmware).Error; err != nil {
		logging.Logf("[FIRMWARE POLLER] Failed to update download status: %v", err)
	}

	logging.Logf("[FIRMWARE POLLER] Downloading firmware %s", firmware.Version)

	// Create HTTP client with context
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", firmware.DownloadURL, nil)
	if err != nil {
		p.markDownloadFailed(firmware, fmt.Sprintf("Failed to create request: %v", err))
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		p.markDownloadFailed(firmware, fmt.Sprintf("HTTP request failed: %v", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Download failed with status %d", resp.StatusCode)
		p.markDownloadFailed(firmware, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Get content length for progress tracking
	contentLength := resp.ContentLength

	// Create output file
	outFile, err := os.Create(filePath)
	if err != nil {
		p.markDownloadFailed(firmware, fmt.Sprintf("Failed to create file: %v", err))
		return err
	}
	defer outFile.Close()

	// Copy data while calculating SHA256 and tracking progress
	hasher := sha256.New()
	teeReader := io.TeeReader(resp.Body, hasher)

	// Wrap reader with progress tracking
	progressReader := &progressReader{
		reader:       teeReader,
		total:        contentLength,
		firmware:     firmware,
		db:           p.db,
		lastProgress: 0,
	}

	_, err = io.Copy(outFile, progressReader)
	if err != nil {
		os.Remove(filePath) // Clean up on error
		p.markDownloadFailed(firmware, fmt.Sprintf("Download failed: %v", err))
		return err
	}

	// Verify checksum if provided
	if firmware.SHA256 != "" {
		calculatedHash := hex.EncodeToString(hasher.Sum(nil))
		if calculatedHash != firmware.SHA256 {
			os.Remove(filePath) // Clean up corrupted file
			errMsg := fmt.Sprintf("Checksum mismatch: expected %s, got %s", firmware.SHA256, calculatedHash)
			p.markDownloadFailed(firmware, errMsg)
			return fmt.Errorf("%s", errMsg)
		}
	}

	// Update database record - success
	firmware.FilePath = filePath
	firmware.IsDownloaded = true
	firmware.DownloadStatus = "downloaded"
	firmware.DownloadProgress = 100
	firmware.DownloadError = ""
	if err := p.db.Save(firmware).Error; err != nil {
		return err
	}

	logging.Logf("[FIRMWARE POLLER] Successfully downloaded firmware %s to %s", firmware.Version, filePath)
	return nil
}

// markDownloadFailed updates the firmware record with failed status
func (p *FirmwarePoller) markDownloadFailed(firmware *database.FirmwareVersion, errorMsg string) {
	firmware.DownloadStatus = "failed"
	firmware.DownloadError = errorMsg
	firmware.IsDownloaded = false
	if err := p.db.Save(firmware).Error; err != nil {
		logging.Logf("[FIRMWARE POLLER] Failed to update failed status: %v", err)
	}
}

// progressReader wraps an io.Reader and tracks download progress
type progressReader struct {
	reader       io.Reader
	total        int64
	read         int64
	firmware     *database.FirmwareVersion
	db           *gorm.DB
	lastProgress int
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)

	// Update progress every 5% to avoid too many database updates
	if pr.total > 0 {
		progress := int((pr.read * 100) / pr.total)
		if progress >= pr.lastProgress+5 || progress == 100 {
			pr.firmware.DownloadProgress = progress
			if dbErr := pr.db.Save(pr.firmware).Error; dbErr == nil {
				pr.lastProgress = progress
				logging.Logf("[FIRMWARE DOWNLOAD] Progress for %s: %d%%", pr.firmware.Version, progress)
			}
		}
	}

	return n, err
}
