package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// WebhookHandler handles webhook data submission for private plugin instances
// Rate limiting and request size limiting should be applied via middleware before calling this handler
func WebhookHandler(c *gin.Context) {
	// Get plugin instance from context (set by rate limiting middleware)
	pluginInstanceInterface, exists := c.Get("plugin_instance")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Plugin instance context not found"})
		return
	}
	pluginInstance := pluginInstanceInterface.(*database.PluginInstance)

	// Check request method (only POST allowed)
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Only POST requests are allowed"})
		return
	}

	// Read request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logging.Error("[WEBHOOK] Failed to read request body", "error", err, "plugin_instance_id", pluginInstance.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Parse JSON data
	var webhookPayload map[string]interface{}
	contentType := c.GetHeader("Content-Type")
	
	if contentType == "application/json" || contentType == "" {
		if err := json.Unmarshal(bodyBytes, &webhookPayload); err != nil {
			logging.Warn("[WEBHOOK] Invalid JSON data", "error", err, "plugin_instance_id", pluginInstance.ID, "ip", c.ClientIP())
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
			return
		}
	} else {
		// For non-JSON content, wrap in merge_variables
		webhookPayload = map[string]interface{}{
			"merge_variables": map[string]interface{}{
				"raw_data": string(bodyBytes),
			},
		}
	}

	// Validate merge_variables exists
	if _, ok := webhookPayload["merge_variables"]; !ok {
		logging.Warn("[WEBHOOK] Missing merge_variables in payload", "plugin_instance_id", pluginInstance.ID, "ip", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook payload must contain merge_variables object"})
		return
	}

	// Determine merge strategy
	mergeStrategy := "default"
	if strategy, ok := webhookPayload["merge_strategy"].(string); ok {
		mergeStrategy = strategy
	}

	// Validate merge strategy
	validStrategies := map[string]bool{
		"default":    true,
		"":           true, // Empty means default
		"deep_merge": true,
		"stream":     true,
	}
	if !validStrategies[mergeStrategy] {
		logging.Warn("[WEBHOOK] Invalid merge strategy", "strategy", mergeStrategy, "plugin_instance_id", pluginInstance.ID, "ip", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid merge strategy: %s. Valid options: default, deep_merge, stream", mergeStrategy)})
		return
	}

	// Create raw data JSON
	rawDataJSON, err := json.Marshal(webhookPayload)
	if err != nil {
		logging.Error("[WEBHOOK] Failed to marshal raw data", "error", err, "plugin_instance_id", pluginInstance.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process webhook data"})
		return
	}

	// Store webhook data using webhook service
	db := database.GetDB()
	webhookService := database.NewWebhookService(db)

	webhookRecord := &database.PrivatePluginWebhookData{
		ID:               pluginInstance.ID.String() + "_webhook_data",
		PluginInstanceID: pluginInstance.ID.String(),
		RawData:          rawDataJSON,
		MergeStrategy:    mergeStrategy,
		ReceivedAt:       time.Now().UTC(),
		ContentType:      contentType,
		ContentSize:      len(bodyBytes),
		SourceIP:         c.ClientIP(),
	}

	if err := webhookService.StoreWebhookData(webhookRecord); err != nil {
		logging.Error("[WEBHOOK] Failed to store webhook data", "error", err, "plugin_instance_id", pluginInstance.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store webhook data"})
		return
	}

	logging.Info("[WEBHOOK] Data received and processed successfully", 
		"plugin_instance_id", pluginInstance.ID, 
		"plugin_instance_name", pluginInstance.Name,
		"merge_strategy", mergeStrategy,
		"content_size", len(bodyBytes),
		"ip", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"message":            "Webhook data received successfully",
		"plugin_instance_id": pluginInstance.ID,
		"merge_strategy":     mergeStrategy,
		"received_at":        webhookRecord.ReceivedAt,
		"size":               len(bodyBytes),
	})
}

// GetWebhookDataHandler retrieves the latest webhook data for a plugin instance (internal use)
func GetWebhookDataHandler(c *gin.Context) {
	pluginInstanceID := c.Query("plugin_instance_id")
	if pluginInstanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin_instance_id parameter is required"})
		return
	}

	db := database.GetDB()
	webhookService := database.NewWebhookService(db)
	
	webhookData, err := webhookService.GetLatestWebhookData(pluginInstanceID)
	if err != nil {
		logging.Error("[WEBHOOK] Failed to get webhook data", "error", err, "plugin_instance_id", pluginInstanceID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve webhook data"})
		return
	}

	if webhookData == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No webhook data found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhook_data": webhookData})
}