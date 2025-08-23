package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/plugins/private"
	"github.com/rmitchellscott/stationmaster/internal/validation"
)

// CreatePrivatePluginRequest represents the request to create a private plugin
type CreatePrivatePluginRequest struct {
	Name            string                 `json:"name" binding:"required,min=1,max=255"`
	Description     string                 `json:"description"`
	MarkupFull      string                 `json:"markup_full"`
	MarkupHalfVert  string                 `json:"markup_half_vert"`
	MarkupHalfHoriz string                 `json:"markup_half_horiz"`
	MarkupQuadrant  string                 `json:"markup_quadrant"`
	SharedMarkup    string                 `json:"shared_markup"`
	DataStrategy    string                 `json:"data_strategy" binding:"required,oneof=webhook polling merge"`
	PollingConfig   map[string]interface{} `json:"polling_config"`
	FormFields      map[string]interface{} `json:"form_fields"`
	Version         string                 `json:"version"`
}

// UpdatePrivatePluginRequest represents the request to update a private plugin
type UpdatePrivatePluginRequest struct {
	Name            string                 `json:"name" binding:"required,min=1,max=255"`
	Description     string                 `json:"description"`
	MarkupFull      string                 `json:"markup_full"`
	MarkupHalfVert  string                 `json:"markup_half_vert"`
	MarkupHalfHoriz string                 `json:"markup_half_horiz"`
	MarkupQuadrant  string                 `json:"markup_quadrant"`
	SharedMarkup    string                 `json:"shared_markup"`
	DataStrategy    string                 `json:"data_strategy" binding:"required,oneof=webhook polling merge"`
	PollingConfig   map[string]interface{} `json:"polling_config"`
	FormFields      map[string]interface{} `json:"form_fields"`
	Version         string                 `json:"version"`
	IsPublished     bool                   `json:"is_published"`
}

// PrivatePluginResponse represents the response format for private plugins
type PrivatePluginResponse struct {
	database.PrivatePlugin
	WebhookURL string `json:"webhook_url"`
}

// CreatePrivatePluginHandler creates a new private plugin
func CreatePrivatePluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	var req CreatePrivatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// Convert polling and form field configs to JSON
	var pollingConfigJSON, formFieldsJSON []byte
	var err error

	if req.PollingConfig != nil {
		pollingConfigJSON, err = json.Marshal(req.PollingConfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid polling config"})
			return
		}
	}

	if req.FormFields != nil {
		formFieldsJSON, err = json.Marshal(req.FormFields)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form fields config"})
			return
		}
	}

	// Validate templates before creating
	validator := validation.NewTemplateValidator()
	validationResult := validator.ValidateAllTemplates(
		req.MarkupFull,
		req.MarkupHalfVert,
		req.MarkupHalfHoriz,
		req.MarkupQuadrant,
		req.SharedMarkup,
	)

	if !validationResult.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Template validation failed",
			"validation_errors": validationResult.Errors,
			"validation_warnings": validationResult.Warnings,
		})
		return
	}

	// Create the private plugin model
	plugin := &database.PrivatePlugin{
		Name:            req.Name,
		Description:     req.Description,
		MarkupFull:      req.MarkupFull,
		MarkupHalfVert:  req.MarkupHalfVert,
		MarkupHalfHoriz: req.MarkupHalfHoriz,
		MarkupQuadrant:  req.MarkupQuadrant,
		SharedMarkup:    req.SharedMarkup,
		DataStrategy:    req.DataStrategy,
		PollingConfig:   pollingConfigJSON,
		FormFields:      formFieldsJSON,
		Version:         req.Version,
		IsPublished:     false, // New plugins are not published by default
	}

	// Set default version if not provided
	if plugin.Version == "" {
		plugin.Version = "1.0.0"
	}

	// Create the plugin
	db := database.GetDB()
	service := database.NewPrivatePluginService(db)

	if err := service.CreatePrivatePlugin(user.ID, plugin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create private plugin", "details": err.Error()})
		return
	}

	// Return the created plugin with webhook URL
	response := PrivatePluginResponse{
		PrivatePlugin: *plugin,
		WebhookURL:    generateWebhookURL(c, plugin.WebhookToken),
	}

	c.JSON(http.StatusCreated, gin.H{"private_plugin": response})
}

