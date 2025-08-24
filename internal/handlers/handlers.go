package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
)

// RenderSchedulerFunc is a function type for scheduling renders
type RenderSchedulerFunc func([]uuid.UUID)

// Global render scheduler function - set by main package
var renderScheduler RenderSchedulerFunc

// SetRenderScheduler sets the global render scheduler function
func SetRenderScheduler(scheduler RenderSchedulerFunc) {
	renderScheduler = scheduler
}

// ScheduleRenderForInstances schedules renders for plugin instances if scheduler is available
func ScheduleRenderForInstances(instanceIDs []uuid.UUID) {
	if renderScheduler != nil {
		renderScheduler(instanceIDs)
	}
}

// ConfigHandler returns application configuration for the frontend
func ConfigHandler(c *gin.Context) {
	oidcEnabled := auth.IsOIDCEnabled()
	oidcSsoOnly := auth.IsOIDCSsoOnlyEnabled()
	oidcButtonText := ""
	if oidcEnabled {
		oidcButtonText = auth.GetOIDCButtonText()
	}

	c.JSON(http.StatusOK, gin.H{
		"authEnabled":      true, // Stationmaster always requires authentication
		"multiUserMode":    true, // Stationmaster always operates in multi-user mode
		"oidcEnabled":      oidcEnabled,
		"oidcSsoOnly":      oidcSsoOnly,
		"oidcButtonText":   oidcButtonText,
		"proxyAuthEnabled": config.Get("PROXY_AUTH_ENABLED", "false") == "true",
		"oidcGroupBasedAdmin": auth.IsOIDCGroupBasedAdminEnabled(),
	})
}

// DashboardHandler returns basic dashboard data
func DashboardHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Welcome to Stationmaster",
		"userID":   user.ID.String(),
		"username": user.Username,
	})
}

// CompleteOnboardingHandler marks the user's onboarding as complete
func CompleteOnboardingHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}

	userService := database.NewUserService(database.DB)
	if err := userService.CompleteOnboarding(user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to complete onboarding",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Onboarding completed successfully",
	})
}
