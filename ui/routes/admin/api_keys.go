package admin

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
	"github.com/like-mike/relai-gateway/ui/auth"
)

func APIKeysHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		acceptHeader := c.GetHeader("Accept")
		if acceptHeader == "application/json" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		} else {
			c.HTML(http.StatusInternalServerError, "api-keys-table.html", gin.H{
				"error": "Database connection error",
			})
		}
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		acceptHeader := c.GetHeader("Accept")
		if acceptHeader == "application/json" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		} else {
			c.HTML(http.StatusInternalServerError, "api-keys-table.html", gin.H{
				"error": "Database connection error",
			})
		}
		return
	}

	// Get organization ID from query parameter
	orgID := c.Query("org_id")

	// Get user context for RBAC
	userContext := auth.GetUserContext(c)
	userID, ok := userContext["id"].(string)
	if !ok || userID == "" {
		log.Printf("No user ID found in context for API keys request")
		acceptHeader := c.GetHeader("Accept")
		if acceptHeader == "application/json" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		} else {
			c.HTML(http.StatusUnauthorized, "api-keys-table.html", gin.H{
				"error": "User authentication required",
			})
		}
		return
	}

	// Get user's organization memberships for RBAC
	memberships, err := db.GetUserOrganizationMemberships(sqlDB, userID)
	if err != nil {
		log.Printf("Failed to get user memberships: %v", err)
		acceptHeader := c.GetHeader("Accept")
		if acceptHeader == "application/json" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user permissions"})
		} else {
			c.HTML(http.StatusInternalServerError, "api-keys-table.html", gin.H{
				"error": "Failed to load user permissions",
			})
		}
		return
	}

	var apiKeys []models.APIKey

	log.Printf("API Keys request - org_id: '%s', user_id: %s", orgID, userID)

	// Get API keys from database - filtered by organization if specified
	if orgID != "" {
		// Validate user has access to the requested organization
		if _, hasAccess := memberships[orgID]; !hasAccess {
			log.Printf("User %s denied access to organization %s", userID, orgID)
			acceptHeader := c.GetHeader("Accept")
			if acceptHeader == "application/json" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to organization"})
			} else {
				c.HTML(http.StatusForbidden, "api-keys-table.html", gin.H{
					"error": "Access denied to organization",
				})
			}
			return
		}

		apiKeys, err = db.GetAPIKeysByOrganization(sqlDB, orgID)
		log.Printf("Found %d API keys for organization %s", len(apiKeys), orgID)
	} else {
		// Get API keys for all organizations the user has access to
		apiKeys, err = db.GetAPIKeysWithOrganizations(sqlDB)
		if err == nil {
			// Filter API keys to only those from organizations the user has access to
			var filteredAPIKeys []models.APIKey
			for _, apiKey := range apiKeys {
				if _, hasAccess := memberships[apiKey.OrganizationID]; hasAccess {
					filteredAPIKeys = append(filteredAPIKeys, apiKey)
				}
			}
			apiKeys = filteredAPIKeys
		}
		log.Printf("Found %d total API keys for user's accessible organizations", len(apiKeys))
	}

	if err != nil {
		log.Printf("Failed to get API keys: %v", err)
		acceptHeader := c.GetHeader("Accept")
		if acceptHeader == "application/json" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load API keys"})
		} else {
			c.HTML(http.StatusInternalServerError, "api-keys-table.html", gin.H{
				"error": "Failed to load API keys",
			})
		}
		return
	}

	// Ensure we have a non-nil slice for template rendering
	if apiKeys == nil {
		apiKeys = []models.APIKey{}
	}

	// Check Accept header to determine response format
	acceptHeader := c.GetHeader("Accept")
	if acceptHeader == "application/json" {
		// Return JSON response for JavaScript consumption
		c.JSON(http.StatusOK, gin.H{
			"api_keys": apiKeys,
		})
		return
	}

	// Render the API keys table template (default behavior for HTMX)
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
	userData := auth.GetUserContext(c)
	if userID, ok := userData["id"].(string); ok && userID != "" {
		log.Printf("Creating API key for user ID: %s", userID)
		req.UserID = &userID

		// Get user's organization memberships for RBAC validation
		memberships, err := db.GetUserOrganizationMemberships(sqlDB, userID)
		if err != nil {
			log.Printf("Failed to get user memberships: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user permissions"})
			return
		}

		// Set organization ID from form or use first accessible organization as default
		if req.OrganizationID == "" {
			log.Printf("No organization ID provided, finding first accessible organization")
			// Get first organization the user has access to as default
			var firstAccessibleOrgID string
			for orgID := range memberships {
				firstAccessibleOrgID = orgID
				break
			}
			if firstAccessibleOrgID == "" {
				log.Printf("ERROR: User has no accessible organizations")
				c.JSON(http.StatusForbidden, gin.H{"error": "No accessible organizations"})
				return
			}
			req.OrganizationID = firstAccessibleOrgID
			log.Printf("Using default accessible organization: %s", req.OrganizationID)
		} else {
			log.Printf("Validating provided organization ID: %s", req.OrganizationID)
			// Validate user has access to the specified organization
			if _, hasAccess := memberships[req.OrganizationID]; !hasAccess {
				log.Printf("User %s denied access to create API key in organization %s", userID, req.OrganizationID)
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to organization"})
				return
			}
			log.Printf("User has access to organization: %s", req.OrganizationID)
		}
	} else {
		log.Printf("No user ID found in context, cannot create API key")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
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

	// Get user context for RBAC
	userContext := auth.GetUserContext(c)
	userID, ok := userContext["id"].(string)
	if !ok || userID == "" {
		log.Printf("No user ID found in context for delete API key request")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	// Get user's organization memberships for RBAC
	memberships, err := db.GetUserOrganizationMemberships(sqlDB, userID)
	if err != nil {
		log.Printf("Failed to get user memberships: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user permissions"})
		return
	}

	// Get API key ID from URL parameter
	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key ID is required"})
		return
	}

	// Get all API keys to find the one we want to delete and check its organization
	allAPIKeys, err := db.GetAPIKeysWithOrganizations(sqlDB)
	if err != nil {
		log.Printf("Failed to get API keys for validation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate API key"})
		return
	}

	// Find the API key and validate organization access
	var targetAPIKey *models.APIKey
	for _, apiKey := range allAPIKeys {
		if apiKey.ID == keyID {
			targetAPIKey = &apiKey
			break
		}
	}

	if targetAPIKey == nil {
		log.Printf("API key %s not found", keyID)
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	// Validate user has access to the API key's organization
	if _, hasAccess := memberships[targetAPIKey.OrganizationID]; !hasAccess {
		log.Printf("User %s denied access to delete API key from organization %s", userID, targetAPIKey.OrganizationID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to organization"})
		return
	}

	// Delete API key (soft delete)
	err = db.DeleteAPIKey(sqlDB, keyID)
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
		// Validate user has access to the requested organization
		if _, hasAccess := memberships[orgID]; !hasAccess {
			log.Printf("User %s denied access to organization %s for refresh", userID, orgID)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to organization"})
			return
		}
		apiKeys, err = db.GetAPIKeysByOrganization(sqlDB, orgID)
	} else {
		// Get API keys for all organizations the user has access to
		apiKeys, err = db.GetAPIKeysWithOrganizations(sqlDB)
		if err == nil {
			// Filter API keys to only those from organizations the user has access to
			var filteredAPIKeys []models.APIKey
			for _, apiKey := range apiKeys {
				if _, hasAccess := memberships[apiKey.OrganizationID]; hasAccess {
					filteredAPIKeys = append(filteredAPIKeys, apiKey)
				}
			}
			apiKeys = filteredAPIKeys
		}
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

	// Get user context for RBAC
	userContext := auth.GetUserContext(c)
	userID, ok := userContext["id"].(string)
	if !ok || userID == "" {
		log.Printf("No user ID found in context for organizations request")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	// Get user's organization memberships
	memberships, err := db.GetUserOrganizationMemberships(sqlDB, userID)
	if err != nil {
		log.Printf("Failed to get user memberships: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user permissions"})
		return
	}

	// Get all organizations and filter by user memberships
	allOrganizations, err := db.GetAllOrganizations(sqlDB)
	if err != nil {
		log.Printf("Failed to get organizations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load organizations"})
		return
	}

	// Filter organizations to only those the user has access to
	var userOrganizations []models.Organization
	for _, org := range allOrganizations {
		if _, hasAccess := memberships[org.ID]; hasAccess {
			userOrganizations = append(userOrganizations, org)
		}
	}

	log.Printf("User %s has access to %d of %d organizations", userID, len(userOrganizations), len(allOrganizations))

	// Return JSON response with filtered organizations
	c.JSON(http.StatusOK, gin.H{
		"organizations": userOrganizations,
	})
}

