package official

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// PluginExecutorRequest represents the request to the Ruby executor
type PluginExecutorRequest struct {
	Plugin   string                 `json:"plugin"`
	Settings map[string]interface{} `json:"settings"`
}

// PluginExecutorResponse represents the response from the Ruby executor
type PluginExecutorResponse struct {
	Success bool                   `json:"success"`
	Plugin  string                 `json:"plugin"`
	Locals  map[string]interface{} `json:"locals"`
	Error   string                 `json:"error"`
}

// OfficialPluginAdapter adapts TRMNL official plugins to the stationmaster plugin interface
type OfficialPluginAdapter struct {
	pluginName        string
	pluginDir         string
	displayName       string
	description       string
	version           string
	configSchema      string
	templates         map[string]string // layout -> liquid template
	sharedMarkup      string            // converted shared markup from partials
	converter         *TemplateConverter
	rubyExecutorPath  string
	requiresProcessing bool
}

// NewOfficialPluginAdapter creates a new adapter for a TRMNL official plugin
func NewOfficialPluginAdapter(pluginName string, options ...func(*OfficialPluginAdapter)) (*OfficialPluginAdapter, error) {
	// Find the project root directory - try multiple possible locations
	projectRoot := ""
	
	// Try environment variable first
	if envRoot := os.Getenv("TRMNL_PLUGINS_ROOT"); envRoot != "" {
		if _, err := os.Stat(filepath.Join(envRoot, "trmnl-plugins")); err == nil {
			projectRoot = envRoot
		}
	}
	
	// Try current working directory and parents
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		
		testRoot := cwd
		for {
			if _, err := os.Stat(filepath.Join(testRoot, "trmnl-plugins")); err == nil {
				projectRoot = testRoot
				break
			}
			parent := filepath.Dir(testRoot)
			if parent == testRoot {
				break
			}
			testRoot = parent
		}
	}
	
	// Try common paths as fallback
	if projectRoot == "" {
		commonPaths := []string{
			"/app",           // Docker container
			"/workspace",     // Some CI environments
			".",             // Current directory
		}
		
		for _, path := range commonPaths {
			if _, err := os.Stat(filepath.Join(path, "trmnl-plugins")); err == nil {
				projectRoot = path
				break
			}
		}
	}
	
	// Final fallback
	if projectRoot == "" {
		cwd, _ := os.Getwd()
		projectRoot = cwd
	}
	
	adapter := &OfficialPluginAdapter{
		pluginName:        pluginName,
		pluginDir:         filepath.Join(projectRoot, "trmnl-plugins", "lib", pluginName),
		displayName:       formatDisplayName(pluginName),
		description:       fmt.Sprintf("Official TRMNL %s plugin", formatDisplayName(pluginName)),
		version:           "1.0.0",
		configSchema:      "{}",
		templates:         make(map[string]string),
		converter:         NewTemplateConverter(os.Getenv("BASE_URL")),
		rubyExecutorPath:  filepath.Join(projectRoot, "scripts", "official_plugin_executor.rb"),
		requiresProcessing: true,
	}
	
	// Apply options
	for _, opt := range options {
		opt(adapter)
	}
	
	// Load templates
	if err := adapter.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}
	
	return adapter, nil
}

// Type returns the plugin type identifier
func (a *OfficialPluginAdapter) Type() string {
	return a.pluginName
}

// PluginType returns that this is an image plugin
func (a *OfficialPluginAdapter) PluginType() plugins.PluginType {
	return plugins.PluginTypeImage
}

// Name returns the human-readable name
func (a *OfficialPluginAdapter) Name() string {
	return a.displayName
}

// Description returns the plugin description
func (a *OfficialPluginAdapter) Description() string {
	return a.description
}

// Author returns the plugin author
func (a *OfficialPluginAdapter) Author() string {
	return "TRMNL"
}

// Version returns the plugin version
func (a *OfficialPluginAdapter) Version() string {
	return a.version
}

// RequiresProcessing returns whether this plugin needs processing
func (a *OfficialPluginAdapter) RequiresProcessing() bool {
	return a.requiresProcessing
}

// ConfigSchema returns the JSON schema for configuration
func (a *OfficialPluginAdapter) ConfigSchema() string {
	return a.configSchema
}

// Validate validates the plugin settings
func (a *OfficialPluginAdapter) Validate(settings map[string]interface{}) error {
	// Basic validation - can be extended per plugin
	return nil
}

