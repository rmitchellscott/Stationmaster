package pollers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

const s3BaseURL = "https://trmnl-fw.s3.us-east-2.amazonaws.com"
const defaultManifestURL = "https://trmnl.com/firmware/releases.json"

type FirmwarePoller struct {
	*BasePoller
	db           *gorm.DB
	manifestURL  string
	s3BucketURL  string
	storageDir   string
	firmwareMode string
}

type FirmwareManifestEntry struct {
	ChipFamily string   `json:"chipFamily"`
	Label      string   `json:"label"`
	Versions   []string `json:"versions"`
}

type s3ListResult struct {
	XMLName  xml.Name  `xml:"ListBucketResult"`
	Contents []s3Entry `xml:"Contents"`
}

type s3Entry struct {
	Key          string    `xml:"Key"`
	LastModified time.Time `xml:"LastModified"`
	Size         int64     `xml:"Size"`
}

type discoveredVersion struct {
	Family      string
	Version     string
	RawVersion  string
	DownloadURL string
	ReleasedAt  time.Time
	FileSize    int64
	IsStable    bool
	ChipFamily  string
	Label       string
}

func NewFirmwarePoller(db *gorm.DB) *FirmwarePoller {
	interval := 6 * time.Hour
	if envInterval := config.Get("FIRMWARE_POLLER_INTERVAL", ""); envInterval != "" {
		if d, err := time.ParseDuration(envInterval); err == nil {
			interval = d
		}
	}

	enabled := config.Get("FIRMWARE_POLLER", "true") != "false"
	manifestURL := config.Get("TRMNL_FIRMWARE_MANIFEST_URL", defaultManifestURL)
	if legacy := config.Get("TRMNL_FIRMWARE_API_URL", ""); legacy != "" && config.Get("TRMNL_FIRMWARE_MANIFEST_URL", "") == "" {
		manifestURL = legacy
	}
	storageDir := config.Get("FIRMWARE_STORAGE_DIR", "/data/firmware")
	firmwareMode := config.Get("FIRMWARE_MODE", "proxy")

	pollerConfig := PollerConfig{
		Name:       "firmware",
		Interval:   interval,
		Enabled:    enabled,
		MaxRetries: 3,
		RetryDelay: 30 * time.Second,
		Timeout:    2 * time.Minute,
	}

	poller := &FirmwarePoller{
		db:           db,
		manifestURL:  manifestURL,
		s3BucketURL:  s3BaseURL,
		storageDir:   storageDir,
		firmwareMode: firmwareMode,
	}

	poller.BasePoller = NewBasePoller(pollerConfig, poller.poll)
	return poller
}

func (p *FirmwarePoller) ExecutePoll(ctx context.Context) error {
	return p.poll(ctx)
}

func (p *FirmwarePoller) DiscoverFirmware(ctx context.Context) error {
	logging.Info("[FIRMWARE DISCOVERY] Starting firmware discovery")

	if err := os.MkdirAll(p.storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	versions, err := p.discoverAllVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover firmware versions: %w", err)
	}

	created := 0
	for _, v := range versions {
		if err := p.upsertFirmwareVersion(ctx, v); err != nil {
			logging.Error("[FIRMWARE DISCOVERY] Error upserting version", "family", v.Family, "version", v.Version, "error", err)
			continue
		}
		created++
	}

	logging.Info("[FIRMWARE DISCOVERY] Firmware discovery completed", "versions", created)
	return nil
}

func (p *FirmwarePoller) StartPendingDownloads(ctx context.Context) error {
	if p.firmwareMode != "download" {
		return nil
	}

	var pending []database.FirmwareVersion
	if err := p.db.Where("download_status IN ?", []string{"pending", "failed"}).Find(&pending).Error; err != nil {
		return fmt.Errorf("failed to fetch pending firmware versions: %w", err)
	}

	if len(pending) == 0 {
		return nil
	}

	if config.Get("FIRMWARE_AUTO_DOWNLOAD", "true") != "true" {
		return nil
	}

	for _, version := range pending {
		if err := p.downloadFirmwareFile(ctx, &version); err != nil {
			logging.Error("[FIRMWARE DOWNLOADS] Failed", "family", version.ModelFamily, "version", version.Version, "error", err)
		}
	}
	return nil
}

