package rendering

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// TemplateEngine handles HTML template rendering
type TemplateEngine struct {
	funcMap template.FuncMap
}

// NewTemplateEngine creates a new template engine with helpful functions
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		funcMap: template.FuncMap{
			// Date/time functions
			"now":        time.Now,
			"formatDate": formatDate,
			"formatTime": formatTime,
			
			// String functions
			"upper":    strings.ToUpper,
			"lower":    strings.ToLower,
			"title":    strings.Title,
			"trim":     strings.TrimSpace,
			"contains": strings.Contains,
			"replace":  strings.ReplaceAll,
			
			// Math functions
			"add":      add,
			"subtract": subtract,
			"multiply": multiply,
			"divide":   divide,
			"round":    round,
			"percent":  percent,
			
			// Formatting functions
			"currency": formatCurrency,
			"number":   formatNumber,
			"bytes":    formatBytes,
		},
	}
}

// RenderTemplate renders a template with the given data
func (te *TemplateEngine) RenderTemplate(templateStr string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New("plugin").Funcs(te.funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	
	return buf.String(), nil
}

// Template helper functions
func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func formatTime(t time.Time) string {
	return t.Format("15:04:05")
}

func add(a, b int) int {
	return a + b
}

func subtract(a, b int) int {
	return a - b
}

func multiply(a, b int) int {
	return a * b
}

func divide(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}

func round(f float64) int {
	return int(f + 0.5)
}

func percent(value, total float64) float64 {
	if total == 0 {
		return 0
	}
	return (value / total) * 100
}

func formatCurrency(value float64, symbol string) string {
	return fmt.Sprintf("%s%.2f", symbol, value)
}

func formatNumber(value interface{}) string {
	switch v := value.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.2f", v)
	case float32:
		return fmt.Sprintf("%.2f", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}