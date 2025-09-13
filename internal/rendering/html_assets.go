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
func (h *HTMLAssetsManager) GenerateTRNMLHeadScripts(assetBaseURL string) string {
	return fmt.Sprintf(`<!-- TRMNL Scripts for core functionality, filters, and rendering -->
    <link rel="stylesheet" href="%s/assets/trmnl/fonts/inter.css">
    <link rel="stylesheet" href="%s/assets/trmnl/css/plugins.css">
    <script src="%s/assets/trmnl/js/plugins.js"></script>
    <script src="%s/assets/trmnl/js/plugin.js"></script>
    <script src="%s/assets/trmnl/js/plugin_legacy.js"></script>
    <script src="%s/assets/trmnl/js/plugin_demo.js"></script>
    <script src="%s/assets/trmnl/plugin-render/plugins_legacy.js"></script>
    <script src="%s/assets/trmnl/plugin-render/dithering.js"></script>
    <script src="%s/assets/trmnl/plugin-render/asset.js"></script>`, 
		assetBaseURL, assetBaseURL, assetBaseURL, assetBaseURL, assetBaseURL, assetBaseURL, assetBaseURL, assetBaseURL, assetBaseURL)
}

// GenerateSharedJavaScript returns exact working JavaScript from private plugin backup
func (h *HTMLAssetsManager) GenerateSharedJavaScript() string {
	return `
        // Server-side rendering completion signal
        setTimeout(() => {
            if (document.body && !document.body.hasAttribute('data-render-complete')) {
                console.log('FALLBACK: Setting completion signal after 10-second timeout');
                document.body.setAttribute('data-render-complete', 'true');
            }
        }, 10000);
        
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

// GenerateFlagDetectionJS returns JavaScript for detecting TRMNL skip flags
func (h *HTMLAssetsManager) GenerateFlagDetectionJS() string {
	return `
		// Check for render completion signal every 500ms
		let checkCount = 0;
		const checkInterval = setInterval(function() {
			checkCount++;
			const hasSignal = document.body && document.body.hasAttribute('data-render-complete');
			
			if (hasSignal) {
				// Check for TRMNL skip flags right before screenshot when ALL JavaScript has executed
				if (typeof window.TRMNL_SKIP_DISPLAY !== 'undefined' && window.TRMNL_SKIP_DISPLAY) {
					document.body.setAttribute('data-trmnl-skip-display', 'true');
					console.log('[TRMNL] SKIP_DISPLAY flag detected - marking for skip display');
				}
				if (typeof window.TRMNL_SKIP_SCREEN_GENERATION !== 'undefined' && window.TRMNL_SKIP_SCREEN_GENERATION) {
					document.body.setAttribute('data-trmnl-skip-screen-generation', 'true');
					console.log('[TRMNL] SKIP_SCREEN_GENERATION flag detected - marking for abort');
				}
				
				clearInterval(checkInterval);
			}
			
			if (checkCount >= 40) { // 20 seconds max
				clearInterval(checkInterval);
			}
		}, 500);
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
func (h *HTMLAssetsManager) GenerateCompleteHTMLDocument(opts HTMLDocumentOptions, assetBaseURL string) string {
	// Screen classes are now handled by UnifiedRenderer.generateHTMLStructure()
	
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
	htmlBuilder.WriteString(h.GenerateTRNMLHeadScripts(assetBaseURL))
	htmlBuilder.WriteString("\n    <style>")
	htmlBuilder.WriteString(additionalStyles)
	htmlBuilder.WriteString("</style>\n</head>\n<body>\n")
	htmlBuilder.WriteString("    <div id=\"loading\">Loading...</div>\n")
	htmlBuilder.WriteString("    <div id=\"output\">")
	// Don't escape opts.Content - it may contain complete HTML with <script> tags that should be preserved as-is
	// Screen classes are now handled by UnifiedRenderer.generateHTMLStructure()
	htmlBuilder.WriteString(opts.Content)
	htmlBuilder.WriteString("</div>\n    \n    <script>\n        ")
	htmlBuilder.WriteString(h.GenerateSharedJavaScript())
	htmlBuilder.WriteString("\n        ")
	htmlBuilder.WriteString(h.GenerateServerSideInitializationJS())
	htmlBuilder.WriteString("\n        ")
	htmlBuilder.WriteString(h.GenerateFlagDetectionJS())
	htmlBuilder.WriteString("\n        ")
	htmlBuilder.WriteString(opts.AdditionalJS)
	htmlBuilder.WriteString("\n    </script>\n</body>\n</html>")
	
	return htmlBuilder.String()
}

// WrapWithTRNMLAssets takes processed content and wraps it with complete TRMNL HTML document
func (h *HTMLAssetsManager) WrapWithTRNMLAssets(content string, title string, width, height int, removeBleedMargin, enableDarkMode bool, assetBaseURL string) string {
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
	}, assetBaseURL)
}