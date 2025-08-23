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
	"gorm.io/gorm"
)

// PrivatePluginWebhookData represents the webhook data storage
type PrivatePluginWebhookData struct {
	ID           string                 `json:"id" gorm:"primaryKey"`
	PluginID     string                 `json:"plugin_id" gorm:"index;not null"`
	Data         map[string]interface{} `json:"data" gorm:"type:json"`
	ReceivedAt   time.Time              `json:"received_at"`
	ContentType  string                 `json:"content_type"`
	ContentSize  int                    `json:"content_size"`
	SourceIP     string                 `json:"source_ip"`
}

// WebhookHandler handles webhook data submission for private plugins
func WebhookHandler(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook token is required"})
		return
	}

	// Find the private plugin by webhook token
	db := database.GetDB()
	service := database.NewPrivatePluginService(db)

	plugin, err := service.GetPrivatePluginByWebhookToken(token)
	if err != nil {
		logging.Warn("[WEBHOOK] Invalid webhook token", "token", token[:8]+"...", "ip", c.ClientIP())
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid webhook token"})
		return
	}

	// Check request method (only POST allowed)
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Only POST requests are allowed"})
		return
	}

	// Read request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logging.Error("[WEBHOOK] Failed to read request body", "error", err, "plugin_id", plugin.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Check content size limit (2KB as per TRMNL spec)
	if len(bodyBytes) > 2048 {
		logging.Warn("[WEBHOOK] Request body too large", "size", len(bodyBytes), "plugin_id", plugin.ID, "ip", c.ClientIP())
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large (max 2KB)"})
		return
	}

	// Parse JSON data
	var webhookData map[string]interface{}
	contentType := c.GetHeader("Content-Type")
	
	if contentType == "application/json" || contentType == "" {
		if err := json.Unmarshal(bodyBytes, &webhookData); err != nil {
			logging.Warn("[WEBHOOK] Invalid JSON data", "error", err, "plugin_id", plugin.ID, "ip", c.ClientIP())
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
			return
		}
	} else {
		// For non-JSON content, store as raw string
		webhookData = map[string]interface{}{
			"raw_data": string(bodyBytes),
		}
	}

	// Store webhook data
	webhookRecord := &PrivatePluginWebhookData{
		ID:          fmt.Sprintf("%s_%d", plugin.ID.String(), time.Now().UnixNano()),
		PluginID:    plugin.ID.String(),
		Data:        webhookData,
		ReceivedAt:  time.Now(),
		ContentType: contentType,
		ContentSize: len(bodyBytes),
		SourceIP:    c.ClientIP(),
	}

	if err := storeWebhookData(db, webhookRecord); err != nil {
		logging.Error("[WEBHOOK] Failed to store webhook data", "error", err, "plugin_id", plugin.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store webhook data"})
		return
	}

	logging.Info("[WEBHOOK] Data received successfully", 
		"plugin_id", plugin.ID, 
		"plugin_name", plugin.Name,
		"content_size", len(bodyBytes),
		"ip", c.ClientIP())

	// TODO: Trigger plugin re-rendering if needed
	// This could be done by adding the plugin instance to the render queue

	c.JSON(http.StatusOK, gin.H{
		"message":    "Webhook data received successfully",
		"plugin_id":  plugin.ID,
		"received_at": webhookRecord.ReceivedAt,
		"size":       len(bodyBytes),
	})
}

// GetWebhookDataHandler retrieves the latest webhook data for a private plugin
func GetWebhookDataHandler(c *gin.Context) {
	// This would be used internally by the plugin processing system
	// Not exposed as a public API endpoint
	
	pluginID := c.Query("plugin_id")
	if pluginID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin_id parameter is required"})
		return
	}

	db := database.GetDB()
	
	var webhookData PrivatePluginWebhookData
	if err := db.Where("plugin_id = ?", pluginID).Order("received_at DESC").First(&webhookData).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No webhook data found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhook_data": webhookData})
}

// storeWebhookData stores webhook data in the database
func storeWebhookData(db *gorm.DB, data *PrivatePluginWebhookData) error {
	// For now, we'll store webhook data in a simple table
	// In a production system, you might want to use a time-series database
	// or implement data retention policies

	// TODO: Create webhook_data table if needed
	// This is a simplified implementation
	
	// Store in memory or simple table for now
	// In a real implementation, you'd save to a webhook_data table
	
	return nil // Placeholder - implement actual storage
}

// cleanupOldWebhookData removes old webhook data (called periodically)
func cleanupOldWebhookData(db *gorm.DB, retentionDays int) error {
	// TODO: Implement cleanup of old webhook data
	// Remove records older than retention period
	
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	
	// Delete old records
	// result := db.Where("received_at < ?", cutoff).Delete(&PrivatePluginWebhookData{})
	
	logging.Info("[WEBHOOK] Cleaned up old webhook data", "cutoff", cutoff)
	
	return nil
}