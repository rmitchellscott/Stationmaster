package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rmitchellscott/stationmaster/internal/config"
)

// ValidationResult represents the result of template validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Message  string   `json:"message"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors"`
}

// TemplateValidator provides validation for liquid templates
type TemplateValidator struct {
}

// NewTemplateValidator creates a new template validator
func NewTemplateValidator() *TemplateValidator {
	return &TemplateValidator{}
}

// ValidateTemplate validates a liquid template for syntax and security
func (v *TemplateValidator) ValidateTemplate(template string, templateName string) ValidationResult {
	return v.ValidateTemplateWithOptions(template, templateName, false)
}

// ValidateTemplateWithOptions validates a liquid template with configurable options
func (v *TemplateValidator) ValidateTemplateWithOptions(template string, templateName string, skipContainer bool) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Message:  "Template validation successful",
		Warnings: []string{},
		Errors:   []string{},
	}

	// Skip validation if template is empty
	if strings.TrimSpace(template) == "" {
		return result
	}

	// 1. Basic liquid syntax validation (pattern-based since we use server-side Ruby Liquid)
	syntaxWarnings := v.validateBasicLiquidSyntax(template, templateName)
	result.Warnings = append(result.Warnings, syntaxWarnings...)

	// Check for advanced Liquid syntax
	if strings.Contains(template, "{%liquid%}") || strings.Contains(template, "{% liquid %}") {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s: Template uses {%%liquid%%} blocks - using server-side Ruby Liquid for full compatibility", templateName))
	}

	// 2. Check for security issues
	securityErrors, securityWarnings := v.checkSecurity(template, templateName)
	result.Errors = append(result.Errors, securityErrors...)
	result.Warnings = append(result.Warnings, securityWarnings...)

	// 3. Verify containerization (skip for shared markup templates)
	if !skipContainer {
		containerErrors, containerWarnings := v.verifyContainerization(template, templateName)
		result.Errors = append(result.Errors, containerErrors...)
		result.Warnings = append(result.Warnings, containerWarnings...)
	}

	// Set overall validity
	result.Valid = len(result.Errors) == 0

	if !result.Valid {
		result.Message = "Template validation failed"
	} else if len(result.Warnings) > 0 {
		result.Message = "Template validation passed with warnings"
	}

	return result
}

// validateBasicLiquidSyntax performs basic pattern-based liquid syntax validation
func (v *TemplateValidator) validateBasicLiquidSyntax(template string, templateName string) []string {
	var warnings []string
	
	// Check for unmatched liquid tags (basic balance checking)
	openTags := regexp.MustCompile(`\{\%\s*(\w+)`)
	closeTags := regexp.MustCompile(`\{\%\s*end(\w+)\s*\%\}`)
	
	openMatches := openTags.FindAllStringSubmatch(template, -1)
	closeMatches := closeTags.FindAllStringSubmatch(template, -1)
	
	// Simple tag balance check for common block tags
	blockTags := map[string]int{}
	for _, match := range openMatches {
		if len(match) > 1 {
			tag := match[1]
			// Only track block tags that need closing
			if tag == "if" || tag == "unless" || tag == "for" || tag == "case" || tag == "capture" || tag == "comment" {
				blockTags[tag]++
			}
		}
	}
	
	for _, match := range closeMatches {
		if len(match) > 1 {
			tag := match[1]
			if count, exists := blockTags[tag]; exists {
				if count > 0 {
					blockTags[tag]--
				}
			}
		}
	}
	
	// Check for unmatched tags
	for tag, count := range blockTags {
		if count != 0 {
			warnings = append(warnings, fmt.Sprintf("%s: Potentially unmatched {%%%% %s %%}} tags detected", templateName, tag))
		}
	}
	
	return warnings
}

// checkSecurity looks for potentially dangerous patterns
func (v *TemplateValidator) checkSecurity(template string, templateName string) ([]string, []string) {
	var errors []string
	var warnings []string

	// Check if external scripts are allowed (development/testing mode)
	if config.Get("ALLOW_EXTERNAL_SCRIPTS", "false") == "true" {
		return errors, warnings // Skip all security checks when explicitly enabled
	}

	// Dangerous patterns to block
	dangerousPatterns := map[string]string{
		`<script[^>]*>`:                    "Script tags are not allowed for security reasons",
		`javascript:`:                      "JavaScript URLs are not allowed",
		`on\w+\s*=`:                        "JavaScript event handlers are not allowed",
		`<iframe[^>]*>`:                    "Iframe tags are not allowed",
		`<object[^>]*>`:                    "Object tags are not allowed", 
		`<embed[^>]*>`:                     "Embed tags are not allowed",
		`<link[^>]*rel\s*=\s*["']?stylesheet`: "External stylesheets are not allowed (use TRMNL framework only)",
		`@import`:                          "CSS imports are not allowed",
		`url\s*\(\s*["']?(?!data:|#)`:     "External URLs in CSS are not allowed",
	}

	// Check for dangerous patterns
	for pattern, message := range dangerousPatterns {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, template); matched {
			errors = append(errors, fmt.Sprintf("%s: %s", templateName, message))
		}
	}

	// Warning patterns
	warningPatterns := map[string]string{
		`<style[^>]*>`:                     "Inline styles found - consider using TRMNL framework classes",
		`style\s*=\s*["'][^"']*["']`:      "Inline style attributes found - consider using TRMNL framework classes",
		`position\s*:\s*fixed`:            "Fixed positioning may not work as expected on e-ink displays",
		`animation`:                       "CSS animations are not supported on e-ink displays",
		`transition`:                      "CSS transitions are not supported on e-ink displays",
	}

	for pattern, message := range warningPatterns {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, template); matched {
			warnings = append(warnings, fmt.Sprintf("%s: %s", templateName, message))
		}
	}

	return errors, warnings
}

