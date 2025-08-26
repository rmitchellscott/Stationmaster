package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"gorm.io/gorm"
)

// WebhookRateLimiter implements rate limiting for webhook endpoints
type WebhookRateLimiter struct {
	db *gorm.DB
	// In-memory tracking for performance (could be replaced with Redis in production)
	userLimits map[string]*UserRateLimit
	mutex      sync.RWMutex
}

type UserRateLimit struct {
	Count      int
	WindowStart time.Time
	WindowSize time.Duration
}

// NewWebhookRateLimiter creates a new webhook rate limiter
func NewWebhookRateLimiter(db *gorm.DB) *WebhookRateLimiter {
	limiter := &WebhookRateLimiter{
		db:         db,
		userLimits: make(map[string]*UserRateLimit),
	}
	
	// Clean up expired entries every hour
	go limiter.cleanupRoutine()
	
	return limiter
}

// RateLimit is a middleware that enforces webhook rate limits per user
func (wrl *WebhookRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get instance ID from URL parameter
		instanceID := c.Param("id")
		if instanceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Instance ID is required"})
			c.Abort()
			return
		}

		// Find the plugin instance by ID to get the user ID
		var pluginInstance database.PluginInstance
		if err := wrl.db.Where("id = ? AND is_active = ?", instanceID, true).First(&pluginInstance).Error; err != nil {
			logging.Warn("[WEBHOOK] Invalid instance ID during rate limiting", "instance_id", instanceID, "ip", c.ClientIP())
			c.JSON(http.StatusNotFound, gin.H{"error": "Invalid instance ID"})
			c.Abort()
			return
		}

		// Get rate limit setting from database
		rateLimit, err := wrl.getRateLimit()
		if err != nil {
			logging.Error("[WEBHOOK] Failed to get rate limit setting", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rate limit"})
			c.Abort()
			return
		}

		// Check rate limit
		userKey := pluginInstance.UserID.String()
		if !wrl.allowRequest(userKey, rateLimit) {
			logging.Warn("[WEBHOOK] Rate limit exceeded", "user_id", pluginInstance.UserID, "plugin_instance_id", pluginInstance.ID, "ip", c.ClientIP())
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":           "Rate limit exceeded",
				"rate_limit":      rateLimit,
				"window_duration": "1 hour",
			})
			c.Abort()
			return
		}

		// Store user ID and plugin instance in context for use by handler
		c.Set("user_id", pluginInstance.UserID)
		c.Set("plugin_instance", &pluginInstance)
		c.Next()
	}
}

// RequestSizeLimit middleware enforces configurable request size limits
func (wrl *WebhookRateLimiter) RequestSizeLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get max request size from database settings
		maxSizeKB, err := wrl.getMaxRequestSizeKB()
		if err != nil {
			logging.Error("[WEBHOOK] Failed to get max request size setting", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check request size limit"})
			c.Abort()
			return
		}

		maxSizeBytes := int64(maxSizeKB * 1024)

		// Check Content-Length header
		if c.Request.ContentLength > maxSizeBytes {
			logging.Warn("[WEBHOOK] Request too large", "size", c.Request.ContentLength, "limit", maxSizeBytes, "ip", c.ClientIP())
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":     "Request payload too large",
				"max_size":  fmt.Sprintf("%dKB", maxSizeKB),
				"your_size": fmt.Sprintf("%dB", c.Request.ContentLength),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// allowRequest checks if a request should be allowed based on rate limiting
func (wrl *WebhookRateLimiter) allowRequest(userKey string, rateLimit int) bool {
	wrl.mutex.Lock()
	defer wrl.mutex.Unlock()

	now := time.Now()
	windowSize := time.Hour

	userLimit, exists := wrl.userLimits[userKey]
	if !exists || now.Sub(userLimit.WindowStart) >= windowSize {
		// New window or user
		wrl.userLimits[userKey] = &UserRateLimit{
			Count:       1,
			WindowStart: now,
			WindowSize:  windowSize,
		}
		return true
	}

	if userLimit.Count >= rateLimit {
		return false
	}

	userLimit.Count++
	return true
}

// getRateLimit fetches the rate limit setting from the database
func (wrl *WebhookRateLimiter) getRateLimit() (int, error) {
	var setting database.SystemSetting
	if err := wrl.db.Where("key = ?", "webhook_rate_limit_per_hour").First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 30, nil // Default value
		}
		return 0, err
	}

	rateLimit, err := strconv.Atoi(setting.Value)
	if err != nil {
		return 30, nil // Default value on parsing error
	}

	return rateLimit, nil
}

// getMaxRequestSizeKB fetches the max request size setting from the database
func (wrl *WebhookRateLimiter) getMaxRequestSizeKB() (int, error) {
	var setting database.SystemSetting
	if err := wrl.db.Where("key = ?", "webhook_max_request_size_kb").First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 5, nil // Default value
		}
		return 0, err
	}

	maxSize, err := strconv.Atoi(setting.Value)
	if err != nil {
		return 5, nil // Default value on parsing error
	}

	return maxSize, nil
}

// cleanupRoutine removes expired rate limit entries
func (wrl *WebhookRateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		wrl.cleanup()
	}
}

// cleanup removes expired entries from the rate limit cache
func (wrl *WebhookRateLimiter) cleanup() {
	wrl.mutex.Lock()
	defer wrl.mutex.Unlock()

	now := time.Now()
	for userKey, userLimit := range wrl.userLimits {
		if now.Sub(userLimit.WindowStart) >= userLimit.WindowSize {
			delete(wrl.userLimits, userKey)
		}
	}
}