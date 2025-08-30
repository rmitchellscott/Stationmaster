package rendering

import (
	"fmt"
)

// BaseHTMLOptions contains configuration for generating HTML documents with embedded JavaScript
type BaseHTMLOptions struct {
	Width              int
	Height             int
	Title              string
	RemoveBleedMargin  bool
	EnableDarkMode     bool
	ScriptLoadStrategy ScriptLoadStrategy
}

// ScriptLoadStrategy defines how TRMNL scripts should be loaded
type ScriptLoadStrategy int

const (
	// ScriptLoadSequential loads scripts one after another (used by server-side rendered content)
	ScriptLoadSequential ScriptLoadStrategy = iota
	// ScriptLoadInHead loads scripts in document head (for immediate script availability)
	ScriptLoadInHead
)

// BaseHTMLRenderer provides shared HTML generation functionality for server-side rendered content
type BaseHTMLRenderer struct{}

// NewBaseHTMLRenderer creates a new base HTML renderer
func NewBaseHTMLRenderer() *BaseHTMLRenderer {
	return &BaseHTMLRenderer{}
}

// GenerateHTML creates a complete HTML document for server-side rendered content (no liquidjs)
func (r *BaseHTMLRenderer) GenerateHTML(opts BaseHTMLOptions, content string, dataJSON []byte, additionalJS string) string {
	// Build screen classes based on options
	screenClasses := []string{"screen"}
	if opts.RemoveBleedMargin {
		screenClasses = append(screenClasses, "screen--no-bleed")
	}
	if opts.EnableDarkMode {
		screenClasses = append(screenClasses, "screen--dark-mode")
	}
	
	// Generate TRMNL scripts section based on strategy
	var scriptsSection string
	if opts.ScriptLoadStrategy == ScriptLoadInHead {
		scriptsSection = r.generateHeadScripts()
	} else {
		scriptsSection = "" // Sequential loading handled in JavaScript
	}
	
	// Create the complete HTML document for server-side rendering
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
        }
    </style>
</head>
<body>
    <div id="loading">Loading...</div>
    <div id="error"></div>
    <div id="output" style="display: none;">%s</div>
    
    %s
    
    <script>
        // Global data and configuration
        const renderData = %s;
        const removeBleedMargin = %t;
        const enableDarkMode = %t;
        const scriptLoadStrategy = %d;
        
        %s
        
        %s
    </script>
    
    <script>
        // Global error handler for uncaught errors
        window.addEventListener('error', function(e) {
            console.error('UNCAUGHT ERROR during rendering:', e.error, e);
            console.error('Error occurred at:', e.filename, 'line:', e.lineno);
            if (document.body && !document.body.hasAttribute('data-render-complete')) {
                console.log('Setting completion due to uncaught error');
                document.body.setAttribute('data-render-complete', 'true');
            }
        });
    </script>
