package auth

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/db"
)

// Middleware provides authentication middleware for the UI
func Middleware() gin.HandlerFunc {
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
			log.Printf("DEBUG: Found name cookie: %s", name)
		} else {
			log.Printf("DEBUG: No name cookie found")
		}

		if email, err := c.Cookie("email"); err == nil && email != "" {
			userEmail = email
			log.Printf("DEBUG: Found email cookie: %s", email)
		} else {
			log.Printf("DEBUG: No email cookie found")
		}

		if role, err := c.Cookie("role"); err == nil && role != "" {
			userRole = role
			log.Printf("DEBUG: Found role cookie: %s", role)
		} else {
			// Default role if not found in cookie
			userRole = "Admin"
			log.Printf("DEBUG: No role cookie found, using default: %s", userRole)
		}

		// Check if we have Azure OID
		var azureOID string
		if oid, err := c.Cookie("oid"); err == nil && oid != "" {
			azureOID = oid
			log.Printf("DEBUG: Found Azure OID in cookie: %s", azureOID)
		} else {
			log.Printf("DEBUG: No Azure OID found in cookie")
		}

		// Get the actual user ID from database using email or Azure OID
		if userEmail != "" {
			database, exists := c.Get("db")
			if exists {
				if sqlDB, ok := database.(*sql.DB); ok {
					log.Printf("DEBUG: Looking up user by email: %s", userEmail)
					user, err := db.GetUserByEmail(sqlDB, userEmail)
					if err == nil && user != nil {
						userID = user.ID
						log.Printf("DEBUG: Found user ID %s for email %s", userID, userEmail)
					} else {
						log.Printf("DEBUG: Failed to get user by email %s: %v", userEmail, err)

						// Try looking up by Azure OID if we have it
						if azureOID != "" {
							log.Printf("DEBUG: Trying lookup by Azure OID: %s", azureOID)
							user, err = db.GetUserByAzureOID(sqlDB, azureOID)
							if err == nil && user != nil {
								userID = user.ID
								log.Printf("DEBUG: Found user ID %s for Azure OID %s", userID, azureOID)
							} else {
								log.Printf("DEBUG: Failed to get user by Azure OID %s: %v", azureOID, err)
								userID = userEmail // Fallback to email as before
							}
						} else {
							userID = userEmail // Fallback to email as before
						}
					}
				} else {
					log.Printf("DEBUG: No valid DB connection")
					userID = userEmail // Fallback to email if no DB connection
				}
			} else {
				log.Printf("DEBUG: No DB found in context")
				userID = userEmail // Fallback to email if no DB in context
			}
		} else {
			log.Printf("DEBUG: No user email found")
		}

		log.Printf("DEBUG: Final userID being set: %s", userID)

		// Set user data in context for all handlers to use
		c.Set("userName", userName)
		c.Set("userEmail", userEmail)
		c.Set("userRole", userRole)
		c.Set("userID", userID)
		c.Set("isAuthenticated", true)

		// Also set enhanced format keys for compatibility
		c.Set("user_name", userName)
		c.Set("user_email", userEmail)
		c.Set("user_id", userID)

		log.Printf("DEBUG: Context set with userID: %s, userEmail: %s, userName: %s", userID, userEmail, userName)

		c.Next()
	}
}
