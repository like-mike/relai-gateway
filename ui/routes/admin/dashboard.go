package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func DashboardHandler(c *gin.Context) {
	var userName, userEmail string
	if session, err := c.Cookie("name"); err == nil && session != "" {
		userName = session
	}
	if session, err := c.Cookie("email"); err == nil && session != "" {
		userEmail = session
	}
	c.HTML(http.StatusOK, "api_keys.html", gin.H{
		"activePage":      "api_keys",
		"isAuthenticated": true,
		"userName":        userName,
		"userEmail":       userEmail,
	})
}
