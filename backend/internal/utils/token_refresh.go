package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"time"
)

// RefreshTokenIfNeeded refreshes OAuth token if expired and updates the connection in database
func RefreshTokenIfNeeded(connection *models.Connection) error {
	if connection.TokenExpiresAt == nil {
		return nil // No expiry set
	}

	// Check if token is expired (with 5 minute buffer to refresh before actual expiry)
	expiryTime := *connection.TokenExpiresAt
	bufferTime := expiryTime.Add(-5 * time.Minute)
	if time.Now().Before(bufferTime) {
		return nil // Token still valid
	}

	// Token expired or about to expire - try to refresh
	if connection.RefreshToken == "" {
		// No refresh token available - mark as expired
		connection.Status = "EXPIRED"
		if err := config.DB.Save(connection).Error; err != nil {
			return fmt.Errorf("failed to update connection status: %v", err)
		}
		return fmt.Errorf("connection token expired and no refresh token available - re-authentication required")
	}

	// Refresh token based on provider
	var newAccessToken string
	var expiresIn int
	var err error

	switch connection.Provider {
	case "google":
		newAccessToken, expiresIn, err = refreshGoogleToken(connection.RefreshToken)
	default:
		return fmt.Errorf("token refresh not implemented for provider: %s", connection.Provider)
	}

	if err != nil {
		// Refresh failed - mark connection as expired
		connection.Status = "EXPIRED"
		if saveErr := config.DB.Save(connection).Error; saveErr != nil {
			return fmt.Errorf("token refresh failed and failed to update status: %v (refresh error: %v)", saveErr, err)
		}
		return fmt.Errorf("token refresh failed: %v", err)
	}

	// Update connection with new token
	connection.AccessToken = newAccessToken
	connection.Status = "ACTIVE"
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	connection.TokenExpiresAt = &expiresAt

	if err := config.DB.Save(connection).Error; err != nil {
		return fmt.Errorf("failed to save refreshed token: %v", err)
	}

	return nil
}

// refreshGoogleToken refreshes a Google OAuth token
func refreshGoogleToken(refreshToken string) (string, int, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return "", 0, fmt.Errorf("Google OAuth credentials not configured")
	}

	tokenURL := "https://oauth2.googleapis.com/token"
	
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create refresh request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to execute refresh request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read refresh response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", 0, fmt.Errorf("failed to parse refresh response: %v", err)
	}

	if tokenResp.AccessToken == "" {
		return "", 0, fmt.Errorf("no access token in refresh response")
	}

	// Default to 3600 seconds (1 hour) if expires_in is not provided
	expiresIn := tokenResp.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}

	return tokenResp.AccessToken, expiresIn, nil
}

