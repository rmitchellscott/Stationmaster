package mashup

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// MashupRenderer handles combining child plugin data into mashup layouts
type MashupRenderer struct {
	layout     string
	childData  map[string]ChildData
	slotConfig []database.MashupSlotInfo
}

// NewMashupRenderer creates a new mashup renderer
func NewMashupRenderer(layout string, childData map[string]ChildData) *MashupRenderer {
	// Generate slot configuration for this layout
	service := database.NewMashupService(database.GetDB())
	slots, _ := service.GetSlotMetadata(layout)
	
	return &MashupRenderer{
		layout:     layout,
		childData:  childData,
		slotConfig: slots,
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
	// Marshal child data and templates to JSON for JavaScript embedding
	childDataJSON, _ := json.Marshal(childData)
	childTemplatesJSON, _ := json.Marshal(childTemplates)
	
	// Build slot divs based on layout
	slotDivs := r.buildSlotDivs()
	
	// Create the complete HTML document with embedded JavaScript
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Mashup</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@100..900&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://usetrmnl.com/css/latest/plugins.css">
    <script src="https://cdn.jsdelivr.net/npm/liquidjs@10.10.1/dist/liquid.browser.umd.js"></script>
    <!-- TRMNL Scripts for dithering and other functionality -->
    <script src="https://usetrmnl.com/assets/plugin-render/plugins-332ca4207dd02576b3641691907cb829ef52a36c4a092a75324a8fc860906967.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/plugins_legacy-a6b0b3aeac32ca71413f1febc053c59a528d4c6bb2173c22bd94ff8e0b9650f1.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/dithering-d697f6229e3bd6e2455425d647e5395bb608999c2039a9837a903c7c7e952d61.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/asset-deduplication-39fa2231b7a5bd5bedf4a1782b6a95d8b87eb3aaaa5e2b6cee287133d858bc96.js"></script>
    <style>
        body { 
            width: %dpx; 
            height: %dpx; 
            margin: 0; 
            padding: 0;
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
    <div class="environment trmnl">
        <div class="screen">
            <div class="mashup mashup--%s">
                %s
            </div>
        </div>
    </div>

    <script>
        // Child data and templates for client-side processing
        const childData = %s;
        const childTemplates = %s;
        const mashupLayout = "%s";

        // TRMNL Compatibility Layer Functions
        function preprocessTRNMLTemplate(template) {
            // Only convert TRMNL's alternative syntax at the START of Liquid expressions
            let processed = template;
            
            // Pattern 1: {{ var: filter, param }} → {{ var | filter: param }}
            processed = processed.replace(/\{\{\s*([^|}]+?):\s*([^,\s]+)\s*,\s*([^}]+?)\s*\}\}/g, '{{ $1 | $2: $3 }}');
            
            // Pattern 2: {{ var: filter }} → {{ var | filter }}  
            processed = processed.replace(/\{\{\s*([^|}]+?):\s*([^}\s]+)\s*\}\}/g, '{{ $1 | $2 }}');
            
            return processed;
        }
        
        function convertStrftimeToIntlOptions(format) {
            const options = {};
            
            // Common strftime patterns
            if (format.includes('%%Y')) options.year = 'numeric';
            if (format.includes('%%y')) options.year = '2-digit';
            if (format.includes('%%m')) options.month = '2-digit';
            if (format.includes('%%B')) options.month = 'long';
            if (format.includes('%%b')) options.month = 'short';
            if (format.includes('%%d')) options.day = '2-digit';
            if (format.includes('%%H')) options.hour = '2-digit';
            if (format.includes('%%M')) options.minute = '2-digit';
            if (format.includes('%%S')) options.second = '2-digit';
            
            return options;
        }

        function registerTRNMLFilters(engine) {
            // l_date: Localized date formatting
            engine.registerFilter('l_date', function(dateValue, format, locale) {
                if (!dateValue) return '';
                
                if (!locale && this.context) {
                    locale = this.context.get(['trmnl', 'user', 'locale']) || 'en';
                }
                
                try {
                    const date = new Date(dateValue);
                    if (isNaN(date.getTime())) return dateValue;
                    
                    const options = convertStrftimeToIntlOptions(format || '%%Y-%%m-%%d');
                    return new Intl.DateTimeFormat(locale, options).format(date);
                } catch (e) {
                    console.warn('l_date filter error:', e);
                    return dateValue;
                }
            });
            
            // json: Convert to JSON
            engine.registerFilter('json', function(value) {
                try {
                    return JSON.stringify(value, null, 2);
                } catch (e) {
                    return String(value);
                }
            });
            
            // parse_json: Parse JSON strings
            engine.registerFilter('parse_json', function(jsonString) {
                try {
                    return JSON.parse(jsonString);
                } catch (e) {
                    console.warn('parse_json filter error:', e);
                    return jsonString;
                }
            });
        }

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
                if (classContent.includes('view--full') || 
                    classContent.includes('view--half') || 
                    classContent.includes('view--quadrant')) {
                    return match;
                }
                
                const enhancedClasses = classContent.replace(/\bview\b/g, 'view view--full');
                return "class='" + enhancedClasses + "'";
            });
            
            return template;
        }

        function handleDitheringTiming() {
            // Check if window.load already fired and handle dithering timing
            if (document.readyState === 'complete') {
                // Page already loaded, manually trigger dithering
                if (typeof window.setup === 'function') {
                    console.log('Triggering dithering via window.setup()');
                    window.setup();
                } else {
                    // Fallback: dispatch load event to trigger dithering
                    console.log('Triggering dithering via window load event');
                    window.dispatchEvent(new Event('load'));
                }
            } else {
                // Wait for page to fully load
                window.addEventListener('load', function() {
                    console.log('Page loaded, triggering dithering');
                    if (typeof window.setup === 'function') {
                        window.setup();
                    }
                });
            }
        }

        // Initialize LiquidJS engine
        const { Liquid } = window.liquidjs;
        const engine = new Liquid();

        // Register TRMNL custom filters
        registerTRNMLFilters(engine);

        // Set a timeout fallback in case anything fails
        setTimeout(() => {
            if (document.body && !document.body.hasAttribute('data-render-complete')) {
                console.log('Fallback: Setting completion signal after timeout');
                document.body.setAttribute('data-render-complete', 'true');
            }
        }, 3000);

        // Process each child template with its data
        document.addEventListener('DOMContentLoaded', async function() {
            console.log('Starting mashup template processing...');
            
            try {
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
                
                // Trigger dithering after all templates are processed
                setTimeout(() => {
                    handleDitheringTiming();
                    
                    // Set completion signal after dithering
                    setTimeout(() => {
                        console.log('Setting render completion signal');
                        document.body.setAttribute('data-render-complete', 'true');
                    }, 200);
                }, 100);
                
            } catch (error) {
                console.error('Error during mashup processing:', error);
                // Always set completion signal even if there are errors
                document.body.setAttribute('data-render-complete', 'true');
            }
        });
    </script>
</body>
</html>`,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
		r.layout,
		slotDivs,
		string(childDataJSON),
		string(childTemplatesJSON),
		r.layout)
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

