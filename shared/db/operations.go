package db

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/lib/pq"
	"github.com/like-mike/relai-gateway/shared/models"
)

// Organizations operations
func GetAllOrganizations(db *sql.DB) ([]models.Organization, error) {
	query := `SELECT id, name, description, is_active, created_at, updated_at 
			  FROM organizations 
			  WHERE is_active = true 
			  ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var organizations []models.Organization
	for rows.Next() {
		var org models.Organization
		err := rows.Scan(&org.ID, &org.Name, &org.Description, &org.IsActive, &org.CreatedAt, &org.UpdatedAt)
		if err != nil {
			return nil, err
		}
		organizations = append(organizations, org)
	}

	return organizations, nil
}

// GetOrganizationByID retrieves a single organization by ID
func GetOrganizationByID(db *sql.DB, id string) (*models.Organization, error) {
	query := `
		SELECT id, name, description, is_active, created_at, updated_at
		FROM organizations
		WHERE id = $1`

	var org models.Organization
	err := db.QueryRow(query, id).Scan(
		&org.ID, &org.Name, &org.Description, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &org, nil
}

// API Keys operations
func GetAPIKeysWithOrganizations(db *sql.DB) ([]models.APIKey, error) {
	query := `
		SELECT
			ak.id, ak.name, ak.description, ak.key_prefix, ak.organization_id,
			ak.user_id, ak.type, ak.max_tokens, ak.permissions, ak.is_active,
			ak.last_used, ak.created_at, ak.updated_at,
			o.name as org_name
		FROM api_keys ak
		JOIN organizations o ON ak.organization_id = o.id
		WHERE ak.is_active = true
		ORDER BY ak.created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apiKeys []models.APIKey
	for rows.Next() {
		var key models.APIKey
		var orgName string

		err := rows.Scan(
			&key.ID, &key.Name, &key.Description, &key.KeyPrefix, &key.OrganizationID,
			&key.UserID, &key.Type, &key.MaxTokens, &key.Permissions, &key.IsActive,
			&key.LastUsed, &key.CreatedAt, &key.UpdatedAt, &orgName,
		)
		if err != nil {
			return nil, err
		}

		// Attach organization info
		key.Organization = &models.Organization{
			ID:   key.OrganizationID,
			Name: orgName,
		}

		apiKeys = append(apiKeys, key)
	}

	return apiKeys, nil
}

func GetAPIKeysByOrganization(db *sql.DB, orgID string) ([]models.APIKey, error) {
	query := `
		SELECT
			ak.id, ak.name, ak.description, ak.key_prefix, ak.organization_id,
			ak.user_id, ak.type, ak.max_tokens, ak.permissions, ak.is_active,
			ak.last_used, ak.created_at, ak.updated_at,
			o.name as org_name
		FROM api_keys ak
		JOIN organizations o ON ak.organization_id = o.id
		WHERE ak.is_active = true AND ak.organization_id = $1
		ORDER BY ak.created_at DESC`

	rows, err := db.Query(query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apiKeys []models.APIKey
	for rows.Next() {
		var key models.APIKey
		var orgName string

		err := rows.Scan(
			&key.ID, &key.Name, &key.Description, &key.KeyPrefix, &key.OrganizationID,
			&key.UserID, &key.Type, &key.MaxTokens, &key.Permissions, &key.IsActive,
			&key.LastUsed, &key.CreatedAt, &key.UpdatedAt, &orgName,
		)
		if err != nil {
			return nil, err
		}

		// Attach organization info
		key.Organization = &models.Organization{
			ID:   key.OrganizationID,
			Name: orgName,
		}

		apiKeys = append(apiKeys, key)
	}

	return apiKeys, nil
}

func CreateAPIKey(db *sql.DB, req models.CreateAPIKeyRequest) (*models.CreateAPIKeyResponse, error) {
	// Generate a secure API key
	fullKey, keyPrefix, keyHash, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Set default max tokens if not provided
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 10000
	}

	// Convert permissions to PostgreSQL array
	permissions := pq.StringArray(req.Permissions)

	query := `
		INSERT INTO api_keys (name, description, key_hash, key_prefix, organization_id, user_id, type, max_tokens, permissions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	var apiKey models.APIKey
	err = db.QueryRow(query,
		req.Name, req.Description, keyHash, keyPrefix,
		req.OrganizationID, req.UserID, req.Type, maxTokens, permissions,
	).Scan(&apiKey.ID, &apiKey.CreatedAt, &apiKey.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Populate the rest of the fields
	apiKey.Name = req.Name
	apiKey.Description = req.Description
	apiKey.KeyPrefix = keyPrefix
	apiKey.OrganizationID = req.OrganizationID
	apiKey.UserID = req.UserID
	apiKey.Type = req.Type
	apiKey.MaxTokens = maxTokens
	apiKey.Permissions = permissions
	apiKey.IsActive = true

	// Get organization name
	var orgName string
	err = db.QueryRow("SELECT name FROM organizations WHERE id = $1", req.OrganizationID).Scan(&orgName)
	if err == nil {
		apiKey.Organization = &models.Organization{
			ID:   req.OrganizationID,
			Name: orgName,
		}
	}

	return &models.CreateAPIKeyResponse{
		APIKey:  apiKey,
		FullKey: fullKey,
		Message: "API key created successfully",
	}, nil
}

func DeleteAPIKey(db *sql.DB, keyID string) error {
	query := `UPDATE api_keys SET is_active = false, updated_at = NOW() WHERE id = $1`
	_, err := db.Exec(query, keyID)
	return err
}

// Models operations
func GetModelsWithOrganizations(db *sql.DB) ([]models.Model, error) {
	// First get all models
	query := `SELECT id, name, description, provider, model_id, is_active, created_at, updated_at 
			  FROM models 
			  ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modelsMap = make(map[string]*models.Model)
	var modelsList []models.Model

	for rows.Next() {
		var model models.Model
		err := rows.Scan(&model.ID, &model.Name, &model.Description, &model.Provider,
			&model.ModelID, &model.IsActive, &model.CreatedAt, &model.UpdatedAt)
		if err != nil {
			return nil, err
		}
		model.Organizations = []models.Organization{}
		modelsMap[model.ID] = &model
		modelsList = append(modelsList, model)
	}

	// Now get organization access for each model
	accessQuery := `
		SELECT moa.model_id, o.id, o.name 
		FROM model_organization_access moa
		JOIN organizations o ON moa.organization_id = o.id
		WHERE o.is_active = true`

	accessRows, err := db.Query(accessQuery)
	if err != nil {
		return modelsList, nil // Return models without organization info if this fails
	}
	defer accessRows.Close()

	for accessRows.Next() {
		var modelID, orgID, orgName string
		err := accessRows.Scan(&modelID, &orgID, &orgName)
		if err != nil {
			continue
		}

		if model, exists := modelsMap[modelID]; exists {
			model.Organizations = append(model.Organizations, models.Organization{
				ID:   orgID,
				Name: orgName,
			})
		}
	}

	// Update the slice with the organization info
	for i, model := range modelsList {
		if modelWithOrgs, exists := modelsMap[model.ID]; exists {
			modelsList[i].Organizations = modelWithOrgs.Organizations
		}
	}

	return modelsList, nil
}

