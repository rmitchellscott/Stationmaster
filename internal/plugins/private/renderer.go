package private

import (
	"encoding/json"
	"fmt"

	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// RenderOptions contains all options needed to render a private plugin to HTML
type RenderOptions struct {
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
}

// PrivatePluginRenderer handles HTML generation for private plugins
type PrivatePluginRenderer struct {
	baseRenderer *rendering.BaseHTMLRenderer
}

// NewPrivatePluginRenderer creates a new private plugin renderer
func NewPrivatePluginRenderer() *PrivatePluginRenderer {
	return &PrivatePluginRenderer{
		baseRenderer: rendering.NewBaseHTMLRenderer(),
	}
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
	
	// Use template data as-is - TRMNL structure is already complete
	templateData := make(map[string]interface{})
	for k, v := range opts.Data {
		templateData[k] = v
	}
	
	// Add template and instance info to data
	privatePluginData := map[string]interface{}{
		"template":     combinedTemplate,
		"data":         templateData,
		"instanceId":   opts.InstanceID,
		"instanceName": opts.InstanceName,
	}
	
	dataJSON, err := json.Marshal(privatePluginData)
	if err != nil {
		return "", fmt.Errorf("failed to encode template data as JSON: %w", err)
	}
	
	// Generate base HTML options
	baseOpts := rendering.BaseHTMLOptions{
		Width:             opts.Width,
		Height:            opts.Height,
		Title:             opts.PluginName,
		RemoveBleedMargin: opts.RemoveBleedMargin,
		EnableDarkMode:    opts.EnableDarkMode,
		ScriptLoadStrategy: rendering.ScriptLoadSequential,
	}
	
	// Private plugin specific JavaScript
	privatePluginJS := `
		// Define the function BEFORE the LiquidJS script loads
        function initializeLiquid() {
            
            // Use liquidjs constructor (we know this exists)
            const engine = new liquidjs.Liquid();
            
            // Register TRMNL custom filters and tags for compatibility
            registerTRNMLExtensions(engine);
            
            // Preprocess template for TRMNL syntax compatibility
            const processedTemplate = preprocessTRNMLTemplate(renderData.template);
        
            // Render template
            engine.parseAndRender(processedTemplate, renderData.data)
                .then(renderedContent => {
                    
                    // Process the rendered content similar to server-side processing
                    let processedTemplate = renderedContent;
                    
                    // Handle view_type variables (fallback)
                    processedTemplate = processedTemplate.replace(/\{\{\s*view_type\s*\}\}/g, 'view--full');
                    
                    // Enhance view classes (same logic as server-side)
                    processedTemplate = enhanceViewClasses(processedTemplate);
                    
                    // Check if template has view classes (after enhancement)
                    const hasViewClass = processedTemplate.includes('class="view') || 
                                       processedTemplate.includes("class='view");
                    
                    
                    // Build screen classes based on options
                    let screenClasses = ['screen'];
                    if (removeBleedMargin) {
                        screenClasses.push('screen--no-bleed');
                    }
                    if (enableDarkMode) {
                        screenClasses.push('screen--dark-mode');
                    }
                    const screenClassString = screenClasses.join(' ');
                    
                    // Wrap user template in TRMNL framework structure
                    let wrappedContent;
                    if (hasViewClass) {
                        wrappedContent = '<div id="plugin-' + renderData.instanceId + '" class="environment trmnl">' +
                            '<div class="' + screenClassString + '">' + processedTemplate + '</div>' +
                            '</div>';
                    } else {
                        wrappedContent = '<div id="plugin-' + renderData.instanceId + '" class="environment trmnl">' +
                            '<div class="' + screenClassString + '">' +
                            '<div class="view view--full">' + processedTemplate + '</div>' +
                            '</div>' +
                            '</div>';
                    }
                    
                    // Hide loading, show output - wait for DOM to be ready
                    function waitForDOMAndShow() {
                        
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
                            
                            // Load TRMNL scripts sequentially and execute innerHTML scripts
                            loadTRNMLScriptsSequentially(() => {
                                executeInnerHTMLScripts(outputEl);
                            });
                        }
                    }
                    
                    waitForDOMAndShow();
                    
                    // Use shared completion signal logic
                    signalRenderingComplete();
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
	`
	
	// Generate HTML using the base renderer
	html := r.baseRenderer.GenerateHTML(baseOpts, "", dataJSON, privatePluginJS)
	
	return html, nil
}


