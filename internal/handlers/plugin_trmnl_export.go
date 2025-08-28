package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// FlexibleStaticData can handle both empty string and map during YAML unmarshaling
type FlexibleStaticData struct {
	Data map[string]interface{}
}

// UnmarshalYAML implements custom YAML unmarshaling for FlexibleStaticData
func (f *FlexibleStaticData) UnmarshalYAML(value *yaml.Node) error {
	// If it's an empty string or just whitespace, set to empty map
	if value.Kind == yaml.ScalarNode && strings.TrimSpace(value.Value) == "" {
		f.Data = make(map[string]interface{})
		return nil
	}
	
	// If it's a map, unmarshal normally
	if value.Kind == yaml.MappingNode {
		var data map[string]interface{}
		if err := value.Decode(&data); err != nil {
			return err
		}
		f.Data = data
		return nil
	}
	
	// For any other scalar, try to treat as empty
	f.Data = make(map[string]interface{})
	return nil
}

// FlexibleHeaders can handle both empty string and map during YAML unmarshaling
type FlexibleHeaders struct {
	Data map[string]string
}

// UnmarshalYAML implements custom YAML unmarshaling for FlexibleHeaders
func (f *FlexibleHeaders) UnmarshalYAML(value *yaml.Node) error {
	// If it's an empty string or just whitespace, set to empty map
	if value.Kind == yaml.ScalarNode && strings.TrimSpace(value.Value) == "" {
		f.Data = make(map[string]string)
		return nil
	}
	
	// If it's a map, unmarshal normally
	if value.Kind == yaml.MappingNode {
		var data map[string]string
		if err := value.Decode(&data); err != nil {
			return err
		}
		f.Data = data
		return nil
	}
	
	// For any other scalar, try to treat as empty
	f.Data = make(map[string]string)
	return nil
}

// FlexibleOptions can handle both simple string arrays and key-value map arrays for form field options
type FlexibleOptions struct {
	Options []string
	OptionsMap map[string]string
}

// UnmarshalYAML implements custom YAML unmarshaling for FlexibleOptions
func (f *FlexibleOptions) UnmarshalYAML(value *yaml.Node) error {
	// Initialize empty structures
	f.Options = nil
	f.OptionsMap = make(map[string]string)
	
	// If it's an empty string or just whitespace, set to empty
	if value.Kind == yaml.ScalarNode && strings.TrimSpace(value.Value) == "" {
		return nil
	}
	
	// If it's a sequence (array), process each item
	if value.Kind == yaml.SequenceNode {
		for _, item := range value.Content {
			// If the item is a scalar (simple string)
			if item.Kind == yaml.ScalarNode {
				f.Options = append(f.Options, item.Value)
			} else if item.Kind == yaml.MappingNode {
				// If the item is a map (key-value pair)
				var itemMap map[string]string
				if err := item.Decode(&itemMap); err != nil {
					return fmt.Errorf("failed to decode option map: %w", err)
				}
				
				// Add each key-value pair to the options map
				for key, value := range itemMap {
					f.OptionsMap[key] = value
				}
			}
		}
		return nil
	}
	
	// For any other case, treat as empty
	return nil
}

// ToStringSlice converts FlexibleOptions to a simple string slice for backward compatibility
func (f *FlexibleOptions) ToStringSlice() []string {
	if len(f.Options) > 0 {
		return f.Options
	}
	
	// If we have a map, convert VALUES to strings (for enum field values)
	if len(f.OptionsMap) > 0 {
		result := make([]string, 0, len(f.OptionsMap))
		for _, value := range f.OptionsMap {
			result = append(result, value)
		}
		return result
	}
	
	return nil
}

