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
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("[GO DEBUG] RUBY LIQUID RENDERER - Processing template\n")
	fmt.Printf(strings.Repeat("=", 80) + "\n")
	fmt.Printf("[GO DEBUG] SharedMarkup length: %d chars\n", len(opts.SharedMarkup))
	fmt.Printf("[GO DEBUG] LayoutTemplate length: %d chars\n", len(opts.LayoutTemplate))
	
	if opts.SharedMarkup != "" {
		fmt.Printf("[GO DEBUG] SharedMarkup preview (first 200 chars):\n%s\n", opts.SharedMarkup[:min(200, len(opts.SharedMarkup))])
	}
	if opts.LayoutTemplate != "" {
		fmt.Printf("[GO DEBUG] LayoutTemplate preview (first 200 chars):\n%s\n", opts.LayoutTemplate[:min(200, len(opts.LayoutTemplate))])
	}
	
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
	
	fmt.Printf("[GO DEBUG] Combined template length: %d chars\n", len(combinedTemplate))
	fmt.Printf("[GO DEBUG] Data structure keys: %v\n", getDataKeys(opts.Data))
	
	// Debug data structure
	for key, value := range opts.Data {
		fmt.Printf("[GO DEBUG] Data[%s]: %T", key, value)
		if m, ok := value.(map[string]interface{}); ok {
			fmt.Printf(" (map with %d keys: %v)", len(m), getMapKeys(m))
		} else if s, ok := value.(string); ok && len(s) < 100 {
			fmt.Printf(" = %q", s)
		} else if s, ok := value.(string); ok {
			fmt.Printf(" = %q... (%d chars)", s[:min(50, len(s))], len(s))
		}
		fmt.Printf("\n")
	}
	
	// Process template with Ruby liquid renderer
	fmt.Printf("[GO DEBUG] Sending to Ruby renderer...\n")
	renderedContent, err := r.liquidRenderer.RenderTemplate(ctx, combinedTemplate, opts.Data)
	if err != nil {
		fmt.Printf("[GO DEBUG] Ruby rendering FAILED: %v\n", err)
		return "", fmt.Errorf("Ruby liquid rendering failed: %w", err)
	}
	
	fmt.Printf("[GO DEBUG] Ruby rendering SUCCESS - output length: %d chars\n", len(renderedContent))
	fmt.Printf("[GO DEBUG] Rendered content preview (first 300 chars):\n%s\n", renderedContent[:min(300, len(renderedContent))])
	
	// Post-process the rendered content similar to the original client-side logic
	processedContent := r.postProcessTemplate(renderedContent, opts)
	
	// Generate base HTML options
	baseOpts := BaseHTMLOptions{
		Width:             opts.Width,
		Height:            opts.Height,
		Title:             opts.PluginName,
		RemoveBleedMargin: opts.RemoveBleedMargin,
		EnableDarkMode:    opts.EnableDarkMode,
		ScriptLoadStrategy: ScriptLoadSequential, // Still need TRMNL scripts for styling
	}
	
	// Create HTML structure without liquidjs - content is already rendered
	htmlContent := r.generateHTMLStructure(processedContent, opts)
	
	// Generate custom HTML without liquidjs since content is already rendered
	html := r.generateServerSideHTML(baseOpts, htmlContent)
	
	return html, nil
}

// PluginRenderOptions contains options for Ruby liquid rendering of plugins
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

// postProcessTemplate applies the same post-processing logic as the original client-side renderer
func (r *RubyLiquidRenderer) postProcessTemplate(renderedContent string, opts PluginRenderOptions) string {
	processedTemplate := renderedContent
	
	// Handle view_type variables (fallback)
	processedTemplate = strings.ReplaceAll(processedTemplate, "{{ view_type }}", "view--full")
	
	// Enhance view classes (same logic as client-side)
	processedTemplate = r.enhanceViewClasses(processedTemplate)
	
	return processedTemplate
}

