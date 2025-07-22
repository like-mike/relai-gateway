package middleware

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessibleModel represents a model that the organization has access to
type AccessibleModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ModelID  string `json:"model_id"`
	Provider string `json:"provider"`
	IsActive bool   `json:"is_active"`
	ApiToken string `json:"api_token"`
}

// ModelCache provides thread-safe caching of organization accessible models
type ModelCache struct {
	cache    map[string][]AccessibleModel
	mutex    sync.RWMutex
	lastLoad time.Time
	isLoaded bool
}

// Global cache instance
var modelCache = &ModelCache{
	cache: make(map[string][]AccessibleModel),
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

		// 2. Get database connection
		db := getDatabaseFromContext(c)
		if db == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			return
		}

		// 3. Validate token and get organization
		orgID, keyID, err := validateAPIKeyAndGetOrg(db, token)
		if err != nil {
			log.Printf("API key validation failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or inactive API key",
			})
			return
		}

		// 4. Query accessible models for the organization
		accessibleModels, err := getAccessibleModels(db, orgID)
		if err != nil {
			log.Printf("Warning: Could not fetch accessible models for org %s: %v", orgID, err)
			accessibleModels = []AccessibleModel{} // Empty but not nil
		}

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

// Public functions for cache management

// InvalidateModelCache clears the entire model cache
func InvalidateModelCache() {
	modelCache.InvalidateCache()
}

// InvalidateModelCacheForOrg clears cache for a specific organization
func InvalidateModelCacheForOrg(orgID string) {
	modelCache.InvalidateOrgCache(orgID)
}

// ReloadModelCache forces a complete cache reload from database
func ReloadModelCache(db *sql.DB) error {
	return modelCache.LoadAllFromDB(db)
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

// InitializeModelCache preloads all organization-model mappings into cache
func InitializeModelCache(db *sql.DB) error {
	log.Printf("Initializing model cache...")
	start := time.Now()

	err := modelCache.LoadAllFromDB(db)
	if err != nil {
		return err
	}

	log.Printf("Model cache initialized in %v with %d organizations",
		time.Since(start), len(modelCache.cache))
	return nil
}

// LoadAllFromDB loads all organization-model mappings from database
func (mc *ModelCache) LoadAllFromDB(db *sql.DB) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// Clear existing cache
	mc.cache = make(map[string][]AccessibleModel)

	// Query all organization-model mappings
	query := `
		SELECT DISTINCT moa.organization_id, m.id, m.name, m.model_id, m.provider, m.is_active, m.api_token
		FROM models m
		JOIN model_organization_access moa ON m.id = moa.model_id
		WHERE m.is_active = true
		ORDER BY moa.organization_id, m.name`

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orgID string
		var model AccessibleModel
		err := rows.Scan(&orgID, &model.ID, &model.Name, &model.ModelID,
			&model.Provider, &model.IsActive, &model.ApiToken)
		if err != nil {
			log.Printf("Error scanning model cache row: %v", err)
			continue
		}

		mc.cache[orgID] = append(mc.cache[orgID], model)
	}

	mc.lastLoad = time.Now()
	mc.isLoaded = true
	return nil
}

// GetModelsForOrg returns cached models for an organization
func (mc *ModelCache) GetModelsForOrg(orgID string) ([]AccessibleModel, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if !mc.isLoaded {
		return nil, false
	}

	models, exists := mc.cache[orgID]
	if !exists {
		// Organization exists but has no accessible models
		return []AccessibleModel{}, true
	}

	// Return a copy to prevent external modification
	result := make([]AccessibleModel, len(models))
	copy(result, models)
	return result, true
}

// InvalidateCache clears the cache and forces reload on next access
func (mc *ModelCache) InvalidateCache() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.cache = make(map[string][]AccessibleModel)
	mc.isLoaded = false
	log.Printf("Model cache invalidated")
}

// InvalidateOrgCache clears cache for a specific organization
func (mc *ModelCache) InvalidateOrgCache(orgID string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	delete(mc.cache, orgID)
	log.Printf("Model cache invalidated for organization: %s", orgID)
}

// getAccessibleModels gets models from cache, falls back to database if cache miss
func getAccessibleModels(db *sql.DB, orgID string) ([]AccessibleModel, error) {
	// Try to get from cache first
	if models, found := modelCache.GetModelsForOrg(orgID); found {
		return models, nil
	}

	// Cache miss - reload entire cache and try again
	log.Printf("Cache miss for org %s, reloading cache...", orgID)
	err := modelCache.LoadAllFromDB(db)
	if err != nil {
		// Cache reload failed, fall back to direct database query
		log.Printf("Cache reload failed, falling back to direct query: %v", err)
		return getAccessibleModelsFromDB(db, orgID)
	}

	// Try cache again after reload
	if models, found := modelCache.GetModelsForOrg(orgID); found {
		return models, nil
	}

	// Organization not found even after cache reload - return empty slice
	return []AccessibleModel{}, nil
}

// getAccessibleModelsFromDB directly queries database (fallback method)
func getAccessibleModelsFromDB(db *sql.DB, orgID string) ([]AccessibleModel, error) {
	query := `
		SELECT DISTINCT m.id, m.name, m.model_id, m.provider, m.is_active, m.api_token
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
		err := rows.Scan(&model.ID, &model.Name, &model.ModelID, &model.Provider, &model.IsActive, &model.ApiToken)
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

		// 5. Store in context for downstream handlers
		c.Set("organization_id", orgID)
		c.Set("api_key_id", keyID)
		c.Set("accessible_models", accessibleModels)
		c.Set("authenticated", true)
		c.Set("api_key", token)

		log.Printf("Optionally authenticated organization %s with access to %d models", orgID, len(accessibleModels))

		// 6. Update last used timestamp (async)
		go updateAPIKeyLastUsed(db, keyID)

		c.Next()
	}
}
