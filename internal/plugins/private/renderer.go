package private

import (
	"encoding/json"
	"fmt"
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
	
	// Use template data as-is - TRMNL structure is already complete
	templateData := make(map[string]interface{})
	for k, v := range opts.Data {
		templateData[k] = v
	}
	
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
        /* Dark mode inversion CSS */
        .screen--dark-mode {
            filter: invert(1);
        }
        .screen--dark-mode .image {
            filter: invert(1);
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
        const removeBleedMargin = %t;
        const enableDarkMode = %t;
        
        // Set a timeout fallback in case anything fails
        setTimeout(() => {
            if (document.body && !document.body.hasAttribute('data-render-complete')) {
                console.log('Fallback: Setting completion signal after timeout');
                document.body.setAttribute('data-render-complete', 'true');
            }
        }, 3000);
        
        // TRMNL Compatibility Layer Functions
        function preprocessTRNMLTemplate(template) {
            // Only convert TRMNL's alternative syntax at the START of Liquid expressions
            // Look for {{ variable: filter }} patterns where there are NO pipes before the colon
            let processed = template;
            
            // Pattern 1: {{ var: filter, param }} → {{ var | filter: param }}
            // Only match if there's no | before the first :
            processed = processed.replace(/\{\{\s*([^|}]+?):\s*([^,\s]+)\s*,\s*([^}]+?)\s*\}\}/g, '{{ $1 | $2: $3 }}');
            
            // Pattern 2: {{ var: filter }} → {{ var | filter }}  
            // Only match if there's no | before the first :
            processed = processed.replace(/\{\{\s*([^|}]+?):\s*([^}\s]+)\s*\}\}/g, '{{ $1 | $2 }}');
            
            return processed;
        }
        
        function registerTRNMLFilters(engine) {
            
            // l_date: Localized date formatting with strftime syntax
            engine.registerFilter('l_date', function(dateValue, format, locale) {
                if (!dateValue) return '';
                
                // Use locale from context if not provided
                if (!locale && this.context) {
                    locale = this.context.get(['trmnl', 'user', 'locale']) || 'en';
                }
                
                try {
                    const date = new Date(dateValue);
                    if (isNaN(date.getTime())) return dateValue;
                    
                    // Convert strftime format to Intl.DateTimeFormat options
                    const options = convertStrftimeToIntlOptions(format || '%Y-%m-%d');
                    return new Intl.DateTimeFormat(locale, options).format(date);
                } catch (e) {
                    console.warn('l_date filter error:', e);
                    return dateValue;
                }
            });
            
            // l_word: Localize common words
            engine.registerFilter('l_word', async function(word, locale) {
                if (!locale && this.context) {
                    locale = this.context.get(['trmnl', 'user', 'locale']) || 'en';
                }
                
                try {
                    const localeData = await loadLocaleData(locale);
                    
                    // Look for the word in the locale data
                    if (localeData[word]) {
                        return localeData[word];
                    }
                    
                    // Fallback to hardcoded translations for common words
                    const fallbackTranslations = getTRNMLWordTranslations();
                    const localeKey = locale.toLowerCase().split('-')[0];
                    
                    if (fallbackTranslations[word] && fallbackTranslations[word][localeKey]) {
                        return fallbackTranslations[word][localeKey];
                    }
                } catch (e) {
                    console.warn('l_word filter error:', e);
                }
                
                return word; // Return original if no translation found
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
            
            // group_by: Group array by key
            engine.registerFilter('group_by', function(array, key) {
                if (!Array.isArray(array)) return array;
                
                const grouped = {};
                array.forEach(item => {
                    const groupKey = item[key] || 'undefined';
                    if (!grouped[groupKey]) grouped[groupKey] = [];
                    grouped[groupKey].push(item);
                });
                
                return Object.keys(grouped).map(k => ({
                    name: k,
                    items: grouped[k],
                    size: grouped[k].length
                }));
            });
            
            // find_by: Find array item by key/value
            engine.registerFilter('find_by', function(array, key, value) {
                if (!Array.isArray(array)) return null;
                return array.find(item => item[key] === value) || null;
            });
            
            // sample: Random array selection
            engine.registerFilter('sample', function(array) {
                if (!Array.isArray(array) || array.length === 0) return null;
                return array[Math.floor(Math.random() * array.length)];
            });
            
            // number_with_delimiter: Format numbers with delimiters
            engine.registerFilter('number_with_delimiter', function(number, delimiter) {
                if (isNaN(number)) return number;
                const delim = delimiter || ',';
                return Number(number).toLocaleString().replace(/,/g, delim);
            });
            
            // number_to_currency: Convert to localized currency
            engine.registerFilter('number_to_currency', function(number, currency, locale) {
                if (isNaN(number)) return number;
                
                if (!locale && this.context) {
                    locale = this.context.get(['trmnl', 'user', 'locale']) || 'en-US';
                }
                
                const options = {
                    style: 'currency',
                    currency: currency || 'USD'
                };
                
                try {
                    return new Intl.NumberFormat(locale, options).format(number);
                } catch (e) {
                    return currency + ' ' + number;
                }
            });
            
            // pluralize: Word inflection based on count
            engine.registerFilter('pluralize', function(word, count, pluralForm) {
                if (count === 1) return word;
                return pluralForm || (word + 's');
            });
            
            // markdown_to_html: Simple markdown conversion
            engine.registerFilter('markdown_to_html', function(markdown) {
                if (!markdown) return '';
                
                // Basic markdown conversion (extend as needed)
                return markdown
                    .replace(/^# (.*$)/gim, '<h1>$1</h1>')
                    .replace(/^## (.*$)/gim, '<h2>$1</h2>')
                    .replace(/^### (.*$)/gim, '<h3>$1</h3>')
                    .replace(/\*\*(.*)\*\*/gim, '<strong>$1</strong>')
                    .replace(/\*(.*)\*/gim, '<em>$1</em>')
                    .replace(/\n/gim, '<br>');
            });
            
            // append_random: Generate unique identifiers
            engine.registerFilter('append_random', function(string) {
                const randomSuffix = Math.random().toString(36).substring(2, 8);
                return string + '_' + randomSuffix;
            });
            
            // days_ago: Generate date X days before current
            engine.registerFilter('days_ago', function(days) {
                const date = new Date();
                date.setDate(date.getDate() - parseInt(days));
                return date.toISOString().split('T')[0];
            });
            
        }
        
        function convertStrftimeToIntlOptions(format) {
            // Convert common strftime patterns to Intl.DateTimeFormat options
            const options = {};
            
            if (format.includes('%A')) options.weekday = 'long';
            else if (format.includes('%a')) options.weekday = 'short';
            
            if (format.includes('%B')) options.month = 'long';
            else if (format.includes('%b')) options.month = 'short';
            else if (format.includes('%m')) options.month = '2-digit';
            
            if (format.includes('%d')) options.day = '2-digit';
            else if (format.includes('%e')) options.day = 'numeric';
            
            if (format.includes('%Y')) options.year = 'numeric';
            else if (format.includes('%y')) options.year = '2-digit';
            
            if (format.includes('%H')) options.hour = '2-digit';
            else if (format.includes('%I')) {
                options.hour = '2-digit';
                options.hour12 = true;
            }
            
            if (format.includes('%M')) options.minute = '2-digit';
            if (format.includes('%S')) options.second = '2-digit';
            
            return options;
        }
        
        // Global locale cache for translations
        let localeCache = {};
        
        async function loadLocaleData(locale) {
            // Check cache first
            if (localeCache[locale]) {
                return localeCache[locale];
            }
            
            try {
                const response = await fetch('/api/locales/' + locale);
                if (response.ok) {
                    const localeData = await response.json();
                    localeCache[locale] = localeData;
                    return localeData;
                }
            } catch (e) {
                console.warn('Failed to load locale data for ' + locale + ':', e);
            }
            
            // Fallback to English if locale fails
            if (locale !== 'en') {
                return loadLocaleData('en');
            }
            
            return {};
        }
        
        function getTRNMLWordTranslations() {
            // This is now just a fallback - real translations come from loadLocaleData
            return {
                'today': { 'en': 'today' },
                'tomorrow': { 'en': 'tomorrow' }, 
                'yesterday': { 'en': 'yesterday' }
            };
        }
        
        // Define the function BEFORE the LiquidJS script loads
        function initializeLiquid() {
            
            // Use liquidjs constructor (we know this exists)
            const engine = new liquidjs.Liquid();
            
            // Register TRMNL custom filters for compatibility
            registerTRNMLFilters(engine);
            
            // Preprocess template for TRMNL syntax compatibility
            const processedTemplate = preprocessTRNMLTemplate(template);
        
            // Render template
            engine.parseAndRender(processedTemplate, data)
                .then(renderedContent => {
                    
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
                        wrappedContent = '<div id="plugin-' + instanceId + '" class="environment trmnl">' +
                            '<div class="' + screenClassString + '">' + processedTemplate + '</div>' +
                            '</div>';
                    } else {
                        wrappedContent = '<div id="plugin-' + instanceId + '" class="environment trmnl">' +
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
                            
                            // Load TRMNL plugins.js AFTER content is shown
                            const script = document.createElement('script');
                            script.src = 'https://usetrmnl.com/js/latest/plugins.js';
                            script.onload = () => {}; // Plugin loaded successfully
                            script.onerror = (e) => console.error('TRMNL plugins.js failed to load:', e);
                            document.head.appendChild(script);
                        }
                    }
                    
                    waitForDOMAndShow();
                    
                    // Wait for fonts to load before setting completion signal
                    function waitForFontsAndComplete() {
                        // Check if fonts are loaded using document.fonts API
                        if (document.fonts && document.fonts.status === 'loaded') {
                            if (document.body) {
                                document.body.setAttribute('data-render-complete', 'true');
                            }
                        } else {
                            // Fallback: wait a bit more for fonts
                            setTimeout(waitForFontsAndComplete, 100);
                        }
                    }
                    
                    // Start font loading check, but also set a maximum wait time
                    waitForFontsAndComplete();
                    
                    // Fallback: set completion signal after 2 seconds even if fonts aren't loaded
                    setTimeout(() => {
                        if (document.body && !document.body.hasAttribute('data-render-complete')) {
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
		opts.InstanceID,
		opts.RemoveBleedMargin,
		opts.EnableDarkMode)
	
	return html, nil
}


