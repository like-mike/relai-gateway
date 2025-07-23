package auth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
)

const (
	defaultAdminUser = "admin"
	defaultAdminPass = "admin"
)

// Config holds authentication configuration.
type Config struct {
	EnableLocalLogin  bool
	EnableAzureAD     bool
	AzureClientID     string
	AzureTenantID     string
	AzureRedirectURI  string
	AzureClientSecret string
}

// LoadConfig loads authentication configuration from environment variables.
func LoadConfig() Config {
	return Config{
		EnableLocalLogin:  os.Getenv("ENABLE_LOCAL_LOGIN") != "false",
		EnableAzureAD:     os.Getenv("ENABLE_AZURE_AD") == "true",
		AzureClientID:     os.Getenv("AZURE_AD_CLIENT_ID"),
		AzureTenantID:     os.Getenv("AZURE_AD_TENANT_ID"),
		AzureRedirectURI:  os.Getenv("AZURE_AD_REDIRECT_URI"),
		AzureClientSecret: os.Getenv("AZURE_AD_CLIENT_SECRET"),
	}
}

// setSessionCookie sets the session cookie.
func setSessionCookie(c *gin.Context, key, value string, maxAge int) {
	c.SetCookie(key, value, maxAge, "/", "", false, true)
}

// RegisterRoutes registers authentication-related routes.
func RegisterRoutes(router gin.IRoutes, config Config) {
	// Only register /admin/logout on authorized group
	if group, ok := router.(*gin.RouterGroup); ok {
		group.GET("/admin/logout", func(c *gin.Context) {
			LogoutHandler(c, config)
		})

		// Add refresh access endpoint
		group.POST("/admin/refresh-access", func(c *gin.Context) {
			RefreshAccessHandler(c, config)
		})
	}
}

// Register public authentication routes (login, azure) on root router only
func RegisterPublicRoutes(router gin.IRoutes, config Config) {
	// Login page
	router.GET("/login", func(c *gin.Context) {
		if config.EnableAzureAD {
			c.Redirect(http.StatusFound, "/auth/azure")
			return
		}
		c.HTML(http.StatusOK, "login.html", gin.H{
			"isAuthenticated":  false,
			"enableLocalLogin": config.EnableLocalLogin,
			"enableAzureAD":    config.EnableAzureAD,
		})
	})

	// Login form submission
	router.POST("/login", func(c *gin.Context) {
		LocalLoginHandler(c, config)
	})

	// Azure AD login
	router.GET("/auth/azure", func(c *gin.Context) {
		AzureLoginHandler(c, config)
	})

	// Azure AD callback
	router.GET("/auth/azure/callback", func(c *gin.Context) {
		AzureCallbackHandler(c, config)
	})
}

// LogoutHandler handles user logout
func LogoutHandler(c *gin.Context, config Config) {
	setSessionCookie(c, "session", "", -1)
	setSessionCookie(c, "email", "", -1)
	setSessionCookie(c, "name", "", -1)
	setSessionCookie(c, "oid", "", -1)

	// Create a proper logout redirect URL that goes to login page, not callback
	logoutURL := "https://login.microsoftonline.com/" + config.AzureTenantID + "/oauth2/v2.0/logout"
	if config.AzureRedirectURI != "" {
		// Extract the base URL and redirect to login instead of callback
		baseURL := config.AzureRedirectURI[:len(config.AzureRedirectURI)-len("/auth/azure/callback")]
		logoutURL += "?post_logout_redirect_uri=" + baseURL + "/login"
	}
	c.Redirect(http.StatusFound, logoutURL)
}

// LocalLoginHandler handles local username/password login
func LocalLoginHandler(c *gin.Context, config Config) {
	adminUser := os.Getenv("ADMIN_USER")
	adminPass := os.Getenv("ADMIN_PASS")
	if adminUser == "" {
		adminUser = defaultAdminUser
	}
	if adminPass == "" {
		adminPass = defaultAdminPass
	}
	username := c.PostForm("username")
	password := c.PostForm("password")

	if config.EnableLocalLogin && username == adminUser && password == adminPass {
		setSessionCookie(c, "session", "dummy-session", 3600)
		c.Redirect(http.StatusFound, "/admin")
		return
	}
	c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Invalid credentials"})
}

// AzureLoginHandler handles Azure AD login initiation
func AzureLoginHandler(c *gin.Context, config Config) {
	if !config.EnableAzureAD {
		c.String(http.StatusNotFound, "Azure AD login disabled")
		return
	}
	authURL := "https://login.microsoftonline.com/" + config.AzureTenantID + "/oauth2/v2.0/authorize" +
		"?client_id=" + config.AzureClientID +
		"&response_type=code" +
		"&redirect_uri=" + config.AzureRedirectURI +
		"&response_mode=query" +
		"&scope=openid email profile" +
		"&state=xyz"
	c.Redirect(http.StatusFound, authURL)
}