// Process executes the plugin logic
func (a *OfficialPluginAdapter) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Validate device model information
	if ctx.Device == nil || ctx.Device.DeviceModel == nil {
		return plugins.CreateErrorResponse("Device model information not available"),
			fmt.Errorf("device model is required for plugin processing")
	}
	
	// Execute Ruby plugin to get locals data
	locals, err := a.executeRubyPlugin(ctx)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to execute plugin: %v", err)),
			fmt.Errorf("failed to execute Ruby plugin: %w", err)
	}
	
	// Determine which template to use based on context (for mashups)
	templateLayout := "full"
	if ctx.GetStringSetting("__layout", "") != "" {
		templateLayout = ctx.GetStringSetting("__layout", "full")
	}
	
	// Get the appropriate template
	template, exists := a.templates[templateLayout]
	if !exists {
		// Fallback to full if specific layout not found
		template = a.templates["full"]
	}
	
	// Prepare template data
	templateData := make(map[string]interface{})
	
	// Add locals from Ruby plugin
	for k, v := range locals {
		templateData[k] = v
	}
	
	// Add instance information
	if ctx.PluginInstance != nil {
		templateData["instance_name"] = ctx.PluginInstance.Name
	}
	
	// Add base_url for image references
	templateData["base_url"] = a.converter.baseURL
	
	// Create TRMNL data structure
	trmnlBuilder := rendering.NewTRNMLDataBuilder()
	trmnlData := trmnlBuilder.BuildTRNMLData(ctx, ctx.PluginInstance, ctx.Settings)
	templateData["trmnl"] = trmnlData
	
	// Render template using Ruby Liquid renderer
	rubyRenderer, err := rendering.NewRubyLiquidRenderer(".")
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to create renderer: %v", err)),
			fmt.Errorf("failed to create Ruby renderer: %w", err)
	}
	
	renderOptions := rendering.PluginRenderOptions{
		SharedMarkup:      a.sharedMarkup,
		LayoutTemplate:    template,
		Data:              templateData,
		Width:             ctx.Device.DeviceModel.ScreenWidth,
		Height:            ctx.Device.DeviceModel.ScreenHeight,
		PluginName:        a.displayName,
		InstanceID:        ctx.PluginInstance.ID.String(),
		InstanceName:      ctx.PluginInstance.Name,
		RemoveBleedMargin: false,
		EnableDarkMode:    false,
	}
	
	html, err := rubyRenderer.RenderToHTML(context.Background(), renderOptions)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render template: %v", err)),
			fmt.Errorf("failed to render template: %w", err)
	}
	
	// Convert HTML to image using browserless
	browserRenderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to create renderer: %v", err)),
			fmt.Errorf("failed to create browserless renderer: %w", err)
	}
	defer browserRenderer.Close()
	
	renderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	imageData, err := browserRenderer.RenderHTML(
		renderCtx,
		html,
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight,
	)
	if err != nil {
		return plugins.CreateErrorResponse(fmt.Sprintf("Failed to render HTML: %v", err)),
			fmt.Errorf("failed to render HTML to image: %w", err)
	}
	
	// Generate filename
	filename := fmt.Sprintf("%s_%s_%dx%d.png",
		a.pluginName,
		time.Now().Format("20060102_150405"),
		ctx.Device.DeviceModel.ScreenWidth,
		ctx.Device.DeviceModel.ScreenHeight)
	
	return plugins.CreateImageDataResponse(imageData, filename), nil
}

// executeRubyPlugin executes the Ruby plugin and returns the locals data
func (a *OfficialPluginAdapter) executeRubyPlugin(ctx plugins.PluginContext) (map[string]interface{}, error) {
	// Prepare request
	request := PluginExecutorRequest{
		Plugin:   a.pluginName,
		Settings: ctx.Settings,
	}
	
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Execute Ruby script
	cmd := exec.Command("ruby", a.rubyExecutorPath)
	cmd.Stdin = bytes.NewReader(requestJSON)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	logging.Debug("[OFFICIAL_PLUGIN] Executing Ruby plugin", "plugin", a.pluginName)
	
	err = cmd.Run()
	if err != nil {
		stderrStr := stderr.String()
		return nil, fmt.Errorf("Ruby execution failed: %w, stderr: %s", err, stderrStr)
	}
	
	// Parse response
	var response PluginExecutorResponse
	err = json.Unmarshal(stdout.Bytes(), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, stdout: %s", err, stdout.String())
	}
	
	if !response.Success {
		return nil, fmt.Errorf("plugin execution failed: %s", response.Error)
	}
	
	logging.Debug("[OFFICIAL_PLUGIN] Plugin executed successfully", "plugin", a.pluginName, "locals_keys", len(response.Locals))
	
	return response.Locals, nil
}

