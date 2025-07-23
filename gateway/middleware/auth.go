package middleware

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AccessibleModel represents a model that the organization has access to
type AccessibleModel struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	ModelID           string   `json:"model_id"`
	Provider          string   `json:"provider"`
	IsActive          bool     `json:"is_active"`
	ApiToken          string   `json:"api_token"`
	ApiEndpoint       string   `json:"api_endpoint"`
	TimeoutSeconds    *int     `json:"timeout_seconds,omitempty"`    // Optional timeout in seconds
	MaxRetries        *int     `json:"max_retries,omitempty"`        // Optional max retries
	RetryDelayMs      *int     `json:"retry_delay_ms,omitempty"`     // Optional retry delay in milliseconds
	BackoffMultiplier *float64 `json:"backoff_multiplier,omitempty"` // Optional backoff
}

// APIKeyAuth validates bearer tokens and stores accessible models in context
func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract bearer token
		token := extractBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing or invalid authorization token",
			})
			return
		}

		log.Println("Validating API key:", token)
		// 2. Get database connection
		db := getDatabaseFromContext(c)
		if db == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			return
		}
		log.Println("Database connection found, proceeding with API key validation")

		// 3. Validate token and get organization
		orgID, keyID, err := validateAPIKeyAndGetOrg(db, token)
		if err != nil {
			log.Printf("API key validation failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or inactive API key",
			})
			return
		}
		log.Printf("API key validated successfully for organization %s", orgID)

		// 4. Query accessible models for the organization
		accessibleModels, err := getAccessibleModels(db, orgID)
		if err != nil {
			log.Printf("Warning: Could not fetch accessible models for org %s: %v", orgID, err)
			accessibleModels = []AccessibleModel{} // Empty but not nil
		}
		log.Printf("Found %d accessible models for organization %s", len(accessibleModels), orgID)

		// 5. Store in context for downstream handlers
		c.Set("organization_id", orgID)
		c.Set("api_key_id", keyID)
		c.Set("accessible_models", accessibleModels)
		c.Set("api_key", token)

		log.Printf("Authenticated organization %s with access to %d models", orgID, len(accessibleModels))

		// 6. Update last used timestamp (async)
		go updateAPIKeyLastUsed(db, keyID)

		c.Next()
	}
}

// extractBearerToken extracts the bearer token from Authorization header
func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// Support both "Bearer sk-..." and "sk-..." formats
	var token string
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		token = authHeader
	}

	// Validate token format
	if !strings.HasPrefix(token, "sk-") {
		return ""
	}

	return token
}

// getDatabaseFromContext gets the database connection from gin context
func getDatabaseFromContext(c *gin.Context) *sql.DB {
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		return nil
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		return nil
	}

	return sqlDB
}

// validateAPIKeyAndGetOrg validates the API key and returns organization ID and key ID
func validateAPIKeyAndGetOrg(db *sql.DB, apiKey string) (orgID, keyID string, err error) {
	query := `
		SELECT id, organization_id
		FROM api_keys
		WHERE api_key = $1 AND is_active = true`

	err = db.QueryRow(query, apiKey).Scan(&keyID, &orgID)
	if err != nil {
		return "", "", err
	}

	return orgID, keyID, nil
}

// getAccessibleModels gets models directly from database
func getAccessibleModels(db *sql.DB, orgID string) ([]AccessibleModel, error) {
	return getAccessibleModelsFromDB(db, orgID)
}

// getAccessibleModelsFromDB directly queries database (fallback method)
func getAccessibleModelsFromDB(db *sql.DB, orgID string) ([]AccessibleModel, error) {
	query := `
		SELECT DISTINCT m.id, 
		m.name, 
		m.model_id, 
		m.provider, 
		m.is_active, 
		m.api_token, 
		m.api_endpoint, 
		m.timeout_seconds,
		m.max_retries,
		m.retry_delay_ms,
		m.backoff_multiplier
		FROM models m
		JOIN model_organization_access moa ON m.id = moa.model_id
		WHERE moa.organization_id = $1 AND m.is_active = true
		ORDER BY m.name`

	rows, err := db.Query(query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []AccessibleModel
	for rows.Next() {
		var model AccessibleModel
		err := rows.Scan(
			&model.ID,
			&model.Name,
			&model.ModelID,
			&model.Provider,
			&model.IsActive,
			&model.ApiToken,
			&model.ApiEndpoint,
			&model.TimeoutSeconds, // Optional, can be nil
			&model.MaxRetries,
			&model.RetryDelayMs,
			&model.BackoffMultiplier, // Optional, can be nil
		)
		if err != nil {
			log.Printf("Error scanning model row: %v", err)
			continue
		}
		models = append(models, model)
	}

	return models, nil
}

// updateAPIKeyLastUsed updates the last_used timestamp for the API key
func updateAPIKeyLastUsed(db *sql.DB, keyID string) {
	query := `UPDATE api_keys SET last_used = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := db.Exec(query, keyID)
	if err != nil {
		log.Printf("Failed to update API key last_used: %v", err)
	}
}

// OptionalAPIKeyAuth is a middleware that validates API keys but doesn't require them
// Useful for endpoints that work with both authenticated and unauthenticated requests
func OptionalAPIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract bearer token
		token := extractBearerToken(c)
		if token == "" {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		// 2. Get database connection
		db := getDatabaseFromContext(c)
		if db == nil {
			// No database connection, continue without authentication
			c.Next()
			return
		}

		// 3. Validate token and get organization
		orgID, keyID, err := validateAPIKeyAndGetOrg(db, token)
		if err != nil {
			log.Println("Invalid API key:", err)
			// Invalid API key, but don't block the request for optional auth
			c.Next()
			return
		}

		// 4. Query accessible models for the organization
		accessibleModels, err := getAccessibleModels(db, orgID)
		if err != nil {
			log.Printf("Warning: Could not fetch accessible models for org %s: %v", orgID, err)
			accessibleModels = []AccessibleModel{} // Empty but not nil
		}

		fmt.Println("hereeee", accessibleModels)

		// 5. Store in context for downstream handlers
		c.Set("organization_id", orgID)
		// c.Set("api_key_id", keyID)
		c.Set("accessible_models", accessibleModels)
		c.Set("authenticated", true)
		// c.Set("api_key", token)

		log.Printf("Optionally authenticated organization %s with access to %d models", orgID, len(accessibleModels))

		// 6. Update last used timestamp (async)
		go updateAPIKeyLastUsed(db, keyID)

		c.Next()
	}
}
