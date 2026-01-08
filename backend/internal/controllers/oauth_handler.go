package controllers

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
	"syncflow-backend/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OAuthHandler struct{}

type OAuthMethods interface {
	InitiateGoogleOAuth(c *gin.Context)
	GoogleOAuthCallback(c *gin.Context)
}

func NewOAuthHandler() OAuthMethods {
	return &OAuthHandler{}
}

// InitiateGoogleOAuth initiates Google OAuth flow
func (h *OAuthHandler) InitiateGoogleOAuth(c *gin.Context) {
	userIDStr := c.Query("user_id")
	returnTo := c.Query("return_to")
	appID := c.Query("app_id")

	// Trim whitespace
	userIDStr = strings.TrimSpace(userIDStr)

	// Default to "temp" if empty - allows login flow
	if userIDStr == "" {
		userIDStr = "temp"
	}

	// Validate: must be "temp" or a valid UUID
	var userID uuid.UUID
	var err error

	if userIDStr == "temp" {
		// "temp" is allowed - will find user by email after OAuth
		// Continue without validation
	} else {
		// Must be a valid UUID
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":    "Invalid user_id - must be 'temp' or a valid UUID",
				"received": userIDStr,
			})
			return
		}
		_ = userID // Use userID if needed later
	}

	// Get Google OAuth credentials from environment
	clientID := getEnv("GOOGLE_CLIENT_ID", "")
	redirectURI := getEnv("GOOGLE_REDIRECT_URI", "http://localhost:8080/api/oauth/google/callback")

	if clientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Google OAuth not configured. Please set GOOGLE_CLIENT_ID",
		})
		return
	}

	// Generate state parameter for security (store user_id, return_to, and app_id in state)
	// Format: user_id|return_to|app_id|random_uuid
	stateValue := userIDStr // Use the original string (either "temp" or actual UUID)
	stateParts := []string{stateValue}
	if returnTo != "" {
		stateParts = append(stateParts, returnTo)
	} else {
		stateParts = append(stateParts, "")
	}
	if appID != "" {
		stateParts = append(stateParts, appID)
	} else {
		stateParts = append(stateParts, "")
	}
	stateParts = append(stateParts, uuid.New().String())
	state := strings.Join(stateParts, "|")

	// Google OAuth URL - include both drive and userinfo scopes
	scopes := "https://www.googleapis.com/auth/drive.readonly https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent&state=%s",
		clientID,
		redirectURI,
		scopes,
		state,
	)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// GoogleOAuthCallback handles the OAuth callback from Google
