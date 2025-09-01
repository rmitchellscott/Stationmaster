package rendering

import (
	"fmt"
	"html/template"
	"strings"
)

// HTMLAssetsManager handles shared HTML generation for all plugin types
type HTMLAssetsManager struct{}

// NewHTMLAssetsManager creates a new HTML assets manager
func NewHTMLAssetsManager() *HTMLAssetsManager {
	return &HTMLAssetsManager{}
}

// GenerateTRNMLHeadScripts returns TRMNL scripts to be loaded in the document head
func (h *HTMLAssetsManager) GenerateTRNMLHeadScripts() string {
	return `<!-- TRMNL Scripts for core functionality, filters, and rendering -->
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@100..900&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://usetrmnl.com/css/latest/plugins.css">
    <script src="https://usetrmnl.com/js/latest/plugins.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-bfbd7e9488fd0d6dff2f619b5cb963c0772a24d6d0b537f60089dc53aa4746ff.js"></script>
    <script src="https://usetrmnl.com/assets/plugin_legacy-0c72702a185603fd7fc5eb915658f49486903cb5c92cd6153a336b8ce3973452.js"></script>
    <script src="https://usetrmnl.com/assets/plugin_demo-25268352c5a400b970985521a5eaa3dc90c736ce0cbf42d749e7e253f0c227f5.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/plugins-332ca4207dd02576b3641691907cb829ef52a36c4a092a75324a8fc860906967.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/plugins_legacy-a6b0b3aeac32ca71413f1febc053c59a528d4c6bb2173c22bd94ff8e0b9650f1.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/dithering-d697f6229e3bd6e2455425d647e5395bb608999c2039a9837a903c7c7e952d61.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/asset-deduplication-39fa2231b7a5bd5bedf4a1782b6a95d8b87eb3aaaa5e2b6cee287133d858bc96.js"></script>`
}

