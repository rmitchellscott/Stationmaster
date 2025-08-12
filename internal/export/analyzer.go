package export

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupAnalysis contains information about a backup file
type BackupAnalysis struct {
	Valid              bool                   `json:"valid"`
	Metadata           *ExportMetadata        `json:"metadata,omitempty"`
	ErrorMessage       string                 `json:"error_message,omitempty"`
	FileSize           int64                  `json:"file_size"`
	CompressedSize     int64                  `json:"compressed_size"`
	DatabaseTables     []TableInfo            `json:"database_tables,omitempty"`
	HasDatabase        bool                   `json:"has_database"`
	HasFilesystem      bool                   `json:"has_filesystem"`
	AnalysisTimestamp  time.Time              `json:"analysis_timestamp"`
	CompatibleVersion  bool                   `json:"compatible_version"`
	RecommendedAction  string                 `json:"recommended_action"`
	Warnings           []string               `json:"warnings,omitempty"`
}

// TableInfo contains information about a database table in the backup
type TableInfo struct {
	Name        string `json:"name"`
	RecordCount int    `json:"record_count"`
	HasData     bool   `json:"has_data"`
}

// AnalyzeBackup analyzes a backup file and returns information about its contents
func AnalyzeBackup(backupPath string) (*BackupAnalysis, error) {
	analysis := &BackupAnalysis{
		AnalysisTimestamp: time.Now().UTC(),
		FileSize:          0,
		CompressedSize:    0,
		Valid:             false,
	}

	// Get file info
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		analysis.ErrorMessage = fmt.Sprintf("Failed to stat backup file: %v", err)
		return analysis, nil
	}
	
	analysis.CompressedSize = fileInfo.Size()
	analysis.FileSize = fileInfo.Size() // For compressed files, this is the same

	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "stationmaster-analyze-*")
	if err != nil {
		analysis.ErrorMessage = fmt.Sprintf("Failed to create temp directory: %v", err)
		return analysis, nil
	}
	defer os.RemoveAll(tempDir)

	// Extract and analyze the archive
	if err := analyzeArchiveContents(backupPath, tempDir, analysis); err != nil {
		analysis.ErrorMessage = fmt.Sprintf("Failed to analyze archive: %v", err)
		return analysis, nil
	}

	// If we got this far, the backup is valid
	analysis.Valid = true
	analysis.RecommendedAction = "Backup is valid and ready to restore"

	return analysis, nil
}

// analyzeArchiveContents extracts and analyzes the contents of the backup archive
func analyzeArchiveContents(archivePath, tempDir string, analysis *BackupAnalysis) error {
	// Open and read the archive
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader (not a valid gzip file): %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Track what we find
	var hasMetadata bool

	// Read through the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Check for metadata.json (should be first file)
		if header.Name == "metadata.json" && !hasMetadata {
			// Extract and read metadata
			metadataPath := filepath.Join(tempDir, "metadata.json")
			if err := extractFileFromTar(tarReader, header, metadataPath); err != nil {
				return fmt.Errorf("failed to extract metadata: %w", err)
			}

			// Parse metadata
			var metadata ExportMetadata
			if err := readJSON(metadataPath, &metadata); err != nil {
				return fmt.Errorf("failed to parse metadata: %w", err)
			}

			hasMetadata = true
			analysis.Metadata = &metadata
			
			// Check version compatibility
			analysis.CompatibleVersion = true // For now, assume all versions are compatible
		}

		// Check for database directory
		if strings.HasPrefix(header.Name, "database/") {
			analysis.HasDatabase = true
		}

		// Check for filesystem directory  
		if strings.HasPrefix(header.Name, "filesystem/") {
			analysis.HasFilesystem = true
		}
	}

	if !hasMetadata {
		return fmt.Errorf("backup does not contain metadata.json")
	}

	// Extract database directory to analyze tables
	if analysis.HasDatabase {
		if err := extractDatabaseForAnalysis(archivePath, tempDir, analysis); err != nil {
			analysis.Warnings = append(analysis.Warnings, fmt.Sprintf("Failed to analyze database tables: %v", err))
		}
	}

	return nil
}

