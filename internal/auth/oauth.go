package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"golang.org/x/oauth2"
)

// OAuthManager handles dynamic OAuth configurations from plugin registrations
type OAuthManager struct {
	providerConfigs map[string]OAuthProviderConfig
}

// OAuthProviderConfig stores OAuth configuration for a provider
type OAuthProviderConfig struct {
	Provider     string   `json:"provider"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	Scopes       []string `json:"scopes"`
	ClientID     string   `json:"client_id"`     // Actual client ID from Ruby
	ClientSecret string   `json:"client_secret"` // Actual client secret from Ruby
}

var oauthManager *OAuthManager

// InitOAuthManager initializes the OAuth manager
func InitOAuthManager() {
	oauthManager = &OAuthManager{
		providerConfigs: make(map[string]OAuthProviderConfig),
	}
}

// RegisterOAuthProvider registers a new OAuth provider configuration
func RegisterOAuthProvider(config OAuthProviderConfig) {
	if oauthManager == nil {
		InitOAuthManager()
	}
	
	// Check if provider already exists
	if existingConfig, exists := oauthManager.providerConfigs[config.Provider]; exists {
		// Merge scopes - combine existing scopes with new ones, avoiding duplicates
		scopeSet := make(map[string]bool)
		
		// Add existing scopes
		for _, scope := range existingConfig.Scopes {
			scopeSet[scope] = true
		}
		
		// Add new scopes
		for _, scope := range config.Scopes {
			scopeSet[scope] = true
		}
		
		// Convert back to slice
		mergedScopes := make([]string, 0, len(scopeSet))
		for scope := range scopeSet {
			mergedScopes = append(mergedScopes, scope)
		}
		
		// Update config with merged scopes, keeping other fields from new config
		config.Scopes = mergedScopes
		logging.Info("[OAUTH] Merging OAuth provider scopes", "provider", config.Provider, "total_scopes", len(mergedScopes), "scopes", mergedScopes)
	} else {
		logging.Info("[OAUTH] Registering new OAuth provider", "provider", config.Provider, "scopes", config.Scopes)
	}
	
	oauthManager.providerConfigs[config.Provider] = config
	logging.Info("[OAUTH] Registered OAuth provider", "provider", config.Provider, "auth_url", config.AuthURL)
}

// GetOAuthManager returns the global OAuth manager instance
func GetOAuthManager() *OAuthManager {
	return oauthManager
}

// OAuthAuthHandler initiates OAuth flow for a specific provider
func OAuthAuthHandler(c *gin.Context) {
	provider := c.Param("provider")
	
	// Get current user from JWT
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	
	user := userInterface.(*database.User)
	
	// Get provider configuration
	providerConfig, exists := oauthManager.providerConfigs[provider]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("OAuth provider '%s' not supported", provider)})
		return
	}
	
	// Use client credentials provided by Ruby service
	clientID := providerConfig.ClientID
	clientSecret := providerConfig.ClientSecret
	
	if clientID == "" || clientSecret == "" {
		logging.Error("[OAUTH] Missing client credentials", "provider", provider)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth not configured for this provider"})
		return
	}
	
	// Create OAuth2 config
	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  fmt.Sprintf("%s/api/oauth/%s/callback", config.GetBaseURL(), provider),
		Endpoint: oauth2.Endpoint{
			AuthURL:  providerConfig.AuthURL,
			TokenURL: providerConfig.TokenURL,
		},
		Scopes: providerConfig.Scopes,
	}
	
	// Generate state parameter for CSRF protection
	state, err := generateOAuthState(user.ID.String(), provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}
	
	// Store state in session cookie
	secure := !allowInsecure()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("oauth_state", state, 600, "/", "", secure, true) // 10 minute expiry
	
	// Build auth URL with additional parameters for better UX
	authURL := oauth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("access_type", "offline"),    // Get refresh tokens
		oauth2.SetAuthURLParam("prompt", "consent"),         // Force consent screen for refresh token
	)
	
	logging.Info("[OAUTH] Redirecting user to OAuth provider", "provider", provider, "user_id", user.ID)
	c.Redirect(http.StatusFound, authURL)
}

// OAuthCallbackHandler handles OAuth callbacks from providers
func OAuthCallbackHandler(c *gin.Context) {
	provider := c.Param("provider")
	
	// Get provider configuration
	providerConfig, exists := oauthManager.providerConfigs[provider]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("OAuth provider '%s' not supported", provider)})
		return
	}
	
	// Verify state parameter
	state := c.Query("state")
	storedState, err := c.Cookie("oauth_state")
	if err != nil || state != storedState {
		logging.Warn("[OAUTH] Invalid OAuth state", "provider", provider, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}
	
	// Clear state cookie
	secure := !allowInsecure()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("oauth_state", "", -1, "/", "", secure, true)
	
	// Handle error from provider
	if errMsg := c.Query("error"); errMsg != "" {
		logging.Warn("[OAUTH] OAuth provider returned error", "provider", provider, "error", errMsg)
		// Redirect to popup error page
		redirectURL := fmt.Sprintf("/oauth-success.html?provider=%s&error=%s", provider, errMsg)
		c.Redirect(http.StatusFound, redirectURL)
		return
	}
	
	// Exchange code for token
	code := c.Query("code")
	if code == "" {
		// Redirect to popup error page
		redirectURL := fmt.Sprintf("/oauth-success.html?provider=%s&error=Missing authorization code", provider)
		c.Redirect(http.StatusFound, redirectURL)
		return
	}
	
	// Extract user ID from state
	userID, err := extractUserIDFromState(state)
	if err != nil {
		logging.Error("[OAUTH] Failed to extract user ID from state", "error", err)
		// Redirect to popup error page
		redirectURL := fmt.Sprintf("/oauth-success.html?provider=%s&error=Invalid state format", provider)
		c.Redirect(http.StatusFound, redirectURL)
		return
	}
	
	// Use client credentials provided by Ruby service
	clientID := providerConfig.ClientID
	clientSecret := providerConfig.ClientSecret
	
	// Create OAuth2 config for token exchange
	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  fmt.Sprintf("%s/api/oauth/%s/callback", config.GetBaseURL(), provider),
		Endpoint: oauth2.Endpoint{
			AuthURL:  providerConfig.AuthURL,
			TokenURL: providerConfig.TokenURL,
		},
		Scopes: providerConfig.Scopes,
	}
	
	// Exchange authorization code for token
	ctx := context.Background()
	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		logging.Error("[OAUTH] Failed to exchange code for token", "provider", provider, "error", err)
		// Redirect to popup error page
		redirectURL := fmt.Sprintf("/oauth-success.html?provider=%s&error=Failed to exchange code for token", provider)
		c.Redirect(http.StatusFound, redirectURL)
		return
	}
	
	// Store refresh token in database
	err = storeUserOAuthToken(userID, provider, token, providerConfig.Scopes)
	if err != nil {
		logging.Error("[OAUTH] Failed to store OAuth token", "provider", provider, "user_id", userID, "error", err)
		// Redirect to popup error page
		redirectURL := fmt.Sprintf("/oauth-success.html?provider=%s&error=Failed to store OAuth token", provider)
		c.Redirect(http.StatusFound, redirectURL)
		return
	}
	
	logging.Info("[OAUTH] Successfully stored OAuth token", "provider", provider, "user_id", userID)
	
	// Redirect to popup success page
	redirectURL := fmt.Sprintf("/oauth-success.html?provider=%s", provider)
	c.Redirect(http.StatusFound, redirectURL)
}

// generateOAuthState generates a secure state parameter with user context
func generateOAuthState(userID, provider string) (string, error) {
	// Create state data with user ID and provider
	stateData := map[string]string{
		"user_id":  userID,
		"provider": provider,
		"nonce":    generateRandomString(16),
	}
	
	stateJSON, err := json.Marshal(stateData)
	if err != nil {
		return "", err
	}
	
	return base64.URLEncoding.EncodeToString(stateJSON), nil
}

// extractUserIDFromState extracts user ID from OAuth state parameter
func extractUserIDFromState(state string) (string, error) {
	stateData, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return "", err
	}
	
	var stateMap map[string]string
	err = json.Unmarshal(stateData, &stateMap)
	if err != nil {
		return "", err
	}
	
	userID, exists := stateMap["user_id"]
	if !exists {
		return "", fmt.Errorf("user_id not found in state")
	}
	
	return userID, nil
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

// storeUserOAuthToken stores the OAuth refresh token for a user
func storeUserOAuthToken(userID, provider string, token *oauth2.Token, scopes []string) error {
	// Convert scopes to JSON string
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}
	
	// Parse user ID to UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}
	
	// Create or update the user's OAuth token
	oauthToken := &database.UserOAuthToken{
		UserID:       userUUID,
		Provider:     provider,
		ServiceName:  provider, // For now, use provider as service name
		RefreshToken: token.RefreshToken,
		Scopes:       string(scopesJSON),
	}
	
	// Upsert the token (create or update if exists)
	result := database.DB.Where("user_id = ? AND provider = ?", userID, provider).
		Assign(map[string]interface{}{
			"refresh_token": token.RefreshToken,
			"scopes":        string(scopesJSON),
			"updated_at":    time.Now(),
		}).
		FirstOrCreate(oauthToken)
	
	return result.Error
}

// GetUserOAuthToken retrieves a user's OAuth refresh token for a provider
func GetUserOAuthToken(userID, provider string) (*database.UserOAuthToken, error) {
	var token database.UserOAuthToken
	err := database.DB.Where("user_id = ? AND provider = ?", userID, provider).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteUserOAuthToken deletes a user's OAuth token for a provider
func DeleteUserOAuthToken(userID, provider string) error {
	return database.DB.Where("user_id = ? AND provider = ?", userID, provider).Delete(&database.UserOAuthToken{}).Error
}

// OAuthStatusHandler checks if user has connected a specific OAuth provider
func OAuthStatusHandler(c *gin.Context) {
	provider := c.Param("provider")
	
	// Get current user from JWT
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	
	user := userInterface.(*database.User)
	
	// Get user's OAuth token for this provider
	token, err := GetUserOAuthToken(user.ID.String(), provider)
	if err != nil {
		// No connection found
		c.JSON(http.StatusNotFound, gin.H{
			"provider":   provider,
			"connected":  false,
		})
		return
	}
	
	// Parse scopes from JSON
	var scopes []string
	if token.Scopes != "" {
		json.Unmarshal([]byte(token.Scopes), &scopes)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"provider":     provider,
		"connected":    true,
		"connected_at": token.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"scopes":       scopes,
	})
}

// OAuthDisconnectHandler disconnects user from an OAuth provider
func OAuthDisconnectHandler(c *gin.Context) {
	provider := c.Param("provider")
	
	// Get current user from JWT
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	
	user := userInterface.(*database.User)
	
	// Delete user's OAuth token
	err := DeleteUserOAuthToken(user.ID.String(), provider)
	if err != nil {
		logging.Error("[OAUTH] Failed to delete OAuth token", "provider", provider, "user_id", user.ID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect OAuth provider"})
		return
	}
	
	logging.Info("[OAUTH] Successfully disconnected OAuth provider", "provider", provider, "user_id", user.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Successfully disconnected"})
}