// GetDisplayNames returns the display names (keys) from the options map
func (f *FlexibleOptions) GetDisplayNames() []string {
	if len(f.Options) > 0 {
		// For simple string options, display names are the same as values
		return f.Options
	}
	
	// If we have a map, return the keys (display names)
	if len(f.OptionsMap) > 0 {
		result := make([]string, 0, len(f.OptionsMap))
		for key := range f.OptionsMap {
			result = append(result, key)
		}
		return result
	}
	
	return nil
}

// TRMNLSettings represents the structure of TRMNL's settings.yml file
// Updated to match actual TRMNL field names and handle flexible types
type TRMNLSettings struct {
	Name             string              `yaml:"name"`
	Strategy         string              `yaml:"strategy"`
	RefreshInterval  int                 `yaml:"refresh_interval,omitempty"`
	
	// TRMNL polling fields
	PollingURL       string              `yaml:"polling_url,omitempty"`
	PollingVerb      string              `yaml:"polling_verb,omitempty"`
	PollingHeaders   FlexibleHeaders     `yaml:"polling_headers,omitempty"`
	PollingBody      string              `yaml:"polling_body,omitempty"`
	
	// TRMNL screen options
	DarkMode         string              `yaml:"dark_mode,omitempty"`
	NoScreenPadding  string              `yaml:"no_screen_padding,omitempty"`
	
	// TRMNL data fields
	StaticData       FlexibleStaticData  `yaml:"static_data,omitempty"`
	
	// Form fields (mapped from custom_fields)
	CustomFields     []TRMNLFormField    `yaml:"custom_fields,omitempty"`
	
	// Legacy fields for backward compatibility
	URL              string              `yaml:"url,omitempty"`
	Headers          map[string]string   `yaml:"headers,omitempty"`
	HTTPVerb         string              `yaml:"http_verb,omitempty"`
	ScreenPadding    string              `yaml:"screen_padding,omitempty"`
	FormFields       []TRMNLFormField    `yaml:"form_fields,omitempty"`
}

// TRMNLFormField represents a form field in TRMNL format
// Updated to match actual TRMNL custom_fields format
type TRMNLFormField struct {
	// TRMNL custom_fields format
	Keyname     string      `yaml:"keyname"`
	Name        string      `yaml:"name"`
	FieldType   string      `yaml:"field_type"`
	Description string      `yaml:"description,omitempty"`
	Optional    bool        `yaml:"optional,omitempty"`
	HelpText    string      `yaml:"help_text,omitempty"`
	
	// Legacy fields for backward compatibility
	ID          string         `yaml:"id"`
	Type        string         `yaml:"type"`
	Label       string         `yaml:"label"`
	Required    bool           `yaml:"required,omitempty"`
	Default     interface{}    `yaml:"default,omitempty"`
	Options     FlexibleOptions `yaml:"options,omitempty"`
	Placeholder string         `yaml:"placeholder,omitempty"`
}

// TRMNLExportService handles conversion between Stationmaster and TRMNL formats
type TRMNLExportService struct{}

// NewTRMNLExportService creates a new TRMNL export service
func NewTRMNLExportService() *TRMNLExportService {
	return &TRMNLExportService{}
}

