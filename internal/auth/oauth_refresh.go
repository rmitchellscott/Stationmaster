package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"golang.org/x/oauth2"
)

// Providers do not all return an expiry, and UserOAuthToken has no expiry column,
// so a conservative lifetime is assumed. Google access tokens last about an hour.
const accessTokenLifetime = 50 * time.Minute

type cachedAccessToken struct {
	token     string
	expiresAt time.Time
}

var (
	accessTokenCache   = map[string]cachedAccessToken{}
	accessTokenCacheMu sync.Mutex
)

// Plugins receive the result and never see the refresh token, so token lifecycle
// stays with the process that owns the database row.
func GetValidAccessToken(ctx context.Context, userID, provider string) (string, error) {
	key := userID + "|" + provider

	accessTokenCacheMu.Lock()
	if cached, ok := accessTokenCache[key]; ok && time.Now().UTC().Before(cached.expiresAt) {
		accessTokenCacheMu.Unlock()
		return cached.token, nil
	}
	accessTokenCacheMu.Unlock()

	stored, err := GetUserOAuthToken(userID, provider)
	if err != nil {
		return "", fmt.Errorf("no stored token for %s: %w", provider, err)
	}
	if stored.RefreshToken == "" {
		return "", fmt.Errorf("stored %s token has no refresh token", provider)
	}

	conf, err := oauthConfigFor(provider)
	if err != nil {
		return "", err
	}

	source := conf.TokenSource(ctx, &oauth2.Token{RefreshToken: stored.RefreshToken})
	fresh, err := source.Token()
	if err != nil {
		return "", fmt.Errorf("failed to refresh %s token: %w", provider, err)
	}

	expiry := fresh.Expiry
	if expiry.IsZero() {
		expiry = time.Now().UTC().Add(accessTokenLifetime)
	}

	accessTokenCacheMu.Lock()
	accessTokenCache[key] = cachedAccessToken{token: fresh.AccessToken, expiresAt: expiry}
	accessTokenCacheMu.Unlock()

	persistRefreshedToken(stored, fresh)

	return fresh.AccessToken, nil
}

// The nested map is the shape the plugin runtime expects inside its trmnl payload.
func AccessTokensForUser(ctx context.Context, userID string) map[string]map[string]string {
	tokens := make(map[string]map[string]string)
	for _, provider := range []string{"google", "todoist"} {
		accessToken, err := GetValidAccessToken(ctx, userID, provider)
		if err != nil {
			logging.Debug("[OAUTH] No access token", "provider", provider, "user_id", userID, "error", err)
			continue
		}
		tokens[provider] = map[string]string{"access_token": accessToken}
	}

	return tokens
}

// A provider that rotates the refresh token invalidates the stored one, so the new
// values are written back rather than left to go stale.
func persistRefreshedToken(stored *database.UserOAuthToken, fresh *oauth2.Token) {
	db := database.GetDB()
	if db == nil {
		return
	}

	updates := map[string]interface{}{"access_token": fresh.AccessToken}
	if fresh.RefreshToken != "" && fresh.RefreshToken != stored.RefreshToken {
		updates["refresh_token"] = fresh.RefreshToken
	}

	if err := db.Model(stored).Updates(updates).Error; err != nil {
		logging.Warn("[OAUTH] Failed to persist refreshed token", "provider", stored.Provider, "error", err)
	}
}

func oauthConfigFor(provider string) (*oauth2.Config, error) {
	manager := GetOAuthManager()
	if manager == nil {
		return nil, fmt.Errorf("oauth manager not initialised")
	}

	cfg, ok := manager.providerConfigs[provider]
	if !ok {
		return nil, fmt.Errorf("no registered oauth provider %q", provider)
	}

	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       cfg.Scopes,
		Endpoint:     oauth2.Endpoint{AuthURL: cfg.AuthURL, TokenURL: cfg.TokenURL},
	}, nil
}