// enhanceViewClasses replicates the JavaScript enhanceViewClasses function
func (r *RubyLiquidRenderer) enhanceViewClasses(template string) string {
	// This is a simplified Go version of the JavaScript enhanceViewClasses function
	// For now, we'll implement basic view class enhancement
	
	// Check if template already has layout modifiers
	if strings.Contains(template, "view--full") || 
		strings.Contains(template, "view--half") || 
		strings.Contains(template, "view--quadrant") {
		return template
	}
	
	// Replace standalone 'view' with 'view view--full' in class attributes
	// This is a basic implementation - could be enhanced with proper HTML parsing if needed
	template = strings.ReplaceAll(template, `class="view"`, `class="view view--full"`)
	template = strings.ReplaceAll(template, `class='view'`, `class='view view--full'`)
	
	return template
}

// generateHTMLStructure creates the TRMNL HTML structure wrapper
func (r *RubyLiquidRenderer) generateHTMLStructure(content string, opts PluginRenderOptions) string {
	// Build screen classes
	screenClasses := []string{"screen"}
	if opts.RemoveBleedMargin {
		screenClasses = append(screenClasses, "screen--no-bleed")
	}
	if opts.EnableDarkMode {
		screenClasses = append(screenClasses, "screen--dark-mode")
	}
	screenClassString := strings.Join(screenClasses, " ")
	
	// Check if content has view classes
	hasViewClass := strings.Contains(content, `class="view`) || strings.Contains(content, `class='view`)
	
	// Get layout mashup info (reuse existing logic)
	mashupClass, slotPosition := getLayoutMashupInfo(opts.Layout)
	
	var wrappedContent string
	
	// Check if this is a non-full layout that needs mashup structure
	if mashupClass != "" && slotPosition != "" {
		// Create mashup structure for positioned layouts
		var innerContent string
		if hasViewClass {
			innerContent = content
		} else {
			// Wrap in appropriate view class based on layout
			viewClass := "view--full"
			switch opts.Layout {
			case "half_vertical":
				viewClass = "view--half_vertical"
			case "half_horizontal":
				viewClass = "view--half_horizontal"
			case "quadrant":
				viewClass = "view--quadrant"
			}
			
			innerContent = fmt.Sprintf(`<div class="view %s">%s</div>`, viewClass, content)
		}
		
		wrappedContent = fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
			<div class="%s">
				<div class="mashup %s">
					<div id="slot-%s" class="view-slot">
						%s
					</div>
				</div>
			</div>
		</div>`, opts.InstanceID, screenClassString, mashupClass, slotPosition, innerContent)
	} else {
		// Standard full layout structure
		if hasViewClass {
			wrappedContent = fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
				<div class="%s">%s</div>
			</div>`, opts.InstanceID, screenClassString, content)
		} else {
			wrappedContent = fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
				<div class="%s">
					<div class="view view--full">%s</div>
				</div>
			</div>`, opts.InstanceID, screenClassString, content)
		}
	}
	
	return wrappedContent
}

// generateServerSideHTML creates complete HTML document for server-side rendered content (no liquidjs)
func (r *RubyLiquidRenderer) generateServerSideHTML(opts BaseHTMLOptions, content string) string {
	// Generate TRMNL scripts section for head loading
	scriptsSection := r.generateHeadScripts()
	
	// Create the complete HTML document without liquidjs
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@100..900&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://usetrmnl.com/css/latest/plugins.css">
    %s
    <style>
        body { 
            width: %dpx; 
            height: %dpx; 
            margin: 0; 
            padding: 0;
        }
        #loading {
            display: flex;
            align-items: center;
            justify-content: center;
            height: 100vh;
            font-family: Arial, sans-serif;
            color: #666;
        }
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
            height: 100%%;
            background: #f5f5f5;
            border: 2px dashed #ccc;
            color: #999;
        }
    </style>
</head>
<body>
    <div id="loading">Loading template...</div>
    <div id="error"></div>
    <div id="output" style="display: block;">%s</div>
    
    <script>
        %s
        %s
    </script>
</body>
</html>`,
		opts.Title,
		scriptsSection,
		opts.Width,
		opts.Height,
		content, // Pre-rendered content goes directly in output div
		r.generateSharedJavaScriptFunctions(),
		r.generateServerSideJS(),
	)
}

