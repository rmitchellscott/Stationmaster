package mashup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/plugins/private"
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
			// Check if stored data is fresh, otherwise actively poll
			maxAge := 5 * time.Minute // Allow some staleness to avoid duplicate polls
			if isFresh, err := pollingService.IsPollingDataFresh(childInstanceID, maxAge); err == nil && isFresh {
				// Use stored polling data
				logging.Debug("[MASHUP] Using fresh stored polling data", "instance_id", childInstanceID, "slot", child.SlotPosition)
				if storedData, err := pollingService.GetPollingDataTemplate(childInstanceID); err == nil {
					for key, value := range storedData {
						templateData[key] = value
					}
				}
			} else {
				// Data is stale or doesn't exist - actively poll fresh data
				logging.Debug("[MASHUP] Stored polling data stale, actively polling", "instance_id", childInstanceID, "slot", child.SlotPosition)
				unifiedRenderer := rendering.NewUnifiedRenderer()
				poller := private.NewEnhancedDataPoller(unifiedRenderer)
				pollingCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				pollStartTime := time.Now().UTC()
				polledResult, pollErr := poller.PollData(pollingCtx, &child.ChildInstance.PluginDefinition, formFieldValues)
				pollDuration := time.Since(pollStartTime)

				if pollErr == nil && polledResult.Success {
					// Merge polling data into template data
					for key, value := range polledResult.Data {
						templateData[key] = value
					}

					// Store the polling data for future use
					rawDataJSON, _ := json.Marshal(polledResult.Data)
					mergedDataJSON, _ := json.Marshal(polledResult.Data)
					errorsJSON, _ := json.Marshal(polledResult.Errors)

					pollingData := &database.PrivatePluginPollingData{
						ID:               childInstanceID + "_polling_data",
						PluginInstanceID: childInstanceID,
						MergedData:       mergedDataJSON,
						RawData:          rawDataJSON,
						PolledAt:         time.Now().UTC(),
						PollDuration:     pollDuration,
						Success:          true,
						Errors:           errorsJSON,
						URLCount:         len(polledResult.Data),
					}

					if storeErr := pollingService.StorePollingData(pollingData); storeErr != nil {
						logging.Warn("[MASHUP] Failed to store polling data for child", "slot", child.SlotPosition, "instance_id", childInstanceID, "error", storeErr)
					} else {
						logging.Debug("[MASHUP] Stored fresh polling data for child", "slot", child.SlotPosition, "instance_id", childInstanceID, "duration", pollDuration)
					}
				} else {
					// Store failed polling attempt
					errorsJSON, _ := json.Marshal(polledResult.Errors)
					if pollErr != nil {
						errorsJSON, _ = json.Marshal([]string{pollErr.Error()})
					}

					pollingData := &database.PrivatePluginPollingData{
						ID:               childInstanceID + "_polling_data",
						PluginInstanceID: childInstanceID,
						MergedData:       []byte("{}"),
						RawData:          []byte("{}"),
						PolledAt:         time.Now().UTC(),
						PollDuration:     pollDuration,
						Success:          false,
						Errors:           errorsJSON,
						URLCount:         0,
					}

					pollingService.StorePollingData(pollingData)

					// Log error but don't fail mashup render
					if pollErr != nil {
						logging.Warn("[MASHUP] Failed to poll data for child", "slot", child.SlotPosition, "instance_id", childInstanceID, "error", pollErr)
					} else if len(polledResult.Errors) > 0 {
						logging.Warn("[MASHUP] Polling errors for child", "slot", child.SlotPosition, "instance_id", childInstanceID, "errors", polledResult.Errors)
					}
				}
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
	
	// Get slot configuration for Ruby renderer  
	slotConfig, err := p.mashupService.GetSlotMetadata(layout)
	if err != nil {
		logging.Warn("[MASHUP] Failed to get slot metadata, using empty config", "layout", layout, "error", err)
		slotConfig = []database.MashupSlotInfo{}
	}
	
	// Render each slot to individual images in parallel
	slotImages, err := p.renderMashupParallel(layout, childData, slotConfig, ctx)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render mashup: %v", err)),
			fmt.Errorf("failed to render mashup: %w", err)
	}

	// Composite images using darken blend mode
	imageData, err := p.compositeDarkenImages(
		slotImages,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
	)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to composite images: %v", err)),
			fmt.Errorf("failed to composite images: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("mashup_%s_%dx%d.png",
		time.Now().UTC().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)

	// Return image data response
	response := plugins.CreateImageDataResponse(imageData, filename)

	return response, nil
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
	// Check if this is an external plugin by looking at the plugin type
	isExternalPlugin := definition.PluginType == "external"
	
	if isExternalPlugin {
		// For external plugins, we need to indicate this is an external plugin
		// The actual rendering will be handled by calling the Ruby service
		// Return a special marker that the mashup rendering can detect
		return "EXTERNAL_PLUGIN"
	}
	
	// For private plugins, use the existing logic
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