// GenerateSharedJavaScript returns exact working JavaScript from private plugin backup
func (h *HTMLAssetsManager) GenerateSharedJavaScript() string {
	return `
        // Server-side rendering completion signal
        setTimeout(() => {
            if (document.body && !document.body.hasAttribute('data-render-complete')) {
                console.log('FALLBACK: Setting completion signal after 3-second timeout');
                document.body.setAttribute('data-render-complete', 'true');
            }
        }, 3000);
        
        console.log('Server-side Ruby rendered template loaded successfully');
        
        let hasExecutedInnerScripts = false;
        
        window.executeInnerHTMLScripts = function(containerElement) {
            // Prevent infinite loop - only execute once
            if (hasExecutedInnerScripts) {
                console.log('Skipping executeInnerHTMLScripts - already executed');
                return;
            }
            hasExecutedInnerScripts = true;
            
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
            
            // CRITICAL: Handle DOMContentLoaded timing issue (from main branch)
            // Since DOM was already loaded when our scripts executed, any DOMContentLoaded 
            // event listeners in the template scripts never fired. Dispatch the event manually.
            setTimeout(() => {
                console.log('Dispatching DOMContentLoaded event for template scripts');
                document.dispatchEvent(new Event('DOMContentLoaded'));
            }, 50);
        }
        
        window.loadTRNMLScriptsSequentially = function(callback) {
            // Scripts are already loaded in head, just trigger callback
            if (callback) callback();
        }
        
        window.handleDitheringTiming = function() {
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
        
        window.waitForFontsAndComplete = function() {
            try {
                if (document.fonts && document.fonts.status === 'loaded') {
                    console.log('Fonts loaded - setting completion signal');
                    if (document.body) {
                        document.body.setAttribute('data-render-complete', 'true');
                    }
                } else {
                    if (document.fonts) {
                        console.log('Waiting for fonts to load...');
                        setTimeout(window.waitForFontsAndComplete, 100);
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
        
        window.signalRenderingComplete = function() {
            try {
                console.log('signalRenderingComplete() called - starting font loading check');
                window.waitForFontsAndComplete();
                
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

// GenerateServerSideInitializationJS returns JavaScript for server-side rendered content initialization
func (h *HTMLAssetsManager) GenerateServerSideInitializationJS() string {
	return `
		// Server-side rendered content - initialize
		function initializeServerRendered() {
			console.log('Server-side rendered template loaded');
			
			const loadingEl = document.getElementById('loading');
			const outputEl = document.getElementById('output');
			
			if (loadingEl) {
				loadingEl.style.display = 'none';
			}
			
			if (outputEl) {
				// Content is already in the output element from server rendering
				outputEl.style.display = 'block';
				// Execute any inline scripts in the rendered content
				window.executeInnerHTMLScripts(outputEl);
			}
			
			// Trigger dithering setup
			window.handleDitheringTiming();
			
			// Signal rendering complete
			window.signalRenderingComplete();
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

// HTMLDocumentOptions contains options for generating complete HTML documents
type HTMLDocumentOptions struct {
	Title             string
	Width             int
	Height            int
	Content           string
	AdditionalCSS     string
	AdditionalJS      string
	RemoveBleedMargin bool
	EnableDarkMode    bool
}

// GenerateCompleteHTMLDocument creates a complete HTML document with TRMNL assets and proper structure
func (h *HTMLAssetsManager) GenerateCompleteHTMLDocument(opts HTMLDocumentOptions) string {
	// Build screen classes based on options
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
	
	// Additional styles for loading states
	additionalStyles := fmt.Sprintf(`
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
        #output {
            display: none;
        }
        %s`, opts.Width, opts.Height, opts.AdditionalCSS)
	
	// Use string concatenation instead of fmt.Sprintf to avoid escaping issues with JavaScript content
	// The opts.Content may contain complete HTML with <script> tags and regex patterns that get mangled by fmt.Sprintf
	var htmlBuilder strings.Builder
	
	htmlBuilder.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	htmlBuilder.WriteString("    <meta charset=\"utf-8\">\n")
	htmlBuilder.WriteString("    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	htmlBuilder.WriteString("    <title>")
	htmlBuilder.WriteString(template.HTMLEscapeString(opts.Title))
	htmlBuilder.WriteString("</title>\n")
	htmlBuilder.WriteString("    ")
	htmlBuilder.WriteString(h.GenerateTRNMLHeadScripts())
	htmlBuilder.WriteString("\n    <style>")
	htmlBuilder.WriteString(additionalStyles)
	htmlBuilder.WriteString("</style>\n</head>\n<body>\n")
	htmlBuilder.WriteString("    <div id=\"loading\">Loading...</div>\n")
	htmlBuilder.WriteString("    <div id=\"output\">")
	// Don't escape opts.Content - it may contain complete HTML with <script> tags that should be preserved as-is
	htmlBuilder.WriteString(opts.Content)
	htmlBuilder.WriteString("</div>\n    \n    <script>\n        ")
	htmlBuilder.WriteString(h.GenerateSharedJavaScript())
	htmlBuilder.WriteString("\n        ")
	htmlBuilder.WriteString(h.GenerateServerSideInitializationJS())
	htmlBuilder.WriteString("\n        ")
	htmlBuilder.WriteString(opts.AdditionalJS)
	htmlBuilder.WriteString("\n    </script>\n</body>\n</html>")
	
	return htmlBuilder.String()
}

// WrapWithTRNMLAssets takes processed content and wraps it with complete TRMNL HTML document
func (h *HTMLAssetsManager) WrapWithTRNMLAssets(content string, title string, width, height int, removeBleedMargin, enableDarkMode bool) string {
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

	return h.GenerateCompleteHTMLDocument(HTMLDocumentOptions{
		Title:             title,
		Width:             width,
		Height:            height,
		Content:           contentWithError,
		AdditionalCSS:     additionalCSS,
		AdditionalJS:      "",
		RemoveBleedMargin: removeBleedMargin,
		EnableDarkMode:    enableDarkMode,
	})
}