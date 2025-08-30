package official

import (
	"fmt"
	"regexp"
	"strings"
)

// TemplateConverter handles conversion from ERB to Liquid syntax
type TemplateConverter struct {
	baseURL string
}

// NewTemplateConverter creates a new template converter
func NewTemplateConverter(baseURL string) *TemplateConverter {
	if baseURL == "" {
		baseURL = "https://app.usetrmnl.com"
	}
	return &TemplateConverter{
		baseURL: baseURL,
	}
}

// ConvertERBToLiquid converts ERB template syntax to Liquid
func (tc *TemplateConverter) ConvertERBToLiquid(template string) string {
	result := template
	
	// Replace Rails.application.credentials.base_url with variable
	result = strings.ReplaceAll(result, "<%= Rails.application.credentials.base_url %>", "{{ base_url }}")
	
	// Convert ERB output tags <%= %> to Liquid {{ }}
	// More complex regex to handle multi-line and nested content
	erbOutputRegex := regexp.MustCompile(`<%=\s*(.*?)\s*%>`)
	result = erbOutputRegex.ReplaceAllStringFunc(result, func(match string) string {
		// Extract content between <%= and %>
		content := erbOutputRegex.FindStringSubmatch(match)[1]
		
		// Handle special cases
		content = tc.convertERBExpression(content)
		
		return fmt.Sprintf("{{ %s }}", content)
	})
	
	// Convert ERB control structures <% %> to Liquid {% %}
	// Handle if/elsif/else/end
	result = tc.convertControlStructures(result)
	
	// Convert render partials
	result = tc.convertPartials(result)
	
	// Convert Rails helpers
	result = tc.convertHelpers(result)
	
	return result
}

// convertERBExpression converts individual ERB expressions to Liquid
func (tc *TemplateConverter) convertERBExpression(expr string) string {
	// Trim spaces
	expr = strings.TrimSpace(expr)
	
	// Handle instance_name specially (common in templates)
	if expr == "instance_name" {
		return "trmnl.plugin_instance.name"
	}
	
	// Handle message, days_left, days_passed, etc. (direct variable access)
	// These come from the plugin's locals method
	if !strings.Contains(expr, ".") && !strings.Contains(expr, "(") && !strings.Contains(expr, "[") {
		return expr
	}
	
	// Handle method calls with parentheses - convert to Liquid filters where appropriate
	if strings.Contains(expr, "t(") {
		// Translation helper - for now just return the key
		// t('renders.days_left_year.days_passed') -> 'Days passed'
		re := regexp.MustCompile(`t\(['"]([^'"]+)['"]\)`)
		if matches := re.FindStringSubmatch(expr); len(matches) > 1 {
			key := matches[1]
			parts := strings.Split(key, ".")
			lastPart := strings.ReplaceAll(parts[len(parts)-1], "_", " ")
			// Convert to sentence case (first letter uppercase, rest lowercase)
			words := strings.Fields(lastPart)
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
				}
			}
			return fmt.Sprintf("'%s'", strings.Join(words, " "))
		}
	}
	
	// Handle array access with brackets
	if strings.Contains(expr, "[") {
		// Convert Ruby array access to Liquid
		// phase[:icon] -> phase.icon
		re := regexp.MustCompile(`(\w+)\[:(\w+)\]`)
		expr = re.ReplaceAllString(expr, "$1.$2")
	}
	
	return expr
}

// convertControlStructures converts ERB control structures to Liquid
func (tc *TemplateConverter) convertControlStructures(template string) string {
	result := template
	
	// Convert if statements
	// <% if condition %> -> {% if condition %}
	ifRegex := regexp.MustCompile(`<%\s*if\s+(.*?)\s*%>`)
	result = ifRegex.ReplaceAllStringFunc(result, func(match string) string {
		condition := ifRegex.FindStringSubmatch(match)[1]
		condition = tc.convertCondition(condition)
		return fmt.Sprintf("{%% if %s %%}", condition)
	})
	
	// Convert elsif statements
	elsifRegex := regexp.MustCompile(`<%\s*elsif\s+(.*?)\s*%>`)
	result = elsifRegex.ReplaceAllStringFunc(result, func(match string) string {
		condition := elsifRegex.FindStringSubmatch(match)[1]
		condition = tc.convertCondition(condition)
		return fmt.Sprintf("{%% elsif %s %%}", condition)
	})
	
	// Convert else
	result = strings.ReplaceAll(result, "<% else %>", "{% else %}")
	
	// Convert each loops
	// <% items.each do |item| %> -> {% for item in items %}
	eachRegex := regexp.MustCompile(`<%\s*(\w+)\.each\s+do\s*\|\s*(\w+)\s*\|\s*%>`)
	result = eachRegex.ReplaceAllString(result, "{% for $2 in $1 %}")
	
	// Convert times loops (used in progress bar)
	// <% max_days.times do |idx| %> -> {% for idx in (0..max_days) %}
	timesRegex := regexp.MustCompile(`<%\s*(\w+)\.times\s+do\s*\|\s*(\w+)\s*\|\s*%>`)
	result = timesRegex.ReplaceAllStringFunc(result, func(match string) string {
		matches := timesRegex.FindStringSubmatch(match)
		variable := matches[1]
		iterator := matches[2]
		// Liquid doesn't have a direct times equivalent, use range
		return fmt.Sprintf("{%% for %s in (1..%s) %%}", iterator, variable)
	})
	
	// Handle ends more intelligently
	result = tc.convertEnds(result)
	
	// Handle unless (convert to if not)
	unlessRegex := regexp.MustCompile(`<%\s*unless\s+(.*?)\s*%>`)
	result = unlessRegex.ReplaceAllStringFunc(result, func(match string) string {
		condition := unlessRegex.FindStringSubmatch(match)[1]
		condition = tc.convertCondition(condition)
		return fmt.Sprintf("{%% unless %s %%}", condition)
	})
	
	return result
}