// loadTemplates loads and converts templates from ERB to Liquid
func (a *OfficialPluginAdapter) loadTemplates() error {
	viewsDir := filepath.Join(a.pluginDir, "views")
	
	// Template layouts to load
	layouts := []string{"full", "half_horizontal", "half_vertical", "quadrant"}
	
	for _, layout := range layouts {
		templatePath := filepath.Join(viewsDir, fmt.Sprintf("%s.html.erb", layout))
		
		// Check if template exists
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			logging.Debug("[OFFICIAL_PLUGIN] Template not found", "plugin", a.pluginName, "layout", layout)
			continue
		}
		
		// Read template content
		content, err := os.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", layout, err)
		}
		
		// Convert ERB to Liquid
		liquidTemplate := a.converter.ConvertERBToLiquid(string(content))
		
		// Store converted template
		a.templates[layout] = liquidTemplate
		
		logging.Debug("[OFFICIAL_PLUGIN] Loaded template", "plugin", a.pluginName, "layout", layout)
	}
	
	// Load partials if they exist
	if err := a.loadPartials(viewsDir); err != nil {
		logging.Warn("[OFFICIAL_PLUGIN] Failed to load partials", "plugin", a.pluginName, "error", err)
	}
	
	if len(a.templates) == 0 {
		// For some plugins like Mondrian, we might not have templates but still want to proceed
		logging.Warn("[OFFICIAL_PLUGIN] No templates found for plugin", "plugin", a.pluginName, "plugin_dir", a.pluginDir)
		
		// Create a minimal default template for plugins that rely purely on JavaScript/CSS
		defaultTemplate := fmt.Sprintf(`<div class="view view--full view--%s">
  <div class="%s-container"></div>
</div>`, a.pluginName, a.pluginName)
		
		a.templates["full"] = defaultTemplate
		
		// You can add more default layouts if needed
		for _, layout := range []string{"half_horizontal", "half_vertical", "quadrant"} {
			a.templates[layout] = strings.ReplaceAll(defaultTemplate, "view--full", fmt.Sprintf("view--%s", layout))
		}
	}
	
	return nil
}

// loadPartials loads partial templates (files starting with _) as shared markup
func (a *OfficialPluginAdapter) loadPartials(viewsDir string) error {
	files, err := os.ReadDir(viewsDir)
	if err != nil {
		return err
	}
	
	var sharedMarkupParts []string
	
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "_") {
			continue
		}
		
		// Remove _ prefix and .html.erb extension for partial name
		partialName := strings.TrimPrefix(file.Name(), "_")
		partialName = strings.TrimSuffix(partialName, ".html.erb")
		
		// Read partial content
		content, err := os.ReadFile(filepath.Join(viewsDir, file.Name()))
		if err != nil {
			logging.Warn("[OFFICIAL_PLUGIN] Failed to read partial", "plugin", a.pluginName, "partial", partialName, "error", err)
			continue
		}
		
		// Convert ERB to Liquid and add to shared markup
		liquidPartial := a.converter.ConvertERBToLiquid(string(content))
		sharedMarkupParts = append(sharedMarkupParts, liquidPartial)
		
		logging.Debug("[OFFICIAL_PLUGIN] Loaded partial as shared markup", "plugin", a.pluginName, "partial", partialName)
	}
	
	// Combine all partials into shared markup
	if len(sharedMarkupParts) > 0 {
		a.sharedMarkup = strings.Join(sharedMarkupParts, "\n\n")
		logging.Debug("[OFFICIAL_PLUGIN] Combined partials into shared markup", "plugin", a.pluginName, "shared_markup_length", len(a.sharedMarkup))
	}
	
	return nil
}

// formatDisplayName converts snake_case to Title Case
func formatDisplayName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	return strings.Join(parts, " ")
}

// Option functions for customizing the adapter

// WithDescription sets a custom description
func WithDescription(desc string) func(*OfficialPluginAdapter) {
	return func(a *OfficialPluginAdapter) {
		a.description = desc
	}
}

// WithConfigSchema sets a custom config schema
func WithConfigSchema(schema string) func(*OfficialPluginAdapter) {
	return func(a *OfficialPluginAdapter) {
		a.configSchema = schema
	}
}

// WithVersion sets a custom version
func WithVersion(version string) func(*OfficialPluginAdapter) {
	return func(a *OfficialPluginAdapter) {
		a.version = version
	}
}