// ConvertToTRMNLSettings converts a Stationmaster PluginDefinition to TRMNL settings format
func (s *TRMNLExportService) ConvertToTRMNLSettings(def *database.PluginDefinition) (*TRMNLSettings, error) {
	settings := &TRMNLSettings{
		Name:     def.Name,
		Strategy: "static", // Default strategy
	}

	// Convert data strategy
	if def.DataStrategy != nil {
		settings.Strategy = *def.DataStrategy
	}

	// Handle polling configuration
	if settings.Strategy == "polling" && def.PollingConfig != nil {
		var pollingConfig map[string]interface{}
		if err := json.Unmarshal(def.PollingConfig, &pollingConfig); err == nil {
			// Handle both legacy single URL format and new URLs array format
			if urls, ok := pollingConfig["urls"].([]interface{}); ok && len(urls) > 0 {
				// New format with URLs array
				if urlObj, ok := urls[0].(map[string]interface{}); ok {
					if url, ok := urlObj["url"].(string); ok {
						settings.PollingURL = url
					}
					if headers, ok := urlObj["headers"].(map[string]interface{}); ok {
						settings.PollingHeaders.Data = make(map[string]string)
						for k, v := range headers {
							if str, ok := v.(string); ok {
								settings.PollingHeaders.Data[k] = str
							}
						}
					}
					if verb, ok := urlObj["method"].(string); ok {
						settings.PollingVerb = strings.ToLower(verb)
					}
				}
			} else {
				// Legacy format with single URL
				if url, ok := pollingConfig["url"].(string); ok {
					settings.PollingURL = url
				}
				if headers, ok := pollingConfig["headers"].(map[string]interface{}); ok {
					settings.PollingHeaders.Data = make(map[string]string)
					for k, v := range headers {
						if str, ok := v.(string); ok {
							settings.PollingHeaders.Data[k] = str
						}
					}
				}
				if verb, ok := pollingConfig["method"].(string); ok {
					settings.PollingVerb = strings.ToLower(verb)
				}
			}

			// Extract refresh interval
			if interval, ok := pollingConfig["interval"].(float64); ok {
				// Convert seconds to minutes for TRMNL format
				intervalMinutes := int(interval / 60)
				// Map to TRMNL's allowed values: 15, 60, 360, 720, or 1440 minutes
				settings.RefreshInterval = mapToTRMNLInterval(intervalMinutes)
			}
		}
	}

	// Handle static data strategy
	if settings.Strategy == "static" {
		settings.StaticData.Data = make(map[string]interface{})
		// Static data would come from form field defaults or sample data
		if def.SampleData != nil {
			var sampleData map[string]interface{}
			if err := json.Unmarshal(def.SampleData, &sampleData); err == nil {
				settings.StaticData.Data = sampleData
			}
		}
	}

	// Handle screen options - use TRMNL field names
	if def.EnableDarkMode != nil && *def.EnableDarkMode {
		settings.DarkMode = "yes"
	} else {
		settings.DarkMode = "no"
	}

	if def.RemoveBleedMargin != nil && *def.RemoveBleedMargin {
		settings.NoScreenPadding = "yes"  // Remove padding
	} else {
		settings.NoScreenPadding = "no"   // Keep padding
	}

	// Convert form fields to TRMNL custom_fields format
	if def.FormFields != nil {
		formFields, err := s.convertFormFieldsToTRMNL(def.FormFields)
		if err != nil {
			return nil, fmt.Errorf("failed to convert form fields: %w", err)
		}
		settings.CustomFields = formFields
	}

	// Add "About This Plugin" field if description exists
	if def.Description != "" {
		aboutField := TRMNLFormField{
			Keyname:     "about_plugin",
			Name:        "About This Plugin",
			FieldType:   "author_bio",
			Description: def.Description,
		}
		settings.CustomFields = append(settings.CustomFields, aboutField)
	}

	// Set default refresh interval if not set
	if settings.RefreshInterval == 0 {
		settings.RefreshInterval = 60 // Default to 60 minutes
	}

	return settings, nil
}

