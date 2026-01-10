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
	InitiateQuickBooksOAuth(c *gin.Context)
	QuickBooksOAuthCallback(c *gin.Context)
}

func NewOAuthHandler() OAuthMethods {
	return &OAuthHandler{}
}

// InitiateGoogleOAuth initiates Google OAuth flow
func (h *OAuthHandler) InitiateGoogleOAuth(c *gin.Context) {
	userIDStr := c.Query("user_id")
	returnTo := c.Query("return_to")
	appID := c.Query("app_id")
	isSource := c.Query("is_source") // "true" or "false" - indicates if this is for source or destination

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

	// Generate state parameter for security (store user_id, return_to, app_id, is_source, and random_uuid in state)
	// Format: user_id|return_to|app_id|is_source|random_uuid
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
	if isSource != "" {
		stateParts = append(stateParts, isSource)
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

	// Parse state to get user_id, return_to, app_id, and is_source (format: "user_id|return_to|app_id|is_source|random_uuid")
	parts := splitState(state)
	// For login flow, user_id might be "temp" - we'll find user by email after OAuth
	var userID uuid.UUID
	returnTo := ""
	appID := ""
	isSource := ""
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
	if len(parts) > 3 {
		isSource = parts[3]
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

	// Determine redirect based on flow type:
	// 1. If returnTo == "pipeline" → connection creation flow → redirect to sheet selection
	// 2. If returnTo == "" or not set → login flow → redirect to dashboard
	// 3. For non-Google apps, always redirect to dashboard

	if googleApp.Name == "googlesheet" || googleApp.Name == "googledrive" {
		// Check if this is a connection creation flow (returnTo == "pipeline")
		if returnTo == "pipeline" {
			// Connection creation flow: redirect to sheet selection page
			// Include is_source to indicate if this is for source or destination
			redirectURL := fmt.Sprintf("%s/connections/select-sheet?connection_id=%s&token=%s&company_id=%s&email=%s&return_to=pipeline",
				frontendURL, connection.ID.String(), token, user.CompanyID.String(), url.QueryEscape(user.Email))

			if appID != "" {
				redirectURL += fmt.Sprintf("&source_app_id=%s", appID)
			}
			if isSource != "" {
				redirectURL += fmt.Sprintf("&is_source=%s", isSource)
			}

			c.Redirect(http.StatusFound, redirectURL)
		} else {
			// Login flow: redirect to dashboard via OAuth callback page
			redirectURL := fmt.Sprintf("%s/oauth?provider=google&email=%s&success=true&token=%s&company_id=%s&company_name=%s&connection_id=%s",
				frontendURL, userInfo.Email, token, user.CompanyID.String(), user.Company.Name, connection.ID.String())
			c.Redirect(http.StatusFound, redirectURL)
		}
	} else {
		// For other apps, redirect to dashboard
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

// InitiateQuickBooksOAuth initiates QuickBooks OAuth flow
func (h *OAuthHandler) InitiateQuickBooksOAuth(c *gin.Context) {
	userIDStr := c.Query("user_id")
	returnTo := c.Query("return_to")
	appID := c.Query("app_id")
	companyIDStr := c.Query("company_id")

	// Get credentials from query params (provided by user) or environment (fallback)
	clientID := c.Query("client_id")
	clientSecret := c.Query("client_secret")

	// If not provided in query, try environment variables
	if clientID == "" {
		clientID = getEnv("QUICKBOOKS_CLIENT_ID", "")
	}
	if clientSecret == "" {
		clientSecret = getEnv("QUICKBOOKS_CLIENT_SECRET", "")
	}

	redirectURI := getEnv("QUICKBOOKS_REDIRECT_URI", "http://localhost:8080/api/oauth/quickbooks/callback")

	// Trim whitespace
	userIDStr = strings.TrimSpace(userIDStr)

	// Default to "temp" if empty
	if userIDStr == "" {
		userIDStr = "temp"
	}

	// Try to get company_id from auth context if not in query
	if companyIDStr == "" {
		if companyID, exists := c.Get("company_id"); exists {
			if companyIDUUID, ok := companyID.(uuid.UUID); ok {
				companyIDStr = companyIDUUID.String()
			}
		}
	}

	fmt.Printf("DEBUG: QuickBooks OAuth initiation\n")
	fmt.Printf("  Client ID provided: %v\n", clientID != "")
	fmt.Printf("  Client Secret provided: %v\n", clientSecret != "")
	fmt.Printf("  Redirect URI: %s\n", redirectURI)

	if clientID == "" {
		fmt.Printf("ERROR: QuickBooks Client ID not provided\n")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "QuickBooks Client ID is required. Please provide it in the connection setup.",
		})
		return
	}

	if clientSecret == "" {
		fmt.Printf("ERROR: QuickBooks Client Secret not provided\n")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "QuickBooks Client Secret is required. Please provide it in the connection setup.",
		})
		return
	}

	// Generate state parameter (format: user_id|return_to|app_id|company_id|client_id|client_secret|random_uuid)
	// We include credentials and company_id in state to use them in callback
	stateValue := userIDStr
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
	// Include company_id in state
	if companyIDStr != "" {
		stateParts = append(stateParts, companyIDStr)
	} else {
		stateParts = append(stateParts, "")
	}
	// Include credentials in state (will be used in callback)
	stateParts = append(stateParts, clientID)
	stateParts = append(stateParts, clientSecret)
	stateParts = append(stateParts, uuid.New().String())
	state := strings.Join(stateParts, "|")

	// QuickBooks OAuth URL (Intuit OAuth 2.0)
	// Scopes: accounting for QuickBooks Online
	scopes := "com.intuit.quickbooks.accounting"
	authURL := fmt.Sprintf(
		"https://appcenter.intuit.com/connect/oauth2?client_id=%s&scope=%s&redirect_uri=%s&response_type=code&state=%s&access_type=offline",
		url.QueryEscape(clientID),
		url.QueryEscape(scopes),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
	)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// QuickBooksOAuthCallback handles the OAuth callback from QuickBooks
func (h *OAuthHandler) QuickBooksOAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	realmID := c.Query("realmId") // QuickBooks provides realmId in callback
	errorParam := c.Query("error")

	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3002")

	if errorParam != "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=%s", frontendURL, errorParam))
		return
	}

	if code == "" || state == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Missing authorization code or state", frontendURL))
		return
	}

	// Parse state to get user_id, return_to, app_id, company_id, client_id, and client_secret
	// Format: user_id|return_to|app_id|company_id|client_id|client_secret|random_uuid
	parts := splitState(state)
	var userID uuid.UUID
	returnTo := ""
	appID := ""
	companyIDStr := ""
	clientID := ""
	clientSecret := ""

	if len(parts) > 0 && parts[0] != "temp" {
		var err error
		userID, err = uuid.Parse(parts[0])
		if err != nil {
			// Continue - will find user by email or create new
		}
	}
	if len(parts) > 1 {
		returnTo = parts[1]
	}
	if len(parts) > 2 {
		appID = parts[2]
	}
	if len(parts) > 3 {
		companyIDStr = parts[3]
	}
	if len(parts) > 4 {
		clientID = parts[4]
	}
	if len(parts) > 5 {
		clientSecret = parts[5]
	}

	// Fallback to environment variables if not in state
	if clientID == "" {
		clientID = getEnv("QUICKBOOKS_CLIENT_ID", "")
	}
	if clientSecret == "" {
		clientSecret = getEnv("QUICKBOOKS_CLIENT_SECRET", "")
	}

	redirectURI := getEnv("QUICKBOOKS_REDIRECT_URI", "http://localhost:8080/api/oauth/quickbooks/callback")

	if clientID == "" || clientSecret == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=QuickBooks OAuth credentials not found", frontendURL))
		return
	}

	tokens, err := exchangeQuickBooksCodeForTokens(code, clientID, clientSecret, redirectURI)
	if err != nil {
		fmt.Printf("ERROR: QuickBooks token exchange failed: %v\n", err)
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to exchange code: %v", frontendURL, err))
		return
	}

	if tokens == nil || tokens.AccessToken == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Token response is invalid", frontendURL))
		return
	}

	fmt.Printf("SUCCESS: QuickBooks token exchange completed\n")
	fmt.Printf("  Access Token length: %d\n", len(tokens.AccessToken))
	fmt.Printf("  Has Refresh Token: %v\n", tokens.RefreshToken != "")
	fmt.Printf("  Realm ID: %s\n", realmID)

	// Get company_id from state, query, auth context first
	var companyID uuid.UUID
	if companyIDStr != "" {
		var err error
		companyID, err = uuid.Parse(companyIDStr)
		if err != nil {
			companyIDStr = "" // Reset if invalid
		}
	}

	// Try query parameter if not in state
	if companyIDStr == "" {
		companyIDStr = c.Query("company_id")
		if companyIDStr != "" {
			var err error
			companyID, err = uuid.Parse(companyIDStr)
			if err != nil {
				companyIDStr = "" // Reset if invalid
			}
		}
	}

	// Try auth context if still not found
	if companyIDStr == "" {
		if companyIDFromContext, exists := c.Get("company_id"); exists {
			if companyIDUUID, ok := companyIDFromContext.(uuid.UUID); ok {
				companyID = companyIDUUID
				companyIDStr = companyID.String()
			}
		}
	}

	// If still no company_id, return error
	if companyIDStr == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Company ID required. Please ensure you are logged in.", frontendURL))
		return
	}

	// Now find user - try auth context first, then state, then find by company
	var user models.User
	userIDFromContext := uuid.Nil

	// Try to get user_id from auth context first (user should be logged in)
	if userIDFromContextVal, exists := c.Get("user_id"); exists {
		if userIDUUID, ok := userIDFromContextVal.(uuid.UUID); ok {
			userIDFromContext = userIDUUID
		}
	}

	// Use user from context if available
	if userIDFromContext != uuid.Nil {
		if err := config.DB.Preload("Company").First(&user, userIDFromContext).Error; err == nil {
			// User found from context
		} else {
			userIDFromContext = uuid.Nil // Reset if not found
		}
	}

	// If not from context, try from state parameter
	if user.ID == uuid.Nil && userID != uuid.Nil {
		if err := config.DB.Preload("Company").First(&user, userID).Error; err == nil {
			// User found from state
		} else {
			userID = uuid.Nil // Reset if not found
		}
	}

	// If still no user, find a user from the company
	if user.ID == uuid.Nil {
		if err := config.DB.Where("company_id = ?", companyID).Preload("Company").First(&user).Error; err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=User not found for company. Please ensure you are logged in.", frontendURL))
			return
		}
	}

	// Get or create app
	var quickbooksApp models.App
	if err := config.DB.Where("name = ?", "quickbooks").First(&quickbooksApp).Error; err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=QuickBooks app not found", frontendURL))
		return
	}

	// Use app_id from state if provided, otherwise use the one we found
	if appID != "" {
		if parsedAppID, err := uuid.Parse(appID); err == nil {
			quickbooksApp.ID = parsedAppID
		}
	}

	// Create or update connection
	var connection models.Connection
	// Check if connection already exists for this company and app
	if err := config.DB.Where("company_id = ? AND app_id = ?", companyID, quickbooksApp.ID).
		First(&connection).Error; err != nil {
		// Create new connection
		connection = models.Connection{
			UserID:       user.ID,
			CompanyID:    companyID,
			AppID:        quickbooksApp.ID,
			Provider:     "quickbooks",
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			RealmID:      realmID,
			IsActive:     true,
			Status:       "ACTIVE",
		}

		// Set token expiry if provided
		if tokens.ExpiresIn > 0 {
			expiry := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
			connection.TokenExpiresAt = &expiry
		}

		if err := config.DB.Create(&connection).Error; err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to create connection: %v", frontendURL, err))
			return
		}
	} else {
		// Update existing connection
		connection.AccessToken = tokens.AccessToken
		connection.RefreshToken = tokens.RefreshToken
		connection.RealmID = realmID
		connection.IsActive = true
		connection.Status = "ACTIVE"

		if tokens.ExpiresIn > 0 {
			expiry := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
			connection.TokenExpiresAt = &expiry
		}

		if err := config.DB.Save(&connection).Error; err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to update connection: %v", frontendURL, err))
			return
		}
	}

	// Generate JWT token for frontend
	token, err := utils.GenerateToken(user.ID, user.CompanyID, user.Email)
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/oauth-error?error=Failed to generate token: %v", frontendURL, err))
		return
	}

	// Redirect to QuickBooks object selection page, then back to pipeline creation
	if returnTo == "pipeline" {
		// First, create a data object for the connection (user will select object type)
		// Then redirect back to pipeline creation with the connection and data object
		// Include user email in URL so frontend can set user properly
		redirectURL := fmt.Sprintf("%s/connections/select-quickbooks-object?connection_id=%s&token=%s&company_id=%s&return_to=pipeline&email=%s",
			frontendURL, connection.ID.String(), token, companyID.String(), url.QueryEscape(user.Email))
		if appID != "" {
			redirectURL += fmt.Sprintf("&source_app_id=%s", appID)
		}
		c.Redirect(http.StatusFound, redirectURL)
	} else {
		redirectURL := fmt.Sprintf("%s/oauth?provider=quickbooks&success=true&token=%s&company_id=%s&connection_id=%s&email=%s",
			frontendURL, token, companyID.String(), connection.ID.String(), url.QueryEscape(user.Email))
		c.Redirect(http.StatusFound, redirectURL)
	}
}

// exchangeQuickBooksCodeForTokens exchanges authorization code for access and refresh tokens
func exchangeQuickBooksCodeForTokens(code, clientID, clientSecret, redirectURI string) (*QuickBooksTokenResponse, error) {
	tokenURL := "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer"

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
	req.Header.Set("Accept", "application/json")

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

	fmt.Printf("QuickBooks token exchange response:\n")
	fmt.Printf("  Status Code: %d\n", resp.StatusCode)
	fmt.Printf("  Response Body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp QuickBooksTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("access token is empty in response")
	}

	return &tokenResp, nil
}

type QuickBooksTokenResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	TokenType             string `json:"token_type"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshTokenExpiresIn int    `json:"x_refresh_token_expires_in"`
	RealmID               string `json:"realmId"`
}
