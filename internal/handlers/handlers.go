package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/config"
)

// ConfigHandler returns application configuration for the frontend
func ConfigHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"authEnabled":         true, // Stationmaster always requires authentication
		"multiUserMode":       true, // Stationmaster always operates in multi-user mode
		"registrationEnabled": config.Get("PUBLIC_REGISTRATION_ENABLED", "false") == "true",
		"oidcEnabled":        config.Get("OIDC_ENABLED", "false") == "true",
		"proxyAuthEnabled":   config.Get("PROXY_AUTH_ENABLED", "false") == "true",
	})
}

// DashboardHandler returns basic dashboard data
func DashboardHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to Stationmaster",
		"userID":  user.ID.String(),
		"username": user.Username,
	})
}