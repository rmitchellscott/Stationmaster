package mashup

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// MashupRenderer handles combining child plugin data into mashup layouts
type MashupRenderer struct {
	layout       string
	childData    map[string]ChildData
	slotConfig   []database.MashupSlotInfo
	baseRenderer *rendering.BaseHTMLRenderer
}

// NewMashupRenderer creates a new mashup renderer
func NewMashupRenderer(layout string, childData map[string]ChildData) *MashupRenderer {
	// Generate slot configuration for this layout
	service := database.NewMashupService(database.GetDB())
	slots, _ := service.GetSlotMetadata(layout)
	
	return &MashupRenderer{
		layout:       layout,
		childData:    childData,
		slotConfig:   slots,
		baseRenderer: rendering.NewBaseHTMLRenderer(),
	}
}

// RenderMashup creates a single HTML document with embedded child data and templates
func (r *MashupRenderer) RenderMashup(ctx plugins.PluginContext) (string, error) {
	logging.Info("[MASHUP_RENDERER] Creating mashup HTML with embedded child data", "layout", r.layout, "children_count", len(r.childData))
	
	// Build JavaScript objects for client-side processing
	childDataJS := make(map[string]interface{})
	childTemplatesJS := make(map[string]string)
	
	for slot, childInfo := range r.childData {
		if !childInfo.Success {
			// Handle error cases
			childDataJS[slot] = map[string]interface{}{"error": childInfo.Error}
			childTemplatesJS[slot] = fmt.Sprintf(`<div class="mashup-error">%s</div>`, childInfo.Error)
			continue
		}
		
		// Add successful child data and template
		childDataJS[slot] = childInfo.Data
		childTemplatesJS[slot] = childInfo.Template
	}
	
	// Generate the complete mashup HTML with embedded JavaScript
	return r.generateMashupHTML(childDataJS, childTemplatesJS, ctx), nil
}

// generateMashupHTML creates the complete HTML document with embedded child data and templates
func (r *MashupRenderer) generateMashupHTML(childData map[string]interface{}, childTemplates map[string]string, ctx plugins.PluginContext) string {
	// Prepare mashup data for JavaScript
	mashupData := map[string]interface{}{
		"childData":      childData,
		"childTemplates": childTemplates,
		"layout":         r.layout,
		"slotConfig":     r.slotConfig,
	}
	
	mashupDataJSON, _ := json.Marshal(mashupData)
	
	// Generate base HTML options
	baseOpts := rendering.BaseHTMLOptions{
		Width:              ctx.Device.DeviceModel.ScreenWidth,
		Height:             ctx.Device.DeviceModel.ScreenHeight,
		Title:              "Mashup",
		RemoveBleedMargin:  false,
		EnableDarkMode:     false,
		ScriptLoadStrategy: rendering.ScriptLoadSequential,
	}
	
	// Mashup-specific JavaScript
	mashupJS := `
        console.log('Mashup JavaScript loaded');
        
        // Define the function BEFORE the LiquidJS script loads (same pattern as private plugins)
        async function initializeLiquid() {
            console.log('initializeLiquid called - starting mashup rendering');
            // Remove the immediate return to actually process templates
            
            // Use liquidjs constructor (we know this exists)
            const engine = new liquidjs.Liquid();

            // Register TRMNL custom filters
            registerTRNMLFilters(engine);
            console.log('Starting mashup template processing...');
            
            // First, create the mashup container structure in the output div
            const outputEl = document.getElementById('output');
            if (!outputEl) {
                console.error('Output element not found');
                signalRenderingComplete();
                return;
            }
            
            // Create mashup structure with slots
            let slotHTML = '';
            const slotConfig = renderData.slotConfig || [];
            for (const slot of slotConfig) {
                slotHTML += '<div id="slot-' + slot.position + '" class="view ' + slot.view_class + '">' +
                           '<div class="mashup-loading">Loading ' + slot.display_name + '...</div>' +
                           '</div>';
            }
            
            outputEl.innerHTML = '<div class="environment trmnl">' +
                                '<div class="screen">' +
                                '<div class="mashup mashup--' + renderData.layout + '">' +
                                slotHTML +
                                '</div>' +
                                '</div>' +
                                '</div>';
            
            // Show the output div
            outputEl.style.display = 'block';
            document.getElementById('loading').style.display = 'none';
            
            try {
                console.log('renderData:', renderData);
                const childData = renderData.childData;
                const childTemplates = renderData.childTemplates;
                console.log('childData:', childData);
                console.log('childTemplates:', childTemplates);
                
                for (const [slot, template] of Object.entries(childTemplates)) {
                    console.log('Processing slot:', slot);
                    const slotElement = document.getElementById('slot-' + slot);
                    
                    if (slotElement && childData[slot]) {
                        try {
                            if (childData[slot].error) {
                                // Handle error case
                                console.log('Error in slot', slot, ':', childData[slot].error);
                                slotElement.innerHTML = template;
                            } else {
                                // Preprocess template for TRMNL syntax compatibility
                                const processedTemplate = preprocessTRNMLTemplate(template);
                                console.log('Processing template for', slot, ':', processedTemplate);
                                console.log('Data for', slot, ':', childData[slot]);
                                
                                // Render template with TRMNL filters
                                const html = await engine.parseAndRender(processedTemplate, childData[slot]);
                                console.log('Rendered HTML for', slot, ':', html);
                                
                                // Enhance view classes like the server does
                                const enhancedHTML = enhanceViewClasses(html);
                                
                                slotElement.innerHTML = enhancedHTML;
                                
                                // Execute any scripts in the rendered content
                                executeInnerHTMLScripts(slotElement);
                            }
                        } catch (error) {
                            console.error('Failed to render slot ' + slot + ':', error);
                            slotElement.innerHTML = '<div class="mashup-error">Template Error: ' + error.message + '</div>';
                        }
                    } else {
                        console.error('Missing slot element or data for:', slot);
                    }
                }
                
                console.log('Mashup template processing complete');
                
                // Load TRMNL scripts (including dithering) then trigger dithering
                loadTRNMLScriptsSequentially(() => {
                    console.log('TRMNL scripts loaded, triggering dithering');
                    
                    // Now handleDitheringTiming will work since scripts are loaded
                    // handleDitheringTiming is called automatically by loadTRNMLScriptsSequentially
                    
                    // Signal completion after dithering
                    setTimeout(() => {
                        console.log('Mashup processing complete, signaling completion');
                        signalRenderingComplete();
                    }, 200);
                });
                
            } catch (error) {
                console.error('Error during mashup processing:', error);
                // Always set completion signal even if there are errors
                document.body.setAttribute('data-render-complete', 'true');
            }
        }
	`
	
	// Generate HTML using the base renderer (pass empty content like private plugins)
	return r.baseRenderer.GenerateHTML(baseOpts, "", mashupDataJSON, mashupJS)
}

// buildSlotDivs creates the slot div structure based on layout configuration
func (r *MashupRenderer) buildSlotDivs() string {
	var slotDivs []string
	
	for _, slot := range r.slotConfig {
		// Create slot div with proper TRMNL classes
		slotDiv := fmt.Sprintf(`<div id="slot-%s" class="view %s">
			<!-- Child content will be rendered here by JavaScript -->
			<div class="mashup-loading">Loading %s...</div>
		</div>`, 
			slot.Position, 
			slot.ViewClass,
			slot.DisplayName)
			
		slotDivs = append(slotDivs, slotDiv)
	}
	
	return strings.Join(slotDivs, "\n")
}

