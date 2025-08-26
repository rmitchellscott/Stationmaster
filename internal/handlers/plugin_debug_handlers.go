package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/validation"
)

// ValidateTRMNLYAMLRequest represents the request to validate TRMNL YAML
type ValidateTRMNLYAMLRequest struct {
	YAML string `json:"yaml" binding:"required"`
}

// ValidateTRMNLYAMLResponse represents the validation response
type ValidateTRMNLYAMLResponse struct {
	Valid           bool                   `json:"valid"`
	ParsedSettings  map[string]interface{} `json:"parsed_settings,omitempty"`
	ValidationInfo  ValidationInfo         `json:"validation_info"`
	ConvertedPlugin map[string]interface{} `json:"converted_plugin,omitempty"`
	Errors          []string               `json:"errors,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
}

// ValidationInfo provides detailed validation information
type ValidationInfo struct {
	RequiredFields     ValidationStatus `json:"required_fields"`
	Strategy           ValidationStatus `json:"strategy"`
	RefreshInterval    ValidationStatus `json:"refresh_interval"`
	PollingConfig      ValidationStatus `json:"polling_config"`
	FormFields         ValidationStatus `json:"form_fields"`
	ScreenOptions      ValidationStatus `json:"screen_options"`
}

// ValidationStatus represents the status of a validation check
type ValidationStatus struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ValidateTRMNLYAMLHandler validates TRMNL YAML without creating a plugin
func ValidateTRMNLYAMLHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	var req ValidateTRMNLYAMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	logging.Info("[TRMNL DEBUG] Starting YAML validation", "user_id", user.ID, "yaml_size", len(req.YAML))

	response := ValidateTRMNLYAMLResponse{
		ValidationInfo: ValidationInfo{},
		Errors:         []string{},
		Warnings:       []string{},
	}

	// Create TRMNL export service for validation
	exportService := NewTRMNLExportService()

	// Step 1: Parse YAML
	def, err := exportService.ParseSettingsYAML([]byte(req.YAML))
	if err != nil {
		logging.Error("[TRMNL DEBUG] YAML parsing failed", "error", err)
		response.Valid = false
		response.Errors = append(response.Errors, "YAML parsing failed: "+err.Error())
		c.JSON(http.StatusOK, response)
		return
	}

	logging.Info("[TRMNL DEBUG] YAML parsing succeeded", "plugin_name", def.Name)

	// Convert settings for display
	settings, err := exportService.ConvertToTRMNLSettings(def)
	if err == nil {
		// Convert to map for JSON response
		response.ParsedSettings = map[string]interface{}{
			"name":               settings.Name,
			"strategy":           settings.Strategy,
			"refresh_interval":   settings.RefreshInterval,
			"polling_url":        settings.PollingURL,
			"polling_verb":       settings.PollingVerb,
			"polling_headers":    settings.PollingHeaders.Data,
			"static_data":        settings.StaticData.Data,
			"dark_mode":          settings.DarkMode,
			"no_screen_padding":  settings.NoScreenPadding,
			"custom_fields":      settings.CustomFields,
			// Legacy fields for backward compatibility
			"url":                settings.URL,
			"http_verb":          settings.HTTPVerb,
			"headers":            settings.Headers,
			"screen_padding":     settings.ScreenPadding,
			"form_fields":        settings.FormFields,
		}
	}

	// Step 2: Detailed validation checks
	response.ValidationInfo.RequiredFields = validateRequiredFields(def)
	response.ValidationInfo.Strategy = validateStrategy(def)
	response.ValidationInfo.RefreshInterval = validateRefreshInterval(def)
	response.ValidationInfo.PollingConfig = validatePollingConfig(def)
	response.ValidationInfo.FormFields = validateFormFields(def)
	response.ValidationInfo.ScreenOptions = validateScreenOptions(def)

	// Step 3: Template validation if available
	if def.MarkupFull != nil || def.MarkupHalfVert != nil || def.MarkupHalfHoriz != nil || def.MarkupQuadrant != nil {
		validator := validation.NewTemplateValidator()
		validationResult := validator.ValidateAllTemplates(
			getStringValue(def.MarkupFull),
			getStringValue(def.MarkupHalfVert),
			getStringValue(def.MarkupHalfHoriz),
			getStringValue(def.MarkupQuadrant),
			getStringValue(def.SharedMarkup),
		)

		if !validationResult.Valid {
			response.Errors = append(response.Errors, validationResult.Errors...)
		}
		response.Warnings = append(response.Warnings, validationResult.Warnings...)
	} else {
		response.Warnings = append(response.Warnings, "No template files found - templates are required for plugin functionality")
	}

	// Convert plugin definition for display
	response.ConvertedPlugin = map[string]interface{}{
		"name":                def.Name,
		"plugin_type":         def.PluginType,
		"version":             def.Version,
		"author":              def.Author,
		"description":         def.Description,
		"data_strategy":       getStringValue(def.DataStrategy),
		"requires_processing": def.RequiresProcessing,
		"enable_dark_mode":    getBoolPointerValue(def.EnableDarkMode),
		"remove_bleed_margin": getBoolPointerValue(def.RemoveBleedMargin),
	}

	// Overall validation status
	response.Valid = len(response.Errors) == 0
	
	if response.Valid {
		logging.Info("[TRMNL DEBUG] YAML validation completed successfully", "plugin_name", def.Name)
	} else {
		logging.Error("[TRMNL DEBUG] YAML validation failed", "errors", response.Errors)
	}

	c.JSON(http.StatusOK, response)
}

// TestTRMNLConversionHandler tests bidirectional conversion between TRMNL and Stationmaster formats
func TestTRMNLConversionHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	var req ValidateTRMNLYAMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	logging.Info("[TRMNL DEBUG] Testing bidirectional conversion", "user_id", user.ID)

	exportService := NewTRMNLExportService()

	// Step 1: Parse TRMNL YAML to Stationmaster format
	def, err := exportService.ParseSettingsYAML([]byte(req.YAML))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse YAML", "details": err.Error()})
		return
	}

	// Step 2: Convert back to TRMNL format
	reconvertedSettings, err := exportService.ConvertToTRMNLSettings(def)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert back to TRMNL format", "details": err.Error()})
		return
	}

	// Step 3: Generate YAML from reconverted settings
	reconvertedYAML, err := exportService.GenerateSettingsYAML(def)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reconverted YAML", "details": err.Error()})
		return
	}

	response := map[string]interface{}{
		"original_yaml":          req.YAML,
		"parsed_plugin":          def,
		"reconverted_settings":   reconvertedSettings,
		"reconverted_yaml":       string(reconvertedYAML),
		"conversion_successful":  true,
	}

	logging.Info("[TRMNL DEBUG] Bidirectional conversion test completed", "plugin_name", def.Name)
	c.JSON(http.StatusOK, response)
}

// Helper functions for validation

func validateRequiredFields(def *database.PluginDefinition) ValidationStatus {
	if def.Name == "" {
		return ValidationStatus{Valid: false, Message: "Plugin name is required"}
	}
	if def.DataStrategy == nil || *def.DataStrategy == "" {
		return ValidationStatus{Valid: false, Message: "Data strategy is required"}
	}
	return ValidationStatus{Valid: true, Message: "All required fields present"}
}

func validateStrategy(def *database.PluginDefinition) ValidationStatus {
	if def.DataStrategy == nil {
		return ValidationStatus{Valid: false, Message: "Strategy not set"}
	}
	
	strategy := *def.DataStrategy
	validStrategies := []string{"polling", "webhook", "static"}
	for _, valid := range validStrategies {
		if strategy == valid {
			return ValidationStatus{Valid: true, Message: "Valid strategy: " + strategy}
		}
	}
	
	return ValidationStatus{Valid: false, Message: "Invalid strategy: " + strategy, Details: "Must be polling, webhook, or static"}
}

func validateRefreshInterval(def *database.PluginDefinition) ValidationStatus {
	if def.DataStrategy == nil || *def.DataStrategy != "polling" {
		return ValidationStatus{Valid: true, Message: "Refresh interval not applicable for non-polling strategy"}
	}
	
	// For polling strategy, we should have polling config with interval
	if def.PollingConfig == nil {
		return ValidationStatus{Valid: false, Message: "Polling config missing for polling strategy"}
	}
	
	return ValidationStatus{Valid: true, Message: "Refresh interval configuration present"}
}

func validatePollingConfig(def *database.PluginDefinition) ValidationStatus {
	if def.DataStrategy == nil || *def.DataStrategy != "polling" {
		return ValidationStatus{Valid: true, Message: "Polling config not applicable"}
	}
	
	if def.PollingConfig == nil {
		return ValidationStatus{Valid: false, Message: "Polling config required for polling strategy"}
	}
	
	return ValidationStatus{Valid: true, Message: "Polling configuration present"}
}

func validateFormFields(def *database.PluginDefinition) ValidationStatus {
	if def.FormFields == nil {
		return ValidationStatus{Valid: true, Message: "No form fields defined (optional)"}
	}
	
	return ValidationStatus{Valid: true, Message: "Form fields configuration present"}
}

func validateScreenOptions(def *database.PluginDefinition) ValidationStatus {
	return ValidationStatus{Valid: true, Message: "Screen options configured", 
		Details: "Dark mode: " + getBoolString(def.EnableDarkMode) + ", Remove bleed margin: " + getBoolString(def.RemoveBleedMargin)}
}

func getBoolString(b *bool) string {
	if b == nil {
		return "not set"
	}
	if *b {
		return "enabled"
	}
	return "disabled"
}

func getBoolPointerValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}