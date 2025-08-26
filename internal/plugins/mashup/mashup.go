package mashup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/plugins/private"
)

// MashupPlugin implements the Plugin interface for mashup functionality
type MashupPlugin struct{}

// NewMashupPlugin creates a new mashup plugin instance
func NewMashupPlugin() *MashupPlugin {
	return &MashupPlugin{}
}

// Type returns the unique type identifier for this plugin
func (m *MashupPlugin) Type() string {
	return "mashup"
}

// PluginType returns whether this is an image or data plugin
func (m *MashupPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeData // Mashups return data for rendering
}

// Name returns the human-readable name of the plugin
func (m *MashupPlugin) Name() string {
	return "Mashup"
}

// Description returns a description of what the plugin does
func (m *MashupPlugin) Description() string {
	return "Grid-based layout containing multiple plugin instances"
}

// Author returns the author/creator of the plugin
func (m *MashupPlugin) Author() string {
	return "Stationmaster"
}

// Version returns the version of the plugin
func (m *MashupPlugin) Version() string {
	return "1.0.0"
}

// ConfigSchema returns the JSON schema for plugin configuration
func (m *MashupPlugin) ConfigSchema() string {
	return `{
		"type": "object",
		"properties": {},
		"additionalProperties": false
	}`
}

// RequiresProcessing returns whether this plugin needs processing
func (m *MashupPlugin) RequiresProcessing() bool {
	return true
}

// RenderTemplate returns the HTML template for rendering the mashup
func (m *MashupPlugin) RenderTemplate() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { margin: 0; padding: 0; font-family: Arial, sans-serif; }
        .mashup-container {
            width: 100%;
            height: 100vh;
            display: grid;
            gap: 2px;
            background: #f0f0f0;
        }
        {{.GridCSS}}
        .mashup-cell {
            background: white;
            overflow: hidden;
            position: relative;
        }
        .mashup-cell-content {
            width: 100%;
            height: 100%;
            overflow: hidden;
        }
        .mashup-cell-error {
            display: flex;
            align-items: center;
            justify-content: center;
            color: #666;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="mashup-container">
        {{range .Children}}
        <div class="mashup-cell" data-position="{{.Position}}">
            {{if .Error}}
            <div class="mashup-cell-error">{{.Error}}</div>
            {{else}}
            <div class="mashup-cell-content">{{.Content}}</div>
            {{end}}
        </div>
        {{end}}
    </div>
</body>
</html>`
}

// DataSchema returns the schema of the data structure returned
func (m *MashupPlugin) DataSchema() string {
	return `{
		"type": "object",
		"properties": {
			"layout": {"type": "string"},
			"children": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"position": {"type": "string"},
						"content": {"type": "string"},
						"error": {"type": "string"}
					}
				}
			}
		}
	}`
}

// MashupData represents the data structure for mashup rendering
type MashupData struct {
	Layout   string       `json:"layout"`
	Children []ChildData  `json:"children"`
	GridCSS  template.CSS `json:"-"`
}

// ChildData represents data for a single child in the mashup
type ChildData struct {
	Position string `json:"position"`
	Content  string `json:"content,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Process executes the mashup plugin logic and returns rendered data
func (m *MashupPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Get mashup definition from plugin instance
	instance := ctx.PluginInstance
	definition := instance.PluginDefinition
	
	if definition.MashupLayout == nil {
		return nil, fmt.Errorf("mashup plugin missing layout configuration")
	}
	
	layout := *definition.MashupLayout
	
	// Get mashup children
	children, err := m.getMashupChildren(instance.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get mashup children: %w", err)
	}
	
	// Process each child plugin
	childData := make([]ChildData, 0, len(children))
	for _, child := range children {
		data := m.processChildPlugin(child)
		childData = append(childData, data)
	}
	
	// Generate CSS for the layout
	gridCSS := m.generateGridCSS(layout)
	
	// Return mashup data for rendering
	return plugins.PluginResponse{
		"layout":    layout,
		"children":  childData,
		"grid_css":  gridCSS,
	}, nil
}

// Validate validates the plugin settings
func (m *MashupPlugin) Validate(settings map[string]interface{}) error {
	// Mashups don't have user-configurable settings
	return nil
}

// getMashupChildren retrieves all children for a mashup instance
func (m *MashupPlugin) getMashupChildren(mashupInstanceID string) ([]database.MashupChild, error) {
	db := database.GetDB()
	var children []database.MashupChild
	
	// Parse string ID to UUID
	instanceUUID, err := uuid.Parse(mashupInstanceID)
	if err != nil {
		return nil, fmt.Errorf("invalid mashup instance ID: %w", err)
	}
	
	err = db.Preload("ChildInstance").
		Preload("ChildInstance.PluginDefinition").
		Where("mashup_instance_id = ?", instanceUUID).
		Order("grid_position").
		Find(&children).Error
	
	return children, err
}