// convertEnds handles converting <% end %> to appropriate Liquid ends
func (tc *TemplateConverter) convertEnds(template string) string {
	lines := strings.Split(template, "\n")
	var stack []string // track what kind of block we're in
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Track opening blocks
		if strings.Contains(trimmed, "{% for ") {
			stack = append(stack, "for")
		} else if strings.Contains(trimmed, "{% if ") {
			stack = append(stack, "if")
		} else if strings.Contains(trimmed, "{% unless ") {
			stack = append(stack, "unless")
		}
		
		// Handle closing blocks
		if strings.Contains(trimmed, "<% end %>") {
			if len(stack) > 0 {
				blockType := stack[len(stack)-1]
				stack = stack[:len(stack)-1] // pop
				
				switch blockType {
				case "for":
					lines[i] = strings.ReplaceAll(lines[i], "<% end %>", "{% endfor %}")
				case "if":
					lines[i] = strings.ReplaceAll(lines[i], "<% end %>", "{% endif %}")
				case "unless":
					lines[i] = strings.ReplaceAll(lines[i], "<% end %>", "{% endunless %}")
				default:
					lines[i] = strings.ReplaceAll(lines[i], "<% end %>", "{% endif %}")
				}
			} else {
				// Default to endif if we can't track the block type
				lines[i] = strings.ReplaceAll(lines[i], "<% end %>", "{% endif %}")
			}
		}
	}
	
	return strings.Join(lines, "\n")
}

// convertCondition converts Ruby conditions to Liquid
func (tc *TemplateConverter) convertCondition(condition string) string {
	// Convert Ruby || to Liquid or
	condition = strings.ReplaceAll(condition, "||", "or")
	
	// Convert Ruby && to Liquid and
	condition = strings.ReplaceAll(condition, "&&", "and")
	
	// Convert Ruby array/hash access
	// phase[:current] -> phase.current
	re := regexp.MustCompile(`(\w+)\[:(\w+)\]`)
	condition = re.ReplaceAllString(condition, "$1.$2")
	
	// Handle method calls (remove parentheses for simple cases)
	// show_days_left? -> show_days_left
	condition = strings.ReplaceAll(condition, "?", "")
	
	return condition
}

// convertPartials converts ERB render calls to Liquid
func (tc *TemplateConverter) convertPartials(template string) string {
	// <%= render 'plugins/days_left_until/progress_bar', percent_passed: percent_passed %>
	// Since we're including partials as SharedMarkup, remove render calls entirely
	// as the content will already be included
	
	renderRegex := regexp.MustCompile(`<%=\s*render\s+['"]([^'"]+)['"](.*?)\s*%>`)
	result := renderRegex.ReplaceAllStringFunc(template, func(match string) string {
		matches := renderRegex.FindStringSubmatch(match)
		partial := matches[1]
		
		// Extract just the partial name (remove path)
		parts := strings.Split(partial, "/")
		partialName := parts[len(parts)-1]
		
		// For common partials that are now included as SharedMarkup, remove the render call
		if partialName == "common" || strings.HasPrefix(partialName, "_common") {
			return "" // Remove the render call - content is in SharedMarkup
		}
		
		// For other partials, keep the render call but convert it
		params := matches[2]
		
		// Remove leading underscore if present
		partialName = strings.TrimPrefix(partialName, "_")
		
		if strings.TrimSpace(params) == "" {
			return fmt.Sprintf("{%% render '%s' %%}", partialName)
		}
		
		// Convert Ruby hash syntax to Liquid
		// , percent_passed: percent_passed -> , percent_passed: percent_passed
		params = strings.ReplaceAll(params, ":", "=")
		params = strings.TrimPrefix(params, ",")
		params = strings.TrimSpace(params)
		
		return fmt.Sprintf("{%% render '%s' %s %%}", partialName, params)
	})
	
	return result
}

// convertHelpers converts Rails helpers to Liquid equivalents
func (tc *TemplateConverter) convertHelpers(template string) string {
	// t() helper for translations - simplified for now
	// We'll handle these as static strings or variables
	
	// l() helper for localization
	// l(date, format: :short) -> date | date: '%b %d'
	lRegex := regexp.MustCompile(`{{\s*l\(([^,)]+)(?:,\s*format:\s*:(\w+))?\)\s*}}`)
	template = lRegex.ReplaceAllStringFunc(template, func(match string) string {
		matches := lRegex.FindStringSubmatch(match)
		dateVar := matches[1]
		format := matches[2]
		
		liquidFormat := "%Y-%m-%d" // default
		switch format {
		case "short":
			liquidFormat = "%b %d"
		case "long":
			liquidFormat = "%B %d, %Y"
		}
		
		return fmt.Sprintf("{{ %s | date: '%s' }}", dateVar, liquidFormat)
	})
	
	return template
}

// ConvertTemplateFiles converts a map of template files from ERB to Liquid
func (tc *TemplateConverter) ConvertTemplateFiles(templates map[string]string) map[string]string {
	converted := make(map[string]string)
	
	for name, content := range templates {
		converted[name] = tc.ConvertERBToLiquid(content)
	}
	
	return converted
}