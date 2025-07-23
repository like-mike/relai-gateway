package admin

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
)

// TokenResponse represents the Azure AD token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// UserProfile represents a user's profile from Microsoft Graph
type UserProfile struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

// EnhancedLoginHandler handles Azure AD login with automatic role synchronization
func EnhancedLoginHandler(c *gin.Context) {
	config := LoadAuthConfig()
	if !config.EnableAzureAD {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Azure AD authentication is disabled"})
		return
	}

	// Generate state parameter for security
	state, err := generateState()
	if err != nil {
		log.Printf("Failed to generate state: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication setup failed"})
		return
	}

	// Store state in session/cookie for validation
	c.SetCookie("auth_state", state, 600, "/", "", false, true) // 10 minutes

	// Build Azure AD authorization URL with additional scopes for group access
	authURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?client_id=%s&response_type=code&redirect_uri=%s&response_mode=query&scope=%s&state=%s",
		config.AzureTenantID,
		config.AzureClientID,
		url.QueryEscape(config.AzureRedirectURI),
		url.QueryEscape("openid profile email User.Read GroupMember.Read.All"), // Added group permissions
		state,
	)

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// EnhancedCallbackHandler handles the Azure AD callback with role synchronization
func EnhancedCallbackHandler(c *gin.Context) {
	config := LoadAuthConfig()
	if !config.EnableAzureAD {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Azure AD authentication is disabled"})
		return
	}

	// Validate state parameter
	state := c.Query("state")
	storedState, err := c.Cookie("auth_state")
	if err != nil || state != storedState {
		log.Printf("State validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	// Clear the state cookie
	c.SetCookie("auth_state", "", -1, "/", "", false, true)

	// Handle error from Azure AD
	if errorCode := c.Query("error"); errorCode != "" {
		errorDesc := c.Query("error_description")
		log.Printf("Azure AD error: %s - %s", errorCode, errorDesc)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Authentication failed: %s", errorDesc)})
		return
	}

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	// Exchange code for tokens
	tokenResp, err := exchangeCodeForTokens(config, code)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token exchange failed"})
		return
	}

	// Extract user info from ID token
	userInfo, err := extractUserFromIDToken(tokenResp.IDToken)
	if err != nil {
		log.Printf("Failed to extract user from ID token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user profile"})
		return
	}

	// Get user's Azure AD groups using the access token from callback
	userGroups, err := getUserGroupsFromToken(tokenResp.AccessToken, userInfo.ID)
	if err != nil {
		log.Printf("Failed to get user groups: %v", err)
		// Continue without groups rather than failing completely
		userGroups = []string{}
	}

	log.Printf("User %s has %d AD groups", userInfo.Mail, len(userGroups))

	// Get database connection
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Create or update user in database
	user, err := db.CreateOrUpdateUser(sqlDB, models.CreateUserRequest{
		AzureOID: userInfo.ID,
		Email:    userInfo.Mail,
		Name:     userInfo.DisplayName,
	})
	if err != nil {
		log.Printf("Failed to create/update user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User creation failed"})
		return
	}

	// Sync user organization memberships based on AD groups
	err = db.SyncUserOrganizationMemberships(sqlDB, user.ID, userGroups)
	if err != nil {
		log.Printf("Failed to sync user organization memberships: %v", err)
		// Log error but don't fail the login - user can still access with previous roles
	}

	// Get updated user memberships
	memberships, err := db.GetUserOrganizationMemberships(sqlDB, user.ID)
	if err != nil {
		log.Printf("Failed to get user memberships: %v", err)
		memberships = make(map[string]string) // Empty map as fallback
	}

	log.Printf("User %s has memberships in %d organizations", user.Email, len(memberships))

	// Update last login time
	err = db.UpdateUserLastLogin(sqlDB, user.ID)
	if err != nil {
		log.Printf("Failed to update last login: %v", err)
		// Non-critical error, continue
	}

	// Create session
	sessionData := map[string]interface{}{
		"user_id":       user.ID,
		"azure_oid":     user.AzureOID,
		"email":         user.Email,
		"name":          user.Name,
		"memberships":   memberships,
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
		"token_expires": time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		"authenticated": true,
		"login_time":    time.Now(),
	}

	// Store session (you might want to use a proper session store)
	sessionJSON, err := json.Marshal(sessionData)
	if err != nil {
		log.Printf("Failed to marshal session data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session creation failed"})
		return
	}

	// Set secure session cookie
	c.SetCookie("session", base64.StdEncoding.EncodeToString(sessionJSON), 86400, "/", "", true, true) // 24 hours

	log.Printf("User %s logged in successfully with %d organization memberships", user.Email, len(memberships))

	// Redirect to dashboard
	c.Redirect(http.StatusTemporaryRedirect, "/admin")
}

// Enhanced token exchange function
func exchangeCodeForTokens(config AuthConfig, code string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", config.AzureTenantID)

	data := url.Values{}
	data.Set("client_id", config.AzureClientID)
	data.Set("client_secret", config.AzureClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", config.AzureRedirectURI)
	data.Set("scope", "openid profile email User.Read GroupMember.Read.All")

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// Extract user info from ID token
func extractUserFromIDToken(idToken string) (*UserProfile, error) {
	token, _, err := jwt.NewParser().ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	userProfile := &UserProfile{
		ID:          claims["oid"].(string),
		DisplayName: claims["name"].(string),
		Mail:        claims["email"].(string),
	}

	return userProfile, nil
}

// Get user groups from token (modified version of existing function)
func getUserGroupsFromToken(accessToken, userID string) ([]string, error) {
	results := []string{}

	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/memberOf", userID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return results, err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return results, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return results, fmt.Errorf("graph request failed: %s", string(body))
	}

	var result struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			OdataType   string `json:"@odata.type"`
		} `json:"value"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("Failed to parse Graph API response: %v", err)
		return results, err
	}

	log.Printf("Found %d directory objects for user %s", len(result.Value), userID)
	for _, item := range result.Value {
		if item.OdataType == "#microsoft.graph.group" {
			results = append(results, item.ID)
			log.Printf("User is member of group: %s (%s)", item.DisplayName, item.ID)
		}
	}

	return results, nil
}

// Helper function to generate secure state parameter
func generateState() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// Enhanced middleware to check authentication and load user context
func EnhancedAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Cookie("session")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			c.Abort()
			return
		}

		sessionData, err := base64.StdEncoding.DecodeString(sessionCookie)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
			c.Abort()
			return
		}

		var session map[string]interface{}
		err = json.Unmarshal(sessionData, &session)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session data"})
			c.Abort()
			return
		}

		// Check if authenticated
		if auth, ok := session["authenticated"].(bool); !ok || !auth {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			c.Abort()
			return
		}

		// Add user context to request
		c.Set("user_id", session["user_id"])
		c.Set("user_email", session["email"])
		c.Set("user_name", session["name"])
		c.Set("user_memberships", session["memberships"])
		c.Set("azure_oid", session["azure_oid"])

		c.Next()
	}
}