// processChildPlugin processes a single child plugin and returns its data
func (m *MashupPlugin) processChildPlugin(child database.MashupChild) ChildData {
	childData := ChildData{
		Position: child.GridPosition,
	}
	
	// Get child plugin instance
	childInstance := &child.ChildInstance
	if childInstance.PluginDefinition.PluginType != "private" {
		childData.Error = "Only private plugins supported in mashups"
		return childData
	}
	
	// Create plugin context for child
	ctx := plugins.PluginContext{
		PluginInstance: childInstance,
		Settings:       make(map[string]interface{}),
	}
	
	// Parse child plugin settings
	if childInstance.Settings != nil {
		if err := json.Unmarshal(childInstance.Settings, &ctx.Settings); err != nil {
			childData.Error = "Failed to parse child plugin settings"
			return childData
		}
	}
	
	// Create private plugin processor
	privatePlugin := private.NewPrivatePlugin(&childInstance.PluginDefinition, childInstance)
	
	// Process child plugin
	result, err := privatePlugin.Process(ctx)
	if err != nil {
		childData.Error = fmt.Sprintf("Child plugin error: %s", err.Error())
		return childData
	}
	
	// Convert result to HTML content
	if content, ok := result["content"].(string); ok {
		childData.Content = content
	} else if html, ok := result["html"].(string); ok {
		childData.Content = html
	} else {
		// Try to render the child plugin using its template
		if dataPlugin, ok := privatePlugin.(plugins.DataPlugin); ok {
			template := dataPlugin.RenderTemplate()
			if rendered, err := m.renderChildTemplate(template, result); err == nil {
				childData.Content = rendered
			} else {
				childData.Error = "Failed to render child plugin"
			}
		} else {
			childData.Error = "Child plugin returned invalid content"
		}
	}
	
	return childData
}

// renderChildTemplate renders a child plugin's template with its data
func (m *MashupPlugin) renderChildTemplate(templateStr string, data interface{}) (string, error) {
	tmpl, err := template.New("child").Parse(templateStr)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

// generateGridCSS generates CSS grid styles based on layout type
func (m *MashupPlugin) generateGridCSS(layout string) string {
	switch strings.ToLower(layout) {
	case "1l1r", "1left1right":
		return `
			.mashup-container {
				grid-template-columns: 1fr 1fr;
				grid-template-rows: 1fr;
				grid-template-areas: "left right";
			}
			.mashup-cell[data-position="left"] { grid-area: left; }
			.mashup-cell[data-position="right"] { grid-area: right; }
		`
	case "2t1b", "2top1bottom":
		return `
			.mashup-container {
				grid-template-columns: 1fr 1fr;
				grid-template-rows: 1fr 1fr;
				grid-template-areas: 
					"top-left top-right"
					"bottom bottom";
			}
			.mashup-cell[data-position="top-left"] { grid-area: top-left; }
			.mashup-cell[data-position="top-right"] { grid-area: top-right; }
			.mashup-cell[data-position="bottom"] { grid-area: bottom; }
		`
	case "1t2b", "1top2bottom":
		return `
			.mashup-container {
				grid-template-columns: 1fr 1fr;
				grid-template-rows: 1fr 1fr;
				grid-template-areas: 
					"top top"
					"bottom-left bottom-right";
			}
			.mashup-cell[data-position="top"] { grid-area: top; }
			.mashup-cell[data-position="bottom-left"] { grid-area: bottom-left; }
			.mashup-cell[data-position="bottom-right"] { grid-area: bottom-right; }
		`
	case "2x2", "quad":
		return `
			.mashup-container {
				grid-template-columns: 1fr 1fr;
				grid-template-rows: 1fr 1fr;
				grid-template-areas: 
					"top-left top-right"
					"bottom-left bottom-right";
			}
			.mashup-cell[data-position="top-left"] { grid-area: top-left; }
			.mashup-cell[data-position="top-right"] { grid-area: top-right; }
			.mashup-cell[data-position="bottom-left"] { grid-area: bottom-left; }
			.mashup-cell[data-position="bottom-right"] { grid-area: bottom-right; }
		`
	default:
		// Default to simple left-right split
		return `
			.mashup-container {
				grid-template-columns: 1fr 1fr;
				grid-template-rows: 1fr;
				grid-template-areas: "left right";
			}
			.mashup-cell[data-position="left"] { grid-area: left; }
			.mashup-cell[data-position="right"] { grid-area: right; }
		`
	}
}

// Register the mashup plugin when this package is imported
func init() {
	// Create a mashup plugin with nil db service initially
	// The db service will be set when the plugin is actually used
	plugins.Register(&MashupPlugin{})
}