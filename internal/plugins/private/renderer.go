package private

import (
	"encoding/json"
	"fmt"
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
}

// NewPrivatePluginRenderer creates a new private plugin renderer
func NewPrivatePluginRenderer() *PrivatePluginRenderer {
	return &PrivatePluginRenderer{}
}

// RenderToClientSideHTML generates HTML with embedded LiquidJS for client-side rendering
func (r *PrivatePluginRenderer) RenderToClientSideHTML(opts RenderOptions) (string, error) {
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
	
	// Add timestamp to template data for client-side access
	templateData := make(map[string]interface{})
	for k, v := range opts.Data {
		templateData[k] = v
	}
	templateData["timestamp"] = time.Now().Format("2006-01-02T15:04:05Z07:00")
	templateData["plugin_name"] = opts.PluginName
	templateData["instance_id"] = opts.InstanceID
	
	// JSON encode template and data for JavaScript
	templateJSON, err := json.Marshal(combinedTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to encode template as JSON: %w", err)
	}
	
	dataJSON, err := json.Marshal(templateData)
	if err != nil {
		return "", fmt.Errorf("failed to encode template data as JSON: %w", err)
	}
	
	// Create complete HTML document with LiquidJS
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@100..900&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://usetrmnl.com/css/latest/plugins.css">
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
    </style>
</head>
<body>
    <div id="loading">Loading template...</div>
    <div id="error"></div>
    <div id="output" style="display: none;"></div>
    
    <script>
        // Template and data definitions (global scope)
        const template = %s;
        const data = %s;
        const instanceId = "%s";
        
        // Set a timeout fallback in case anything fails
        setTimeout(() => {
            if (document.body && !document.body.hasAttribute('data-render-complete')) {
                console.log('Fallback: Setting completion signal after timeout');
                document.body.setAttribute('data-render-complete', 'true');
            }
        }, 3000);
        
        // Define the function BEFORE the LiquidJS script loads
        function initializeLiquid() {
            console.log('LiquidJS loaded! Starting template rendering...');
            console.log('liquidjs type:', typeof liquidjs);
            console.log('Template length:', template.length);
            console.log('Data keys:', Object.keys(data));
            
            // Use liquidjs constructor (we know this exists)
            const engine = new liquidjs.Liquid();
        
            // Render template
            console.log('Starting template render with liquidjs...');
            engine.parseAndRender(template, data)
                .then(renderedContent => {
                    console.log('Template rendered successfully! Content length:', renderedContent.length);
                    console.log('Rendered content preview:', renderedContent.substring(0, 200) + '...');
                    
                    // Process the rendered content similar to server-side processing
                    let processedTemplate = renderedContent;
                    
                    // Handle view_type variables (fallback)
                    processedTemplate = processedTemplate.replace(/\{\{\s*view_type\s*\}\}/g, 'view--full');
                    
                    // Enhance view classes (same logic as server-side)
                    function enhanceViewClasses(template) {
                        // Process double quotes
                        template = template.replace(/class="([^"]*\bview\b[^"]*)"/g, function(match, classContent) {
                            // Check if already has layout modifiers
                            if (classContent.includes('view--full') || 
                                classContent.includes('view--half') || 
                                classContent.includes('view--quadrant')) {
                                return match;
                            }
                            
                            // Replace standalone 'view' with 'view view--full'
                            const enhancedClasses = classContent.replace(/\bview\b/g, 'view view--full');
                            return 'class="' + enhancedClasses + '"';
                        });
                        
                        // Process single quotes
                        template = template.replace(/class='([^']*\bview\b[^']*)'/g, function(match, classContent) {
                            // Check if already has layout modifiers
                            if (classContent.includes('view--full') || 
                                classContent.includes('view--half') || 
                                classContent.includes('view--quadrant')) {
                                return match;
                            }
                            
                            // Replace standalone 'view' with 'view view--full'
                            const enhancedClasses = classContent.replace(/\bview\b/g, 'view view--full');
                            return "class='" + enhancedClasses + "'";
                        });
                        
                        return template;
                    }
                    
                    processedTemplate = enhanceViewClasses(processedTemplate);
                    
                    // Check if template has view classes (after enhancement)
                    const hasViewClass = processedTemplate.includes('class="view') || 
                                       processedTemplate.includes("class='view");
                    
                    console.log('Has view class:', hasViewClass);
                    
                    // Wrap user template in TRMNL framework structure
                    let wrappedContent;
                    if (hasViewClass) {
                        wrappedContent = '<div id="plugin-' + instanceId + '" class="environment trmnl">' +
                            '<div class="screen">' + processedTemplate + '</div>' +
                            '</div>';
                    } else {
                        wrappedContent = '<div id="plugin-' + instanceId + '" class="environment trmnl">' +
                            '<div class="screen">' +
                            '<div class="view view--full">' + processedTemplate + '</div>' +
                            '</div>' +
                            '</div>';
                    }
                    
                    // Hide loading, show output - wait for DOM to be ready
                    function waitForDOMAndShow() {
                        console.log('Debug: Document ready state:', document.readyState);
                        
                        if (document.readyState === 'loading') {
                            // DOM still loading, wait a bit more
                            setTimeout(waitForDOMAndShow, 10);
                            return;
                        }
                        
                        const loadingEl = document.getElementById('loading');
                        const outputEl = document.getElementById('output');
                        
                        if (loadingEl) {
                            loadingEl.style.display = 'none';
                        }
                        
                        if (outputEl) {
                            outputEl.style.display = 'block';
                            outputEl.innerHTML = wrappedContent;
                            
                            // Load TRMNL plugins.js AFTER content is shown
                            const script = document.createElement('script');
                            script.src = 'https://usetrmnl.com/js/latest/plugins.js';
                            script.onload = () => console.log('TRMNL plugins.js loaded successfully');
                            script.onerror = (e) => console.log('TRMNL plugins.js failed to load:', e);
                            document.head.appendChild(script);
                        }
                    }
                    
                    waitForDOMAndShow();
                    
                    // Wait for fonts to load before setting completion signal
                    function waitForFontsAndComplete() {
                        // Check if fonts are loaded using document.fonts API
                        if (document.fonts && document.fonts.status === 'loaded') {
                            console.log('Fonts loaded, setting completion signal');
                            if (document.body) {
                                document.body.setAttribute('data-render-complete', 'true');
                            }
                        } else {
                            // Fallback: wait a bit more for fonts
                            setTimeout(waitForFontsAndComplete, 100);
                        }
                    }
                    
                    console.log('Template rendered successfully, waiting for fonts...');
                    // Start font loading check, but also set a maximum wait time
                    waitForFontsAndComplete();
                    
                    // Fallback: set completion signal after 2 seconds even if fonts aren't loaded
                    setTimeout(() => {
                        if (document.body && !document.body.hasAttribute('data-render-complete')) {
                            console.log('Font loading timeout, setting completion signal anyway');
                            document.body.setAttribute('data-render-complete', 'true');
                        }
                    }, 2000);
                })
                .catch(err => {
                    console.error('Liquid rendering error:', err);
                    console.error('Error details:', {
                        message: err.message,
                        stack: err.stack,
                        name: err.name
                    });
                    
                    const loadingEl = document.getElementById('loading');
                    const errorEl = document.getElementById('error');
                    
                    if (loadingEl) loadingEl.style.display = 'none';
                    if (errorEl) {
                        errorEl.style.display = 'block';
                        errorEl.textContent = 'Template Error: ' + err.message + '\n\nStack: ' + (err.stack || 'No stack trace available');
                    }
                    
                    // Set completion signal even in error case so browserless doesn't hang
                    if (document.body) {
                        document.body.setAttribute('data-render-complete', 'true');
                    }
                });
        }
    </script>
    <script src="https://cdn.jsdelivr.net/npm/liquidjs/dist/liquid.browser.min.js" onload="initializeLiquid()"></script>
    
    <script>
        // Fallback: If LiquidJS CDN fails to load
        setTimeout(() => {
            if (typeof liquidjs === 'undefined') {
                console.error('LiquidJS failed to load from CDN');
                const loadingEl = document.getElementById('loading');
                const errorEl = document.getElementById('error');
                
                if (loadingEl) loadingEl.style.display = 'none';
                if (errorEl) {
                    errorEl.style.display = 'block';
                    errorEl.textContent = 'LiquidJS library failed to load from CDN';
                }
                if (document.body) {
                    document.body.setAttribute('data-render-complete', 'true');
                }
            }
        }, 5000);
    </script>
</body>
</html>`,
		opts.PluginName,
		opts.Width, 
		opts.Height,
		string(templateJSON),
		string(dataJSON),
		opts.InstanceID)
	
	return html, nil
}


