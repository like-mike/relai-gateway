package admin

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/config"
)

// GetUserContext extracts user data from context set by auth middleware
func GetUserContext(c *gin.Context) gin.H {
	// Get data from enhanced auth middleware context keys
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

func DashboardHandler(c *gin.Context) {
	userData := GetUserContext(c)
	userData["activePage"] = "api_keys"
	userData["title"] = "API Keys"

	c.HTML(http.StatusOK, "api-keys.html", userData)
}