// renderMashupParallel renders each slot HTML, then renders N full mashup images (one per slot)
// Returns map of slot position -> PNG image bytes for compositing
func (p *MashupPlugin) renderMashupParallel(layout string, childData map[string]ChildData, slotConfig []database.MashupSlotInfo, ctx plugins.PluginContext) (map[string][]byte, error) {
	logging.Info("[MASHUP] Starting parallel mashup rendering", "layout", layout, "children_count", len(childData))

	// Step 1: Render each slot's content HTML in parallel
	type slotHTMLResult struct {
		position string
		html     string
		err      error
	}

	htmlResultChan := make(chan slotHTMLResult, len(slotConfig))

	// Launch parallel rendering for each slot's content
	for _, slot := range slotConfig {
		go func(slotInfo database.MashupSlotInfo) {
			childInfo, exists := childData[slotInfo.Position]

			if !exists || !childInfo.Success {
				errorMsg := "No content available"
				if exists && !childInfo.Success {
					errorMsg = childInfo.Error
				}

				errorHTML := fmt.Sprintf(`<div class="mashup-error">%s</div>`, errorMsg)
				htmlResultChan <- slotHTMLResult{
					position: slotInfo.Position,
					html:     errorHTML,
					err:      nil,
				}
				return
			}

			var slotHTML string
			var err error

			if childInfo.Template == "EXTERNAL_PLUGIN" {
				slotHTML, err = p.fetchExternalPluginSlotHTML(childInfo, slotInfo, ctx)
				if err != nil {
					htmlResultChan <- slotHTMLResult{
						position: slotInfo.Position,
						html:     "",
						err:      fmt.Errorf("failed to render external plugin slot %s: %w", slotInfo.Position, err),
					}
					return
				}
			} else {
				unifiedRenderer := rendering.NewUnifiedRenderer()

				var sharedMarkup string
				if childInfo.Instance != nil && childInfo.Instance.PluginDefinition.SharedMarkup != nil {
					sharedMarkup = *childInfo.Instance.PluginDefinition.SharedMarkup
				}

				renderOptions := rendering.PluginRenderOptions{
					SharedMarkup:      sharedMarkup,
					LayoutTemplate:    childInfo.Template,
					Data:              childInfo.Data,
					Width:             ctx.Device.DeviceModel.ScreenWidth,
					Height:            ctx.Device.DeviceModel.ScreenHeight,
					PluginName:        fmt.Sprintf("%s (slot: %s)", p.Name(), slotInfo.Position),
					InstanceID:        fmt.Sprintf("%s-%s", p.instance.ID.String(), slotInfo.Position),
					InstanceName:      fmt.Sprintf("%s-%s", p.Name(), slotInfo.Position),
					RemoveBleedMargin: false,
					EnableDarkMode:    false,
				}

				slotHTML, err = unifiedRenderer.ProcessTemplate(context.Background(), renderOptions)
				if err != nil {
					htmlResultChan <- slotHTMLResult{
						position: slotInfo.Position,
						html:     "",
						err:      fmt.Errorf("failed to render private plugin slot %s: %w", slotInfo.Position, err),
					}
					return
				}
			}

			htmlResultChan <- slotHTMLResult{
				position: slotInfo.Position,
				html:     slotHTML,
				err:      nil,
			}
		}(slot)
	}

	// Collect slot HTML results
	renderedSlots := make(map[string]string)
	for i := 0; i < len(slotConfig); i++ {
		result := <-htmlResultChan
		if result.err != nil {
			return nil, result.err
		}
		renderedSlots[result.position] = result.html
	}

	logging.Info("[MASHUP] Slot HTML rendering completed", "slots_rendered", len(renderedSlots))

	// Step 2: Render N full mashup images (one per slot) in parallel
	type imageResult struct {
		position  string
		imageData []byte
		err       error
	}

	imageResultChan := make(chan imageResult, len(slotConfig))
	browserlessRenderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create browserless renderer: %w", err)
	}
	defer browserlessRenderer.Close()

	for _, slot := range slotConfig {
		go func(slotInfo database.MashupSlotInfo) {
			// Build full mashup HTML with only this slot active (others empty)
			fullHTML := p.buildMashupHTML(layout, renderedSlots, slotConfig, ctx, slotInfo.Position)

			// Render to PNG using browserless
			imageData, err := browserlessRenderer.RenderHTML(
				context.Background(),
				fullHTML,
				ctx.Device.DeviceModel.ScreenWidth,
				ctx.Device.DeviceModel.ScreenHeight,
			)

			if err != nil {
				imageResultChan <- imageResult{
					position:  slotInfo.Position,
					imageData: nil,
					err:       fmt.Errorf("failed to render mashup image for slot %s: %w", slotInfo.Position, err),
				}
				return
			}

			imageResultChan <- imageResult{
				position:  slotInfo.Position,
				imageData: imageData,
				err:       nil,
			}
		}(slot)
	}

	// Collect image results
	slotImages := make(map[string][]byte)
	for i := 0; i < len(slotConfig); i++ {
		result := <-imageResultChan
		if result.err != nil {
			return nil, result.err
		}
		slotImages[result.position] = result.imageData
	}

	logging.Info("[MASHUP] Image rendering completed", "images_rendered", len(slotImages))

	return slotImages, nil
}