func CreateModel(db *sql.DB, req models.CreateModelRequest) (*models.Model, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create the model
	query := `
		INSERT INTO models (name, description, provider, model_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	var model models.Model
	err = tx.QueryRow(query, req.Name, req.Description, req.Provider, req.ModelID).
		Scan(&model.ID, &model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Set the fields
	model.Name = req.Name
	model.Description = req.Description
	model.Provider = req.Provider
	model.ModelID = req.ModelID
	model.IsActive = true

	// Add organization access
	if len(req.OrgIDs) > 0 {
		accessQuery := `INSERT INTO model_organization_access (model_id, organization_id) VALUES ($1, $2)`
		for _, orgID := range req.OrgIDs {
			_, err = tx.Exec(accessQuery, model.ID, orgID)
			if err != nil {
				return nil, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &model, nil
}

func DeleteModel(db *sql.DB, modelID string) error {
	query := `UPDATE models SET is_active = false, updated_at = NOW() WHERE id = $1`
	_, err := db.Exec(query, modelID)
	return err
}

// Quota operations
func GetOrganizationQuota(db *sql.DB, orgID string) (*models.OrganizationQuota, error) {
	query := `SELECT id, organization_id, total_quota, used_tokens, reset_date, created_at, updated_at 
			  FROM organization_quotas 
			  WHERE organization_id = $1`

	var quota models.OrganizationQuota
	err := db.QueryRow(query, orgID).Scan(
		&quota.ID, &quota.OrganizationID, &quota.TotalQuota,
		&quota.UsedTokens, &quota.ResetDate, &quota.CreatedAt, &quota.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &quota, nil
}

func GetQuotaStatsForFirstOrg(db *sql.DB) (*models.QuotaStats, error) {
	// Get the first organization's quota for demo purposes
	query := `SELECT total_quota, used_tokens 
			  FROM organization_quotas 
			  ORDER BY created_at 
			  LIMIT 1`

	var quota models.OrganizationQuota
	err := db.QueryRow(query).Scan(&quota.TotalQuota, &quota.UsedTokens)
	if err != nil {
		// Return default stats if no quota found
		return &models.QuotaStats{
			TotalUsage:     "0",
			RemainingQuota: "100K",
			PercentUsed:    "0.0%",
		}, nil
	}

	stats := quota.CalculateQuotaStats()
	return &stats, nil
}

// Helper functions
func generateAPIKey() (fullKey, prefix, hash string, err error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	_, err = rand.Read(bytes)
	if err != nil {
		return "", "", "", err
	}

	// Create the full key with sk- prefix
	fullKey = "sk-" + hex.EncodeToString(bytes)

	// Extract prefix (first 7 characters for display)
	prefix = fullKey[:7] + "..."

	// Create hash for storage
	hasher := sha256.New()
	hasher.Write([]byte(fullKey))
	hash = hex.EncodeToString(hasher.Sum(nil))

	return fullKey, prefix, hash, nil
}

// UserOperations
func GetUserByUsername(db *sql.DB, username string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, is_active, last_login, created_at, updated_at 
			  FROM users 
			  WHERE username = $1 AND is_active = true`

	var user models.User
	err := db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func UpdateUserLastLogin(db *sql.DB, userID string) error {
	query := `UPDATE users SET last_login = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := db.Exec(query, userID)
	return err
}
