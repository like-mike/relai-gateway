//go:build !test

package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultAdminUser = "admin"
	defaultAdminPass = "admin"
)

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	EnableLocalLogin  bool
	EnableAzureAD     bool
	AzureClientID     string
	AzureTenantID     string
	AzureRedirectURI  string
	AzureClientSecret string
}

// LoadAuthConfig loads authentication configuration from environment variables.
func LoadAuthConfig() AuthConfig {
	return AuthConfig{
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

// RegisterAuthRoutes registers authentication-related routes.
func RegisterAuthRoutes(router gin.IRoutes, config AuthConfig) {
	// Only register /admin/logout on authorized group
	if group, ok := router.(*gin.RouterGroup); ok {
		group.GET("/admin/logout", func(c *gin.Context) {
			setSessionCookie(c, "session", "", -1)
			logoutURL := "https://login.microsoftonline.com/" + config.AzureTenantID + "/oauth2/v2.0/logout" +
				"?post_logout_redirect_uri=" + config.AzureRedirectURI
			c.Redirect(http.StatusFound, logoutURL)
		})
	}
}

// Register public authentication routes (login, azure) on root router only
func RegisterPublicAuthRoutes(router gin.IRoutes, config AuthConfig) {
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
	})

	// Azure AD login
	router.GET("/auth/azure", func(c *gin.Context) {
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
	})

	// Azure AD callback
	router.GET("/auth/azure/callback", func(c *gin.Context) {
		fmt.Println("yoyoyoyoyoy")
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

		// TODO: Validate JWT signature with Azure public keys for production

		setSessionCookie(c, "session", "dummy-session", 3600)

		c.Redirect(http.StatusFound, "/admin")
	})
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

	// var result struct {
	// 	Value []struct {
	// 		ID          string `json:"id"`
	// 		DisplayName string `json:"displayName"`
	// 		OdataType   string `json:"@odata.type"`
	// 	} `json:"value"`
	// }
	// err = json.Unmarshal(body, &result)
	// if err != nil {
	// 	return err
	// }

	fmt.Println(string(body))

	// fmt.Printf("Groups for user %s:\n", userID)
	// for _, item := range result.Value {
	// 	if item.OdataType == "#microsoft.graph.group" {
	// 		fmt.Printf("- %s (%s)\n", item.DisplayName, item.ID)
	// 	}
	// }
	return results, nil
}