func (p *FirmwarePoller) DownloadFirmware(ctx context.Context, firmware *database.FirmwareVersion) error {
	return p.downloadFirmwareFile(ctx, firmware)
}

func (p *FirmwarePoller) poll(ctx context.Context) error {
	logging.Info("[FIRMWARE POLLER] Starting firmware update check")

	if err := os.MkdirAll(p.storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	versions, err := p.discoverAllVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover firmware versions: %w", err)
	}

	for _, v := range versions {
		if err := p.upsertFirmwareVersion(ctx, v); err != nil {
			logging.Error("[FIRMWARE POLLER] Error upserting version", "family", v.Family, "version", v.Version, "error", err)
		}
	}

	if p.firmwareMode == "download" && config.Get("FIRMWARE_AUTO_DOWNLOAD", "true") == "true" {
		p.StartPendingDownloads(ctx)
	}

	logging.Info("[FIRMWARE POLLER] Firmware update check completed")
	return nil
}

// discoverAllVersions merges S3 bucket listing with releases.json manifest.
// S3 is the source of truth for what exists. releases.json marks which are stable.
func (p *FirmwarePoller) discoverAllVersions(ctx context.Context) ([]discoveredVersion, error) {
	// Fetch S3 listing — source of truth for available binaries
	s3Versions, err := p.fetchS3Versions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch S3 listing: %w", err)
	}

	// Fetch manifest — marks stable versions and provides metadata
	manifest, err := p.fetchManifest(ctx)
	if err != nil {
		logging.Warn("[FIRMWARE POLLER] Failed to fetch manifest, using S3 only", "error", err)
		manifest = nil
	}

	// Build stable version set from manifest
	stableSet := map[string]map[string]bool{}          // family -> version -> true
	familyMeta := map[string]FirmwareManifestEntry{}    // family -> metadata
	latestStable := map[string]string{}                 // family -> latest stable version
	if manifest != nil {
		for family, entry := range manifest {
			stableSet[family] = map[string]bool{}
			familyMeta[family] = entry
			for _, v := range entry.Versions {
				stableSet[family][cleanVersion(v)] = true
			}
			if len(entry.Versions) > 0 {
				latestStable[family] = cleanVersion(entry.Versions[len(entry.Versions)-1])
			}
		}
	}

	// Merge: S3 versions enriched with manifest metadata
	var results []discoveredVersion
	latestByFamily := map[string]string{} // track latest version per family from S3

	for _, sv := range s3Versions {
		// Determine if latest in S3 (by date — s3Versions are already sorted newest first per family)
		if _, exists := latestByFamily[sv.Family]; !exists {
			latestByFamily[sv.Family] = sv.Version
		}

		stable := false
		if familyStable, ok := stableSet[sv.Family]; ok {
			stable = familyStable[sv.Version]
		}

		meta := familyMeta[sv.Family]

		results = append(results, discoveredVersion{
			Family:      sv.Family,
			Version:     sv.Version,
			RawVersion:  sv.RawVersion,
			DownloadURL: sv.DownloadURL,
			ReleasedAt:  sv.ReleasedAt,
			FileSize:    sv.FileSize,
			IsStable:    stable,
			ChipFamily:  meta.ChipFamily,
			Label:       meta.Label,
		})
	}

	return results, nil
}

type s3Version struct {
	Family      string
	Version     string
	RawVersion  string
	DownloadURL string
	ReleasedAt  time.Time
	FileSize    int64
}