</body>
</html>`,
		opts.Title,
		scriptsSection,
		opts.Width,
		opts.Height,
		content,
		string(dataJSON),
		opts.RemoveBleedMargin,
		opts.EnableDarkMode,
		int(opts.ScriptLoadStrategy),
		r.generateSharedJavaScriptFunctions(),
		additionalJS,
	)
}

// generateHeadScripts returns TRMNL scripts to be loaded in the document head (no liquidjs)
func (r *BaseHTMLRenderer) generateHeadScripts() string {
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

// generateSharedJavaScriptFunctions returns common JavaScript functions for server-side rendered content
func (r *BaseHTMLRenderer) generateSharedJavaScriptFunctions() string {
	return `
        // CRITICAL: Immediate fallback timer - starts when page loads regardless of other code
        setTimeout(() => {
            if (document.body && !document.body.hasAttribute('data-render-complete')) {
                console.log('FALLBACK: Setting completion signal after 3-second timeout');
                document.body.setAttribute('data-render-complete', 'true');
            }
        }, 3000);
        
        // Additional logging to debug JavaScript execution
        console.log('Server-side renderer JavaScript loaded successfully');
        console.log('renderData available:', typeof renderData !== 'undefined');
        
        // Essential functions for server-side rendered content
        function signalRenderingComplete() {
            console.log('DEBUG: signalRenderingComplete() called');
            console.log('DEBUG: document.body exists:', !!document.body);
            console.log('DEBUG: document.readyState:', document.readyState);
            
            try {
                // Show the output and hide loading
                const outputEl = document.getElementById('output');
                const loadingEl = document.getElementById('loading');
                
                console.log('DEBUG: outputEl found:', !!outputEl);
                console.log('DEBUG: loadingEl found:', !!loadingEl);
                
                if (outputEl) {
                    outputEl.style.display = 'block';
                    console.log('DEBUG: outputEl display set to block');
                }
                if (loadingEl) {
                    loadingEl.style.display = 'none';
                    console.log('DEBUG: loadingEl display set to none');
                }
                
                // Signal completion to external systems (like browserless)
                if (document.body) {
                    console.log('DEBUG: Setting data-render-complete attribute on body');
                    document.body.setAttribute('data-render-complete', 'true');
                    console.log('DEBUG: Attribute set. Current value:', document.body.getAttribute('data-render-complete'));
                    
                    // Verify the attribute was actually set
                    setTimeout(() => {
                        const currentValue = document.body.getAttribute('data-render-complete');
                        console.log('DEBUG: Post-set verification - attribute value:', currentValue);
                        console.log('DEBUG: Body element:', document.body.outerHTML.substring(0, 200));
                    }, 100);
                } else {
                    console.error('DEBUG: document.body is null - cannot set completion attribute');
                }
                console.log('DEBUG: signalRenderingComplete() completed successfully');
            } catch (error) {
                console.error('DEBUG: Error in signalRenderingComplete():', error);
                console.error('DEBUG: Error stack:', error.stack);
                // Immediately set completion on error
                if (document.body) {
                    document.body.setAttribute('data-render-complete', 'true');
                    console.log('DEBUG: Fallback completion attribute set due to error');
                }
            }
        }
        
        // Load TRMNL scripts sequentially and handle completion
        function loadTRNMLScriptsSequentially(onComplete) {
            console.log('Loading TRMNL scripts sequentially...');
            
            // For server-side rendering, content is already rendered
            // We just need to load TRMNL scripts for styling/dithering
            const scripts = [
                'https://usetrmnl.com/js/latest/plugins.js',
                'https://usetrmnl.com/assets/plugin-render/plugins-332ca4207dd02576b3641691907cb829ef52a36c4a092a75324a8fc860906967.js',
                'https://usetrmnl.com/assets/plugin-render/dithering-d697f6229e3bd6e2455425d647e5395bb608999c2039a9837a903c7c7e952d61.js'
            ];
            
            let loadedCount = 0;
            
            function loadNext() {
                if (loadedCount >= scripts.length) {
                    console.log('All TRMNL scripts loaded');
                    
                    // Handle dithering timing after all scripts are loaded
                    if (typeof handleDitheringTiming === 'function') {
                        try {
                            handleDitheringTiming();
                        } catch (error) {
                            console.error('Error in handleDitheringTiming:', error);
                        }
                    }
                    
                    if (onComplete) onComplete();
                    return;
                }
                
                const script = document.createElement('script');
                script.src = scripts[loadedCount];
                script.onload = () => {
                    loadedCount++;
                    loadNext();
                };
                script.onerror = () => {
                    console.error('Failed to load script:', scripts[loadedCount]);
                    loadedCount++;
                    loadNext();
                };
                
                document.head.appendChild(script);
            }
            
            loadNext();
        }
        
        // Execute scripts within innerHTML (for dynamic content)
        function executeInnerHTMLScripts(container) {
            const scripts = container.querySelectorAll('script');
            for (let script of scripts) {
                try {
                    const newScript = document.createElement('script');
                    newScript.textContent = script.textContent;
                    if (script.src) newScript.src = script.src;
                    document.head.appendChild(newScript);
                    document.head.removeChild(newScript);
                } catch (error) {
                    console.error('Error executing script:', error);
                }
            }
        }
	`
}