package private

import (
	"fmt"
	"html"
	"strings"

	"github.com/osteele/liquid"
)

// LiquidRenderer handles liquid template rendering with security and TRMNL-specific features
type LiquidRenderer struct {
	engine *liquid.Engine
}

// NewLiquidRenderer creates a new liquid renderer with security configurations
func NewLiquidRenderer() *LiquidRenderer {
	engine := liquid.NewEngine()
	
	// Add custom filters for TRMNL
	engine.RegisterFilter("truncate", truncateFilter)
	engine.RegisterFilter("escape_html", escapeHTMLFilter)
	engine.RegisterFilter("safe", safeFilter)
	engine.RegisterFilter("format_date", formatDateFilter)
	engine.RegisterFilter("format_time", formatTimeFilter)
	
	// Add security restrictions
	// TODO: Add template sandbox restrictions to prevent arbitrary code execution
	
	return &LiquidRenderer{
		engine: engine,
	}
}

// RenderTemplate renders a liquid template with the provided context data
func (r *LiquidRenderer) RenderTemplate(template string, context map[string]interface{}) (string, error) {
	// Validate template for security
	if err := r.validateTemplate(template); err != nil {
		return "", fmt.Errorf("template validation failed: %w", err)
	}
	
	// Parse template
	tpl, err := r.engine.ParseString(template)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}
	
	// Render with context
	result, err := tpl.Render(context)
	if err != nil {
		return "", fmt.Errorf("template render error: %w", err)
	}
	
	return string(result), nil
}

// validateTemplate performs basic security validation on templates
func (r *LiquidRenderer) validateTemplate(template string) error {
	// Check for potentially dangerous patterns
	dangerous := []string{
		"<script",
		"javascript:",
		"eval(",
		"setTimeout(",
		"setInterval(",
	}
	
	templateLower := strings.ToLower(template)
	for _, pattern := range dangerous {
		if strings.Contains(templateLower, pattern) {
			return fmt.Errorf("template contains potentially dangerous content: %s", pattern)
		}
	}
	
	// TODO: Add more comprehensive template validation
	// - Check for infinite loops
	// - Validate variable access patterns
	// - Ensure proper containerization
	
	return nil
}

// Custom filters for TRMNL templates

// truncateFilter truncates text to specified length
func truncateFilter(input interface{}, length interface{}) interface{} {
	str, ok := input.(string)
	if !ok {
		return input
	}
	
	lengthInt, ok := length.(int)
	if !ok {
		return input
	}
	
	if len(str) <= lengthInt {
		return str
	}
	
	return str[:lengthInt] + "..."
}

// escapeHTMLFilter escapes HTML characters
func escapeHTMLFilter(input interface{}) interface{} {
	str, ok := input.(string)
	if !ok {
		return input
	}
	
	return html.EscapeString(str)
}

// safeFilter marks content as safe (bypasses escaping)
func safeFilter(input interface{}) interface{} {
	// In a real implementation, this would use liquid's safe string type
	// For now, just return the input as-is
	return input
}

// formatDateFilter formats dates
func formatDateFilter(input interface{}, format interface{}) interface{} {
	// TODO: Implement proper date formatting
	// For now, just return the input
	return input
}

// formatTimeFilter formats time values
func formatTimeFilter(input interface{}, format interface{}) interface{} {
	// TODO: Implement proper time formatting
	// For now, just return the input
	return input
}