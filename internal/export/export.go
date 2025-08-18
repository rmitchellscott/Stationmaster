package export

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"gorm.io/gorm"
)

// ExportMetadata contains information about the export
type ExportMetadata struct {
	StationmasterVersion string    `json:"stationmaster_version"`
	GitCommit            string    `json:"git_commit"`
	ExportTimestamp      time.Time `json:"export_timestamp"`
	DatabaseType         string    `json:"database_type"`
	UsersExported        []string  `json:"users_exported"`
	TotalUsers           int       `json:"total_users"`
	TotalAPIKeys         int       `json:"total_api_keys"`
	ExportedTables       []string  `json:"exported_tables"`
}

// ExportOptions configures what to include in the export
type ExportOptions struct {
	IncludeDatabase bool
	IncludeFiles    bool
	IncludeConfigs  bool
	UserIDs         []uuid.UUID // If specified, only export these users
}

// ImportOptions configures how to handle the import
type ImportOptions struct {
	OverwriteFiles    bool
	OverwriteDatabase bool
	UserIDs           []uuid.UUID // If specified, only import these users
}

// Exporter handles creating complete backups
type Exporter struct {
	db      *gorm.DB
	dataDir string
}

// NewExporter creates a new exporter instance
func NewExporter(db *gorm.DB, dataDir string) *Exporter {
	return &Exporter{
		db:      db,
		dataDir: dataDir,
	}
}

// Export creates a complete backup archive
func (e *Exporter) Export(outputPath string, options ExportOptions) error {
	// Create temporary directory for staging
	tempDir, err := os.MkdirTemp("", "stationmaster-export-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Create directory structure
	dbDir := filepath.Join(tempDir, "database")

	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	var metadata ExportMetadata
	var exportedUsers []string

	// Export database if requested
	if options.IncludeDatabase {
		if err := e.exportDatabase(dbDir, &metadata, options); err != nil {
			return fmt.Errorf("failed to export database: %w", err)
		}
	}

	// Get exported user list
	if len(options.UserIDs) > 0 {
		for _, userID := range options.UserIDs {
			exportedUsers = append(exportedUsers, userID.String())
		}
	} else {
		// Export all users
		var users []database.User
		if err := e.db.Find(&users).Error; err == nil {
			for _, user := range users {
				exportedUsers = append(exportedUsers, user.ID.String())
			}
		}
	}

	// Create metadata
	metadata.StationmasterVersion = "1.0.0" // TODO: Get from version package
	metadata.GitCommit = "unknown"          // TODO: Get from version package
	metadata.ExportTimestamp = time.Now().UTC()
	metadata.DatabaseType = "sqlite" // TODO: Get from database config
	metadata.UsersExported = exportedUsers

	// Write metadata
	metadataFile := filepath.Join(tempDir, "metadata.json")
	if err := writeJSON(metadataFile, metadata); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Create compressed archive
	if err := createTarGz(tempDir, outputPath); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	return nil
}

// exportDatabase exports all database tables to JSON files
func (e *Exporter) exportDatabase(dbDir string, metadata *ExportMetadata, options ExportOptions) error {
	models := database.GetAllModels()
	var exportedTables []string

	for _, model := range models {
		tableName := getTableName(model)

		// Get all records for this model
		var records []map[string]interface{}

		query := e.db

		// Filter by user ID if specified and model has UserID field
		if len(options.UserIDs) > 0 && hasUserIDField(model) {
			query = query.Where("user_id IN ?", options.UserIDs)
		}

		if err := query.Model(model).Find(&records).Error; err != nil {
			return fmt.Errorf("failed to export table %s: %w", tableName, err)
		}

		// Write to JSON file
		outputFile := filepath.Join(dbDir, tableName+".json")
		if err := writeJSON(outputFile, records); err != nil {
			return fmt.Errorf("failed to write %s: %w", tableName, err)
		}

		exportedTables = append(exportedTables, tableName)
	}

	metadata.ExportedTables = exportedTables

	// Count total users and API keys for metadata
	var userCount int64
	var apiKeyCount int64

	// Count all users (not just exported ones)
	if err := e.db.Model(&database.User{}).Count(&userCount).Error; err != nil {
		fmt.Printf("[WARNING] Failed to count users for metadata: %v\n", err)
	}
	metadata.TotalUsers = int(userCount)

	// Count all API keys
	if err := e.db.Model(&database.APIKey{}).Count(&apiKeyCount).Error; err != nil {
		fmt.Printf("[WARNING] Failed to count API keys for metadata: %v\n", err)
	}
	metadata.TotalAPIKeys = int(apiKeyCount)

	return nil
}

// Helper functions

func getTableName(model interface{}) string {
	switch model.(type) {
	case *database.User:
		return "users"
	case *database.APIKey:
		return "api_keys"
	case *database.UserSession:
		return "user_sessions"
	case *database.SystemSetting:
		return "system_settings"
	case *database.LoginAttempt:
		return "login_attempts"
	case *database.BackupJob:
		return "backup_jobs"
	case *database.RestoreUpload:
		return "restore_uploads"
	case *database.RestoreExtractionJob:
		return "restore_extraction_jobs"
	default:
		return "unknown"
	}
}

func hasUserIDField(model interface{}) bool {
	switch model.(type) {
	case *database.User, *database.SystemSetting, *database.LoginAttempt:
		return false
	default:
		return true
	}
}

func writeJSON(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// addFileToTar adds a single file to the tar archive
func addFileToTar(tarWriter *tar.Writer, filePath, nameInArchive string) error {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Create tar header
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = nameInArchive

	// Write header
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	// Write file contents
	_, err = io.Copy(tarWriter, file)
	return err
}

func createTarGz(sourceDir, outputPath string) error {
	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// First, add metadata.json if it exists - MUST be first file in archive
	metadataPath := filepath.Join(sourceDir, "metadata.json")
	if _, err := os.Stat(metadataPath); err == nil {
		if err := addFileToTar(tarWriter, metadataPath, "metadata.json"); err != nil {
			return fmt.Errorf("failed to add metadata.json: %w", err)
		}
		fmt.Printf("[EXPORT] Added metadata.json as first file in archive\n")
	}

	// Walk through source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path for tar
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Skip metadata.json as we've already added it
		if relPath == "metadata.json" {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Write file contents for regular files
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
