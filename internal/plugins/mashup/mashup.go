package mashup

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// Register the mashup plugin factory when this package is imported
func init() {
	plugins.RegisterMashupPluginFactory(NewMashupPlugin)
}

// MashupPlugin implements the Plugin interface for mashup plugins
type MashupPlugin struct {
	definition *database.PluginDefinition
	instance   *database.PluginInstance
	mashupService *database.MashupService
}

// NewMashupPlugin creates a new mashup plugin instance
func NewMashupPlugin(definition *database.PluginDefinition, instance *database.PluginInstance) plugins.Plugin {
	db := database.GetDB()
	return &MashupPlugin{
		definition: definition,
		instance:   instance,
		mashupService: database.NewMashupService(db),
	}
}

// Type returns the plugin type identifier
func (p *MashupPlugin) Type() string {
	return fmt.Sprintf("mashup_%s", p.definition.ID)
}

// PluginType returns that this is an image plugin
func (p *MashupPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the instance name
func (p *MashupPlugin) Name() string {
	if p.instance != nil {
		return p.instance.Name
	}
	return p.definition.Name
}

// Description returns the plugin description
func (p *MashupPlugin) Description() string {
	return p.definition.Description
}

// Author returns the plugin author
func (p *MashupPlugin) Author() string {
	return p.definition.Author
}

// Version returns the plugin version
func (p *MashupPlugin) Version() string {
	return p.definition.Version
}

// RequiresProcessing returns true since mashups need HTML rendering
func (p *MashupPlugin) RequiresProcessing() bool {
	return true
}

// ConfigSchema returns the JSON schema for configuration
func (p *MashupPlugin) ConfigSchema() string {
	return p.definition.ConfigSchema
}

// Process executes the mashup logic - combines child plugin outputs
func (p *MashupPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	if p.instance == nil {
		return plugins.CreateErrorResponse("No plugin instance provided"), 
			fmt.Errorf("mashup plugin requires an instance")
	}
	
	// Validate device model information
	if ctx.Device == nil || ctx.Device.DeviceModel == nil {
		return plugins.CreateErrorResponse("Device model information not available"),
			fmt.Errorf("device model is required for mashup processing")
	}
	
	// Get mashup layout
	if p.definition.MashupLayout == nil {
		return plugins.CreateErrorResponse("No mashup layout defined"),
			fmt.Errorf("mashup layout is required")
	}
	layout := *p.definition.MashupLayout
	
	// Get child plugin instances
	children, err := p.mashupService.GetChildren(p.instance.ID)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to get mashup children: %v", err)),
			fmt.Errorf("failed to get mashup children: %w", err)
	}
	
	if len(children) == 0 {
		return plugins.CreateErrorResponse("No child plugins configured"),
			fmt.Errorf("mashup has no child plugins configured")
	}
	
	logging.Info("[MASHUP] Processing mashup", "layout", layout, "children_count", len(children))
	
	// Build child data directly from stored data (no plugin processing)
	childData := make(map[string]ChildData)
	pollingService := database.NewPollingDataService(database.GetDB())
	webhookService := database.NewWebhookService(database.GetDB())
	
	for _, child := range children {
		// Build template data for this child
		templateData := make(map[string]interface{})
		
		// Parse form field values from instance settings
		var formFieldValues map[string]interface{}
		if child.ChildInstance.Settings != nil {
			if err := json.Unmarshal(child.ChildInstance.Settings, &formFieldValues); err != nil {
				formFieldValues = make(map[string]interface{})
			}
		} else {
			formFieldValues = make(map[string]interface{})
		}
		
		// Fetch external data based on child plugin's data strategy
		childInstanceID := child.ChildInstance.ID.String()
		logging.Debug("[MASHUP] Processing child plugin data", 
			"slot", child.SlotPosition, 
			"plugin_name", child.ChildInstance.Name,
			"instance_id", childInstanceID,
			"data_strategy", child.ChildInstance.PluginDefinition.DataStrategy)
			
		switch dataStrategy := child.ChildInstance.PluginDefinition.DataStrategy; {
		case dataStrategy != nil && *dataStrategy == "polling":
			// Use stored polling data
			logging.Debug("[MASHUP] Querying stored polling data", "instance_id", childInstanceID)
			if storedData, err := pollingService.GetPollingDataTemplate(childInstanceID); err == nil {
				logging.Debug("[MASHUP] Retrieved polling data", 
					"instance_id", childInstanceID,
					"data_keys", getMapKeys(storedData),
					"full_data", storedData)
				for key, value := range storedData {
					templateData[key] = value
				}
			} else {
				logging.Warn("[MASHUP] Failed to get polling data for child", "slot", child.SlotPosition, "instance_id", childInstanceID, "error", err)
			}
		case dataStrategy != nil && *dataStrategy == "webhook":
			// Use stored webhook data
			if webhookData, err := webhookService.GetWebhookDataTemplate(childInstanceID); err == nil {
				for key, value := range webhookData {
					templateData[key] = value
				}
			} else {
				logging.Warn("[MASHUP] Failed to get webhook data for child", "slot", child.SlotPosition, "error", err)
			}
		case dataStrategy != nil && *dataStrategy == "static":
			// Static strategy uses only form fields and trmnl struct
			// No external data fetching needed
		}
		
		// Create TRMNL data structure for this child using shared builder
		trmnlBuilder := rendering.NewTRNMLDataBuilder()
		trmnlData := trmnlBuilder.BuildTRNMLData(ctx, &child.ChildInstance, formFieldValues)
		templateData["trmnl"] = trmnlData
		
		// Get appropriate template markup based on slot position
		templateMarkup := p.getTemplateMarkupForSlot(layout, child.SlotPosition, &child.ChildInstance.PluginDefinition)
		if templateMarkup == "" {
			logging.Error("[MASHUP] No template markup found for child", "slot", child.SlotPosition)
			childData[child.SlotPosition] = ChildData{
				Success:  false,
				Error:    "No template markup available",
				Template: "<div class='mashup-error'>Template not found</div>",
				Data:     make(map[string]interface{}),
			}
			continue
		}
		
		// Store child data for rendering
		childData[child.SlotPosition] = ChildData{
			Success:  true,
			Template: templateMarkup,
			Data:     templateData,
			Instance: &child.ChildInstance,
		}
		
		logging.Info("[MASHUP] Child data prepared successfully", "slot", child.SlotPosition, "plugin", child.ChildInstance.Name)
	}
	
	// Generate mashup HTML using the MashupRenderer with child data
	renderer := NewMashupRenderer(layout, childData)
	finalHTML, err := renderer.RenderMashup(ctx)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render mashup: %v", err)),
			fmt.Errorf("failed to render mashup: %w", err)
	}
	
	// Create browserless renderer for HTML to image conversion
	browserRenderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to create renderer: %v", err)),
			fmt.Errorf("failed to create browserless renderer: %w", err)
	}
	defer browserRenderer.Close()
	
	// Render HTML to image using browserless (like private plugins do)
	renderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	imageData, err := browserRenderer.RenderHTML(
		renderCtx,
		finalHTML,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
	)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render HTML: %v", err)),
			fmt.Errorf("failed to render HTML to image: %w", err)
	}
	
	// Generate filename
	filename := fmt.Sprintf("mashup_%s_%dx%d.png",
		time.Now().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)
	
	// Return image data response (like private plugins do)
	return plugins.CreateImageDataResponse(imageData, filename), nil
}