func (p *FirmwarePoller) fetchS3Versions(ctx context.Context) ([]s3Version, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", p.s3BucketURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("S3 returned status %d", resp.StatusCode)
	}

	var listing s3ListResult
	if err := xml.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, fmt.Errorf("failed to decode S3 listing: %w", err)
	}

	var versions []s3Version
	for _, entry := range listing.Contents {
		if !strings.HasSuffix(entry.Key, ".bin") || entry.Size == 0 {
			continue
		}

		family, rawVersion := parseS3Key(entry.Key)
		if rawVersion == "" {
			continue
		}

		versions = append(versions, s3Version{
			Family:      family,
			Version:     cleanVersion(rawVersion),
			RawVersion:  rawVersion,
			DownloadURL: fmt.Sprintf("%s/%s", p.s3BucketURL, entry.Key),
			ReleasedAt:  entry.LastModified,
			FileSize:    entry.Size,
		})
	}

	return versions, nil
}

// parseS3Key extracts family and version from S3 key.
// "FW1.7.8.bin" -> ("trmnl", "FW1.7.8")
// "trmnl_x/FW1.7.7.bin" -> ("trmnl_x", "FW1.7.7")
func parseS3Key(key string) (string, string) {
	key = strings.TrimSuffix(key, ".bin")
	if idx := strings.LastIndex(key, "/"); idx >= 0 {
		family := key[:idx]
		version := key[idx+1:]
		if version == "" {
			return "", ""
		}
		return family, version
	}
	return "trmnl", key
}

func (p *FirmwarePoller) fetchManifest(ctx context.Context) (map[string]FirmwareManifestEntry, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", p.manifestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest returned status %d", resp.StatusCode)
	}

	var manifest map[string]FirmwareManifestEntry
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	return manifest, nil
}

func cleanVersion(version string) string {
	return strings.TrimPrefix(version, "FW")
}

func (p *FirmwarePoller) upsertFirmwareVersion(ctx context.Context, v discoveredVersion) error {
	var existing database.FirmwareVersion
	err := p.db.Where("version = ? AND model_family = ?", v.Version, v.Family).First(&existing).Error

	if err == nil {
		changed := false
		if existing.IsStable != v.IsStable {
			existing.IsStable = v.IsStable
			changed = true
		}
		if existing.ChipFamily != v.ChipFamily && v.ChipFamily != "" {
			existing.ChipFamily = v.ChipFamily
			changed = true
		}
		if existing.FamilyLabel != v.Label && v.Label != "" {
			existing.FamilyLabel = v.Label
			changed = true
		}
		if existing.FileSize != v.FileSize && v.FileSize > 0 {
			existing.FileSize = v.FileSize
			changed = true
		}
		if changed {
			return p.db.Save(&existing).Error
		}
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error: %w", err)
	}

	fw := database.FirmwareVersion{
		Version:        v.Version,
		ModelFamily:    v.Family,
		ChipFamily:     v.ChipFamily,
		FamilyLabel:    v.Label,
		DownloadURL:    v.DownloadURL,
		FileSize:       v.FileSize,
		IsLatest:       false,
		IsStable:       v.IsStable,
		IsDownloaded:   false,
		DownloadStatus: "pending",
		ReleasedAt:     v.ReleasedAt,
	}

	if err := p.db.Create(&fw).Error; err != nil {
		return fmt.Errorf("failed to create firmware version: %w", err)
	}

	logging.Info("[FIRMWARE POLLER] Added firmware version", "family", v.Family, "version", v.Version, "stable", v.IsStable)

	// Update is_latest: the most recently released stable version per family
	return p.updateLatestForFamily(v.Family)
}