// generateHeadScripts returns TRMNL scripts to be loaded in the document head
func (r *RubyLiquidRenderer) generateHeadScripts() string {
	return `<!-- TRMNL Scripts for core functionality, filters, and rendering -->
    <script src="https://usetrmnl.com/js/latest/plugins.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-bfbd7e9488fd0d6dff2f619b5cb963c0772a24d6d0b537f60089dc53aa4746ff.js"></script>
    <script src="https://usetrmnl.com/assets/plugin_legacy-0c72702a185603fd7fc5eb915658f49486903cb5c92cd6153a336b8ce3973452.js"></script>
    <script src="https://usetrmnl.com/assets/plugin_demo-25268352c5a400b970985521a5eaa3dc90c736ce0cbf42d749e7e253f0c227f5.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/plugins-332ca4207dd02576b3641691907cb829ef52a36c4a092a75324a8fc860906967.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/plugins_legacy-a6b0b3aeac32ca71413f1febc053c59a528d4c6bb2173c22bd94ff8e0b9650f1.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/dithering-d697f6229e3bd6e2455425d647e5395bb608999c2039a9837a903c7c7e952d61.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/asset-deduplication-39fa2231b7a5bd5bedf4a1782b6a95d8b87eb3aaaa5e2b6cee287133d858bc96.js"></script>`
}

// generateSharedJavaScriptFunctions returns shared JavaScript functions (without liquid processing)
func (r *RubyLiquidRenderer) generateSharedJavaScriptFunctions() string {
	return `
        // Server-side rendering completion signal
        setTimeout(() => {
            if (document.body && !document.body.hasAttribute('data-render-complete')) {
                console.log('FALLBACK: Setting completion signal after 3-second timeout');
                document.body.setAttribute('data-render-complete', 'true');
            }
        }, 3000);
        
        console.log('Server-side Ruby rendered template loaded successfully');
        
        function executeInnerHTMLScripts(containerElement) {
            // Execute any script tags that were inserted via innerHTML
            const scripts = containerElement.querySelectorAll('script');
            scripts.forEach(script => {
                console.log('Executing template script:', script.textContent.substring(0, 100) + '...');
                let scriptContent = script.textContent;
                
                // Make functions globally accessible
                scriptContent = scriptContent.replace(/const\s+(\w+)\s*=\s*\(/g, 'window.$1 = (');
                
                const newScript = document.createElement('script');
                newScript.textContent = scriptContent;
                document.body.appendChild(newScript);
            });
            
            // Handle DOMContentLoaded timing issue
            setTimeout(() => {
                console.log('Dispatching DOMContentLoaded event for template scripts');
                document.dispatchEvent(new Event('DOMContentLoaded'));
            }, 50);
        }
        
        function loadTRNMLScriptsSequentially(callback) {
            // Scripts are already loaded in head, just trigger callback
            if (callback) callback();
        }
        
        function handleDitheringTiming() {
            // Check if window.load already fired and handle dithering timing
            if (document.readyState === 'complete') {
                if (typeof window.setup === 'function') {
                    console.log('Triggering dithering via window.setup()');
                    window.setup();
                } else {
                    console.log('Triggering dithering via window load event');
                    window.dispatchEvent(new Event('load'));
                }
            } else {
                window.addEventListener('load', function() {
                    console.log('Page loaded, triggering dithering');
                    if (typeof window.setup === 'function') {
                        window.setup();
                    }
                });
            }
        }
        
        function waitForFontsAndComplete() {
            try {
                if (document.fonts && document.fonts.status === 'loaded') {
                    console.log('Fonts loaded - setting completion signal');
                    if (document.body) {
                        document.body.setAttribute('data-render-complete', 'true');
                    }
                } else {
                    if (document.fonts) {
                        console.log('Waiting for fonts to load...');
                        setTimeout(waitForFontsAndComplete, 100);
                    } else {
                        console.log('document.fonts API not available - completing immediately');
                        if (document.body) {
                            document.body.setAttribute('data-render-complete', 'true');
                        }
                    }
                }
            } catch (error) {
                console.error('Error in waitForFontsAndComplete():', error);
                if (document.body) {
                    document.body.setAttribute('data-render-complete', 'true');
                }
            }
        }
        
        function signalRenderingComplete() {
            try {
                console.log('signalRenderingComplete() called - starting font loading check');
                waitForFontsAndComplete();
                
                setTimeout(() => {
                    if (document.body && !document.body.hasAttribute('data-render-complete')) {
                        console.log('Font loading timeout - completing anyway');
                        document.body.setAttribute('data-render-complete', 'true');
                    }
                }, 1500);
            } catch (error) {
                console.error('Error in signalRenderingComplete():', error);
                if (document.body) {
                    document.body.setAttribute('data-render-complete', 'true');
                }
            }
        }
	`
}

