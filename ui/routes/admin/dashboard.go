package admin

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/config"
)

// GetUserContext extracts user data from context set by auth middleware
func GetUserContext(c *gin.Context) gin.H {
	userName, _ := c.Get("userName")
	userEmail, _ := c.Get("userEmail")
	userRole, _ := c.Get("userRole")
	userID, _ := c.Get("userID")
	isAuthenticated, _ := c.Get("isAuthenticated")

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
		"id":              userID,
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
