package auth

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/config"
)

// UserContext represents the authenticated user's context data
type UserContext struct {
	UserName        string
	UserEmail       string
	UserRole        string
	UserID          string
	AzureOID        string
	Memberships     interface{}
	IsAuthenticated bool
	Theme           interface{}
	Config          interface{}
	ThemeKey        interface{}
}

// GetUserContext extracts user data from context set by auth middleware
func GetUserContext(c *gin.Context) gin.H {
	// Get data from auth middleware context keys
	userName, _ := c.Get("user_name")
	userEmail, _ := c.Get("user_email")
	userID, _ := c.Get("user_id")
	azureOID, _ := c.Get("azure_oid")
	userMemberships, _ := c.Get("user_memberships")

	// Default role (enhanced middleware doesn't set role, but we can default to Admin)
	userRole := "Admin"
	isAuthenticated := true // If we get here, user is authenticated

	// Get theme data
	themeData, err := config.GetThemeContextData()
	if err != nil {
		log.Printf("Warning: Failed to get theme data: %v", err)
		themeData = nil
	}

	context := gin.H{
		"userName":        userName,
		"userEmail":       userEmail,
		"userRole":        userRole,
		"id":              userID, // This is now the actual user UUID from database
		"azure_oid":       azureOID,
		"memberships":     userMemberships,
		"isAuthenticated": isAuthenticated,
	}

	// Add theme data if available
	if themeData != nil {
		context["Theme"] = themeData.ActiveTheme
		context["Config"] = themeData.Config
		context["ThemeKey"] = themeData.ThemeKey
	}

	return context
}

// RequireAuth is a helper that can be used to check if user is authenticated
func RequireAuth(c *gin.Context) bool {
	isAuth, exists := c.Get("isAuthenticated")
	return exists && isAuth == true
}

// GetUserID safely extracts the user ID from context
func GetUserID(c *gin.Context) (string, bool) {
	if userID, exists := c.Get("userID"); exists {
		if id, ok := userID.(string); ok && id != "" {
			return id, true
		}
	}

	// Try alternative key
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok && id != "" {
			return id, true
		}
	}

	return "", false
}

// GetUserEmail safely extracts the user email from context
func GetUserEmail(c *gin.Context) (string, bool) {
	if userEmail, exists := c.Get("userEmail"); exists {
		if email, ok := userEmail.(string); ok && email != "" {
			return email, true
		}
	}

	// Try alternative key
	if userEmail, exists := c.Get("user_email"); exists {
		if email, ok := userEmail.(string); ok && email != "" {
			return email, true
		}
	}

	return "", false
}