// ConvertFromTRMNLSettings converts TRMNL settings to Stationmaster PluginDefinition fields
func (s *TRMNLExportService) ConvertFromTRMNLSettings(settings *TRMNLSettings) (*database.PluginDefinition, error) {
	logging.Info("[TRMNL IMPORT] Starting conversion from TRMNL settings", "plugin_name", settings.Name, "strategy", settings.Strategy)
	
	def := &database.PluginDefinition{
		Name:        settings.Name,
		PluginType:  "private",
		Version:     "1.0.0",
	}

	// Convert data strategy
	strategy := settings.Strategy
	def.DataStrategy = &strategy
	logging.Info("[TRMNL IMPORT] Set data strategy", "strategy", strategy)

	// Handle polling configuration
	if settings.Strategy == "polling" {
		// Prioritize new TRMNL field names, fall back to legacy
		url := settings.PollingURL
		if url == "" {
			url = settings.URL
		}
		
		method := strings.ToUpper(settings.PollingVerb)
		if method == "" {
			method = strings.ToUpper(settings.HTTPVerb)
		}
		if method == "" {
			method = "GET"
		}
		
		// Get headers from flexible type or legacy field
		var headers map[string]string
		if settings.PollingHeaders.Data != nil && len(settings.PollingHeaders.Data) > 0 {
			headers = settings.PollingHeaders.Data
		} else if settings.Headers != nil && len(settings.Headers) > 0 {
			headers = settings.Headers
		}
		
		logging.Info("[TRMNL IMPORT] Processing polling configuration", 
			"url", url, 
			"method", method,
			"headers_count", len(headers),
			"refresh_interval", settings.RefreshInterval)
		
		pollingConfig := make(map[string]interface{})
		
		// Create URLs array format (new format)
		urlConfig := map[string]interface{}{
			"url":    url,
			"method": method,
		}
		
		if headers != nil && len(headers) > 0 {
			urlConfig["headers"] = headers
			logging.Info("[TRMNL IMPORT] Added headers to polling config", "headers", headers)
		}
		
		pollingConfig["urls"] = []interface{}{urlConfig}
		
		// Convert refresh interval from minutes to seconds
		intervalSeconds := 3600 // Default to 1 hour
		if settings.RefreshInterval > 0 {
			intervalSeconds = settings.RefreshInterval * 60
		}
		pollingConfig["interval"] = intervalSeconds
		logging.Info("[TRMNL IMPORT] Set polling interval", "minutes", settings.RefreshInterval, "seconds", intervalSeconds)

		pollingConfigJSON, err := json.Marshal(pollingConfig)
		if err != nil {
			logging.Error("[TRMNL IMPORT] Failed to marshal polling config", "error", err, "config", pollingConfig)
			return nil, fmt.Errorf("failed to marshal polling config: %w", err)
		}
		def.PollingConfig = pollingConfigJSON
		logging.Info("[TRMNL IMPORT] Successfully created polling config JSON")
	}

	// Handle screen options
	darkMode := settings.DarkMode == "yes"
	def.EnableDarkMode = &darkMode
	logging.Info("[TRMNL IMPORT] Set dark mode option", "dark_mode", darkMode, "raw_value", settings.DarkMode)
	
	// Handle screen padding - prioritize NoScreenPadding over ScreenPadding
	var removeBleedMargin bool
	if settings.NoScreenPadding != "" {
		removeBleedMargin = settings.NoScreenPadding == "yes"
		logging.Info("[TRMNL IMPORT] Set bleed margin option from no_screen_padding", "remove_bleed_margin", removeBleedMargin, "raw_value", settings.NoScreenPadding)
	} else {
		removeBleedMargin = settings.ScreenPadding == "no"
		logging.Info("[TRMNL IMPORT] Set bleed margin option from screen_padding", "remove_bleed_margin", removeBleedMargin, "raw_value", settings.ScreenPadding)
	}
	def.RemoveBleedMargin = &removeBleedMargin

	// Convert form fields - prioritize CustomFields over FormFields
	var fieldsToConvert []TRMNLFormField
	if settings.CustomFields != nil && len(settings.CustomFields) > 0 {
		fieldsToConvert = settings.CustomFields
		logging.Info("[TRMNL IMPORT] Converting custom_fields", "field_count", len(settings.CustomFields))
	} else if settings.FormFields != nil && len(settings.FormFields) > 0 {
		fieldsToConvert = settings.FormFields
		logging.Info("[TRMNL IMPORT] Converting form_fields", "field_count", len(settings.FormFields))
	}

	// Extract description from any author_bio field (not just about_plugin keyname)
	if fieldsToConvert != nil {
		for _, field := range fieldsToConvert {
			// TRMNL format: any field with field_type: author_bio
			if field.FieldType == "author_bio" && field.Description != "" {
				def.Description = field.Description
				logging.Info("[TRMNL IMPORT] Extracted author_bio description", 
					"keyname", field.Keyname, 
					"description", field.Description)
				break
			}
			// Legacy format: any field with type: author_bio  
			if field.Type == "author_bio" && field.Description != "" {
				if def.Description == "" {
					def.Description = field.Description
					logging.Info("[TRMNL IMPORT] Extracted author_bio description from legacy field", 
						"id", field.ID,
						"description", field.Description)
				}
				break
			}
		}
	}

	if fieldsToConvert != nil && len(fieldsToConvert) > 0 {
		// Filter out author_bio fields since they're handled separately as description
		var filteredFields []TRMNLFormField
		for _, field := range fieldsToConvert {
			// Skip any author_bio fields - they're converted to description above
			if field.FieldType == "author_bio" {
				continue
			}
			if field.Type == "author_bio" {
				continue
			}
			filteredFields = append(filteredFields, field)
		}
		
		if len(filteredFields) > 0 {
			// Convert TRMNL fields back to YAML for UI display
			yamlString, err := s.convertTRMNLFieldsToYAML(filteredFields)
			if err != nil {
				logging.Error("[TRMNL IMPORT] Failed to convert form fields to YAML", "error", err, "field_count", len(filteredFields))
				return nil, fmt.Errorf("failed to convert form fields to YAML: %w", err)
			}
			
			// Store form fields in the format expected by UI: {yaml: "..."}
			formFieldsWrapper := map[string]string{
				"yaml": yamlString,
			}
			formFieldsJSON, err := json.Marshal(formFieldsWrapper)
			if err != nil {
				logging.Error("[TRMNL IMPORT] Failed to marshal form fields wrapper", "error", err)
				return nil, fmt.Errorf("failed to marshal form fields wrapper: %w", err)
			}
			
			def.FormFields = formFieldsJSON
			logging.Info("[TRMNL IMPORT] Successfully converted form fields to YAML format", 
				"field_count", len(filteredFields), "yaml_length", len(yamlString))
		} else {
			logging.Info("[TRMNL IMPORT] No form fields to convert after filtering author_bio fields")
		}
	} else {
		logging.Info("[TRMNL IMPORT] No form fields to convert")
	}

	// Handle static data
	if settings.Strategy == "static" && settings.StaticData.Data != nil && len(settings.StaticData.Data) > 0 {
		logging.Info("[TRMNL IMPORT] Processing static data", "data_keys", len(settings.StaticData.Data))
		
		sampleDataJSON, err := json.Marshal(settings.StaticData.Data)
		if err != nil {
			logging.Error("[TRMNL IMPORT] Failed to marshal static data", "error", err, "data", settings.StaticData.Data)
			return nil, fmt.Errorf("failed to marshal static data: %w", err)
		}
		def.SampleData = sampleDataJSON
		logging.Info("[TRMNL IMPORT] Successfully processed static data")
	}

	logging.Info("[TRMNL IMPORT] Conversion completed successfully", "plugin_name", def.Name)
	return def, nil
}