// POST /api/completions-proxy
func CompletionsProxyHandler(c *gin.Context) {
	type ProxyRequest struct {
		OrganizationID string `json:"organization_id"`
		APIKeyID       string `json:"api_key_id"`
		ModelID        string `json:"model_id"`
		Message        string `json:"message"`
		Stream         bool   `json:"stream"`
	}
	var req ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ProxyHandler: Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	log.Printf("ProxyHandler: Incoming request: %+v", req)

	fmt.Println()
	// Lookup API key securely from DB using req.APIKeyID
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}
	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid database connection"})
		return
	}
	apiKey, err := db.GetAPIKeyByID(sqlDB, req.APIKeyID)
	if err != nil || apiKey == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	// Build the request to the completions API
	payload := map[string]interface{}{
		"model":    req.ModelID,
		"messages": []map[string]string{{"role": "user", "content": req.Message}},
		"stream":   req.Stream,
	}
	body, _ := json.Marshal(payload)
	log.Printf("ProxyHandler: Upstream payload: %s", string(body))
	providerURL := "http://localhost:8081/v1/chat/completions"

	httpReq, err := http.NewRequest("POST", providerURL, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build upstream request"})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Upstream provider error"})
		return
	}
	defer resp.Body.Close()

	// Handle streaming vs non-streaming responses
	if req.Stream {
		// Copy all headers from upstream response
		for k, v := range resp.Header {
			for _, vv := range v {
				c.Writer.Header().Add(k, vv)
			}
		}
		c.Status(resp.StatusCode)

		// Stream response with proper flushing for real-time delivery
		buffer := make([]byte, 1024)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
					log.Printf("ProxyHandler: Error writing streaming chunk: %v", writeErr)
					return
				}
				// Flush immediately to ensure real-time streaming
				if flusher, ok := c.Writer.(http.Flusher); ok {
					flusher.Flush()
				}
			}
			if err != nil {
				if err == io.EOF {
					log.Printf("ProxyHandler: Streaming completed")
					break
				}
				log.Printf("ProxyHandler: Error reading streaming response: %v", err)
				break
			}
		}
	} else {
		// Forward non-streaming response
		c.Status(resp.StatusCode)
		for k, v := range resp.Header {
			for _, vv := range v {
				c.Writer.Header().Add(k, vv)
			}
		}
		io.Copy(c.Writer, resp.Body)
	}
}