// Validate validates the plugin settings (currently no special validation needed)
func (p *MashupPlugin) Validate(settings map[string]interface{}) error {
	return nil
}

// GetInstance returns the plugin instance
func (p *MashupPlugin) GetInstance() *database.PluginInstance {
	return p.instance
}


// ChildData represents the data and template for a child plugin in mashup
type ChildData struct {
	Success  bool                         // Whether data preparation was successful
	Error    string                       // Error message if failed
	Template string                       // Template markup for this child
	Data     map[string]interface{}       // Template data including TRMNL structure
	Instance *database.PluginInstance     // Plugin instance data
}


// getTemplateMarkupForSlot returns the appropriate template markup based on slot position and layout
func (p *MashupPlugin) getTemplateMarkupForSlot(layout string, slotPosition string, definition *database.PluginDefinition) string {
	// Map slot positions to template types based on their ViewClass from slot metadata
	templateType := p.getTemplateTypeForSlot(layout, slotPosition)
	
	switch templateType {
	case "half_vertical":
		if definition.MarkupHalfVert != nil {
			return *definition.MarkupHalfVert
		}
	case "half_horizontal":
		if definition.MarkupHalfHoriz != nil {
			return *definition.MarkupHalfHoriz
		}
	case "quadrant":
		if definition.MarkupQuadrant != nil {
			return *definition.MarkupQuadrant
		}
	}
	
	// Fallback to MarkupFull if no appropriate template found
	if definition.MarkupFull != nil {
		return *definition.MarkupFull
	}
	
	return ""
}

// getTemplateTypeForSlot determines the template type based on layout and slot position
func (p *MashupPlugin) getTemplateTypeForSlot(layout string, slotPosition string) string {
	switch layout {
	case "1Lx1R":
		// Both left and right use half_vertical
		return "half_vertical"
		
	case "1Tx1B":
		// Both top and bottom use half_horizontal
		return "half_horizontal"
		
	case "1Lx2R":
		switch slotPosition {
		case "left":
			return "half_vertical"
		case "right-top", "right-bottom":
			return "quadrant"
		}
		
	case "2Lx1R":
		switch slotPosition {
		case "left-top", "left-bottom":
			return "quadrant"
		case "right":
			return "half_vertical"
		}
		
	case "2Tx1B":
		switch slotPosition {
		case "top-left", "top-right":
			return "quadrant"
		case "bottom":
			return "half_horizontal"
		}
		
	case "1Tx2B":
		switch slotPosition {
		case "top":
			return "half_horizontal"
		case "bottom-left", "bottom-right":
			return "quadrant"
		}
		
	case "2x2":
		// All quadrants use quadrant template
		return "quadrant"
	}
	
	// Default fallback
	return "full"
}

// getMapKeys returns the keys of a map for debugging purposes
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}