// convertFormFieldsToTRMNL converts Stationmaster JSON schema form fields to TRMNL YAML format
func (s *TRMNLExportService) convertFormFieldsToTRMNL(formFieldsJSON []byte) ([]TRMNLFormField, error) {
	var stationmasterFields map[string]interface{}
	if err := json.Unmarshal(formFieldsJSON, &stationmasterFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal form fields: %w", err)
	}

	var trmnlFields []TRMNLFormField

	// Extract properties from JSON schema
	if properties, ok := stationmasterFields["properties"].(map[string]interface{}); ok {
		for fieldID, fieldDef := range properties {
			if fieldDefMap, ok := fieldDef.(map[string]interface{}); ok {
				trmnlField := TRMNLFormField{
					ID: fieldID,
				}

				// Extract type
				if fieldType, ok := fieldDefMap["type"].(string); ok {
					trmnlField.Type = convertTypeToTRMNL(fieldType)
				}

				// Extract title/label
				if title, ok := fieldDefMap["title"].(string); ok {
					trmnlField.Label = title
				} else if description, ok := fieldDefMap["description"].(string); ok {
					trmnlField.Label = description
				} else {
					trmnlField.Label = fieldID
				}

				// Extract default value
				if defaultVal, ok := fieldDefMap["default"]; ok {
					trmnlField.Default = defaultVal
				}

				// Extract enum options for select fields
				if enum, ok := fieldDefMap["enum"].([]interface{}); ok {
					options := make([]string, len(enum))
					for i, option := range enum {
						if str, ok := option.(string); ok {
							options[i] = str
						}
					}
					// Set FlexibleOptions with simple string options
					trmnlField.Options.Options = options
					trmnlField.Options.OptionsMap = make(map[string]string)
					trmnlField.Type = "select"
				}

				// Check if field is required
				if required, ok := stationmasterFields["required"].([]interface{}); ok {
					for _, reqField := range required {
						if reqFieldStr, ok := reqField.(string); ok && reqFieldStr == fieldID {
							trmnlField.Required = true
							break
						}
					}
				}

				trmnlFields = append(trmnlFields, trmnlField)
			}
		}
	}

	return trmnlFields, nil
}

