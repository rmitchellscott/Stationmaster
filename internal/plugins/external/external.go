package external

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/validation"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// ExternalPlugin implements the Plugin interface for externally-sourced plugins
type ExternalPlugin struct {
	definition *database.PluginDefinition
	instance   *database.PluginInstance
	serviceURL string
}

// NewExternalPlugin creates a new external plugin instance
func NewExternalPlugin(definition *database.PluginDefinition, instance *database.PluginInstance) plugins.Plugin {
	serviceURL := os.Getenv("EXTERNAL_PLUGIN_SERVICES")
	if serviceURL == "" {
		serviceURL = "http://stationmaster-plugins:3000"
	}
	
	return &ExternalPlugin{
		definition: definition,
		instance:   instance,
		serviceURL: serviceURL,
	}
}

// Type returns the plugin type identifier based on the definition
func (p *ExternalPlugin) Type() string {
	return fmt.Sprintf("external_%s", p.definition.ID)
}

// PluginType returns that this is an image plugin (uses templates + data rendering)
func (p *ExternalPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the instance name if available, otherwise definition name
func (p *ExternalPlugin) Name() string {
	if p.instance != nil {
		return p.instance.Name
	}
	return p.definition.Name
}

// Description returns the plugin description
func (p *ExternalPlugin) Description() string {
	return p.definition.Description
}

// Author returns the plugin author (always TRMNL for external plugins)
func (p *ExternalPlugin) Author() string {
	return p.definition.Author
}

// Version returns the plugin version
func (p *ExternalPlugin) Version() string {
	return p.definition.Version
}

// RequiresProcessing returns true since external plugins need HTML rendering
func (p *ExternalPlugin) RequiresProcessing() bool {
	return p.definition.RequiresProcessing
}

// ConfigSchema returns the JSON schema for form fields
func (p *ExternalPlugin) ConfigSchema() string {
	logging.Debug("[EXTERNAL_PLUGIN] ConfigSchema called", "plugin", p.definition.Identifier)
	
	if p.definition.FormFields == nil {
		logging.Debug("[EXTERNAL_PLUGIN] FormFields is nil", "plugin", p.definition.Identifier)
		return `{"type": "object", "properties": {}}`
	}
	
	logging.Debug("[EXTERNAL_PLUGIN] FormFields found", "plugin", p.definition.Identifier, "formFields", string(p.definition.FormFields))
	
	// Parse the FormFields JSON and convert YAML to JSON schema
	var formFieldsData interface{}
	if err := json.Unmarshal(p.definition.FormFields, &formFieldsData); err != nil {
		logging.Error("[EXTERNAL_PLUGIN] Failed to parse FormFields JSON", "plugin", p.definition.Identifier, "error", err, "formFields", string(p.definition.FormFields))
		return `{"type": "object", "properties": {}}`
	}
	
	logging.Debug("[EXTERNAL_PLUGIN] FormFields JSON parsed successfully", "plugin", p.definition.Identifier, "parsedData", formFieldsData)
	
	// Use the validation function to convert YAML form fields to JSON schema
	jsonSchema, err := validation.ValidateFormFields(formFieldsData)
	if err != nil {
		logging.Error("[EXTERNAL_PLUGIN] Failed to convert form fields to JSON schema", "plugin", p.definition.Identifier, "error", err, "formFieldsData", formFieldsData)
		return `{"type": "object", "properties": {}}`
	}
	
	return jsonSchema
}

// Process executes the plugin logic - fetches fully rendered HTML from Ruby service
func (p *ExternalPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Validate device model information
	if ctx.Device == nil || ctx.Device.DeviceModel == nil {
		return plugins.CreateErrorResponse("Device model information not available"),
			fmt.Errorf("device model is required for external plugin processing")
	}
	
	// Parse form field values from instance settings
	var formFieldValues map[string]interface{}
	if p.instance != nil && p.instance.Settings != nil {
		if err := json.Unmarshal(p.instance.Settings, &formFieldValues); err != nil {
			formFieldValues = make(map[string]interface{})
		}
	} else {
		formFieldValues = make(map[string]interface{})
	}

	// For standalone external plugins, use "full" layout
	layout := "full"
	
	// Fetch processed HTML from Ruby service (includes plugin execution + ERB rendering)
	processedContent, err := p.fetchRenderedHTML(formFieldValues, layout, ctx)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to fetch rendered HTML: %v", err)),
			fmt.Errorf("failed to fetch rendered HTML for external plugin %s: %w", p.definition.ID, err)
	}
	
	// Wrap content with same structure as private plugins get from generateHTMLStructure()
	// This provides the .environment.trmnl and .screen wrappers needed for proper CSS layout
	// Note: Don't add extra .view wrapper since external plugin content already has it
	structuredContent := fmt.Sprintf(`<div id="plugin-%s" class="environment trmnl">
		<div class="screen">
			%s
		</div>
	</div>`, p.instance.ID.String(), processedContent)
	
	// Wrap with TRMNL assets like private plugins do (same as UnifiedRenderer does)
	assetsManager := rendering.NewHTMLAssetsManager()
	assetBaseURL := config.GetAssetBaseURL()
	wrappedHTML := assetsManager.WrapWithTRNMLAssets(
		structuredContent,
		p.Name(),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
		false, // removeBleedMargin - TODO: Make configurable if needed
		false, // enableDarkMode - TODO: Make configurable if needed
		assetBaseURL,
	)
	
	// Create browserless renderer
	browserRenderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to create renderer: %v", err)),
			fmt.Errorf("failed to create browserless renderer: %w", err)
	}
	defer browserRenderer.Close()
	
	// Render wrapped HTML to image using browserless (same as private plugins)
	renderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	renderResult, err := browserRenderer.RenderHTMLWithResult(
		renderCtx,
		wrappedHTML,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
	)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render HTML: %v", err)),
			fmt.Errorf("failed to render HTML to image: %w", err)
	}
	
	imageData := renderResult.ImageData
	flags := renderResult.Flags
	
	// Generate filename
	filename := fmt.Sprintf("external_plugin_%s_%dx%d.png",
		time.Now().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)
	
	// Return image data response (RenderWorker will handle storage)
	response := plugins.CreateImageDataResponse(imageData, filename)
	// Add flags to response metadata if needed
	if flags.SkipDisplay {
		response["skip_display"] = true
	}
	
	return response, nil
}


