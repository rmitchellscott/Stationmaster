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
	DeviceModelName   string
	BitDepth          int
	ScreenOrientation string
}

// UnifiedRenderer handles template rendering using embedded Ruby renderer with TRMNL asset wrapping
type UnifiedRenderer struct {
	embeddedRenderer *EmbeddedLiquidRenderer
}

// NewUnifiedRenderer creates a new unified renderer
func NewUnifiedRenderer() *UnifiedRenderer {
	return &UnifiedRenderer{
		embeddedRenderer: NewEmbeddedLiquidRenderer(),
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

	// Process template with embedded Ruby renderer
	renderedContent, err := r.embeddedRenderer.RenderTemplate(ctx, combinedTemplate, opts.Data)
	if err != nil {
		return "", fmt.Errorf("embedded Ruby renderer failed: %w", err)
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
	screenClassStr := BuildScreenClasses(ScreenClassOptions{
		ModelName:         opts.DeviceModelName,
		BitDepth:          opts.BitDepth,
		ScreenWidth:       opts.Width,
		ScreenHeight:      opts.Height,
		ScreenOrientation: opts.ScreenOrientation,
		RemoveBleedMargin: opts.RemoveBleedMargin,
		EnableDarkMode:    opts.EnableDarkMode,
	})

	var inner string
	if slots := mashupSlots(opts.Layout); slots > 0 {
		viewClass, mashupClass := layoutToViewClass(opts.Layout)
		var slotBuilder strings.Builder
		slotBuilder.WriteString(fmt.Sprintf(`<div class="mashup %s">`, mashupClass))
		slotBuilder.WriteString(fmt.Sprintf(`<div class="view %s">%s</div>`, viewClass, content))
		for i := 1; i < slots; i++ {
			slotBuilder.WriteString(fmt.Sprintf(`<div class="view %s"></div>`, viewClass))
		}
		slotBuilder.WriteString(`</div>`)
		inner = slotBuilder.String()
	} else {
		inner = fmt.Sprintf(`<div class="view view--full">%s</div>`, content)
	}

	wrappedContent := fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
		<div class="%s">
			%s
		</div>
	</div>`, opts.InstanceID, screenClassStr, inner)

	return wrappedContent
}

func layoutToViewClass(layout string) (string, string) {
	switch layout {
	case "half_vertical":
		return "view--half_vertical", "mashup--1Lx1R"
	case "half_horizontal":
		return "view--half_horizontal", "mashup--1Tx1B"
	case "quadrant":
		return "view--quadrant", "mashup--2x2"
	default:
		return "view--full", ""
	}
}

func mashupSlots(layout string) int {
	switch layout {
	case "half_vertical", "half_horizontal":
		return 2
	case "quadrant":
		return 4
	default:
		return 0
	}
}

// ValidateTemplate validates a liquid template using embedded renderer
func (r *UnifiedRenderer) ValidateTemplate(ctx context.Context, template string) error {
	// Try to render with empty data to validate syntax
	_, err := r.embeddedRenderer.RenderTemplate(ctx, template, map[string]interface{}{})
	return err
}

// IsServiceAvailable checks if the embedded Ruby renderer is available
func (r *UnifiedRenderer) IsServiceAvailable(ctx context.Context) bool {
	return r.embeddedRenderer.IsAvailable()
}