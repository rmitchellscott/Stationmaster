package rendering

import (
	"fmt"
)

// BaseHTMLOptions contains configuration for generating HTML documents with embedded JavaScript
type BaseHTMLOptions struct {
	Width             int
	Height            int
	Title             string
	RemoveBleedMargin bool
	EnableDarkMode    bool
	ScriptLoadStrategy ScriptLoadStrategy
}

// ScriptLoadStrategy defines how TRMNL scripts should be loaded
type ScriptLoadStrategy int

const (
	// ScriptLoadSequential loads scripts one after another (used by private plugins)
	ScriptLoadSequential ScriptLoadStrategy = iota
	// ScriptLoadInHead loads all scripts in document head (used by mashup plugins)
	ScriptLoadInHead
)

// BaseHTMLRenderer provides shared HTML generation functionality
type BaseHTMLRenderer struct{}

// NewBaseHTMLRenderer creates a new base HTML renderer
func NewBaseHTMLRenderer() *BaseHTMLRenderer {
	return &BaseHTMLRenderer{}
}

// GenerateHTML creates a complete HTML document with embedded TRMNL functionality
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
	
	// Create the complete HTML document
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
    <div id="output" style="display: none;"></div>
    
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
        
        // Global error handler for uncaught errors
        window.addEventListener('error', function(e) {
            console.error('UNCAUGHT ERROR during rendering:', e.error, e);
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

// generateHeadScripts returns TRMNL scripts to be loaded in the document head
func (r *BaseHTMLRenderer) generateHeadScripts() string {
	return `<script src="https://cdn.jsdelivr.net/npm/liquidjs@10.10.1/dist/liquid.browser.umd.js"></script>
    <!-- TRMNL Scripts for core functionality, filters, and rendering -->
    <script src="https://usetrmnl.com/js/latest/plugins.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-bfbd7e9488fd0d6dff2f619b5cb963c0772a24d6d0b537f60089dc53aa4746ff.js"></script>
    <script src="https://usetrmnl.com/assets/plugin_legacy-0c72702a185603fd7fc5eb915658f49486903cb5c92cd6153a336b8ce3973452.js"></script>
    <script src="https://usetrmnl.com/assets/plugin_demo-25268352c5a400b970985521a5eaa3dc90c736ce0cbf42d749e7e253f0c227f5.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/plugins-332ca4207dd02576b3641691907cb829ef52a36c4a092a75324a8fc860906967.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/plugins_legacy-a6b0b3aeac32ca71413f1febc053c59a528d4c6bb2173c22bd94ff8e0b9650f1.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/dithering-d697f6229e3bd6e2455425d647e5395bb608999c2039a9837a903c7c7e952d61.js"></script>
    <script src="https://usetrmnl.com/assets/plugin-render/asset-deduplication-39fa2231b7a5bd5bedf4a1782b6a95d8b87eb3aaaa5e2b6cee287133d858bc96.js"></script>`
}

// generateSharedJavaScriptFunctions returns common JavaScript functions used by both renderers
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
        console.log('Base renderer JavaScript loaded successfully');
        console.log('renderData available:', typeof renderData !== 'undefined');
        console.log('LiquidJS loading...');
        
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
        
        function registerTRNMLExtensions(engine) {
            
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
                    const options = convertStrftimeToIntlOptions(format || '%%Y-%%m-%%d');
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
            
            // Apply TRMNL template processing (two-pass approach)
            processTRNMLTemplates(engine);
        }
        
        function processTRNMLTemplates(engine) {
            // Two-pass TRMNL template processing
            // This function will be called before template rendering to extract and register templates
            
            // Store original parseAndRender method
            const originalParseAndRender = engine.parseAndRender.bind(engine);
            
            // Override parseAndRender to implement two-pass processing
            engine.parseAndRender = async function(template, data) {
                console.log('TRMNL two-pass template processing...');
                
                // PASS 1: Extract template definitions using regex
                const templates = {};
                const templateRegex = /\{%\s*template\s+([a-zA-Z0-9_\/]+)\s*%\}([\s\S]*?)\{%\s*endtemplate\s*%\}/g;
                let match;
                
                while ((match = templateRegex.exec(template)) !== null) {
                    const templateName = match[1];
                    const templateContent = match[2].trim();
                    templates[templateName] = templateContent;
                    console.log('Extracted TRMNL template:', templateName);
                }
                
                // PASS 2: Create new engine with extracted templates if any were found
                if (Object.keys(templates).length > 0) {
                    console.log('Creating engine with', Object.keys(templates).length, 'TRMNL templates');
                    
                    // Create new engine with extracted templates
                    const templatedEngine = new liquidjs.Liquid({
                        templates: templates
                    });
                    
                    // Re-register all TRMNL filters on the new engine
                    registerTRNMLFiltersOnly(templatedEngine);
                    
                    // Remove template definitions from main template
                    const cleanedTemplate = template.replace(templateRegex, '').trim();
                    console.log('Rendering cleaned template with', Object.keys(templates).length, 'registered templates');
                    
                    // Render using the new engine with templates
                    return await templatedEngine.parseAndRender(cleanedTemplate, data);
                } else {
                    console.log('No TRMNL templates found, using original rendering');
                    // No templates found, use original rendering
                    return await originalParseAndRender(template, data);
                }
            };
        }
        
        // Separate function to register only the filters (not the template processing)
        function registerTRNMLFiltersOnly(engine) {
            
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
                    const options = convertStrftimeToIntlOptions(format || '%%Y-%%m-%%d');
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
            
            if (format.includes('%%A')) options.weekday = 'long';
            else if (format.includes('%%a')) options.weekday = 'short';
            
            if (format.includes('%%B')) options.month = 'long';
            else if (format.includes('%%b')) options.month = 'short';
            else if (format.includes('%%m')) options.month = '2-digit';
            
            if (format.includes('%%d')) options.day = '2-digit';
            else if (format.includes('%%e')) options.day = 'numeric';
            
            if (format.includes('%%Y')) options.year = 'numeric';
            else if (format.includes('%%y')) options.year = '2-digit';
            
            if (format.includes('%%H')) options.hour = '2-digit';
            else if (format.includes('%%I')) {
                options.hour = '2-digit';
                options.hour12 = true;
            }
            
            if (format.includes('%%M')) options.minute = '2-digit';
            if (format.includes('%%S')) options.second = '2-digit';
            
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
        
        function executeInnerHTMLScripts(containerElement) {
            // CRITICAL FIX: Execute any script tags that were inserted via innerHTML
            // Browsers don't execute script tags when added through innerHTML for security
            const scripts = containerElement.querySelectorAll('script');
            scripts.forEach(script => {
                console.log('Executing template script:', script.textContent.substring(0, 100) + '...');
                let scriptContent = script.textContent;
                
                // IMPORTANT: Make functions globally accessible
                // Convert 'const functionName =' to 'window.functionName =' 
                scriptContent = scriptContent.replace(/const\s+(\w+)\s*=\s*\(/g, 'window.$1 = (');
                
                const newScript = document.createElement('script');
                newScript.textContent = scriptContent;
                document.body.appendChild(newScript);
            });
            
            // CRITICAL: Handle DOMContentLoaded timing issue
            // Since DOM was already loaded when our scripts executed, any DOMContentLoaded 
            // event listeners in the template scripts never fired. Dispatch the event manually.
            setTimeout(() => {
                console.log('Dispatching DOMContentLoaded event for template scripts');
                document.dispatchEvent(new Event('DOMContentLoaded'));
            }, 50);
        }
        
        function loadTRNMLScriptsSequentially(callback) {
            const scriptUrls = [
                // Main plugins.js (layout/typography utilities)
                'https://usetrmnl.com/js/latest/plugins.js',
                
                // Core Plugin Files
                'https://usetrmnl.com/assets/plugin-bfbd7e9488fd0d6dff2f619b5cb963c0772a24d6d0b537f60089dc53aa4746ff.js',
                'https://usetrmnl.com/assets/plugin_legacy-0c72702a185603fd7fc5eb915658f49486903cb5c92cd6153a336b8ce3973452.js',
                'https://usetrmnl.com/assets/plugin_demo-25268352c5a400b970985521a5eaa3dc90c736ce0cbf42d749e7e253f0c227f5.js',
                
                // Plugin Render Files
                'https://usetrmnl.com/assets/plugin-render/plugins-332ca4207dd02576b3641691907cb829ef52a36c4a092a75324a8fc860906967.js',
                'https://usetrmnl.com/assets/plugin-render/plugins_legacy-a6b0b3aeac32ca71413f1febc053c59a528d4c6bb2173c22bd94ff8e0b9650f1.js',
                'https://usetrmnl.com/assets/plugin-render/dithering-d697f6229e3bd6e2455425d647e5395bb608999c2039a9837a903c7c7e952d61.js',
                'https://usetrmnl.com/assets/plugin-render/asset-deduplication-39fa2231b7a5bd5bedf4a1782b6a95d8b87eb3aaaa5e2b6cee287133d858bc96.js'
            ];
            
            function loadScriptsSequentially(urls, index = 0) {
                if (index >= urls.length) {
                    // All scripts loaded - handle dithering timing
                    setTimeout(() => {
                        handleDitheringTiming();
                        if (callback) callback();
                    }, 100);
                    return;
                }
                
                const script = document.createElement('script');
                script.src = urls[index];
                script.onload = () => loadScriptsSequentially(urls, index + 1);
                script.onerror = (e) => {
                    console.error('TRMNL script failed to load:', urls[index], e);
                    loadScriptsSequentially(urls, index + 1); // Continue loading other scripts
                };
                document.head.appendChild(script);
            }
            
            loadScriptsSequentially(scriptUrls);
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
        
        function waitForFontsAndComplete() {
            try {
                // Check if fonts are loaded using document.fonts API
                if (document.fonts && document.fonts.status === 'loaded') {
                    console.log('Fonts loaded - setting completion signal');
                    if (document.body) {
                        document.body.setAttribute('data-render-complete', 'true');
                    }
                } else {
                    // Check if document.fonts API is available
                    if (document.fonts) {
                        console.log('Waiting for fonts to load...');
                        // Fallback: wait a bit more for fonts
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
        
        // Centralized completion function for plugins to call
        function signalRenderingComplete() {
            try {
                console.log('signalRenderingComplete() called - starting font loading check');
                // Start font loading check for optimal completion timing
                waitForFontsAndComplete();
                
                // Additional safety: if fonts take too long, complete anyway
                setTimeout(() => {
                    if (document.body && !document.body.hasAttribute('data-render-complete')) {
                        console.log('Font loading timeout - completing anyway');
                        document.body.setAttribute('data-render-complete', 'true');
                    }
                }, 1500); // Shorter than the 3-second absolute fallback
            } catch (error) {
                console.error('Error in signalRenderingComplete():', error);
                // Immediately set completion on error
                if (document.body) {
                    document.body.setAttribute('data-render-complete', 'true');
                }
            }
        }
	`
}