package export

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"gorm.io/gorm"
)

// Importer handles restoring from backup archives
type Importer struct {
	db      *gorm.DB
	dataDir string
}

// NewImporter creates a new importer instance
func NewImporter(db *gorm.DB, dataDir string) *Importer {
	return &Importer{
		db:      db,
		dataDir: dataDir,
	}
}

// Import restores from a backup archive
func (i *Importer) Import(archivePath string, options ImportOptions) (*ExportMetadata, error) {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "stationmaster-import-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract archive
	if err := ExtractTarGz(archivePath, tempDir); err != nil {
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	// Read metadata
	metadataPath := filepath.Join(tempDir, "metadata.json")
	var metadata ExportMetadata
	if err := readJSON(metadataPath, &metadata); err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Validate compatibility (optional - you might want version checks here)
	if err := i.validateMetadata(&metadata); err != nil {
		return nil, fmt.Errorf("backup validation failed: %w", err)
	}

	// Import database if present
	dbDir := filepath.Join(tempDir, "database")
	if _, err := os.Stat(dbDir); err == nil {
		if err := i.importDatabase(dbDir, options); err != nil {
			return nil, fmt.Errorf("failed to import database: %w", err)
		}
	}

	return &metadata, nil
}

// ImportFromExtractedDirectory imports from an already-extracted directory
func (i *Importer) ImportFromExtractedDirectory(extractedDir string, options ImportOptions) (*ExportMetadata, error) {
	// Read metadata
	metadataPath := filepath.Join(extractedDir, "metadata.json")
	var metadata ExportMetadata
	if err := readJSON(metadataPath, &metadata); err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Validate compatibility
	if err := i.validateMetadata(&metadata); err != nil {
		return nil, fmt.Errorf("backup validation failed: %w", err)
	}

	// Import database if present
	dbDir := filepath.Join(extractedDir, "database")
	if _, err := os.Stat(dbDir); err == nil {
		if err := i.importDatabase(dbDir, options); err != nil {
			return nil, fmt.Errorf("failed to import database: %w", err)
		}
	}

	return &metadata, nil
}

// validateMetadata checks if the backup is compatible
func (i *Importer) validateMetadata(metadata *ExportMetadata) error {
	// Basic validation - you might want more sophisticated version checking
	if metadata.ExportTimestamp.IsZero() {
		return fmt.Errorf("invalid backup: missing export timestamp")
	}

	// Could add version compatibility checks here
	return nil
}

// importDatabase imports all database tables from JSON files
func (i *Importer) importDatabase(dbDir string, options ImportOptions) error {
	// Get list of JSON files in database directory
	files, err := filepath.Glob(filepath.Join(dbDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list database files: %w", err)
	}

	// Clear existing data if overwrite is enabled
	if options.OverwriteDatabase {
		if err := i.clearDatabase(); err != nil {
			return fmt.Errorf("failed to clear database: %w", err)
		}
	}

	// Import each table
	for _, file := range files {
		tableName := strings.TrimSuffix(filepath.Base(file), ".json")
		if err := i.importTable(file, tableName, options); err != nil {
			return fmt.Errorf("failed to import table %s: %w", tableName, err)
		}
	}

	return nil
}

// importTable imports a single table from a JSON file
func (i *Importer) importTable(jsonFile, tableName string, options ImportOptions) error {
	// Read JSON data
	var records []map[string]interface{}
	if err := readJSON(jsonFile, &records); err != nil {
		return fmt.Errorf("failed to read %s: %w", jsonFile, err)
	}

	if len(records) == 0 {
		fmt.Printf("[IMPORT] Skipping empty table: %s\n", tableName)
		return nil
	}

	// Get the model for this table
	model := getModelForTable(tableName)
	if model == nil {
		fmt.Printf("[IMPORT] Warning: Unknown table %s, skipping\n", tableName)
		return nil
	}

	// Insert records using raw SQL to preserve IDs and timestamps
	for _, record := range records {
		// Filter by user ID if specified and record has user_id field
		if len(options.UserIDs) > 0 {
			if userIDStr, ok := record["user_id"].(string); ok {
				// Check if this user should be imported
				shouldImport := false
				for _, allowedUserID := range options.UserIDs {
					if userIDStr == allowedUserID.String() {
						shouldImport = true
						break
					}
				}
				if !shouldImport {
					continue
				}
			}
		}

		// Use GORM to create the record, which handles type conversion properly
		if err := i.db.Table(tableName).Create(record).Error; err != nil {
			// Skip duplicates in non-overwrite mode
			if !options.OverwriteDatabase && strings.Contains(err.Error(), "UNIQUE") {
				continue
			}
			return fmt.Errorf("failed to insert record into %s: %w", tableName, err)
		}
	}

	fmt.Printf("[IMPORT] Imported %d records into %s\n", len(records), tableName)
	return nil
}

// clearDatabase removes all existing data
func (i *Importer) clearDatabase() error {
	models := database.GetAllModels()

	// Delete in reverse order to handle foreign key constraints
	for idx := len(models) - 1; idx >= 0; idx-- {
		model := models[idx]
		tableName := getTableName(model)

		// Use Unscoped() to permanently delete (not soft delete)
		if err := i.db.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			return fmt.Errorf("failed to clear table %s: %w", tableName, err)
		}
		fmt.Printf("[IMPORT] Cleared table: %s\n", tableName)
	}

	return nil
}

// getModelForTable returns the GORM model for a table name
func getModelForTable(tableName string) interface{} {
	switch tableName {
	case "users":
		return &database.User{}
	case "api_keys":
		return &database.APIKey{}
	case "user_sessions":
		return &database.UserSession{}
	case "system_settings":
		return &database.SystemSetting{}
	case "login_attempts":
		return &database.LoginAttempt{}
	case "backup_jobs":
		return &database.BackupJob{}
	case "restore_uploads":
		return &database.RestoreUpload{}
	case "restore_extraction_jobs":
		return &database.RestoreExtractionJob{}
	default:
		return nil
	}
}

// ExtractTarGz extracts a tar.gz archive to a destination directory
func ExtractTarGz(archivePath, destDir string) error {
	// Open archive
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Calculate destination path
		destPath := filepath.Join(destDir, header.Name)

		// Security check: ensure path is within destination directory
		if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(destPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}

		case tar.TypeReg:
			// Create file
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", destPath, err)
			}

			outFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", destPath, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", destPath, err)
			}

			outFile.Close()

			// Set file permissions
			if err := os.Chmod(destPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set permissions for %s: %w", destPath, err)
			}
		}
	}

	return nil
}

// readJSON reads JSON data from a file
func readJSON(filename string, data interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(data)
}