// convertFormFieldsFromTRMNL converts TRMNL YAML form fields to Stationmaster JSON schema
func (s *TRMNLExportService) convertFormFieldsFromTRMNL(trmnlFields []TRMNLFormField) ([]byte, error) {
	logging.Info("[TRMNL IMPORT] Starting form fields conversion", "field_count", len(trmnlFields))
	
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
	}

	var required []string
	properties := schema["properties"].(map[string]interface{})

	for i, field := range trmnlFields {
		// Use TRMNL field names with fallback to legacy names
		id := field.Keyname
		if id == "" {
			id = field.ID
		}
		
		fieldType := field.FieldType
		if fieldType == "" {
			fieldType = field.Type
		}
		
		label := field.Name
		if label == "" {
			label = field.Label
		}

		logging.Info("[TRMNL IMPORT] Processing form field", 
			"index", i,
			"keyname", field.Keyname,
			"field_type", field.FieldType,
			"name", field.Name,
			"description", field.Description,
			"help_text", field.HelpText,
			"optional", field.Optional,
			"id", id,
			"type", fieldType,
			"label", label,
			"required", field.Required,
			"has_default", field.Default != nil,
			"has_options", len(field.Options.ToStringSlice()) > 0)

		if id == "" {
			logging.Error("[TRMNL IMPORT] Form field missing ID/keyname", "index", i, "field", field)
			return nil, fmt.Errorf("form field at index %d is missing required 'id' or 'keyname' field", i)
		}

		if fieldType == "" {
			logging.Error("[TRMNL IMPORT] Form field missing type", "index", i, "id", id)
			return nil, fmt.Errorf("form field '%s' is missing required 'type' or 'field_type' field", id)
		}

		fieldDef := map[string]interface{}{
			"type":  convertTypeFromTRMNL(fieldType),
			"title": label,
		}

		if field.Default != nil {
			fieldDef["default"] = field.Default
			logging.Info("[TRMNL IMPORT] Added default value", "field", id, "default", field.Default)
		}

		// Use description from TRMNL, help_text, or placeholder as description
		description := field.Description
		if description == "" && field.HelpText != "" {
			description = field.HelpText
		}
		if description == "" && field.Placeholder != "" {
			description = field.Placeholder
		}
		if description != "" {
			fieldDef["description"] = description
		}

		// Handle flexible options - convert to string slice for JSON schema
		optionsList := field.Options.ToStringSlice()
		displayNames := field.Options.GetDisplayNames()
		if len(optionsList) > 0 {
			fieldDef["enum"] = optionsList
			
			// Add display names if they differ from values (i.e., we have a map)
			if len(field.Options.OptionsMap) > 0 && len(displayNames) == len(optionsList) {
				fieldDef["enumNames"] = displayNames
				logging.Info("[TRMNL IMPORT] Added options with display names", 
					"field", id, 
					"options", optionsList, 
					"display_names", displayNames)
			} else {
				logging.Info("[TRMNL IMPORT] Added options", "field", id, "options", optionsList)
			}
		}

		properties[id] = fieldDef

		// Handle required logic: field is required if not explicitly marked as optional
		// Support both TRMNL format (optional) and legacy format (required)
		isRequired := false
		if field.FieldType != "" || field.Keyname != "" {
			// TRMNL format: required unless explicitly optional
			isRequired = !field.Optional
		} else {
			// Legacy format: use explicit Required field
			isRequired = field.Required
		}
		
		if isRequired {
			required = append(required, id)
		}
	}

	if len(required) > 0 {
		schema["required"] = required
		logging.Info("[TRMNL IMPORT] Set required fields", "required", required)
	}

	result, err := json.Marshal(schema)
	if err != nil {
		logging.Error("[TRMNL IMPORT] Failed to marshal form fields schema", "error", err, "schema", schema)
		return nil, fmt.Errorf("failed to marshal form fields schema: %w", err)
	}

	logging.Info("[TRMNL IMPORT] Successfully converted form fields to JSON schema")
	return result, nil
}

