package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUserContext extracts user data from context set by auth middleware
func GetUserContext(c *gin.Context) gin.H {
	userName, _ := c.Get("userName")
	userEmail, _ := c.Get("userEmail")
	userRole, _ := c.Get("userRole")
	userID, _ := c.Get("userID")
	isAuthenticated, _ := c.Get("isAuthenticated")

	return gin.H{
		"userName":        userName,
		"userEmail":       userEmail,
		"userRole":        userRole,
		"id":              userID,
		"isAuthenticated": isAuthenticated,
	}
}

func DashboardHandler(c *gin.Context) {
	userData := GetUserContext(c)
	userData["activePage"] = "api_keys"
	userData["title"] = "API Keys"

	c.HTML(http.StatusOK, "api-keys.html", userData)
}
