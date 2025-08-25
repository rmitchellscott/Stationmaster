package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rmitchellscott/stationmaster/internal/locales"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// LocaleHandler handles locale-related HTTP requests
type LocaleHandler struct {
	localeManager *locales.LocaleManager
}

// NewLocaleHandler creates a new locale handler
func NewLocaleHandler(localeManager *locales.LocaleManager) *LocaleHandler {
	return &LocaleHandler{
		localeManager: localeManager,
	}
}

// GetLocaleData returns translation data for a specific locale
func (h *LocaleHandler) GetLocaleData(c *gin.Context) {
	locale := c.Param("locale")
	if locale == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "locale parameter is required"})
		return
	}

	// Get locale data as JSON
	localeData, err := h.localeManager.GetLocaleJSON(locale)
	if err != nil {
		logging.Warn("[LOCALE_HANDLER] Locale not found", "locale", locale, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "locale not found"})
		return
	}

	// Set appropriate headers
	c.Header("Content-Type", "application/json")
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	
	// Return raw JSON data
	c.Data(http.StatusOK, "application/json", localeData)
}

// GetAvailableLocales returns a list of all available locales
func (h *LocaleHandler) GetAvailableLocales(c *gin.Context) {
	locales := h.localeManager.GetAvailableLocales()
	
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	c.JSON(http.StatusOK, gin.H{
		"locales": locales,
		"count":   len(locales),
	})
}

// GetTranslation returns a specific translation for a locale and key
func (h *LocaleHandler) GetTranslation(c *gin.Context) {
	locale := c.Param("locale")
	key := c.Query("key")
	
	if locale == "" || key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "locale and key parameters are required"})
		return
	}

	translation, found := h.localeManager.GetTranslation(locale, key)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "translation not found"})
		return
	}

	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	c.JSON(http.StatusOK, gin.H{
		"locale":      locale,
		"key":         key,
		"translation": translation,
	})
}

// RegisterLocaleRoutes registers all locale-related routes (public - no auth required)
func RegisterLocaleRoutes(r *gin.Engine, localeManager *locales.LocaleManager) {
	handler := NewLocaleHandler(localeManager)
	
	// Public locale API routes (needed by browserless for template rendering)
	localeGroup := r.Group("/api/locales")
	{
		localeGroup.GET("", handler.GetAvailableLocales)
		localeGroup.GET("/:locale", handler.GetLocaleData)
		localeGroup.GET("/:locale/translate", handler.GetTranslation)
	}
}