package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/validation"
)




// CreatePrivatePluginRequest represents the request to create a private plugin
type CreatePrivatePluginRequest struct {
	Name              string                 `json:"name" binding:"required,min=1,max=255"`
	Description       string                 `json:"description"`
	MarkupFull        string                 `json:"markup_full"`
	MarkupHalfVert    string                 `json:"markup_half_vert"`
	MarkupHalfHoriz   string                 `json:"markup_half_horiz"`
	MarkupQuadrant    string                 `json:"markup_quadrant"`
	SharedMarkup      string                 `json:"shared_markup"`
	DataStrategy      string                 `json:"data_strategy" binding:"required,oneof=webhook polling static"`
	PollingConfig     map[string]interface{} `json:"polling_config"`
	FormFields        map[string]interface{} `json:"form_fields"`
	Version           string                 `json:"version"`
	RemoveBleedMargin bool                   `json:"remove_bleed_margin"`
	EnableDarkMode    bool                   `json:"enable_dark_mode"`
}

// UpdatePrivatePluginRequest represents the request to update a private plugin
type UpdatePrivatePluginRequest struct {
	Name              string                 `json:"name" binding:"required,min=1,max=255"`
	Description       string                 `json:"description"`
	MarkupFull        string                 `json:"markup_full"`
	MarkupHalfVert    string                 `json:"markup_half_vert"`
	MarkupHalfHoriz   string                 `json:"markup_half_horiz"`
	MarkupQuadrant    string                 `json:"markup_quadrant"`
	SharedMarkup      string                 `json:"shared_markup"`
	DataStrategy      string                 `json:"data_strategy" binding:"required,oneof=webhook polling static"`
	PollingConfig     map[string]interface{} `json:"polling_config"`
	FormFields        map[string]interface{} `json:"form_fields"`
	Version           string                 `json:"version"`
	IsPublished       bool                   `json:"is_published"`
	RemoveBleedMargin bool                   `json:"remove_bleed_margin"`
	EnableDarkMode    bool                   `json:"enable_dark_mode"`
}

// PrivatePluginResponse represents the response format for private plugins  
type PrivatePluginResponse struct {
	database.PrivatePlugin
	WebhookToken string `json:"webhook_token"`
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

	// Validate and process polling configuration
	if err := ValidatePollingConfig(req.PollingConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Polling config validation failed", "details": err.Error()})
		return
	}

	// Validate and process form fields configuration
	if _, err := ValidateFormFields(req.FormFields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form fields validation failed", "details": err.Error()})
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
		Name:              req.Name,
		Description:       req.Description,
		MarkupFull:        req.MarkupFull,
		MarkupHalfVert:    req.MarkupHalfVert,
		MarkupHalfHoriz:   req.MarkupHalfHoriz,
		MarkupQuadrant:    req.MarkupQuadrant,
		SharedMarkup:      req.SharedMarkup,
		DataStrategy:      req.DataStrategy,
		PollingConfig:     pollingConfigJSON,
		FormFields:        formFieldsJSON,
		Version:           req.Version,
		RemoveBleedMargin: req.RemoveBleedMargin,
		EnableDarkMode:    req.EnableDarkMode,
		IsPublished:       false, // New plugins are not published by default
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
	
	// Sync to unified plugin system
	unifiedService := database.NewUnifiedPluginService(db)
	if _, err := unifiedService.MigratePrivatePlugin(plugin); err != nil {
		// Log but don't fail the creation - unified system is secondary
		fmt.Printf("Warning: Failed to sync private plugin to unified system: %v\n", err)
	}

	// Return the created plugin with webhook token
	response := PrivatePluginResponse{
		PrivatePlugin: *plugin,
		WebhookToken:  plugin.WebhookToken,
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

	// Convert to response format with webhook tokens
	var responses []PrivatePluginResponse
	for _, plugin := range plugins {
		responses = append(responses, PrivatePluginResponse{
			PrivatePlugin: plugin,
			WebhookToken:  plugin.WebhookToken,
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
		WebhookToken:  plugin.WebhookToken,
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

	// Validate and process polling configuration
	if err := ValidatePollingConfig(req.PollingConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Polling config validation failed", "details": err.Error()})
		return
	}

	// Validate and process form fields configuration
	if _, err := ValidateFormFields(req.FormFields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form fields validation failed", "details": err.Error()})
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
		Name:              req.Name,
		Description:       req.Description,
		MarkupFull:        req.MarkupFull,
		MarkupHalfVert:    req.MarkupHalfVert,
		MarkupHalfHoriz:   req.MarkupHalfHoriz,
		MarkupQuadrant:    req.MarkupQuadrant,
		SharedMarkup:      req.SharedMarkup,
		DataStrategy:      req.DataStrategy,
		PollingConfig:     pollingConfigJSON,
		FormFields:        formFieldsJSON,
		Version:           req.Version,
		IsPublished:       req.IsPublished,
		RemoveBleedMargin: req.RemoveBleedMargin,
		EnableDarkMode:    req.EnableDarkMode,
	}

	db := database.GetDB()
	service := database.NewPrivatePluginService(db)

	if err := service.UpdatePrivatePlugin(id, user.ID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update private plugin", "details": err.Error()})
		return
	}

	// Fetch updated plugin for unified system sync
	plugin, err := service.GetPrivatePluginByID(id, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated plugin"})
		return
	}

	// Sync to unified plugin system
	unifiedService := database.NewUnifiedPluginService(db)
	if _, err := unifiedService.MigratePrivatePlugin(plugin); err != nil {
		// Log but don't fail the update - unified system is secondary
		fmt.Printf("Warning: Failed to sync private plugin update to unified system: %v\n", err)
	}

	response := PrivatePluginResponse{
		PrivatePlugin: *plugin,
		WebhookToken:  plugin.WebhookToken,
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