// TEMP: Test endpoint for debugging streaming without auth
func TestStreamingHandler(c *gin.Context) {
	log.Printf("TestStreamingHandler: Starting test streaming endpoint")

	type TestRequest struct {
		Message string `json:"message"`
		Stream  bool   `json:"stream"`
	}

	var req TestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("TestStreamingHandler: Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Hardcode values for testing
	testMessage := req.Message
	if testMessage == "" {
		testMessage = "Hello, how are you?"
	}

	log.Printf("TestStreamingHandler: Message: %s, Stream: %v", testMessage, req.Stream)

	// Build the request to the completions API
	payload := map[string]interface{}{
		"model":    "gpt-4",
		"messages": []map[string]string{{"role": "user", "content": testMessage}},
		"stream":   req.Stream,
	}
	body, _ := json.Marshal(payload)
	log.Printf("TestStreamingHandler: Upstream payload: %s", string(body))

	// Hardcode provider URL and API key (replace with your actual OpenAI API key)
	providerURL := "https://api.openai.com/v1/chat/completions"
	hardcodedAPIKey := "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" // REPLACE THIS WITH YOUR ACTUAL API KEY

	httpReq, err := http.NewRequest("POST", providerURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("TestStreamingHandler: Failed to build request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build upstream request"})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+hardcodedAPIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Printf("TestStreamingHandler: Upstream request failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Upstream provider error"})
		return
	}
	defer resp.Body.Close()

	log.Printf("TestStreamingHandler: Upstream response status: %d", resp.StatusCode)

	// Handle streaming vs non-streaming responses
	if req.Stream {
		log.Printf("TestStreamingHandler: Setting up streaming response")
		// Copy all headers from upstream response
		for k, v := range resp.Header {
			for _, vv := range v {
				c.Writer.Header().Add(k, vv)
			}
		}
		c.Status(resp.StatusCode)

		// Stream response with proper flushing for real-time delivery
		buffer := make([]byte, 1024)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
					log.Printf("TestStreamingHandler: Error writing streaming chunk: %v", writeErr)
					return
				}
				// Flush immediately to ensure real-time streaming
				if flusher, ok := c.Writer.(http.Flusher); ok {
					flusher.Flush()
				}
				log.Printf("TestStreamingHandler: Wrote and flushed %d bytes", n)
			}
			if err != nil {
				if err == io.EOF {
					log.Printf("TestStreamingHandler: Streaming completed")
					break
				}
				log.Printf("TestStreamingHandler: Error reading streaming response: %v", err)
				break
			}
		}
	} else {
		log.Printf("TestStreamingHandler: Setting up non-streaming response")
		// Forward non-streaming response
		c.Status(resp.StatusCode)
		for k, v := range resp.Header {
			for _, vv := range v {
				c.Writer.Header().Add(k, vv)
			}
		}
		io.Copy(c.Writer, resp.Body)
	}

	log.Printf("TestStreamingHandler: Response complete")
}
