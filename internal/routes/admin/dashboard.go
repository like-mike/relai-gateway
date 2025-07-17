package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func DashboardHandler(c *gin.Context) {
	userEmail := ""
	if session, err := c.Cookie("session"); err == nil && session != "" {
		userEmail = session
	}
	c.HTML(http.StatusOK, "api_keys.html", gin.H{
		"activePage":      "api_keys",
		"isAuthenticated": true,
		"userEmail":       userEmail,
	})
}