// convertTRMNLFieldsToYAML converts TRMNL form fields back to YAML string for UI display
func (s *TRMNLExportService) convertTRMNLFieldsToYAML(trmnlFields []TRMNLFormField) (string, error) {
	if len(trmnlFields) == 0 {
		return "", nil
	}
	
	yamlBytes, err := yaml.Marshal(trmnlFields)
	if err != nil {
		return "", fmt.Errorf("failed to marshal TRMNL fields to YAML: %w", err)
	}
	
	return string(yamlBytes), nil
}

// convertTypeToTRMNL converts JSON schema types to TRMNL form field types
func convertTypeToTRMNL(jsonType string) string {
	switch jsonType {
	case "string":
		return "text"
	case "number", "integer":
		return "number"
	case "boolean":
		return "checkbox"
	default:
		return "text"
	}
}

// convertTypeFromTRMNL converts TRMNL form field types to JSON schema types
func convertTypeFromTRMNL(trmnlType string) string {
	switch trmnlType {
	case "text", "textarea":
		return "string"
	case "number":
		return "number"
	case "checkbox":
		return "boolean"
	case "select":
		return "string"
	case "author_bio":
		return "string"
	default:
		return "string"
	}
}

// mapToTRMNLInterval maps an interval in minutes to TRMNL's allowed values
func mapToTRMNLInterval(minutes int) int {
	allowedIntervals := []int{15, 60, 360, 720, 1440}
	
	// Find the closest allowed interval
	closest := allowedIntervals[0]
	minDiff := abs(minutes - closest)
	
	for _, interval := range allowedIntervals {
		diff := abs(minutes - interval)
		if diff < minDiff {
			minDiff = diff
			closest = interval
		}
	}
	
	return closest
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GenerateSettingsYAML converts a PluginDefinition to YAML format
func (s *TRMNLExportService) GenerateSettingsYAML(def *database.PluginDefinition) ([]byte, error) {
	settings, err := s.ConvertToTRMNLSettings(def)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to TRMNL settings: %w", err)
	}

	yamlData, err := yaml.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return yamlData, nil
}