// fetchRenderedHTML fetches fully rendered HTML from the external plugin service
func (p *ExternalPlugin) fetchRenderedHTML(settings map[string]interface{}, layout string, ctx plugins.PluginContext) (string, error) {
	// Build URL for plugin execution - use plugin identifier as name
	url := fmt.Sprintf("%s/api/plugins/%s/execute", p.serviceURL, p.definition.Identifier)
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Create TRMNL data structure using shared builder
	trmnlBuilder := rendering.NewTRNMLDataBuilder()
	trmnlData := trmnlBuilder.BuildTRNMLData(ctx, p.instance, settings)
	
	// Inject OAuth tokens for external service integration
	if ctx.User != nil {
		oauthTokens, err := p.getOAuthTokensForUser(ctx.User.ID.String())
		if err == nil && len(oauthTokens) > 0 {
			trmnlData["oauth_tokens"] = oauthTokens
		}
	}
	
	// Prepare POST request with settings and layout info
	requestBody := map[string]interface{}{
		"settings": settings,
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
	logging.Debug("[EXTERNAL_PLUGIN] Fetched rendered HTML", "plugin", p.definition.Identifier, "html_length", len(html))
	
	return html, nil
}

// Validate validates the plugin settings against the form fields schema
func (p *ExternalPlugin) Validate(settings map[string]interface{}) error {
	// TODO: Implement JSON schema validation against FormFields
	return nil
}

// GetInstance returns the plugin instance
func (p *ExternalPlugin) GetInstance() *database.PluginInstance {
	return p.instance
}

// getOAuthTokensForUser retrieves OAuth refresh tokens for the user to inject into plugin execution
func (p *ExternalPlugin) getOAuthTokensForUser(userID string) (map[string]map[string]string, error) {
	// Import auth package to access OAuth token functions
	// We need to get OAuth tokens that might be relevant to this plugin

	logging.Debug("[EXTERNAL_PLUGIN] Getting OAuth tokens for user", "user_id", userID)
	tokens := make(map[string]map[string]string)

	// Try to get Google OAuth token (for Google Analytics, YouTube Analytics)
	if googleToken, err := getOAuthTokenFromAuth(userID, "google"); err == nil && googleToken != nil {
		logging.Debug("[EXTERNAL_PLUGIN] Found Google token", "access_len", len(googleToken.AccessToken), "refresh_len", len(googleToken.RefreshToken))
		tokens["google"] = map[string]string{
			"access_token":  googleToken.AccessToken,
			"refresh_token": googleToken.RefreshToken,
		}
	} else {
		logging.Debug("[EXTERNAL_PLUGIN] No Google token found", "error", err)
	}

	// Try to get Todoist OAuth token
	if todoistToken, err := getOAuthTokenFromAuth(userID, "todoist"); err == nil && todoistToken != nil {
		logging.Debug("[EXTERNAL_PLUGIN] Found Todoist token", "access_len", len(todoistToken.AccessToken), "refresh_len", len(todoistToken.RefreshToken))
		tokens["todoist"] = map[string]string{
			"access_token":  todoistToken.AccessToken,
			"refresh_token": todoistToken.RefreshToken,
		}
	} else {
		logging.Debug("[EXTERNAL_PLUGIN] No Todoist token found", "error", err)
	}

	// Add other providers as needed

	logging.Debug("[EXTERNAL_PLUGIN] Returning tokens", "token_count", len(tokens), "providers", func() []string {
		keys := make([]string, 0, len(tokens))
		for k := range tokens {
			keys = append(keys, k)
		}
		return keys
	}())

	return tokens, nil
}

// getOAuthTokenFromAuth is a helper function to get OAuth tokens from the auth package
func getOAuthTokenFromAuth(userID, provider string) (*database.UserOAuthToken, error) {
	// This function needs to access the auth package to retrieve tokens
	// We'll implement this by directly querying the database to avoid circular imports
	var token database.UserOAuthToken
	err := database.DB.Where("user_id = ? AND provider = ?", userID, provider).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// Register the external plugin factory when this package is imported
func init() {
	plugins.RegisterExternalPluginFactory(NewExternalPlugin)
}