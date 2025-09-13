package rendering

import (
	"context"
	"fmt"
	"strings"
	
	"github.com/rmitchellscott/stationmaster/internal/config"
)

// PluginRenderOptions contains options for rendering a plugin template
type PluginRenderOptions struct {
	SharedMarkup      string
	LayoutTemplate    string
	Data              map[string]interface{}
	Width             int
	Height            int
	PluginName        string
	InstanceID        string
	InstanceName      string
	RemoveBleedMargin bool
	EnableDarkMode    bool
	Layout            string // Layout type for proper mashup CSS structure
	LayoutWidth       int    // Layout-specific width for positioning
	LayoutHeight      int    // Layout-specific height for positioning
}

// UnifiedRenderer handles template rendering using external Ruby service with TRMNL asset wrapping
type UnifiedRenderer struct {
	externalRubyService *ExternalRubyService
}

// NewUnifiedRenderer creates a new unified renderer
func NewUnifiedRenderer() *UnifiedRenderer {
	return &UnifiedRenderer{
		externalRubyService: NewExternalRubyService(),
	}
}

// RenderToHTML processes a liquid template and data, returning complete HTML ready for browserless
func (r *UnifiedRenderer) RenderToHTML(ctx context.Context, opts PluginRenderOptions) (string, error) {
	// Process template using external Ruby service
	processedContent, err := r.ProcessTemplate(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("failed to process template: %w", err)
	}

	// Generate HTML structure with proper TRMNL wrapper
	htmlContent := r.generateHTMLStructure(processedContent, opts)

	// Use external function to wrap with TRMNL assets
	assetsManager := NewHTMLAssetsManager()
	assetBaseURL := config.GetAssetBaseURL()
	html := assetsManager.WrapWithTRNMLAssets(
		htmlContent,
		opts.PluginName,
		opts.Width,
		opts.Height,
		opts.RemoveBleedMargin,
		opts.EnableDarkMode,
		assetBaseURL,
	)

	return html, nil
}

// ProcessTemplate processes a liquid template and returns just the processed content without CSS/JS injection
func (r *UnifiedRenderer) ProcessTemplate(ctx context.Context, opts PluginRenderOptions) (string, error) {
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

	// Process template with external Ruby service
	renderedContent, err := r.externalRubyService.RenderTemplate(ctx, combinedTemplate, opts.Data)
	if err != nil {
		return "", fmt.Errorf("external Ruby service rendering failed: %w", err)
	}

	// Post-process the rendered content
	processedContent := r.postProcessTemplate(renderedContent, opts)

	return processedContent, nil
}

// postProcessTemplate performs post-processing on rendered template content
func (r *UnifiedRenderer) postProcessTemplate(content string, opts PluginRenderOptions) string {
	// Apply view class enhancements
	replacements := map[string]string{
		"view-title":                         "view-title view-title--sm color-fg-secondary",
		"view-quote":                         "view-quote view-quote--md",
		"view-text":                          "view-text view-text--md",
		"view-text view-text--md view-text--lg": "view-text view-text--lg", // Avoid double enhancement
		"view-text--sm":                      "view-text view-text--sm",
		"view-label":                         "view-label view-label--xs color-fg-tertiary",
	}

	enhanced := content
	for old, new := range replacements {
		if !strings.Contains(enhanced, new) { // Avoid double enhancement
			enhanced = strings.ReplaceAll(enhanced, old, new)
		}
	}

	return enhanced
}

// generateHTMLStructure wraps processed content with TRMNL-appropriate structure
func (r *UnifiedRenderer) generateHTMLStructure(content string, opts PluginRenderOptions) string {
	// Build screen classes based on options
	screenClasses := []string{"screen"}
	if opts.RemoveBleedMargin {
		screenClasses = append(screenClasses, "screen--no-bleed")
	}
	if opts.EnableDarkMode {
		screenClasses = append(screenClasses, "screen--dark-mode")
	}

	screenClassStr := ""
	for i, class := range screenClasses {
		if i > 0 {
			screenClassStr += " "
		}
		screenClassStr += class
	}

	// Always wrap content with TRMNL structure including screen classes
	wrappedContent := fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
		<div class="%s">
			<div class="view view--full">%s</div>
		</div>
	</div>`, opts.InstanceID, screenClassStr, content)

	return wrappedContent
}

// ValidateTemplate validates a liquid template using external service
func (r *UnifiedRenderer) ValidateTemplate(ctx context.Context, template string) error {
	return r.externalRubyService.ValidateTemplate(ctx, template)
}

// IsServiceAvailable checks if the external Ruby service is available
func (r *UnifiedRenderer) IsServiceAvailable(ctx context.Context) bool {
	return r.externalRubyService.IsServiceAvailable(ctx)
}