// ParseSettingsYAML parses TRMNL settings.yml and returns a PluginDefinition
func (s *TRMNLExportService) ParseSettingsYAML(yamlData []byte) (*database.PluginDefinition, error) {
	logging.Info("[TRMNL IMPORT] Starting YAML parsing", "size", len(yamlData))
	
	// Log a snippet of the YAML for debugging (first 200 chars)
	yamlSnippet := string(yamlData)
	if len(yamlSnippet) > 200 {
		yamlSnippet = yamlSnippet[:200] + "..."
	}
	logging.Info("[TRMNL IMPORT] YAML content snippet", "content", yamlSnippet)

	var settings TRMNLSettings
	if err := yaml.Unmarshal(yamlData, &settings); err != nil {
		logging.Error("[TRMNL IMPORT] YAML unmarshaling failed", "error", err, "yaml_snippet", yamlSnippet)
		return nil, fmt.Errorf("failed to unmarshal YAML - invalid YAML format: %w", err)
	}

	logging.Info("[TRMNL IMPORT] YAML parsed successfully", 
		"name", settings.Name, 
		"strategy", settings.Strategy,
		"refresh_interval", settings.RefreshInterval,
		"has_form_fields", len(settings.FormFields) > 0,
		"has_url", settings.URL != "",
		"has_headers", len(settings.Headers) > 0,
		"dark_mode", settings.DarkMode,
		"screen_padding", settings.ScreenPadding)

	// Validate required fields
	if settings.Name == "" {
		logging.Error("[TRMNL IMPORT] Validation failed: missing plugin name")
		return nil, fmt.Errorf("plugin name is required in settings.yml")
	}

	if settings.Strategy == "" {
		logging.Error("[TRMNL IMPORT] Validation failed: missing strategy")
		return nil, fmt.Errorf("strategy is required in settings.yml")
	}

	// Validate strategy
	validStrategies := map[string]bool{
		"polling": true,
		"webhook": true,
		"static":  true,
	}
	if !validStrategies[settings.Strategy] {
		logging.Error("[TRMNL IMPORT] Validation failed: invalid strategy", "strategy", settings.Strategy)
		return nil, fmt.Errorf("invalid strategy '%s' in settings.yml (must be polling, webhook, or static)", settings.Strategy)
	}

	// Validate refresh interval for polling strategy
	if settings.Strategy == "polling" {
		allowedIntervals := []int{15, 60, 360, 720, 1440}
		validInterval := false
		for _, interval := range allowedIntervals {
			if settings.RefreshInterval == interval {
				validInterval = true
				break
			}
		}
		if !validInterval {
			logging.Error("[TRMNL IMPORT] Validation failed: invalid refresh interval", 
				"interval", settings.RefreshInterval, 
				"allowed", allowedIntervals)
			return nil, fmt.Errorf("invalid refresh_interval %d minutes in settings.yml (must be 15, 60, 360, 720, or 1440 minutes)", settings.RefreshInterval)
		}

		// Validate polling-specific fields - check both TRMNL and legacy field names
		if settings.Strategy == "polling" && settings.PollingURL == "" && settings.URL == "" {
			logging.Error("[TRMNL IMPORT] Validation failed: polling strategy missing URL")
			return nil, fmt.Errorf("polling_url or url is required for polling strategy in settings.yml")
		}
	}

	logging.Info("[TRMNL IMPORT] YAML validation passed, starting conversion to PluginDefinition")
	
	result, err := s.ConvertFromTRMNLSettings(&settings)
	if err != nil {
		logging.Error("[TRMNL IMPORT] Conversion to PluginDefinition failed", "error", err)
		return nil, fmt.Errorf("failed to convert TRMNL settings to plugin definition: %w", err)
	}
	
	logging.Info("[TRMNL IMPORT] Successfully converted to PluginDefinition", 
		"plugin_name", result.Name,
		"plugin_type", result.PluginType,
		"data_strategy", getStringValue(result.DataStrategy))
	
	return result, nil
}

// getStringValue safely extracts string value from pointer for logging
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}