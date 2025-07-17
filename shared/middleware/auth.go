package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Gin middleware for authentication
func AuthMiddlewareGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for session cookie
		session, err := c.Cookie("session")
		if err != nil || session == "" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		// TODO: Validate session value
		c.Next()
	}
}
