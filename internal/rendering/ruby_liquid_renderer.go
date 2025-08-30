package rendering

import (
	"context"
	"fmt"
	"strings"

	"github.com/rmitchellscott/stationmaster/internal/ruby"
)

// RubyLiquidRenderer handles server-side liquid template rendering using Ruby trmnl-liquid
type RubyLiquidRenderer struct {
	liquidRenderer *ruby.LiquidRenderer
	baseRenderer   *BaseHTMLRenderer
}

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

// NewRubyLiquidRenderer creates a new Ruby liquid renderer
func NewRubyLiquidRenderer(appDir string) (*RubyLiquidRenderer, error) {
	liquidRenderer, err := ruby.NewLiquidRenderer(appDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ruby liquid renderer: %w", err)
	}
	
	return &RubyLiquidRenderer{
		liquidRenderer: liquidRenderer,
		baseRenderer:   NewBaseHTMLRenderer(),
	}, nil
}

// RenderToHTML processes a liquid template and data, returning complete HTML ready for browserless
func (r *RubyLiquidRenderer) RenderToHTML(ctx context.Context, opts PluginRenderOptions) (string, error) {
	// Use the new separated flow: ProcessTemplate + WrapWithTRNMLAssets
	processedContent, err := r.ProcessTemplate(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("failed to process template: %w", err)
	}
	
	// Generate HTML structure with proper TRMNL wrapper for private plugins
	htmlContent := r.generateHTMLStructure(processedContent, opts)
	
	// Use external function to wrap with TRMNL assets
	assetsManager := NewHTMLAssetsManager()
	html := assetsManager.WrapWithTRNMLAssets(
		htmlContent,
		opts.PluginName,
		opts.Width,
		opts.Height,
		opts.RemoveBleedMargin,
		opts.EnableDarkMode,
	)
	
	return html, nil
}

// ProcessTemplate processes a liquid template and returns just the processed content without CSS/JS injection
func (r *RubyLiquidRenderer) ProcessTemplate(ctx context.Context, opts PluginRenderOptions) (string, error) {
	
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
	
	
	// Process template with Ruby liquid renderer
	renderedContent, err := r.liquidRenderer.RenderTemplate(ctx, combinedTemplate, opts.Data)
	if err != nil {
		return "", fmt.Errorf("Ruby liquid rendering failed: %w", err)
	}
	
	// Post-process the rendered content
	processedContent := r.postProcessTemplate(renderedContent, opts)
	
	// For ProcessTemplate, don't add TRMNL wrapper - let caller handle final structure
	// Just return the post-processed content
	return processedContent, nil
}

// postProcessTemplate performs post-processing on rendered template content
func (r *RubyLiquidRenderer) postProcessTemplate(content string, opts PluginRenderOptions) string {
	// Apply view class enhancements (from private plugin renderer)
	replacements := map[string]string{
		"view-title":          "view-title view-title--sm color-fg-secondary",
		"view-quote":          "view-quote view-quote--md",
		"view-text":           "view-text view-text--md",
		"view-text view-text--md view-text--lg": "view-text view-text--lg", // Avoid double enhancement
		"view-text--sm":       "view-text view-text--sm",
		"view-label":          "view-label view-label--xs color-fg-tertiary",
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
func (r *RubyLiquidRenderer) generateHTMLStructure(content string, opts PluginRenderOptions) string {
	// Build screen classes based on options (copied from working backup)
	screenClasses := []string{"screen"}
	if opts.RemoveBleedMargin {
		screenClasses = append(screenClasses, "screen--no-bleed")
	}
	if opts.EnableDarkMode {
		screenClasses = append(screenClasses, "screen--dark")
	}
	
	screenClassStr := ""
	for i, class := range screenClasses {
		if i > 0 {
			screenClassStr += " "
		}
		screenClassStr += class
	}
	
	// Check if content already has view class (like the working backup did)
	hasViewClass := strings.Contains(content, "class=\"view")
	
	var wrappedContent string
	if hasViewClass {
		// Content already has view class, just wrap with environment and screen
		wrappedContent = fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
			<div class="%s">%s</div>
		</div>`, opts.InstanceID, screenClassStr, content)
	} else {
		// Content needs view wrapper
		wrappedContent = fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
			<div class="%s">
				<div class="view view--full">%s</div>
			</div>
		</div>`, opts.InstanceID, screenClassStr, content)
	}
	
	return wrappedContent
}

// generateServerSideHTML creates complete HTML document using shared utility
func (r *RubyLiquidRenderer) generateServerSideHTML(opts BaseHTMLOptions, content string) string {
	// Use shared HTML assets manager for consistent document generation
	assetsManager := NewHTMLAssetsManager()
	
	// Additional CSS for error handling and mashup-specific styles
	additionalCSS := `
        #error {
            display: none;
            padding: 20px;
            background: #fee;
            border: 1px solid #fcc;
            color: #c00;
            font-family: monospace;
            font-size: 12px;
            white-space: pre-wrap;
        }
        /* Dark mode inversion CSS */
        .screen--dark-mode {
            filter: invert(1);
        }
        .screen--dark-mode .image {
            filter: invert(1);
        }
        .mashup-error {
            padding: 10px;
            background: #fee;
            border: 1px solid #fcc;
            color: #c00;
            font-family: monospace;
            font-size: 12px;
        }
        .mashup-empty-slot {
            display: flex;
            align-items: center;
            justify-content: center;
            height: 100%;
            background: #f5f5f5;
            border: 2px dashed #ccc;
            color: #999;
        }
    `
	
	// Wrap content with error div for compatibility
	contentWithError := `<div id="error"></div>` + content
	
	return assetsManager.GenerateCompleteHTMLDocument(HTMLDocumentOptions{
		Title:             opts.Title,
		Width:             opts.Width,
		Height:            opts.Height,
		Content:           contentWithError,
		AdditionalCSS:     additionalCSS,
		AdditionalJS:      "",
		RemoveBleedMargin: opts.RemoveBleedMargin,
		EnableDarkMode:    opts.EnableDarkMode,
	})
}

// Deprecated methods kept for backward compatibility - delegate to shared utility
func (r *RubyLiquidRenderer) generateHeadScripts() string {
	assetsManager := NewHTMLAssetsManager()
	return assetsManager.GenerateTRNMLHeadScripts()
}

func (r *RubyLiquidRenderer) generateSharedJavaScriptFunctions() string {
	assetsManager := NewHTMLAssetsManager()
	return assetsManager.GenerateSharedJavaScript()
}

func (r *RubyLiquidRenderer) generateServerSideJS() string {
	assetsManager := NewHTMLAssetsManager()
	return assetsManager.GenerateServerSideInitializationJS()
}

// ValidateTemplate validates a liquid template
func (r *RubyLiquidRenderer) ValidateTemplate(ctx context.Context, template string) error {
	return r.liquidRenderer.ValidateTemplate(ctx, template)
}

// Helper functions
func getDataKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}