// AzureCallbackHandler handles Azure AD callback
func AzureCallbackHandler(c *gin.Context, config Config) {
	code := c.Query("code")
	if code == "" {
		c.String(http.StatusBadRequest, "Missing code")
		return
	}
	// Exchange code for token, validate, create session
	tokenEndpoint := "https://login.microsoftonline.com/" + config.AzureTenantID + "/oauth2/v2.0/token"
	resp, err := http.PostForm(tokenEndpoint, map[string][]string{
		"client_id":     {config.AzureClientID},
		"client_secret": {config.AzureClientSecret},
		"scope":         {"openid email profile"},
		"code":          {code},
		"redirect_uri":  {config.AzureRedirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil || resp.StatusCode != http.StatusOK {
		c.String(http.StatusUnauthorized, "Azure AD token exchange failed")
		return
	}
	defer resp.Body.Close()
	var tokenResp struct {
		IDToken     string `json:"id_token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		c.String(http.StatusUnauthorized, "Failed to parse Azure token response")
		return
	}
	// Validate ID token (JWT)
	token, _, err := jwt.NewParser().ParseUnverified(tokenResp.IDToken, jwt.MapClaims{})
	if err != nil {
		c.String(http.StatusUnauthorized, "Invalid Azure ID token")
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.String(http.StatusUnauthorized, "Invalid Azure token claims")
		return
	}
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	oid, _ := claims["oid"].(string)

	setSessionCookie(c, "email", email, 3600)
	setSessionCookie(c, "name", name, 3600)
	setSessionCookie(c, "oid", oid, 3600)

	// Get user groups
	accessToken, err := getAccessToken(config.AzureTenantID, config.AzureClientID, config.AzureClientSecret)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get access token")
		return
	}
	results, err := getUserGroups(accessToken, oid)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get user groups")
		return
	}
	fmt.Println("User groups:", results)

	setSessionCookie(c, "session", "dummy-session", 3600)

	c.Redirect(http.StatusFound, "/admin")
}

func getAccessToken(tenantID, clientID, clientSecret string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("scope", "https://graph.microsoft.com/.default")

	url := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	resp, err := http.Post(url, "application/x-www-form-urlencoded", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("token request failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return "", err
	}
	return tokenResp.AccessToken, nil
}

func getUserGroups(accessToken, userID string) ([]string, error) {
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

// RefreshAccessHandler handles refresh access requests
func RefreshAccessHandler(c *gin.Context, config Config) {
	// Get user info from session cookies
	email, _ := c.Cookie("email")
	name, _ := c.Cookie("name")
	oid, _ := c.Cookie("oid")

	if email == "" || oid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	log.Printf("=== REFRESH ACCESS REQUEST for %s (%s) ===", name, email)

	// Get fresh access token and user groups
	accessToken, err := getAccessToken(config.AzureTenantID, config.AzureClientID, config.AzureClientSecret)
	if err != nil {
		log.Printf("Failed to get access token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get access token",
		})
		return
	}

	userGroups, err := getUserGroups(accessToken, oid)
	if err != nil {
		log.Printf("Failed to get user groups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get user groups",
		})
		return
	}

	log.Printf("=== REFRESH: Found %d AD groups for user %s ===", len(userGroups), email)
	for i, group := range userGroups {
		log.Printf("Refresh Group %d: %s", i+1, group)
	}

	// Call the enhanced refresh logic (reusing the enhanced auth logic)
	RefreshUserAccess(c, email, name, oid, userGroups)
}

// RefreshUserAccess handles the refresh logic by reusing enhanced authentication logic
func RefreshUserAccess(c *gin.Context, email, name, oid string, userGroups []string) {
	// Get database connection
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection error",
		})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection error",
		})
		return
	}

	// Get or create user
	user, err := db.CreateOrUpdateUser(sqlDB, models.CreateUserRequest{
		AzureOID: oid,
		Email:    email,
		Name:     name,
	})
	if err != nil {
		log.Printf("Failed to create/update user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "User update failed",
		})
		return
	}

	// Sync user organization memberships based on AD groups
	err = db.SyncUserOrganizationMemberships(sqlDB, user.ID, userGroups)
	if err != nil {
		log.Printf("Failed to sync user organization memberships: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to sync organization memberships",
		})
		return
	}

	// Get updated user memberships
	memberships, err := db.GetUserOrganizationMemberships(sqlDB, user.ID)
	if err != nil {
		log.Printf("Failed to get user memberships: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get updated memberships",
		})
		return
	}

	log.Printf("REFRESH SUCCESS: User %s now has memberships in %d organizations", user.Email, len(memberships))
	for orgID, roleName := range memberships {
		log.Printf("Membership: Org %s -> Role %s", orgID, roleName)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Access refreshed successfully",
		"memberships": len(memberships),
	})
}

// GetAccessToken gets an access token for Microsoft Graph API calls
func GetAccessToken(tenantID, clientID, clientSecret string) (string, error) {
	return getAccessToken(tenantID, clientID, clientSecret)
}