// generateServerSideJS creates JavaScript for the server-side rendered content
func (r *RubyLiquidRenderer) generateServerSideJS() string {
	return `
		// Server-side rendered content - no liquidjs needed
		function initializeServerRendered() {
			console.log('Server-side rendered template loaded');
			
			const loadingEl = document.getElementById('loading');
			const outputEl = document.getElementById('output');
			
			if (loadingEl) {
				loadingEl.style.display = 'none';
			}
			
			if (outputEl) {
				// Content is already in the output element from server rendering
				// Execute any inline scripts in the rendered content
				executeInnerHTMLScripts(outputEl);
			}
			
			// Trigger dithering setup
			handleDitheringTiming();
			
			// Signal rendering complete
			signalRenderingComplete();
		}
		
		// Initialize when DOM is ready
		if (document.readyState === 'loading') {
			document.addEventListener('DOMContentLoaded', initializeServerRendered);
		} else {
			initializeServerRendered();
		}
		
		// Global error handler for uncaught errors
		window.addEventListener('error', function(e) {
			console.error('UNCAUGHT ERROR during server-side rendering:', e.error, e);
			console.error('Error occurred at:', e.filename, 'line:', e.lineno);
			if (document.body && !document.body.hasAttribute('data-render-complete')) {
				console.log('Setting completion signal due to JavaScript error');
				document.body.setAttribute('data-render-complete', 'true');
			}
		});
		
		// Also catch unhandled promise rejections
		window.addEventListener('unhandledrejection', function(e) {
			console.error('UNHANDLED PROMISE REJECTION:', e.reason);
			if (document.body && !document.body.hasAttribute('data-render-complete')) {
				console.log('Setting completion signal due to promise rejection');
				document.body.setAttribute('data-render-complete', 'true');
			}
		});
	`
}

// ValidateTemplate validates a liquid template
func (r *RubyLiquidRenderer) ValidateTemplate(ctx context.Context, template string) error {
	return r.liquidRenderer.ValidateTemplate(ctx, template)
}

// getLayoutMashupInfo returns mashup CSS class and slot position (reuse existing function)
func getLayoutMashupInfo(layout string) (string, string) {
	switch layout {
	case "half_vertical":
		return "mashup--1Lx1R", "left"
	case "half_horizontal":
		return "mashup--1Tx1B", "top"
	case "quadrant":
		return "mashup--2Lx2R", "q1"
	default:
		return "", ""
	}
}

// Helper functions for debugging
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