package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

const (
	// TRMNL file size limits
	MaxTemplateFileSize = 1024 * 1024 // 1MB per template file
	MaxTotalZipSize     = 10 * 1024 * 1024 // 10MB total ZIP size
)

// TRMNLZipService handles ZIP file creation and extraction for TRMNL compatibility
type TRMNLZipService struct {
	exportService *TRMNLExportService
}

// NewTRMNLZipService creates a new ZIP service
func NewTRMNLZipService() *TRMNLZipService {
	return &TRMNLZipService{
		exportService: NewTRMNLExportService(),
	}
}

// ZipExportData represents the data to be exported in a ZIP file
type ZipExportData struct {
	SettingsYAML   []byte
	FullTemplate   string
	HalfHorizontal string
	HalfVertical   string
	QuadrantTemplate string
}

// CreateTRMNLZip creates a TRMNL-compatible ZIP file from a PluginDefinition
func (s *TRMNLZipService) CreateTRMNLZip(def *database.PluginDefinition) (*bytes.Buffer, error) {
	// Generate settings.yml
	settingsYAML, err := s.exportService.GenerateSettingsYAML(def)
	if err != nil {
		return nil, fmt.Errorf("failed to generate settings.yml: %w", err)
	}

	// Create ZIP buffer
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add settings.yml (required)
	if err := s.addFileToZip(zipWriter, "settings.yml", settingsYAML); err != nil {
		zipWriter.Close()
		return nil, fmt.Errorf("failed to add settings.yml: %w", err)
	}

	// Add template files
	templates := map[string]string{
		"full.liquid":          getTemplateContent(def.MarkupFull),
		"half_horizontal.liquid": getTemplateContent(def.MarkupHalfHoriz),
		"half_vertical.liquid":   getTemplateContent(def.MarkupHalfVert),
		"quadrant.liquid":        getTemplateContent(def.MarkupQuadrant),
	}

	for filename, content := range templates {
		if content != "" {
			// Validate file size
			if len(content) > MaxTemplateFileSize {
				zipWriter.Close()
				return nil, fmt.Errorf("template file %s exceeds 1MB limit", filename)
			}

			if err := s.addFileToZip(zipWriter, filename, []byte(content)); err != nil {
				zipWriter.Close()
				return nil, fmt.Errorf("failed to add %s: %w", filename, err)
			}
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close ZIP writer: %w", err)
	}

	return buf, nil
}

// ExtractTRMNLZip extracts and validates a TRMNL-compatible ZIP file
func (s *TRMNLZipService) ExtractTRMNLZip(file multipart.File, header *multipart.FileHeader) (*ZipExportData, error) {
	logging.Info("[TRMNL IMPORT] Starting ZIP extraction", 
		"filename", header.Filename,
		"size", header.Size)

	// Check total file size
	if header.Size > MaxTotalZipSize {
		logging.Error("[TRMNL IMPORT] ZIP file too large", "size", header.Size, "max", MaxTotalZipSize)
		return nil, fmt.Errorf("ZIP file too large: %d bytes (max %d bytes)", header.Size, MaxTotalZipSize)
	}

	// Read file contents
	fileBytes := make([]byte, header.Size)
	if _, err := io.ReadFull(file, fileBytes); err != nil {
		logging.Error("[TRMNL IMPORT] Failed to read uploaded file", "error", err)
		return nil, fmt.Errorf("failed to read uploaded file: %w", err)
	}

	// Create ZIP reader
	zipReader, err := zip.NewReader(bytes.NewReader(fileBytes), header.Size)
	if err != nil {
		logging.Error("[TRMNL IMPORT] Failed to read ZIP file", "error", err)
		return nil, fmt.Errorf("failed to read ZIP file: %w", err)
	}

	logging.Info("[TRMNL IMPORT] ZIP file opened successfully", "file_count", len(zipReader.File))

	// Extract files
	exportData := &ZipExportData{}
	foundFiles := make(map[string]bool)

	for i, f := range zipReader.File {
		logging.Info("[TRMNL IMPORT] Processing ZIP file", 
			"index", i,
			"name", f.Name,
			"size", f.UncompressedSize64,
			"compressed_size", f.CompressedSize64)

		// Validate file path (flat structure only)
		if filepath.Dir(f.Name) != "." && f.Name != filepath.Base(f.Name) {
			logging.Error("[TRMNL IMPORT] Invalid file structure", "file", f.Name, "dir", filepath.Dir(f.Name))
			return nil, fmt.Errorf("invalid file structure: subdirectories not allowed (found: %s)", f.Name)
		}

		// Check file size
		if f.UncompressedSize64 > MaxTemplateFileSize {
			logging.Error("[TRMNL IMPORT] File too large", "file", f.Name, "size", f.UncompressedSize64, "max", MaxTemplateFileSize)
			return nil, fmt.Errorf("file %s exceeds 1MB limit", f.Name)
		}

		// Open and read file
		rc, err := f.Open()
		if err != nil {
			logging.Error("[TRMNL IMPORT] Failed to open file", "file", f.Name, "error", err)
			return nil, fmt.Errorf("failed to open file %s: %w", f.Name, err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			logging.Error("[TRMNL IMPORT] Failed to read file content", "file", f.Name, "error", err)
			return nil, fmt.Errorf("failed to read file %s: %w", f.Name, err)
		}

		logging.Info("[TRMNL IMPORT] Successfully read file", "file", f.Name, "content_size", len(content))

		// Assign content based on filename
		switch strings.ToLower(f.Name) {
		case "settings.yml":
			exportData.SettingsYAML = content
			foundFiles["settings.yml"] = true
			logging.Info("[TRMNL IMPORT] Found settings.yml", "size", len(content))
		case "full.liquid":
			exportData.FullTemplate = string(content)
			foundFiles["full.liquid"] = true
			logging.Info("[TRMNL IMPORT] Found full.liquid template", "size", len(content))
		case "half_horizontal.liquid":
			exportData.HalfHorizontal = string(content)
			foundFiles["half_horizontal.liquid"] = true
			logging.Info("[TRMNL IMPORT] Found half_horizontal.liquid template", "size", len(content))
		case "half_vertical.liquid":
			exportData.HalfVertical = string(content)
			foundFiles["half_vertical.liquid"] = true
			logging.Info("[TRMNL IMPORT] Found half_vertical.liquid template", "size", len(content))
		case "quadrant.liquid":
			exportData.QuadrantTemplate = string(content)
			foundFiles["quadrant.liquid"] = true
			logging.Info("[TRMNL IMPORT] Found quadrant.liquid template", "size", len(content))
		default:
			logging.Error("[TRMNL IMPORT] Unexpected file in ZIP", "file", f.Name)
			return nil, fmt.Errorf("unexpected file in ZIP: %s", f.Name)
		}
	}

	logging.Info("[TRMNL IMPORT] File extraction completed", "found_files", foundFiles)

	// Validate required files
	if !foundFiles["settings.yml"] {
		logging.Error("[TRMNL IMPORT] Missing required settings.yml")
		return nil, fmt.Errorf("settings.yml is required but not found in ZIP")
	}

	// At least one template file must be present
	templateFiles := []string{"full.liquid", "half_horizontal.liquid", "half_vertical.liquid", "quadrant.liquid"}
	hasTemplate := false
	for _, template := range templateFiles {
		if foundFiles[template] {
			hasTemplate = true
			break
		}
	}

	if !hasTemplate {
		logging.Error("[TRMNL IMPORT] No template files found", "available_templates", templateFiles)
		return nil, fmt.Errorf("at least one template file is required (full.liquid, half_horizontal.liquid, half_vertical.liquid, or quadrant.liquid)")
	}

	logging.Info("[TRMNL IMPORT] ZIP extraction and validation completed successfully")
	return exportData, nil
}

// ConvertZipDataToPluginDefinition converts extracted ZIP data to a PluginDefinition
func (s *TRMNLZipService) ConvertZipDataToPluginDefinition(zipData *ZipExportData) (*database.PluginDefinition, error) {
	logging.Info("[TRMNL IMPORT] Converting ZIP data to PluginDefinition", 
		"has_settings", len(zipData.SettingsYAML) > 0,
		"has_full", zipData.FullTemplate != "",
		"has_half_h", zipData.HalfHorizontal != "",
		"has_half_v", zipData.HalfVertical != "",
		"has_quadrant", zipData.QuadrantTemplate != "")

	// Parse settings.yml using the TRMNL export service
	def, err := s.exportService.ParseSettingsYAML(zipData.SettingsYAML)
	if err != nil {
		logging.Error("[TRMNL IMPORT] Failed to parse settings.yml", "error", err)
		return nil, fmt.Errorf("failed to parse settings.yml: %w", err)
	}

	logging.Info("[TRMNL IMPORT] Successfully parsed settings.yml", "plugin_name", def.Name, "description", def.Description)

	// Set template content
	templateCount := 0
	if zipData.FullTemplate != "" {
		def.MarkupFull = &zipData.FullTemplate
		templateCount++
		logging.Info("[TRMNL IMPORT] Set full template", "size", len(zipData.FullTemplate))
	}
	if zipData.HalfHorizontal != "" {
		def.MarkupHalfHoriz = &zipData.HalfHorizontal
		templateCount++
		logging.Info("[TRMNL IMPORT] Set half horizontal template", "size", len(zipData.HalfHorizontal))
	}
	if zipData.HalfVertical != "" {
		def.MarkupHalfVert = &zipData.HalfVertical
		templateCount++
		logging.Info("[TRMNL IMPORT] Set half vertical template", "size", len(zipData.HalfVertical))
	}
	if zipData.QuadrantTemplate != "" {
		def.MarkupQuadrant = &zipData.QuadrantTemplate
		templateCount++
		logging.Info("[TRMNL IMPORT] Set quadrant template", "size", len(zipData.QuadrantTemplate))
	}

	logging.Info("[TRMNL IMPORT] Set template content", "template_count", templateCount)

	// Set plugin metadata
	def.PluginType = "private"
	def.Author = "Imported from TRMNL"
	
	// Preserve the description extracted from YAML custom fields (about_plugin with field_type: author_bio)
	// If no description was extracted, leave it empty

	// Default processing requirements
	requiresProcessing := true
	def.RequiresProcessing = requiresProcessing

	// Validate form fields if they exist to ensure valid JSON schema
	if def.FormFields != nil {
		// Convert form fields to interface{} for validation
		var formFieldsInterface interface{}
		if err := json.Unmarshal(def.FormFields, &formFieldsInterface); err != nil {
			logging.Error("[TRMNL IMPORT] Failed to unmarshal form fields for validation", "error", err)
			return nil, fmt.Errorf("invalid form fields JSON: %w", err)
		}
		
		// Validate form fields using existing validation logic
		_, err := ValidateFormFields(formFieldsInterface)
		if err != nil {
			logging.Error("[TRMNL IMPORT] Form fields validation failed", "error", err)
			return nil, fmt.Errorf("form fields validation failed: %w", err)
		}
		
		logging.Info("[TRMNL IMPORT] Form fields validation passed")
	}

	logging.Info("[TRMNL IMPORT] Successfully converted ZIP data to PluginDefinition", 
		"plugin_name", def.Name,
		"plugin_type", def.PluginType,
		"requires_processing", def.RequiresProcessing,
		"has_form_fields", def.FormFields != nil)

	return def, nil
}

// addFileToZip adds a file to the ZIP writer
func (s *TRMNLZipService) addFileToZip(zipWriter *zip.Writer, filename string, content []byte) error {
	fileWriter, err := zipWriter.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s in ZIP: %w", filename, err)
	}

	if _, err := fileWriter.Write(content); err != nil {
		return fmt.Errorf("failed to write content to %s: %w", filename, err)
	}

	return nil
}

// getTemplateContent safely extracts template content from pointer
func getTemplateContent(template *string) string {
	if template == nil {
		return ""
	}
	return *template
}

// ValidateZipStructure performs basic validation on uploaded ZIP file
func (s *TRMNLZipService) ValidateZipStructure(file multipart.File, header *multipart.FileHeader) error {
	// Check file extension
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		return fmt.Errorf("file must be a ZIP archive")
	}

	// Check file size
	if header.Size == 0 {
		return fmt.Errorf("uploaded file is empty")
	}

	if header.Size > MaxTotalZipSize {
		return fmt.Errorf("ZIP file too large: %d bytes (max %d bytes)", header.Size, MaxTotalZipSize)
	}

	return nil
}

// GenerateExportFilename creates a filename for exported ZIP files
func (s *TRMNLZipService) GenerateExportFilename(pluginName string) string {
	// Sanitize plugin name for filename
	sanitized := strings.ReplaceAll(pluginName, " ", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	
	// Remove any non-alphanumeric characters except underscore and hyphen
	var result strings.Builder
	for _, char := range sanitized {
		if (char >= 'a' && char <= 'z') || 
		   (char >= 'A' && char <= 'Z') || 
		   (char >= '0' && char <= '9') || 
		   char == '_' || char == '-' {
			result.WriteRune(char)
		}
	}
	
	filename := result.String()
	if filename == "" {
		filename = "private_plugin"
	}
	
	return filename + ".zip"
}