// verifyContainerization ensures templates use proper containerization
func (v *TemplateValidator) verifyContainerization(template string, templateName string) ([]string, []string) {
	var errors []string
	var warnings []string

	// Check for container div with unique ID (Stationmaster format)
	hasContainer := regexp.MustCompile(`<div[^>]*id\s*=\s*["']plugin-{{ instance_id }}["'][^>]*>`).MatchString(template)
	hasContainerClass := regexp.MustCompile(`<div[^>]*class\s*=\s*["'][^"']*plugin-container[^"']*["'][^>]*>`).MatchString(template)
	
	// Check for TRMNL layout format (which is what our actual plugins use)
	hasTrmnlLayout := regexp.MustCompile(`<div[^>]*class\s*=\s*["'][^"']*layout[^"']*["'][^>]*>`).MatchString(template)

	if !hasContainer && !hasContainerClass && !hasTrmnlLayout {
		errors = append(errors, fmt.Sprintf("%s: Template must include a container div with either id='plugin-{{ instance_id }}', class='plugin-container', or TRMNL's class='layout'", templateName))
	}

	// Check for proper TRMNL framework usage
	hasTrmnlClasses := regexp.MustCompile(`class\s*=\s*["'][^"']*view--[^"']*["']`).MatchString(template)
	if !hasTrmnlClasses {
		warnings = append(warnings, fmt.Sprintf("%s: Consider using TRMNL framework classes (view--, text--, etc.)", templateName))
	}

	// Check for absolute positioning which might break containerization
	hasAbsolutePos := regexp.MustCompile(`position\s*:\s*absolute`).MatchString(template)
	if hasAbsolutePos {
		warnings = append(warnings, fmt.Sprintf("%s: Absolute positioning detected - ensure it works within container boundaries", templateName))
	}

	return errors, warnings
}

// mergeTemplates combines shared markup with a layout template
func (v *TemplateValidator) mergeTemplates(sharedMarkup, layoutTemplate string) string {
	sharedTrimmed := strings.TrimSpace(sharedMarkup)
	layoutTrimmed := strings.TrimSpace(layoutTemplate)
	
	if sharedTrimmed == "" {
		return layoutTemplate
	}
	if layoutTrimmed == "" {
		return sharedMarkup
	}
	
	// Merge shared markup with layout template
	// Shared markup typically contains common elements, layout has specific content
	return sharedTrimmed + "\n" + layoutTrimmed
}

// ValidateAllTemplates validates all layout templates for a private plugin
func (v *TemplateValidator) ValidateAllTemplates(fullTemplate, halfVertTemplate, halfHorizTemplate, quadrantTemplate, sharedTemplate string) ValidationResult {
	combinedResult := ValidationResult{
		Valid:    true,
		Message:  "All templates validated successfully",
		Warnings: []string{},
		Errors:   []string{},
	}

	// First, validate shared markup separately if it exists (skip container requirement)
	if strings.TrimSpace(sharedTemplate) != "" {
		sharedResult := v.ValidateTemplateWithOptions(sharedTemplate, "shared markup", true)
		combinedResult.Errors = append(combinedResult.Errors, sharedResult.Errors...)
		combinedResult.Warnings = append(combinedResult.Warnings, sharedResult.Warnings...)
		
		if !sharedResult.Valid {
			combinedResult.Valid = false
		}
	}

	// Define layout templates to validate
	layoutTemplates := map[string]string{
		"full layout":       fullTemplate,
		"half vertical":     halfVertTemplate,
		"half horizontal":   halfHorizTemplate,
		"quadrant":         quadrantTemplate,
	}

	// Validate each layout template (merged with shared markup if applicable)
	for name, template := range layoutTemplates {
		if strings.TrimSpace(template) == "" {
			continue // Skip empty templates
		}

		// Merge with shared markup and validate the combined result
		mergedTemplate := v.mergeTemplates(sharedTemplate, template)
		result := v.ValidateTemplate(mergedTemplate, name)
		
		combinedResult.Errors = append(combinedResult.Errors, result.Errors...)
		combinedResult.Warnings = append(combinedResult.Warnings, result.Warnings...)
		
		if !result.Valid {
			combinedResult.Valid = false
		}
	}

	// Check that at least one layout template is provided
	hasAtLeastOneLayout := strings.TrimSpace(fullTemplate) != "" ||
		strings.TrimSpace(halfVertTemplate) != "" ||
		strings.TrimSpace(halfHorizTemplate) != "" ||
		strings.TrimSpace(quadrantTemplate) != ""

	if !hasAtLeastOneLayout {
		combinedResult.Valid = false
		combinedResult.Errors = append(combinedResult.Errors, "At least one layout template must be provided")
	}

	// Set final message
	if !combinedResult.Valid {
		combinedResult.Message = "Template validation failed"
	} else if len(combinedResult.Warnings) > 0 {
		combinedResult.Message = "Templates validated successfully with warnings"
	}

	return combinedResult
}