func (h *OAuthHandler) GoogleOAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3001")

	if errorParam != "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=%s", frontendURL, errorParam))
		return
	}

	if code == "" || state == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Missing authorization code or state", frontendURL))
		return
	}

	// Parse state to get user_id, return_to, and app_id (format: "user_id|return_to|app_id|random_uuid")
	parts := splitState(state)
	// For login flow, user_id might be "temp" - we'll find user by email after OAuth
	var userID uuid.UUID
	returnTo := ""
	appID := ""
	if len(parts) > 0 && parts[0] != "temp" {
		var err error
		userID, err = uuid.Parse(parts[0])
		if err != nil {
			// Continue with temp user - will find by email
		}
	}
	if len(parts) > 1 {
		returnTo = parts[1]
	}
	if len(parts) > 2 {
		appID = parts[2]
	}

	// Exchange code for tokens
	clientID := getEnv("GOOGLE_CLIENT_ID", "")
	clientSecret := getEnv("GOOGLE_CLIENT_SECRET", "")
	redirectURI := getEnv("GOOGLE_REDIRECT_URI", "http://localhost:8080/api/oauth/google/callback")

	if clientID == "" || clientSecret == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Google OAuth not configured", frontendURL))
		return
	}

	tokens, err := exchangeGoogleCodeForTokens(code, clientID, clientSecret, redirectURI)
	if err != nil {
		fmt.Printf("ERROR: Token exchange failed: %v\n", err)
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to exchange code: %v", frontendURL, err))
		return
	}

	// Validate access token
	if tokens == nil {
		fmt.Printf("ERROR: Token response is nil\n")
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Token response is nil", frontendURL))
		return
	}

	if tokens.AccessToken == "" {
		fmt.Printf("ERROR: Access token is empty in response\n")
		fmt.Printf("  Token response: %+v\n", tokens)
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Access token is empty", frontendURL))
		return
	}

	fmt.Printf("SUCCESS: Token exchange completed\n")
	fmt.Printf("  Access Token length: %d\n", len(tokens.AccessToken))
	fmt.Printf("  Has Refresh Token: %v\n", tokens.RefreshToken != "")

	// Get user info from Google
	fmt.Printf("Attempting to get user info with access token...\n")
	userInfo, err := getGoogleUserInfo(tokens.AccessToken)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Failed to get user info:\n")
		tokenPreview := tokens.AccessToken
		if len(tokenPreview) > 20 {
			tokenPreview = tokenPreview[:20] + "..."
		}
		fmt.Printf("  Access Token (first 20 chars): %s\n", tokenPreview)
		fmt.Printf("  Error: %v\n", err)
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to get user info: %v", frontendURL, err))
		return
	}

	// For login flow, find or create user by email
	var user models.User
	if userID == uuid.Nil || (len(parts) > 0 && parts[0] == "temp") {
		// Try to find user by email
		result := config.DB.Where("email = ?", userInfo.Email).Preload("Company").First(&user)
		if result.Error != nil {
			// User doesn't exist - create company and user
			companyName := utils.ExtractCompanyName(userInfo.Email)
			companyDomain := utils.ExtractCompanyDomain(userInfo.Email)

			if companyName == "" {
				c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Invalid email format", frontendURL))
				return
			}

			// Find or create company
			var company models.Company
			companyResult := config.DB.Where("domain = ? OR name = ?", companyDomain, companyName).First(&company)
			if companyResult.Error != nil {
				// Create new company
				company = models.Company{
					Name:     companyName,
					Domain:   companyDomain,
					IsActive: true,
				}
				if err := config.DB.Create(&company).Error; err != nil {
					c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to create company: %v", frontendURL, err))
					return
				}
			}

			// Create user
			// Split name into first and last (simple split on first space)
			firstName := userInfo.Name
			lastName := ""
			if parts := strings.SplitN(userInfo.Name, " ", 2); len(parts) > 1 {
				firstName = parts[0]
				lastName = parts[1]
			}

			user = models.User{
				CompanyID: company.ID,
				Email:     userInfo.Email,
				FirstName: firstName,
				LastName:  lastName,
				Role:      "user",
				IsActive:  true,
			}
			if err := config.DB.Create(&user).Error; err != nil {
				c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to create user: %v", frontendURL, err))
				return
			}
		}
		userID = user.ID
	} else {
		// Load existing user
		if err := config.DB.First(&user, userID).Error; err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=User not found", frontendURL))
			return
		}
	}

	// Calculate token expiration
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	// Get or find Google app
	var googleApp models.App
	if err := config.DB.Where("name = ?", "googledrive").First(&googleApp).Error; err != nil {
		// If app doesn't exist, try googlesheet
		if err := config.DB.Where("name = ?", "googlesheet").First(&googleApp).Error; err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Google app not configured", frontendURL))
			return
		}
	}

	// Ensure company is loaded for existing users
	if user.CompanyID != uuid.Nil && user.Company.ID == uuid.Nil {
		if err := config.DB.Preload("Company").First(&user, userID).Error; err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=User not found", frontendURL))
			return
		}
	}

	// Save or update connection
	connection := models.Connection{
		UserID:         user.ID,
		CompanyID:      user.CompanyID,
		AppID:          googleApp.ID,
		Provider:       "google", // Set provider for Google OAuth
		ProviderUserID: userInfo.ID,
		AccessToken:    tokens.AccessToken,
		RefreshToken:   tokens.RefreshToken,
		TokenExpiresAt: &expiresAt,
		IsActive:       true,
	}

	// Check if connection already exists
	var existing models.Connection
	// Use the global DB from config (temporary - should use dependency injection)
	result := config.DB.Where("user_id = ? AND company_id = ? AND app_id = ?",
		user.ID, user.CompanyID, googleApp.ID).First(&existing)

	if result.Error == nil {
		// Update existing
		existing.AccessToken = tokens.AccessToken
		existing.RefreshToken = tokens.RefreshToken
		existing.TokenExpiresAt = &expiresAt
		existing.UpdatedAt = time.Now()
		config.DB.Save(&existing)
		connection = existing

		// Update metadata
		metadataJSON := fmt.Sprintf(`{"email": "%s", "name": "%s"}`, userInfo.Email, userInfo.Name)
		var metadata models.Metadata
		if err := config.DB.Where("connection_id = ?", connection.ID).First(&metadata).Error; err == nil {
			metadata.Data = metadataJSON
			config.DB.Save(&metadata)
		} else {
			// Create metadata if it doesn't exist
			metadata = models.Metadata{
				ConnectionID: connection.ID,
				Data:         metadataJSON,
			}
			config.DB.Create(&metadata)
		}
	} else {
		// Create new connection
		if err := config.DB.Create(&connection).Error; err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to create connection: %v", frontendURL, err))
			return
		}

		// Reload connection to ensure ID is set
		if err := config.DB.First(&connection, connection.ID).Error; err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to retrieve connection: %v", frontendURL, err))
			return
		}

		// Create metadata
		metadataJSON := fmt.Sprintf(`{"email": "%s", "name": "%s"}`, userInfo.Email, userInfo.Name)
		metadata := models.Metadata{
			ConnectionID: connection.ID,
			Data:         metadataJSON,
		}
		if err := config.DB.Create(&metadata).Error; err != nil {
			// Log error but don't fail - metadata is optional
			fmt.Printf("WARNING: Failed to create metadata: %v\n", err)
		}
	}

	// Generate JWT token for authentication
	token, err := utils.GenerateToken(user.ID, user.CompanyID, user.Email)
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to generate token: %v", frontendURL, err))
		return
	}

	// Check if this OAuth was initiated from connection creation (check state or query param)
	// For now, always redirect to sheet selection if it's a Google Sheets/Drive connection
	// Otherwise, redirect to dashboard

	if googleApp.Name == "googlesheet" || googleApp.Name == "googledrive" {
		// Redirect to sheet selection page for Google Sheets/Drive connections
		// Use return_to and app_id from state parameter
		redirectURL := fmt.Sprintf("%s/connections/select-sheet?connection_id=%s&token=%s&company_id=%s",
			frontendURL, connection.ID.String(), token, user.CompanyID.String())

		if returnTo != "" {
			redirectURL += fmt.Sprintf("&return_to=%s", returnTo)
		}
		if appID != "" {
			redirectURL += fmt.Sprintf("&source_app_id=%s", appID)
		}

		c.Redirect(http.StatusFound, redirectURL)
	} else {
		// For other apps or regular login, redirect to dashboard
		redirectURL := fmt.Sprintf("%s/oauth?provider=google&email=%s&success=true&token=%s&company_id=%s&company_name=%s&connection_id=%s",
			frontendURL, userInfo.Email, token, user.CompanyID.String(), user.Company.Name, connection.ID.String())
		c.Redirect(http.StatusFound, redirectURL)
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func splitState(state string) []string {
	// Simple split on |
	parts := []string{}
	current := ""
	for _, char := range state {
		if char == '|' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type GoogleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func exchangeGoogleCodeForTokens(code, clientID, clientSecret, redirectURI string) (*GoogleTokenResponse, error) {
	tokenURL := "https://oauth2.googleapis.com/token"

	// Google OAuth2 token endpoint expects form-urlencoded, not JSON
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Token exchange response:\n")
	fmt.Printf("  Status Code: %d\n", resp.StatusCode)
	fmt.Printf("  Response Body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		// Log the error for debugging
		fmt.Printf("ERROR: Google OAuth token exchange failed:\n")
		fmt.Printf("  Status: %d\n", resp.StatusCode)
		fmt.Printf("  Response: %s\n", string(body))
		fmt.Printf("  Client ID: %s\n", clientID)
		fmt.Printf("  Redirect URI: %s\n", redirectURI)
		fmt.Printf("  Client Secret present: %v\n", clientSecret != "")
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp GoogleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	// Validate token response
	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("access token is empty in response")
	}

	fmt.Printf("Token exchange successful:\n")
	fmt.Printf("  Access Token received: %v\n", tokenResp.AccessToken != "")
	fmt.Printf("  Refresh Token received: %v\n", tokenResp.RefreshToken != "")
	fmt.Printf("  Expires In: %d seconds\n", tokenResp.ExpiresIn)

	return &tokenResp, nil
}

func getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is empty")
	}

	// Use the newer OAuth2 userinfo endpoint
	url := "https://www.googleapis.com/oauth2/v2/userinfo"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set authorization header - make sure token is properly formatted
	authHeader := "Bearer " + strings.TrimSpace(accessToken)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")

	fmt.Printf("Making userinfo request:\n")
	fmt.Printf("  URL: %s\n", url)
	authPreview := authHeader
	if len(authPreview) > 30 {
		authPreview = authPreview[:30] + "..."
	}
	fmt.Printf("  Auth Header (first 30 chars): %s\n", authPreview)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// Log detailed error
		fmt.Printf("Google userinfo API error:\n")
		fmt.Printf("  Status: %d\n", resp.StatusCode)
		fmt.Printf("  Response: %s\n", string(body))
		tokenPreview := accessToken
		if len(tokenPreview) > 20 {
			tokenPreview = tokenPreview[:20] + "..."
		}
		fmt.Printf("  Access Token (first 20 chars): %s\n", tokenPreview)
		return nil, fmt.Errorf("userinfo request failed: %s", string(body))
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}
