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

		// Extract user information from cookies and set in context
		var userName, userEmail, userRole, userID string

		if name, err := c.Cookie("name"); err == nil && name != "" {
			userName = name
		}

		if email, err := c.Cookie("email"); err == nil && email != "" {
			userEmail = email
		}

		if role, err := c.Cookie("role"); err == nil && role != "" {
			userRole = role
		} else {
			// Default role if not found in cookie
			userRole = "Admin"
		}

		// if id, err := c.Cookie("id"); err == nil && id != "" {
		// 	userID = id
		// }

		userID = userEmail

		// Set user data in context for all handlers to use
		c.Set("userName", userName)
		c.Set("userEmail", userEmail)
		c.Set("userRole", userRole)
		c.Set("userID", userID)
		c.Set("isAuthenticated", true)

		// TODO: Validate session value
		c.Next()
	}
}

// package middleware

// import (
// 	"log"
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// )

// // Gin middleware for authentication
// func AuthMiddlewareGin() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		// Check for session cookie
// 		session, err := c.Cookie("session")
// 		if err != nil || session == "" {
// 			c.Redirect(http.StatusFound, "/login")
// 			c.Abort()
// 			return
// 		}

// 		// For now, set a hardcoded user context (replace with real auth later)
// 		// userName := "Demo User"
// 		// userEmail := "demo@example.com"
// 		// userRole := "Admin"
// 		// userID := "demo-user-123" // Simple string ID

// 		// Try to get real data from cookies if available
// 		if name, err := c.Cookie("name"); err == nil && name != "" {
// 			userName = name
// 		}

// 		if email, err := c.Cookie("email"); err == nil && email != "" {
// 			userEmail = email
// 		}

// 		if role, err := c.Cookie("role"); err == nil && role != "" {
// 			userRole = role
// 		}

// 		if id, err := c.Cookie("id"); err == nil && id != "" {
// 			userID = id
// 		}

// 		log.Printf("Using user context - ID: %s, Name: %s, Email: %s, Role: %s", userID, userName, userEmail, userRole)

// 		// Set user data in context for all handlers to use
// 		c.Set("userName", userName)
// 		c.Set("userEmail", userEmail)
// 		c.Set("userRole", userRole)
// 		c.Set("userID", userID)
// 		c.Set("isAuthenticated", true)

// 		// TODO: Validate session value
// 		c.Next()
// 	}
// }