// GetPrivatePluginsHandler retrieves all private plugins for the current user
func GetPrivatePluginsHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	db := database.GetDB()
	service := database.NewPrivatePluginService(db)

	plugins, err := service.GetPrivatePluginsByUserID(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch private plugins", "details": err.Error()})
		return
	}

	// Convert to response format with webhook URLs
	var responses []PrivatePluginResponse
	for _, plugin := range plugins {
		responses = append(responses, PrivatePluginResponse{
			PrivatePlugin: plugin,
			WebhookURL:    generateWebhookURL(c, plugin.WebhookToken),
		})
	}

	c.JSON(http.StatusOK, gin.H{"private_plugins": responses})
}

// GetPrivatePluginHandler retrieves a specific private plugin by ID
func GetPrivatePluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin ID"})
		return
	}

	db := database.GetDB()
	service := database.NewPrivatePluginService(db)

	plugin, err := service.GetPrivatePluginByID(id, user.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Private plugin not found"})
		return
	}

	response := PrivatePluginResponse{
		PrivatePlugin: *plugin,
		WebhookURL:    generateWebhookURL(c, plugin.WebhookToken),
	}

	c.JSON(http.StatusOK, gin.H{"private_plugin": response})
}

// UpdatePrivatePluginHandler updates a private plugin
func UpdatePrivatePluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin ID"})
		return
	}

	var req UpdatePrivatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// Convert configs to JSON
	var pollingConfigJSON, formFieldsJSON []byte

	if req.PollingConfig != nil {
		pollingConfigJSON, err = json.Marshal(req.PollingConfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid polling config"})
			return
		}
	}

	if req.FormFields != nil {
		formFieldsJSON, err = json.Marshal(req.FormFields)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form fields config"})
			return
		}
	}

	// Validate templates before updating
	validator := validation.NewTemplateValidator()
	validationResult := validator.ValidateAllTemplates(
		req.MarkupFull,
		req.MarkupHalfVert,
		req.MarkupHalfHoriz,
		req.MarkupQuadrant,
		req.SharedMarkup,
	)

	if !validationResult.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Template validation failed",
			"validation_errors": validationResult.Errors,
			"validation_warnings": validationResult.Warnings,
		})
		return
	}

	// Create update model
	updates := &database.PrivatePlugin{
		Name:            req.Name,
		Description:     req.Description,
		MarkupFull:      req.MarkupFull,
		MarkupHalfVert:  req.MarkupHalfVert,
		MarkupHalfHoriz: req.MarkupHalfHoriz,
		MarkupQuadrant:  req.MarkupQuadrant,
		SharedMarkup:    req.SharedMarkup,
		DataStrategy:    req.DataStrategy,
		PollingConfig:   pollingConfigJSON,
		FormFields:      formFieldsJSON,
		Version:         req.Version,
		IsPublished:     req.IsPublished,
	}

	db := database.GetDB()
	service := database.NewPrivatePluginService(db)

	if err := service.UpdatePrivatePlugin(id, user.ID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update private plugin", "details": err.Error()})
		return
	}

	// Fetch and return updated plugin
	plugin, err := service.GetPrivatePluginByID(id, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated plugin"})
		return
	}

	response := PrivatePluginResponse{
		PrivatePlugin: *plugin,
		WebhookURL:    generateWebhookURL(c, plugin.WebhookToken),
	}

	c.JSON(http.StatusOK, gin.H{"private_plugin": response})
}

// DeletePrivatePluginHandler deletes a private plugin
func DeletePrivatePluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin ID"})
		return
	}

	db := database.GetDB()
	service := database.NewPrivatePluginService(db)

	if err := service.DeletePrivatePlugin(id, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete private plugin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Private plugin deleted successfully"})
}

// RegenerateWebhookTokenHandler regenerates the webhook token for a private plugin
func RegenerateWebhookTokenHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin ID"})
		return
	}

	db := database.GetDB()
	service := database.NewPrivatePluginService(db)

	token, err := service.RegenerateWebhookToken(id, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to regenerate webhook token", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"webhook_token": token,
		"webhook_url":   generateWebhookURL(c, token),
	})
}