// compositeDarkenImages combines multiple PNG images using darken blend mode
// Each image should be the same size. Empty areas are white (255,255,255).
// Darken blend takes the minimum value for each RGB channel, so white pixels
// from empty slots don't affect content pixels from active slots.
func (p *MashupPlugin) compositeDarkenImages(slotImages map[string][]byte, width, height int) ([]byte, error) {
	logging.Info("[MASHUP] Starting image composition", "image_count", len(slotImages))

	if len(slotImages) == 0 {
		return nil, fmt.Errorf("no images to composite")
	}

	// Decode all PNG images
	var images []image.Image
	for pos, imgData := range slotImages {
		img, err := png.Decode(bytes.NewReader(imgData))
		if err != nil {
			return nil, fmt.Errorf("failed to decode image for slot %s: %w", pos, err)
		}
		images = append(images, img)
	}

	// Create output image
	bounds := image.Rect(0, 0, width, height)
	output := image.NewRGBA(bounds)

	// Composite using darken blend: result = min(all images)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var minR, minG, minB uint32 = 65535, 65535, 65535

			// Find minimum value for each channel across all images
			for _, img := range images {
				r, g, b, _ := img.At(x, y).RGBA()
				if r < minR {
					minR = r
				}
				if g < minG {
					minG = g
				}
				if b < minB {
					minB = b
				}
			}

			// Convert from 16-bit to 8-bit and set pixel
			output.SetRGBA(x, y, color.RGBA{
				R: uint8(minR >> 8),
				G: uint8(minG >> 8),
				B: uint8(minB >> 8),
				A: 255,
			})
		}
	}

	logging.Info("[MASHUP] Image composition completed")

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, output); err != nil {
		return nil, fmt.Errorf("failed to encode composited image: %w", err)
	}

	return buf.Bytes(), nil
}

