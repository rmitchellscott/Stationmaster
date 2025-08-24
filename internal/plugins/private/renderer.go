package private

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// RenderOptions contains all options needed to render a private plugin to HTML
type RenderOptions struct {
	SharedMarkup   string
	LayoutTemplate string
	Data           map[string]interface{}
	Width          int
	Height         int
	PluginName     string
	InstanceID     string
}

// PrivatePluginRenderer handles HTML generation for private plugins
type PrivatePluginRenderer struct {
	liquidRenderer *LiquidRenderer
}

// NewPrivatePluginRenderer creates a new private plugin renderer
func NewPrivatePluginRenderer() *PrivatePluginRenderer {
	return &PrivatePluginRenderer{
		liquidRenderer: NewLiquidRenderer(),
	}
}

// RenderToHTML generates complete HTML document for a private plugin
func (r *PrivatePluginRenderer) RenderToHTML(opts RenderOptions) (string, error) {
	// Combine shared markup with layout template
	combinedTemplate := opts.SharedMarkup
	if opts.LayoutTemplate != "" {
		if combinedTemplate != "" {
			combinedTemplate += "\n" + opts.LayoutTemplate
		} else {
			combinedTemplate = opts.LayoutTemplate
		}
	}
	
	if combinedTemplate == "" {
		return "", fmt.Errorf("no template content provided")
	}
	
	// Render template with liquid
	renderedContent, err := r.liquidRenderer.RenderTemplate(combinedTemplate, opts.Data)
	if err != nil {
		return "", fmt.Errorf("template render error: %w", err)
	}
	
	// Basic template variable substitution for compatibility
	processedTemplate := renderedContent
	processedTemplate = strings.ReplaceAll(processedTemplate, "{{ timestamp }}", time.Now().Format("2006-01-02 15:04:05"))
	processedTemplate = strings.ReplaceAll(processedTemplate, "{{timestamp}}", time.Now().Format("2006-01-02 15:04:05"))
	
	// Handle view_type variables
	if strings.Contains(processedTemplate, "{{view_type}}") {
		processedTemplate = strings.ReplaceAll(processedTemplate, "{{view_type}}", "view--full")
	}
	if strings.Contains(processedTemplate, "{{ view_type }}") {
		processedTemplate = strings.ReplaceAll(processedTemplate, "{{ view_type }}", "view--full")
	}
	
	// Enhanced view class detection and merging
	processedTemplate = enhanceViewClasses(processedTemplate)
	
	// After enhancement, check if we have view classes (they should now be properly formatted)
	hasViewClass := strings.Contains(processedTemplate, "class=\"view") || 
		strings.Contains(processedTemplate, "class='view")
	
	// Wrap user template in TRMNL framework structure
	var wrappedContent string
	if hasViewClass {
		// Template has view classes (enhanced with proper layout modifiers), just wrap with environment
		wrappedContent = fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
    <div class="screen">
        %s
    </div>
</div>`, opts.InstanceID, processedTemplate)
	} else {
		// No view classes found, auto-inject view wrapper
		wrappedContent = fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
    <div class="screen">
        <div class="view view--full">
            %s
        </div>
    </div>
</div>`, opts.InstanceID, processedTemplate)
	}
	
	// Create complete HTML document with TRMNL framework
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s</title>
    <link rel="stylesheet" href="https://usetrmnl.com/css/latest/plugins.css">
    <style>
        body { 
            width: %dpx; 
            height: %dpx; 
            margin: 0; 
            padding: 0;
        }
    </style>
</head>
<body>
    %s
    <script src="https://usetrmnl.com/js/latest/plugins.js"></script>
</body>
</html>`,
		opts.PluginName,
		opts.Width,
		opts.Height,
		wrappedContent)
	
	return html, nil
}

// enhanceViewClasses intelligently merges view--full with existing view classes
// that don't already have layout modifiers
func enhanceViewClasses(template string) string {
	// Regex to find class attributes (both single and double quotes)
	doubleQuoteRegex, err := regexp.Compile(`class="([^"]*view[^"]*)"`)
	if err != nil {
		return template
	}
	
	singleQuoteRegex, err := regexp.Compile(`class='([^']*view[^']*)'`)
	if err != nil {
		return template
	}
	
	// Regex to match standalone "view" class (word boundaries)
	viewWordRegex, err := regexp.Compile(`\bview\b`)
	if err != nil {
		return template
	}
	
	// Process double quotes first
	template = doubleQuoteRegex.ReplaceAllStringFunc(template, func(match string) string {
		parts := doubleQuoteRegex.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		
		classContent := parts[1]
		
		// Check if already has layout modifiers
		if strings.Contains(classContent, "view--full") ||
		   strings.Contains(classContent, "view--half") ||
		   strings.Contains(classContent, "view--quadrant") {
			return match
		}
		
		// Check if it contains standalone "view" class
		if viewWordRegex.MatchString(classContent) {
			enhancedClasses := viewWordRegex.ReplaceAllString(classContent, "view view--full")
			return fmt.Sprintf(`class="%s"`, enhancedClasses)
		}
		
		return match
	})
	
	// Process single quotes
	template = singleQuoteRegex.ReplaceAllStringFunc(template, func(match string) string {
		parts := singleQuoteRegex.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		
		classContent := parts[1]
		
		// Check if already has layout modifiers
		if strings.Contains(classContent, "view--full") ||
		   strings.Contains(classContent, "view--half") ||
		   strings.Contains(classContent, "view--quadrant") {
			return match
		}
		
		// Check if it contains standalone "view" class
		if viewWordRegex.MatchString(classContent) {
			enhancedClasses := viewWordRegex.ReplaceAllString(classContent, "view view--full")
			return fmt.Sprintf(`class='%s'`, enhancedClasses)
		}
		
		return match
	})
	
	return template
}