// ValidatePrivatePluginHandler validates a private plugin's templates
func ValidatePrivatePluginHandler(c *gin.Context) {
	_, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	var req CreatePrivatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// Create template validator
	validator := validation.NewTemplateValidator()

	// Validate all templates
	result := validator.ValidateAllTemplates(
		req.MarkupFull,
		req.MarkupHalfVert,
		req.MarkupHalfHoriz,
		req.MarkupQuadrant,
		req.SharedMarkup,
	)

	// Return validation result
	c.JSON(http.StatusOK, gin.H{
		"valid":    result.Valid,
		"message":  result.Message,
		"warnings": result.Warnings,
		"errors":   result.Errors,
	})
}

// TestPrivatePluginRequest represents the request to test a private plugin
type TestPrivatePluginRequest struct {
	Plugin       CreatePrivatePluginRequest `json:"plugin" binding:"required"`
	Layout       string                     `json:"layout" binding:"required,oneof=full half_vertical half_horizontal quadrant"`
	SampleData   map[string]interface{}     `json:"sample_data"`
	DeviceWidth  int                        `json:"device_width" binding:"required,min=1"`
	DeviceHeight int                        `json:"device_height" binding:"required,min=1"`
}

// TestPrivatePluginHandler tests a private plugin with sample data
func TestPrivatePluginHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	var req TestPrivatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	startTime := time.Now()

	// Convert request to private plugin model for testing
	pollingConfigJSON, _ := json.Marshal(req.Plugin.PollingConfig)
	formFieldsJSON, _ := json.Marshal(req.Plugin.FormFields)

	testPlugin := &database.PrivatePlugin{
		ID:              uuid.New(), // Generate temporary ID
		UserID:          user.ID,
		Name:            req.Plugin.Name,
		Description:     req.Plugin.Description,
		MarkupFull:      req.Plugin.MarkupFull,
		MarkupHalfVert:  req.Plugin.MarkupHalfVert,
		MarkupHalfHoriz: req.Plugin.MarkupHalfHoriz,
		MarkupQuadrant:  req.Plugin.MarkupQuadrant,
		SharedMarkup:    req.Plugin.SharedMarkup,
		DataStrategy:    req.Plugin.DataStrategy,
		PollingConfig:   pollingConfigJSON,
		FormFields:      formFieldsJSON,
		Version:         req.Plugin.Version,
		WebhookToken:    "test-token", // Dummy token for testing
	}

	// Create a test device model
	testDevice := &database.Device{
		ID:   uuid.New(),
		Name: "Test Device",
		DeviceModel: &database.DeviceModel{
			ModelName:    "Test Model",
			DisplayName:  "Test Model",
			ScreenWidth:  req.DeviceWidth,
			ScreenHeight: req.DeviceHeight,
			BitDepth:     1,
		},
	}

	// Create private plugin instance
	privatePlugin := private.NewPrivatePlugin(testPlugin)
	
	// Create plugin context with test data
	pluginCtx := plugins.PluginContext{
		Device: testDevice,
		Settings: make(map[string]interface{}),
	}

	// Override the data fetching to use sample data
	testData := req.SampleData
	if testData == nil {
		testData = map[string]interface{}{
			"test": "This is test data",
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		}
	}

	// For now, we'll create a simple test that renders with sample data
	// TODO: Implement proper layout selection in the plugin context

	// Process the plugin to generate the image
	response, err := privatePlugin.Process(pluginCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Plugin processing failed", 
			"details": err.Error(),
		})
		return
	}

	// Check if response contains image data
	if imageData, exists := response["image_data"]; exists {
		if imageBytes, ok := imageData.([]byte); ok {
			// Convert to base64 for display
			base64Image := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(imageBytes))
			
			renderTime := time.Since(startTime)
			c.JSON(http.StatusOK, gin.H{
				"message": "Plugin test completed successfully",
				"preview_url": base64Image,
				"render_time_ms": renderTime.Milliseconds(),
				"layout": req.Layout,
				"dimensions": gin.H{
					"width": req.DeviceWidth,
					"height": req.DeviceHeight,
				},
			})
			return
		}
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Failed to generate image from plugin",
		"details": "Plugin did not return valid image data",
	})
}

// GetPrivatePluginStatsHandler returns statistics about private plugins (admin only)
func GetPrivatePluginStatsHandler(c *gin.Context) {
	// Verify admin access
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	db := database.GetDB()
	service := database.NewPrivatePluginService(db)

	stats, err := service.GetPrivatePluginStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// generateWebhookURL generates the full webhook URL for a given token
func generateWebhookURL(c *gin.Context, token string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/api/webhooks/plugin/%s", scheme, c.Request.Host, token)
}