// buildMashupHTML builds complete mashup HTML for multi-pass rendering
// activeSlotPosition: which slot gets real content (others get empty divs for white background)
func (p *MashupPlugin) buildMashupHTML(layout string, renderedSlots map[string]string, slotConfig []database.MashupSlotInfo, ctx plugins.PluginContext, activeSlotPosition string) string {
	var contentBuilder strings.Builder

	assetsManager := rendering.NewHTMLAssetsManager()
	assetBaseURL := config.GetAssetBaseURL()

	// Build the mashup container content
	contentBuilder.WriteString(fmt.Sprintf(`<div class="environment trmnl">
	<div class="screen">
		<div class="mashup mashup--%s">`, layout))

	// Add each slot: real content for activeSlotPosition, empty div for others
	for _, slot := range slotConfig {
		var slotContent string

		if slot.Position == activeSlotPosition {
			// This is the active slot - use real rendered content
			slotContent = renderedSlots[slot.Position]
			if slotContent == "" {
				slotContent = fmt.Sprintf(`<div class="mashup-empty-slot">No content for %s</div>`, slot.DisplayName)
			}
		} else {
			// Inactive slot - empty div renders as white background
			slotContent = ""
		}

		contentBuilder.WriteString(fmt.Sprintf(`
		<div id="slot-%s" class="view %s">
			%s
		</div>`, slot.Position, slot.ViewClass, slotContent))
	}

	// Close the mashup structure
	contentBuilder.WriteString(`
		</div>
	</div>
</div>`)

	mashupContent := contentBuilder.String()

	// Wrap with TRMNL assets
	return assetsManager.WrapWithTRNMLAssets(
		mashupContent,
		p.Name(),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
		false,
		false,
		assetBaseURL,
	)
}

// fetchExternalPluginSlotHTML fetches rendered HTML from Ruby service for external plugin slots in mashup
func (p *MashupPlugin) fetchExternalPluginSlotHTML(childInfo ChildData, slotInfo database.MashupSlotInfo, ctx plugins.PluginContext) (string, error) {
	// Get service URL (same as external plugin)
	serviceURL := os.Getenv("EXTERNAL_PLUGIN_SERVICES")
	if serviceURL == "" {
		serviceURL = "http://stationmaster-plugins:3000"
	}
	
	// Get plugin identifier from definition
	pluginIdentifier := childInfo.Instance.PluginDefinition.Identifier
	
	// Build URL for plugin execution
	url := fmt.Sprintf("%s/api/plugins/%s/execute", serviceURL, pluginIdentifier)
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Parse settings from child instance
	var formFieldValues map[string]interface{}
	if childInfo.Instance.Settings != nil {
		if err := json.Unmarshal(childInfo.Instance.Settings, &formFieldValues); err != nil {
			formFieldValues = make(map[string]interface{})
		}
	} else {
		formFieldValues = make(map[string]interface{})
	}
	
	// Determine layout based on slot template type
	layout := p.getLayoutForSlot(slotInfo.Position, &childInfo.Instance.PluginDefinition)
	
	// Create TRMNL data structure using shared builder
	trmnlBuilder := rendering.NewTRNMLDataBuilder()
	trmnlData := trmnlBuilder.BuildTRNMLData(ctx, childInfo.Instance, formFieldValues)
	
	// Prepare POST request with settings and layout info
	requestBody := map[string]interface{}{
		"settings": formFieldValues,
		"layout":   layout,
		"trmnl":    trmnlData,
	}
	
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}
	
	// Create POST request
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("plugin service returned status %d", resp.StatusCode)
	}
	
	// Read response as plain text (HTML)
	var buf strings.Builder
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	
	html := buf.String()
	logging.Debug("[MASHUP] Fetched rendered HTML for external plugin slot", 
		"plugin", pluginIdentifier, 
		"slot", slotInfo.Position,
		"layout", layout,
		"html_length", len(html))
	
	return html, nil
}

// getLayoutForSlot maps slot position to the appropriate layout name for external plugins
func (p *MashupPlugin) getLayoutForSlot(slotPosition string, definition *database.PluginDefinition) string {
	// Get the template type that would be used for this slot
	templateType := p.getTemplateTypeForSlot(*p.definition.MashupLayout, slotPosition)
	
	// Map template types to layout names that the Ruby service expects
	switch templateType {
	case "half_vertical":
		return "half_vertical"
	case "half_horizontal":
		return "half_horizontal"  
	case "quadrant":
		return "quadrant"
	default:
		return "full" // fallback
	}
}

// getMapKeys returns the keys of a map for debugging purposes
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}