// extractDatabaseForAnalysis extracts just the database directory to analyze table contents
func extractDatabaseForAnalysis(archivePath, tempDir string, analysis *BackupAnalysis) error {
	// Re-open the archive to extract database files
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Extract only database files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Only extract database JSON files
		if strings.HasPrefix(header.Name, "database/") && strings.HasSuffix(header.Name, ".json") {
			destPath := filepath.Join(tempDir, header.Name)
			if err := extractFileFromTar(tarReader, header, destPath); err != nil {
				return err
			}
		}
	}

	// Analyze database tables
	dbDir := filepath.Join(tempDir, "database")
	if _, err := os.Stat(dbDir); err == nil {
		tables, err := analyzeDatabaseTables(dbDir)
		if err != nil {
			return err
		}
		analysis.DatabaseTables = tables
	}

	return nil
}

// analyzeDatabaseTables analyzes the JSON files in the database directory
func analyzeDatabaseTables(dbDir string) ([]TableInfo, error) {
	var tables []TableInfo

	// Get list of JSON files
	files, err := filepath.Glob(filepath.Join(dbDir, "*.json"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		tableName := strings.TrimSuffix(filepath.Base(file), ".json")
		
		// Read and count records
		var records []interface{}
		if err := readJSON(file, &records); err != nil {
			// If we can't read it, still include it but mark as having no data
			tables = append(tables, TableInfo{
				Name:        tableName,
				RecordCount: 0,
				HasData:     false,
			})
			continue
		}

		tables = append(tables, TableInfo{
			Name:        tableName,
			RecordCount: len(records),
			HasData:     len(records) > 0,
		})
	}

	return tables, nil
}

// extractFileFromTar extracts a single file from a tar reader
func extractFileFromTar(tarReader *tar.Reader, header *tar.Header, destPath string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Create destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Copy data
	_, err = io.Copy(outFile, tarReader)
	if err != nil {
		return err
	}

	// Set file permissions
	return os.Chmod(destPath, os.FileMode(header.Mode))
}

// AnalyzeBackupFromExtractedDir analyzes a backup from an already extracted directory
func AnalyzeBackupFromExtractedDir(extractedDir string) (*BackupAnalysis, error) {
	analysis := &BackupAnalysis{
		AnalysisTimestamp: time.Now().UTC(),
		Valid:             false,
	}

	// Check for metadata
	metadataPath := filepath.Join(extractedDir, "metadata.json")
	if _, err := os.Stat(metadataPath); err != nil {
		analysis.ErrorMessage = "No metadata.json found in extracted directory"
		return analysis, nil
	}

	// Read metadata
	var metadata ExportMetadata
	if err := readJSON(metadataPath, &metadata); err != nil {
		analysis.ErrorMessage = fmt.Sprintf("Failed to parse metadata: %v", err)
		return analysis, nil
	}

	analysis.Metadata = &metadata
	analysis.CompatibleVersion = true

	// Check for database directory
	dbDir := filepath.Join(extractedDir, "database")
	if _, err := os.Stat(dbDir); err == nil {
		analysis.HasDatabase = true
		
		// Analyze database tables
		tables, err := analyzeDatabaseTables(dbDir)
		if err != nil {
			analysis.Warnings = append(analysis.Warnings, fmt.Sprintf("Failed to analyze database tables: %v", err))
		} else {
			analysis.DatabaseTables = tables
		}
	}

	// Check for filesystem directory
	fsDir := filepath.Join(extractedDir, "filesystem")
	if _, err := os.Stat(fsDir); err == nil {
		analysis.HasFilesystem = true
	}

	analysis.Valid = true
	analysis.RecommendedAction = "Backup is valid and ready to restore"

	return analysis, nil
}