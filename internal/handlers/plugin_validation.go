package handlers

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// PollingURLConfig represents a single URL configuration for polling
type PollingURLConfig struct {
	URL     string            `json:"url" binding:"required"`
	Method  string            `json:"method" binding:"oneof=GET POST"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// PollingConfig represents the complete polling configuration
type PollingConfig struct {
	URLs        []PollingURLConfig `json:"urls" binding:"required,min=1"`
	Interval    int                `json:"interval" binding:"min=60"`      // minimum 60 seconds
	Timeout     int                `json:"timeout" binding:"min=1,max=60"` // 1-60 seconds
	MaxSize     int                `json:"max_size" binding:"min=1024"`    // minimum 1KB
	UserAgent   string             `json:"user_agent"`
	RetryCount  int                `json:"retry_count" binding:"min=0,max=5"`
}

// FormField represents a single form field configuration in YAML format
type FormField struct {
	Keyname      string                 `json:"keyname" yaml:"keyname" binding:"required"`
	FieldType    string                 `json:"field_type" yaml:"field_type" binding:"required"`
	Name         string                 `json:"name" yaml:"name" binding:"required"`
	Description  string                 `json:"description" yaml:"description,omitempty"`
	Optional     bool                   `json:"optional" yaml:"optional,omitempty"`
	Default      interface{}            `json:"default" yaml:"default,omitempty"`
	Placeholder  string                 `json:"placeholder" yaml:"placeholder,omitempty"`
	HelpText     string                 `json:"help_text" yaml:"help_text,omitempty"`
	Options      []FormFieldOption      `json:"options" yaml:"options,omitempty"` // For select fields
	Validation   map[string]interface{} `json:"validation" yaml:"validation,omitempty"`
}

// FormFieldOption represents an option for select fields
type FormFieldOption struct {
	Label string `json:"label" yaml:"label" binding:"required"`
	Value string `json:"value" yaml:"value" binding:"required"`
}

// FormFieldsConfig represents the complete form fields configuration
type FormFieldsConfig struct {
	Fields []FormField `json:"fields" yaml:"fields"`
}

// ValidatePollingConfig validates the polling configuration structure
func ValidatePollingConfig(config interface{}) error {
	if config == nil {
		return nil // Polling config is optional
	}

	// Convert to map if it's not already
	var configMap map[string]interface{}
	
	switch v := config.(type) {
	case map[string]interface{}:
		configMap = v
	case string:
		if v == "" {
			return nil // Empty string means no config
		}
		return fmt.Errorf("polling config should be an object, not a string")
	default:
		// Try to convert via JSON marshaling
		jsonData, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("invalid polling config format: %w", err)
		}
		if err := json.Unmarshal(jsonData, &configMap); err != nil {
			return fmt.Errorf("polling config must be an object: %w", err)
		}
	}

	// Convert to JSON and back to validate structure
	jsonData, err := json.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("invalid polling config format: %w", err)
	}

	var pollingConfig PollingConfig
	if err := json.Unmarshal(jsonData, &pollingConfig); err != nil {
		return fmt.Errorf("polling config validation failed: %w", err)
	}

	// Additional validation
	for i, urlConfig := range pollingConfig.URLs {
		if urlConfig.Method == "" {
			pollingConfig.URLs[i].Method = "GET" // Default method
		}
		if urlConfig.Method == "POST" && urlConfig.Body != "" {
			// Validate JSON body if provided
			var jsonBody interface{}
			if err := json.Unmarshal([]byte(urlConfig.Body), &jsonBody); err != nil {
				return fmt.Errorf("invalid JSON body for URL %d: %w", i+1, err)
			}
		}
	}

	return nil
}

// ValidateFormFields validates the form fields configuration and converts YAML to JSON schema
func ValidateFormFields(formFields interface{}) (string, error) {
	if formFields == nil {
		return `{"type": "object", "properties": {}}`, nil
	}

	// Convert to map if it's not already
	var formFieldsMap map[string]interface{}
	
	switch v := formFields.(type) {
	case map[string]interface{}:
		formFieldsMap = v
	case string:
		if v == "" {
			return `{"type": "object", "properties": {}}`, nil
		}
		return "", fmt.Errorf("form fields should be an object, not a string")
	default:
		// Try to convert via JSON marshaling
		jsonData, err := json.Marshal(formFields)
		if err != nil {
			return "", fmt.Errorf("invalid form fields format: %w", err)
		}
		if err := json.Unmarshal(jsonData, &formFieldsMap); err != nil {
			return "", fmt.Errorf("form fields must be an object: %w", err)
		}
	}

	// Try to parse as YAML string first
	if yamlStr, ok := formFieldsMap["yaml"].(string); ok {
		return convertYAMLFormFieldsToJSONSchema(yamlStr)
	}

	// If not YAML string, treat as direct form field config
	jsonData, err := json.Marshal(formFieldsMap)
	if err != nil {
		return "", fmt.Errorf("invalid form fields format: %w", err)
	}

	var config FormFieldsConfig
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return "", fmt.Errorf("form fields validation failed: %w", err)
	}

	return generateJSONSchemaFromFormFields(config)
}

// convertYAMLFormFieldsToJSONSchema converts YAML form field definitions to JSON schema
func convertYAMLFormFieldsToJSONSchema(yamlContent string) (string, error) {
	var config FormFieldsConfig
	if err := yaml.Unmarshal([]byte(yamlContent), &config); err != nil {
		return "", fmt.Errorf("invalid YAML format: %w", err)
	}

	return generateJSONSchemaFromFormFields(config)
}

// generateJSONSchemaFromFormFields generates a JSON schema from form field configuration
func generateJSONSchemaFromFormFields(config FormFieldsConfig) (string, error) {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
		"required":   []string{},
	}

	properties := schema["properties"].(map[string]interface{})
	var required []string

	for _, field := range config.Fields {
		fieldSchema := generateFieldSchema(field)
		properties[field.Keyname] = fieldSchema

		if !field.Optional {
			required = append(required, field.Keyname)
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return "", fmt.Errorf("failed to generate JSON schema: %w", err)
	}

	return string(schemaJSON), nil
}

// generateFieldSchema generates JSON schema for a single form field
func generateFieldSchema(field FormField) map[string]interface{} {
	schema := map[string]interface{}{
		"title":       field.Name,
		"description": field.Description,
	}

	if field.Default != nil {
		schema["default"] = field.Default
	}

	switch field.FieldType {
	case "string", "url", "author_bio", "copyable", "copyable_webhook_url":
		schema["type"] = "string"
		if field.Placeholder != "" {
			schema["placeholder"] = field.Placeholder
		}
	case "text", "code":
		schema["type"] = "string"
		schema["format"] = "textarea"
	case "number":
		schema["type"] = "number"
	case "password":
		schema["type"] = "string"
		schema["format"] = "password"
	case "date":
		schema["type"] = "string"
		schema["format"] = "date"
	case "time":
		schema["type"] = "string"
		schema["format"] = "time"
	case "time_zone":
		schema["type"] = "string"
		schema["format"] = "timezone"
	case "select":
		schema["type"] = "string"
		if len(field.Options) > 0 {
			enum := make([]string, len(field.Options))
			enumNames := make([]string, len(field.Options))
			for i, option := range field.Options {
				enum[i] = option.Value
				enumNames[i] = option.Label
			}
			schema["enum"] = enum
			schema["enumNames"] = enumNames
		}
	case "xhrSelect", "xhrSelectSearch":
		schema["type"] = "string"
		schema["format"] = "xhr-select"
	default:
		schema["type"] = "string"
	}

	// Add validation rules
	if field.Validation != nil {
		for key, value := range field.Validation {
			schema[key] = value
		}
	}

	return schema
}