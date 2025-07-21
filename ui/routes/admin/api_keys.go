package admin

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
)

func APIKeysHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		c.HTML(http.StatusInternalServerError, "api-keys-table.html", gin.H{
			"error": "Database connection error",
		})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		c.HTML(http.StatusInternalServerError, "api-keys-table.html", gin.H{
			"error": "Database connection error",
		})
		return
	}

	// Get organization ID from query parameter
	orgID := c.Query("org_id")

	var apiKeys []models.APIKey
	var err error

	log.Printf("API Keys request - org_id: '%s'", orgID)

	// Get API keys from database - filtered by organization if specified
	if orgID != "" {
		apiKeys, err = db.GetAPIKeysByOrganization(sqlDB, orgID)
		log.Printf("Found %d API keys for organization %s", len(apiKeys), orgID)
	} else {
		apiKeys, err = db.GetAPIKeysWithOrganizations(sqlDB)
		log.Printf("Found %d total API keys", len(apiKeys))
	}

	if err != nil {
		log.Printf("Failed to get API keys: %v", err)
		c.HTML(http.StatusInternalServerError, "api-keys-table.html", gin.H{
			"error": "Failed to load API keys",
		})
		return
	}

	// Ensure we have a non-nil slice for template rendering
	if apiKeys == nil {
		apiKeys = []models.APIKey{}
	}

	// Render the API keys table template
	log.Printf("Rendering template - apiKeys: %v (len: %d)", apiKeys != nil, len(apiKeys))
	c.HTML(http.StatusOK, "api-keys-table.html", gin.H{
		"apiKeys": apiKeys,
	})
}

func CreateAPIKeyHandler(c *gin.Context) {
	log.Printf("=== CREATE API KEY REQUEST ===")
	log.Printf("Method: %s", c.Request.Method)
	log.Printf("Content-Type: %s", c.GetHeader("Content-Type"))

	// Log raw form data
	c.Request.ParseForm()
	log.Printf("Raw Form Data: %+v", c.Request.Form)

	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		log.Printf("ERROR: Database connection not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("ERROR: Invalid database connection type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Parse form data
	var req models.CreateAPIKeyRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("ERROR: Failed to bind API key request: %v", err)
		log.Printf("Request body: %+v", req)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	log.Printf("SUCCESS: Parsed request: %+v", req)

	// Get current user from context and set as creator
	userData := GetUserContext(c)
	if userID, ok := userData["id"].(string); ok && userID != "" {
		log.Printf("Creating API key for user ID: %s", userID)
		req.UserID = &userID
	} else {
		log.Printf("No user ID found in context, creating as system")
	}

	// Set organization ID from form or use first organization as default
	if req.OrganizationID == "" {
		log.Printf("No organization ID provided, getting first organization as default")
		// Get first organization as default
		orgs, err := db.GetAllOrganizations(sqlDB)
		if err != nil || len(orgs) == 0 {
			log.Printf("ERROR: No organizations available: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "No organizations available"})
			return
		}
		req.OrganizationID = orgs[0].ID
		log.Printf("Using default organization: %s", req.OrganizationID)
	} else {
		log.Printf("Using provided organization ID: %s", req.OrganizationID)
	}

	// Create API key in database
	log.Printf("Creating API key with request: %+v", req)
	response, err := db.CreateAPIKey(sqlDB, req)
	if err != nil {
		log.Printf("ERROR: Failed to create API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	log.Printf("SUCCESS: API key created: %+v", response)

	// Return success response with the new key for modal display
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": response.Message,
		"newKey":  response.FullKey,
		"keyName": response.APIKey.Name,
	})
}

func DeleteAPIKeyHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Get API key ID from URL parameter
	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key ID is required"})
		return
	}

	// Delete API key (soft delete)
	err := db.DeleteAPIKey(sqlDB, keyID)
	if err != nil {
		log.Printf("Failed to delete API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
		return
	}

	// Get organization ID from query parameter for filtering
	orgID := c.Query("org_id")

	var apiKeys []models.APIKey
	// Get updated API keys list and return the table HTML
	if orgID != "" {
		apiKeys, err = db.GetAPIKeysByOrganization(sqlDB, orgID)
	} else {
		apiKeys, err = db.GetAPIKeysWithOrganizations(sqlDB)
	}

	if err != nil {
		log.Printf("Failed to get updated API keys: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh API keys"})
		return
	}

	// Return the updated table HTML for HTMX
	c.HTML(http.StatusOK, "api-keys-table.html", gin.H{
		"apiKeys": apiKeys,
		"message": "API key deleted successfully",
	})
}

func OrganizationsHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Get organizations from database
	organizations, err := db.GetAllOrganizations(sqlDB)
	if err != nil {
		log.Printf("Failed to get organizations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load organizations"})
		return
	}

	// Return JSON response
	c.JSON(http.StatusOK, gin.H{
		"organizations": organizations,
	})
}