func (p *FirmwarePoller) updateLatestForFamily(family string) error {
	tx := p.db.Begin()

	// Clear latest for this family
	if err := tx.Model(&database.FirmwareVersion{}).Where("model_family = ?", family).Update("is_latest", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Find most recent stable version for this family
	var latest database.FirmwareVersion
	err := tx.Where("model_family = ? AND is_stable = ?", family, true).Order("released_at DESC").First(&latest).Error
	if err == gorm.ErrRecordNotFound {
		// No stable versions — use most recent overall
		err = tx.Where("model_family = ?", family).Order("released_at DESC").First(&latest).Error
	}
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return tx.Commit().Error
		}
		return err
	}

	if err := tx.Model(&database.FirmwareVersion{}).Where("id = ?", latest.ID).Update("is_latest", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (p *FirmwarePoller) downloadFirmwareFile(ctx context.Context, firmware *database.FirmwareVersion) error {
	familyDir := p.storageDir
	if firmware.ModelFamily != "" && firmware.ModelFamily != "trmnl" {
		familyDir = filepath.Join(p.storageDir, firmware.ModelFamily)
	}
	if err := os.MkdirAll(familyDir, 0755); err != nil {
		return fmt.Errorf("failed to create family directory: %w", err)
	}

	filename := fmt.Sprintf("firmware_%s.bin", firmware.Version)
	filePath := filepath.Join(familyDir, filename)

	if firmware.IsDownloaded && firmware.FilePath != "" {
		if _, err := os.Stat(firmware.FilePath); err == nil {
			firmware.DownloadStatus = "downloaded"
			firmware.DownloadProgress = 100
			p.db.Save(firmware)
			return nil
		}
	}

	firmware.DownloadStatus = "downloading"
	firmware.DownloadProgress = 0
	firmware.DownloadError = ""
	p.db.Save(firmware)

	logging.Info("[FIRMWARE POLLER] Downloading firmware", "family", firmware.ModelFamily, "version", firmware.Version)

	client := &http.Client{Timeout: 5 * time.Minute}
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

	outFile, err := os.Create(filePath)
	if err != nil {
		p.markDownloadFailed(firmware, fmt.Sprintf("Failed to create file: %v", err))
		return err
	}
	defer outFile.Close()

	hasher := sha256.New()
	teeReader := io.TeeReader(resp.Body, hasher)

	pr := &progressReader{
		reader:       teeReader,
		total:        resp.ContentLength,
		firmware:     firmware,
		db:           p.db,
		lastProgress: 0,
	}

	written, err := io.Copy(outFile, pr)
	if err != nil {
		os.Remove(filePath)
		p.markDownloadFailed(firmware, fmt.Sprintf("Download failed: %v", err))
		return err
	}

	if firmware.SHA256 != "" {
		calculated := hex.EncodeToString(hasher.Sum(nil))
		if calculated != firmware.SHA256 {
			os.Remove(filePath)
			errMsg := fmt.Sprintf("Checksum mismatch: expected %s, got %s", firmware.SHA256, calculated)
			p.markDownloadFailed(firmware, errMsg)
			return fmt.Errorf("%s", errMsg)
		}
	}

	firmware.FilePath = filePath
	firmware.FileSize = written
	firmware.IsDownloaded = true
	firmware.DownloadStatus = "downloaded"
	firmware.DownloadProgress = 100
	firmware.DownloadError = ""
	firmware.SHA256 = hex.EncodeToString(hasher.Sum(nil))
	if err := p.db.Save(firmware).Error; err != nil {
		return err
	}

	logging.Info("[FIRMWARE POLLER] Downloaded firmware", "family", firmware.ModelFamily, "version", firmware.Version, "path", filePath)
	return nil
}

func (p *FirmwarePoller) markDownloadFailed(firmware *database.FirmwareVersion, errorMsg string) {
	firmware.DownloadStatus = "failed"
	firmware.DownloadError = errorMsg
	firmware.IsDownloaded = false
	p.db.Save(firmware)
}

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

	if pr.total > 0 {
		progress := int((pr.read * 100) / pr.total)
		if progress >= pr.lastProgress+5 || progress == 100 {
			pr.firmware.DownloadProgress = progress
			if dbErr := pr.db.Save(pr.firmware).Error; dbErr == nil {
				pr.lastProgress = progress
			}
		}
	}

	return n, err
}
