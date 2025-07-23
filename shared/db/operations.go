package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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

// SyncUserOrganizationMemberships syncs user's organization memberships based on AD groups
func SyncUserOrganizationMemberships(db *sql.DB, userID string, userADGroups []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get current user organization memberships
	currentMemberships := make(map[string]string) // orgID -> roleType
	membershipQuery := `
		SELECT organization_id, role_name
		FROM user_organizations
		WHERE user_id = $1`

	rows, err := tx.Query(membershipQuery, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var orgID, roleName string
			if err := rows.Scan(&orgID, &roleName); err == nil {
				currentMemberships[orgID] = roleName
			}
		}
	}

	// Get organization AD group mappings
	// Changed to handle multiple roles per group: orgID -> {groupID -> []roleType}
	orgMappings := make(map[string]map[string][]string) // orgID -> {groupID -> []roleType}

	// Enhanced debug logging
	fmt.Printf("=== SYNC DEBUG: Looking for organizations mapped to user's %d AD groups ===\n", len(userADGroups))
	for i, group := range userADGroups {
		fmt.Printf("User AD Group %d: %s\n", i+1, group)
	}

	// First, let's check what's actually in the organization_ad_groups table
	debugQuery := `SELECT organization_id, ad_group_id, role_type FROM organization_ad_groups WHERE is_active = true`
	debugRows, err := tx.Query(debugQuery)
	if err == nil {
		defer debugRows.Close()
		fmt.Printf("=== All active AD group mappings in database: ===\n")
		for debugRows.Next() {
			var orgID, groupID, roleType string
			if err := debugRows.Scan(&orgID, &groupID, &roleType); err == nil {
				fmt.Printf("DB Mapping: Org=%s, Group=%s, Role=%s\n", orgID, groupID, roleType)
			}
		}
	} else {
		fmt.Printf("Error querying organization_ad_groups: %v\n", err)
	}

	mappingQuery := `
		SELECT organization_id, ad_group_id, role_type
		FROM organization_ad_groups
		WHERE is_active = true AND ad_group_id = ANY($1)`

	if len(userADGroups) > 0 {
		fmt.Printf("Executing query with user groups: %v\n", userADGroups)
		rows, err = tx.Query(mappingQuery, pq.Array(userADGroups))
		if err != nil {
			fmt.Printf("Error in AD group mapping query: %v\n", err)
		} else {
			defer rows.Close()
			matchCount := 0
			for rows.Next() {
				var orgID, groupID, roleType string
				if err := rows.Scan(&orgID, &groupID, &roleType); err == nil {
					if orgMappings[orgID] == nil {
						orgMappings[orgID] = make(map[string][]string)
					}
					orgMappings[orgID][groupID] = append(orgMappings[orgID][groupID], roleType)
					matchCount++
					fmt.Printf("MATCHED: User group %s -> Org %s with role %s\n", groupID, orgID, roleType)
				}
			}
			fmt.Printf("Total matches found: %d\n", matchCount)
		}
	} else {
		fmt.Printf("No user AD groups to check\n")
	}

	// Determine new memberships based on AD groups
	newMemberships := make(map[string]string) // orgID -> roleType
	fmt.Printf("=== PROCESSING NEW MEMBERSHIPS ===\n")

	for orgID, groupMappings := range orgMappings {
		fmt.Printf("Processing organization: %s\n", orgID)
		userRolesInOrg := []string{} // Collect all roles user has in this org

		for groupID, roleTypes := range groupMappings {
			fmt.Printf("  Checking group %s with roles %v\n", groupID, roleTypes)
			for _, userGroup := range userADGroups {
				if userGroup == groupID {
					fmt.Printf("  USER MATCH: User is in group %s, found roles %v\n", groupID, roleTypes)
					userRolesInOrg = append(userRolesInOrg, roleTypes...)
				}
			}
		}

		// Now determine the highest privilege role for this organization
		if len(userRolesInOrg) > 0 {
			finalRole := "member" // default to lowest privilege
			for _, role := range userRolesInOrg {
				if role == "admin" {
					finalRole = "admin" // admin always wins
					break
				}
			}
			fmt.Printf("  FINAL ROLE for org %s: %s (from roles: %v)\n", orgID, finalRole, userRolesInOrg)
			newMemberships[orgID] = finalRole
		}
	}

	fmt.Printf("=== FINAL NEW MEMBERSHIPS ===\n")
	for orgID, roleType := range newMemberships {
		fmt.Printf("Org %s -> Role %s\n", orgID, roleType)
	}

	// Remove user from organizations they should no longer be in
	for orgID := range currentMemberships {
		if _, shouldBeIn := newMemberships[orgID]; !shouldBeIn {
			_, err = tx.Exec(`DELETE FROM user_organizations WHERE user_id = $1 AND organization_id = $2`, userID, orgID)
			if err != nil {
				return err
			}
		}
	}

	// Add or update user memberships for organizations they should be in
	for orgID, roleType := range newMemberships {
		// Insert or update membership using role_name directly
		_, err = tx.Exec(`
			INSERT INTO user_organizations (user_id, organization_id, role_name)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, organization_id)
			DO UPDATE SET role_name = EXCLUDED.role_name`, userID, orgID, roleType)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetUserOrganizationMemberships gets user's current organization memberships
func GetUserOrganizationMemberships(db *sql.DB, userID string) (map[string]string, error) {
	memberships := make(map[string]string) // orgID -> roleName

	query := `
		SELECT o.id, o.name, uo.role_name
		FROM user_organizations uo
		JOIN organizations o ON uo.organization_id = o.id
		WHERE uo.user_id = $1 AND o.is_active = true`

	rows, err := db.Query(query, userID)
	if err != nil {
		return memberships, err
	}
	defer rows.Close()

	for rows.Next() {
		var orgID, orgName, roleName string
		if err := rows.Scan(&orgID, &orgName, &roleName); err == nil {
			memberships[orgID] = roleName
		}
	}

	return memberships, nil
}

// GetAPIKeyByID fetches an API key by its ID
func GetAPIKeyByID(db *sql.DB, id string) (string, error) {
	var apiKey string
	err := db.QueryRow(`SELECT api_key FROM api_keys WHERE id = $1`, id).Scan(&apiKey)
	if err != nil {
		return "", err
	}
	return apiKey, nil
}

// GetOrganizationByID retrieves a single organization by ID
func GetOrganizationByID(db *sql.DB, id string) (*models.Organization, error) {
	query := `
		SELECT id, name, description, is_active, created_at, updated_at,
		       ad_admin_group_id, ad_admin_group_name, ad_member_group_id, ad_member_group_name
		FROM organizations
		WHERE id = $1`

	var org models.Organization
	err := db.QueryRow(query, id).Scan(
		&org.ID, &org.Name, &org.Description, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
		&org.AdAdminGroupID, &org.AdAdminGroupName, &org.AdMemberGroupID, &org.AdMemberGroupName,
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
			ak.id, ak.name, ak.organization_id, ak.is_active,
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
			&key.ID, &key.Name, &key.OrganizationID, &key.IsActive,
			&key.LastUsed, &key.CreatedAt, &key.UpdatedAt, &orgName,
		)
		if err != nil {
			return nil, err
		}

		// Create a display prefix from the key ID
		key.KeyPrefix = "sk-" + key.ID[:8] + "..."

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
			ak.id, ak.name, ak.organization_id, ak.is_active,
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
			&key.ID, &key.Name, &key.OrganizationID, &key.IsActive,
			&key.LastUsed, &key.CreatedAt, &key.UpdatedAt, &orgName,
		)
		if err != nil {
			return nil, err
		}

		// Create a display prefix from the key ID
		key.KeyPrefix = "sk-" + key.ID[:8] + "..."

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
	fullKey, keyPrefix, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	query := `
		INSERT INTO api_keys (name, organization_id, api_key)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`

	var apiKey models.APIKey
	err = db.QueryRow(query, req.Name, req.OrganizationID, fullKey).Scan(&apiKey.ID, &apiKey.CreatedAt, &apiKey.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Populate the rest of the fields
	apiKey.Name = req.Name
	apiKey.Description = req.Description
	apiKey.KeyPrefix = keyPrefix
	apiKey.OrganizationID = req.OrganizationID
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
	query := `SELECT id, name, description, provider, model_id, api_endpoint, api_token,
	          input_cost_per_1m, output_cost_per_1m, max_retries, timeout_seconds,
	          retry_delay_ms, backoff_multiplier, is_active, created_at, updated_at
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
			&model.ModelID, &model.APIEndpoint, &model.APIToken,
			&model.InputCostPer1M, &model.OutputCostPer1M,
			&model.MaxRetries, &model.TimeoutSeconds, &model.RetryDelayMs, &model.BackoffMultiplier,
			&model.IsActive, &model.CreatedAt, &model.UpdatedAt)
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

	// Convert cost strings to floats
	var inputCost, outputCost *float64
	if req.InputCostPer1M != nil && *req.InputCostPer1M != "" {
		if cost, err := strconv.ParseFloat(*req.InputCostPer1M, 64); err == nil {
			inputCost = &cost
		}
	}
	if req.OutputCostPer1M != nil && *req.OutputCostPer1M != "" {
		if cost, err := strconv.ParseFloat(*req.OutputCostPer1M, 64); err == nil {
			outputCost = &cost
		}
	}

	// Convert retry/timeout strings to ints/floats
	var maxRetries, timeoutSeconds, retryDelayMs *int
	var backoffMultiplier *float64

	if req.MaxRetries != nil && *req.MaxRetries != "" {
		if retries, err := strconv.Atoi(*req.MaxRetries); err == nil {
			maxRetries = &retries
		}
	}
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds != "" {
		if timeout, err := strconv.Atoi(*req.TimeoutSeconds); err == nil {
			// Enforce maximum timeout limit of 5 minutes (300 seconds)
			if timeout > 300 {
				return nil, fmt.Errorf("timeout_seconds cannot exceed 300 seconds (5 minutes)")
			}
			if timeout < 5 {
				return nil, fmt.Errorf("timeout_seconds cannot be less than 5 seconds")
			}
			timeoutSeconds = &timeout
		}
	}
	if req.RetryDelayMs != nil && *req.RetryDelayMs != "" {
		if delay, err := strconv.Atoi(*req.RetryDelayMs); err == nil {
			retryDelayMs = &delay
		}
	}
	if req.BackoffMultiplier != nil && *req.BackoffMultiplier != "" {
		if multiplier, err := strconv.ParseFloat(*req.BackoffMultiplier, 64); err == nil {
			backoffMultiplier = &multiplier
		}
	}

	// Create the model
	query := `
		INSERT INTO models (name, description, provider, model_id, api_endpoint, api_token,
		                   input_cost_per_1m, output_cost_per_1m, max_retries, timeout_seconds,
		                   retry_delay_ms, backoff_multiplier)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	var model models.Model
	err = tx.QueryRow(query, req.Name, req.Description, req.Provider, req.ModelID, req.APIEndpoint, req.APIToken,
		inputCost, outputCost, maxRetries, timeoutSeconds, retryDelayMs, backoffMultiplier).
		Scan(&model.ID, &model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Set the fields
	model.Name = req.Name
	model.Description = req.Description
	model.Provider = req.Provider
	model.ModelID = req.ModelID
	model.APIEndpoint = req.APIEndpoint
	model.APIToken = req.APIToken
	model.InputCostPer1M = inputCost
	model.OutputCostPer1M = outputCost
	model.MaxRetries = maxRetries
	model.TimeoutSeconds = timeoutSeconds
	model.RetryDelayMs = retryDelayMs
	model.BackoffMultiplier = backoffMultiplier
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

func UpdateModel(db *sql.DB, modelID string, req models.UpdateModelRequest) (*models.Model, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}
	if req.Provider != nil {
		setParts = append(setParts, fmt.Sprintf("provider = $%d", argIndex))
		args = append(args, *req.Provider)
		argIndex++
	}
	if req.ModelID != nil {
		setParts = append(setParts, fmt.Sprintf("model_id = $%d", argIndex))
		args = append(args, *req.ModelID)
		argIndex++
	}
	if req.APIEndpoint != nil {
		setParts = append(setParts, fmt.Sprintf("api_endpoint = $%d", argIndex))
		args = append(args, *req.APIEndpoint)
		argIndex++
	}
	if req.APIToken != nil {
		setParts = append(setParts, fmt.Sprintf("api_token = $%d", argIndex))
		args = append(args, *req.APIToken)
		argIndex++
	}
	if req.InputCostPer1M != nil && *req.InputCostPer1M != "" {
		if cost, err := strconv.ParseFloat(*req.InputCostPer1M, 64); err == nil {
			setParts = append(setParts, fmt.Sprintf("input_cost_per_1m = $%d", argIndex))
			args = append(args, cost)
			argIndex++
		}
	}
	if req.OutputCostPer1M != nil && *req.OutputCostPer1M != "" {
		if cost, err := strconv.ParseFloat(*req.OutputCostPer1M, 64); err == nil {
			setParts = append(setParts, fmt.Sprintf("output_cost_per_1m = $%d", argIndex))
			args = append(args, cost)
			argIndex++
		}
	}
	if req.MaxRetries != nil && *req.MaxRetries != "" {
		if retries, err := strconv.Atoi(*req.MaxRetries); err == nil {
			setParts = append(setParts, fmt.Sprintf("max_retries = $%d", argIndex))
			args = append(args, retries)
			argIndex++
		}
	}
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds != "" {
		if timeout, err := strconv.Atoi(*req.TimeoutSeconds); err == nil {
			// Enforce maximum timeout limit of 5 minutes (300 seconds)
			if timeout > 300 {
				return nil, fmt.Errorf("timeout_seconds cannot exceed 300 seconds (5 minutes)")
			}
			if timeout < 5 {
				return nil, fmt.Errorf("timeout_seconds cannot be less than 5 seconds")
			}
			setParts = append(setParts, fmt.Sprintf("timeout_seconds = $%d", argIndex))
			args = append(args, timeout)
			argIndex++
		}
	}
	if req.RetryDelayMs != nil && *req.RetryDelayMs != "" {
		if delay, err := strconv.Atoi(*req.RetryDelayMs); err == nil {
			setParts = append(setParts, fmt.Sprintf("retry_delay_ms = $%d", argIndex))
			args = append(args, delay)
			argIndex++
		}
	}
	if req.BackoffMultiplier != nil && *req.BackoffMultiplier != "" {
		if multiplier, err := strconv.ParseFloat(*req.BackoffMultiplier, 64); err == nil {
			setParts = append(setParts, fmt.Sprintf("backoff_multiplier = $%d", argIndex))
			args = append(args, multiplier)
			argIndex++
		}
	}
	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(setParts) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Add updated_at
	setParts = append(setParts, "updated_at = NOW()")

	// Add WHERE clause
	args = append(args, modelID)
	whereClause := fmt.Sprintf("id = $%d", argIndex)

	query := fmt.Sprintf(
		`UPDATE models SET %s WHERE %s RETURNING id, name, description, provider, model_id, api_endpoint, api_token, input_cost_per_1m, output_cost_per_1m, max_retries, timeout_seconds, retry_delay_ms, backoff_multiplier, is_active, created_at, updated_at`,
		strings.Join(setParts, ", "),
		whereClause,
	)

	var model models.Model
	err = tx.QueryRow(query, args...).Scan(
		&model.ID, &model.Name, &model.Description, &model.Provider,
		&model.ModelID, &model.APIEndpoint, &model.APIToken,
		&model.InputCostPer1M, &model.OutputCostPer1M,
		&model.MaxRetries, &model.TimeoutSeconds, &model.RetryDelayMs, &model.BackoffMultiplier,
		&model.IsActive, &model.CreatedAt, &model.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Handle organization access updates if provided
	if len(req.OrgIDs) > 0 {
		// Remove existing organization access
		_, err = tx.Exec("DELETE FROM model_organization_access WHERE model_id = $1", modelID)
		if err != nil {
			return nil, err
		}

		// Add new organization access
		accessQuery := `INSERT INTO model_organization_access (model_id, organization_id) VALUES ($1, $2)`
		for _, orgID := range req.OrgIDs {
			_, err = tx.Exec(accessQuery, modelID, orgID)
			if err != nil {
				return nil, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Get the model with organization access for the response
	modelWithOrgs, err := GetModelWithOrganizations(db, modelID)
	if err != nil {
		// Return the model without organizations if we can't get them
		return &model, nil
	}

	return modelWithOrgs, nil
}

func GetModelWithOrganizations(db *sql.DB, modelID string) (*models.Model, error) {
	// Get the model
	query := `SELECT id, name, description, provider, model_id, api_endpoint, api_token,
	          input_cost_per_1m, output_cost_per_1m, max_retries, timeout_seconds,
	          retry_delay_ms, backoff_multiplier, is_active, created_at, updated_at
			  FROM models WHERE id = $1`

	var model models.Model
	err := db.QueryRow(query, modelID).Scan(
		&model.ID, &model.Name, &model.Description, &model.Provider,
		&model.ModelID, &model.APIEndpoint, &model.APIToken,
		&model.InputCostPer1M, &model.OutputCostPer1M,
		&model.MaxRetries, &model.TimeoutSeconds, &model.RetryDelayMs, &model.BackoffMultiplier,
		&model.IsActive, &model.CreatedAt, &model.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Get organization access
	orgQuery := `
		SELECT o.id, o.name
		FROM model_organization_access moa
		JOIN organizations o ON moa.organization_id = o.id
		WHERE moa.model_id = $1 AND o.is_active = true`

	orgRows, err := db.Query(orgQuery, modelID)
	if err != nil {
		return &model, nil // Return model without organizations if query fails
	}
	defer orgRows.Close()

	var organizations []models.Organization
	for orgRows.Next() {
		var org models.Organization
		err := orgRows.Scan(&org.ID, &org.Name)
		if err != nil {
			continue
		}
		organizations = append(organizations, org)
	}

	model.Organizations = organizations
	return &model, nil
}

func ManageModelAccess(db *sql.DB, modelID string, changes []ModelAccessChange) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, change := range changes {
		switch change.Action {
		case "add":
			// Add organization access (ignore if already exists)
			_, err = tx.Exec(
				`INSERT INTO model_organization_access (model_id, organization_id)
				 VALUES ($1, $2)
				 ON CONFLICT (model_id, organization_id) DO NOTHING`,
				modelID, change.OrgID,
			)
			if err != nil {
				return fmt.Errorf("failed to add organization access: %w", err)
			}
		case "remove":
			// Remove organization access
			_, err = tx.Exec(
				`DELETE FROM model_organization_access
				 WHERE model_id = $1 AND organization_id = $2`,
				modelID, change.OrgID,
			)
			if err != nil {
				return fmt.Errorf("failed to remove organization access: %w", err)
			}
		default:
			return fmt.Errorf("invalid action: %s", change.Action)
		}
	}

	return tx.Commit()
}

// ModelAccessChange represents a change to model organization access
type ModelAccessChange struct {
	OrgID  string `json:"orgId"`
	Action string `json:"action"` // "add" or "remove"
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
func generateAPIKey() (fullKey, prefix string, err error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	_, err = rand.Read(bytes)
	if err != nil {
		return "", "", err
	}

	// Create the full key with sk- prefix
	fullKey = "sk-" + hex.EncodeToString(bytes)

	// Extract prefix (first 7 characters for display)
	prefix = fullKey[:7] + "..."

	return fullKey, prefix, nil
}

// RBAC User Operations
func GetUserByAzureOID(db *sql.DB, azureOID string) (*models.User, error) {
	query := `SELECT id, azure_oid, email, name, is_active, last_login, created_at, updated_at
		      FROM users
		      WHERE azure_oid = $1 AND is_active = true`

	var user models.User
	err := db.QueryRow(query, azureOID).Scan(
		&user.ID, &user.AzureOID, &user.Email, &user.Name,
		&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByEmail(db *sql.DB, email string) (*models.User, error) {
	query := `SELECT id, azure_oid, email, name, is_active, last_login, created_at, updated_at
		      FROM users
		      WHERE email = $1 AND is_active = true`

	var user models.User
	err := db.QueryRow(query, email).Scan(
		&user.ID, &user.AzureOID, &user.Email, &user.Name,
		&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func CreateOrUpdateUser(db *sql.DB, req models.CreateUserRequest) (*models.User, error) {
	query := `
		INSERT INTO users (azure_oid, email, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (azure_oid)
		DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			last_login = NOW(),
			updated_at = NOW()
		RETURNING id, azure_oid, email, name, is_active, last_login, created_at, updated_at`

	var user models.User
	err := db.QueryRow(query, req.AzureOID, req.Email, req.Name).Scan(
		&user.ID, &user.AzureOID, &user.Email, &user.Name,
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

func GetUserByID(db *sql.DB, userID string) (*models.User, error) {
	query := `SELECT id, azure_oid, email, name, is_active, last_login, created_at, updated_at
		      FROM users
		      WHERE id = $1`

	var user models.User
	err := db.QueryRow(query, userID).Scan(
		&user.ID, &user.AzureOID, &user.Email, &user.Name,
		&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func AssignUserToOrganization(db *sql.DB, userID, orgID, roleName string, createdBy *string) error {
	query := `
		INSERT INTO user_organizations (user_id, organization_id, role_name, created_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, organization_id)
		DO UPDATE SET role_name = EXCLUDED.role_name, created_by = EXCLUDED.created_by`

	_, err := db.Exec(query, userID, orgID, roleName, createdBy)
	return err
}

func AssignSystemRole(db *sql.DB, userID, roleID string, createdBy *string) error {
	query := `
		INSERT INTO user_system_roles (user_id, role_id, created_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING`

	_, err := db.Exec(query, userID, roleID, createdBy)
	return err
}

// Legacy user function for backwards compatibility
func GetUserByUsername(db *sql.DB, username string) (*models.LegacyUser, error) {
	// This function is kept for backwards compatibility but should not be used in new code
	// The new RBAC system uses Azure OID instead of username
	return nil, fmt.Errorf("legacy user lookup not supported in RBAC system")
}

// Endpoints operations
func GetEndpointsWithModels(db *sql.DB) ([]models.Endpoint, error) {
	query := `
		SELECT
			e.id, e.organization_id, e.name, e.path_prefix, e.description,
			e.primary_model_id, e.fallback_model_id, e.is_active, e.created_at, e.updated_at,
			pm.name as primary_model_name, fm.name as fallback_model_name
		FROM endpoints e
		LEFT JOIN models pm ON e.primary_model_id = pm.id
		LEFT JOIN models fm ON e.fallback_model_id = fm.id
		WHERE e.is_active = true
		ORDER BY e.created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []models.Endpoint
	for rows.Next() {
		var endpoint models.Endpoint
		err := rows.Scan(
			&endpoint.ID, &endpoint.OrganizationID, &endpoint.Name, &endpoint.PathPrefix,
			&endpoint.Description, &endpoint.PrimaryModelID, &endpoint.FallbackModelID,
			&endpoint.IsActive, &endpoint.CreatedAt, &endpoint.UpdatedAt,
			&endpoint.PrimaryModelName, &endpoint.FallbackModelName,
		)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}

func GetEndpointsByOrganization(db *sql.DB, orgID string) ([]models.Endpoint, error) {
	query := `
		SELECT
			e.id, e.organization_id, e.name, e.path_prefix, e.description,
			e.primary_model_id, e.fallback_model_id, e.is_active, e.created_at, e.updated_at,
			pm.name as primary_model_name, fm.name as fallback_model_name
		FROM endpoints e
		LEFT JOIN models pm ON e.primary_model_id = pm.id
		LEFT JOIN models fm ON e.fallback_model_id = fm.id
		WHERE e.is_active = true AND e.organization_id = $1
		ORDER BY e.created_at DESC`

	rows, err := db.Query(query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []models.Endpoint
	for rows.Next() {
		var endpoint models.Endpoint
		err := rows.Scan(
			&endpoint.ID, &endpoint.OrganizationID, &endpoint.Name, &endpoint.PathPrefix,
			&endpoint.Description, &endpoint.PrimaryModelID, &endpoint.FallbackModelID,
			&endpoint.IsActive, &endpoint.CreatedAt, &endpoint.UpdatedAt,
			&endpoint.PrimaryModelName, &endpoint.FallbackModelName,
		)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}

func CreateEndpoint(db *sql.DB, req models.EndpointCreate, orgID string) (*models.Endpoint, error) {
	// Set default active status if not provided
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	query := `
		INSERT INTO endpoints (organization_id, name, path_prefix, description, primary_model_id, fallback_model_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	var endpoint models.Endpoint
	err := db.QueryRow(query,
		orgID, req.Name, req.PathPrefix, req.Description,
		req.PrimaryModelID, req.FallbackModelID, isActive,
	).Scan(&endpoint.ID, &endpoint.CreatedAt, &endpoint.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Populate the fields
	endpoint.OrganizationID = orgID
	endpoint.Name = req.Name
	endpoint.PathPrefix = req.PathPrefix
	endpoint.Description = req.Description
	endpoint.PrimaryModelID = req.PrimaryModelID
	endpoint.FallbackModelID = req.FallbackModelID
	endpoint.IsActive = isActive

	return &endpoint, nil
}

func UpdateEndpoint(db *sql.DB, endpointID string, req models.EndpointUpdate) (*models.Endpoint, error) {
	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.PathPrefix != nil {
		setParts = append(setParts, fmt.Sprintf("path_prefix = $%d", argIndex))
		args = append(args, *req.PathPrefix)
		argIndex++
	}
	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}
	if req.PrimaryModelID != nil {
		setParts = append(setParts, fmt.Sprintf("primary_model_id = $%d", argIndex))
		args = append(args, *req.PrimaryModelID)
		argIndex++
	}
	if req.FallbackModelID != nil {
		setParts = append(setParts, fmt.Sprintf("fallback_model_id = $%d", argIndex))
		args = append(args, *req.FallbackModelID)
		argIndex++
	}
	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(setParts) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = NOW()"))

	// Add WHERE clause
	args = append(args, endpointID)
	whereClause := fmt.Sprintf("id = $%d", argIndex)

	query := fmt.Sprintf(
		`UPDATE endpoints SET %s WHERE %s RETURNING id, organization_id, name, path_prefix, description, primary_model_id, fallback_model_id, is_active, created_at, updated_at`,
		fmt.Sprintf("%s", setParts),
		whereClause,
	)

	var endpoint models.Endpoint
	err := db.QueryRow(query, args...).Scan(
		&endpoint.ID, &endpoint.OrganizationID, &endpoint.Name, &endpoint.PathPrefix,
		&endpoint.Description, &endpoint.PrimaryModelID, &endpoint.FallbackModelID,
		&endpoint.IsActive, &endpoint.CreatedAt, &endpoint.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &endpoint, nil
}

func DeleteEndpoint(db *sql.DB, endpointID string) error {
	query := `UPDATE endpoints SET is_active = false, updated_at = NOW() WHERE id = $1`
	_, err := db.Exec(query, endpointID)
	return err
}

func GetEndpointByID(db *sql.DB, endpointID string) (*models.Endpoint, error) {
	query := `
		SELECT
			e.id, e.organization_id, e.name, e.path_prefix, e.description,
			e.primary_model_id, e.fallback_model_id, e.is_active, e.created_at, e.updated_at,
			pm.name as primary_model_name, fm.name as fallback_model_name
		FROM endpoints e
		LEFT JOIN models pm ON e.primary_model_id = pm.id
		LEFT JOIN models fm ON e.fallback_model_id = fm.id
		WHERE e.id = $1`

	var endpoint models.Endpoint
	err := db.QueryRow(query, endpointID).Scan(
		&endpoint.ID, &endpoint.OrganizationID, &endpoint.Name, &endpoint.PathPrefix,
		&endpoint.Description, &endpoint.PrimaryModelID, &endpoint.FallbackModelID,
		&endpoint.IsActive, &endpoint.CreatedAt, &endpoint.UpdatedAt,
		&endpoint.PrimaryModelName, &endpoint.FallbackModelName,
	)

	if err != nil {
		return nil, err
	}

	return &endpoint, nil
}

// Usage tracking operations

// CreateUsageLog records API usage for billing and analytics
func CreateUsageLog(db *sql.DB, req CreateUsageLogRequest) error {
	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO usage_logs (
			organization_id, api_key_id, model_id, endpoint,
			prompt_tokens, completion_tokens, total_tokens,
			request_id, response_status, response_time_ms, cost_usd, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = db.Exec(query,
		req.OrganizationID, req.APIKeyID, req.ModelID, req.Endpoint,
		req.PromptTokens, req.CompletionTokens, req.TotalTokens,
		req.RequestID, req.ResponseStatus, req.ResponseTimeMS, req.CostUSD, metadataJSON,
	)

	return err
}

// CreateUsageLogRequest represents the data needed to create a usage log
type CreateUsageLogRequest struct {
	OrganizationID   string                 `json:"organization_id"`
	APIKeyID         string                 `json:"api_key_id"`
	ModelID          string                 `json:"model_id"`
	Endpoint         string                 `json:"endpoint"`
	PromptTokens     int                    `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	TotalTokens      int                    `json:"total_tokens"`
	RequestID        *string                `json:"request_id"`
	ResponseStatus   int                    `json:"response_status"`
	ResponseTimeMS   *int                   `json:"response_time_ms"`
	CostUSD          *float64               `json:"cost_usd"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// UpdateOrganizationUsage updates the organization's token usage and quota
func UpdateOrganizationUsage(db *sql.DB, orgID string, tokensUsed int) error {
	query := `
		UPDATE organization_quotas
		SET used_tokens = used_tokens + $1, updated_at = NOW()
		WHERE organization_id = $2`

	_, err := db.Exec(query, tokensUsed, orgID)
	return err
}

// GetUsageStatsByOrganization retrieves usage statistics for an organization
func GetUsageStatsByOrganization(db *sql.DB, orgID string, days int) (int64, int64, int64, int64, float64, error) {
	query := `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(AVG(response_time_ms), 0) as avg_response_time
		FROM usage_logs
		WHERE organization_id = $1
		AND created_at >= NOW() - INTERVAL '%d days'`

	var totalRequests, totalTokens, promptTokens, completionTokens int64
	var avgResponseTime float64

	err := db.QueryRow(fmt.Sprintf(query, days), orgID).Scan(
		&totalRequests, &totalTokens, &promptTokens, &completionTokens, &avgResponseTime,
	)

	if err != nil {
		return 0, 0, 0, 0, 0, err
	}

	return totalRequests, totalTokens, promptTokens, completionTokens, avgResponseTime, nil
}

// GetUsageByModelForOrganization retrieves usage statistics grouped by model
func GetUsageByModelForOrganization(db *sql.DB, orgID string, days int) ([]ModelUsageStats, error) {
	query := `
		SELECT
			ul.model_id,
			m.name as model_name,
			m.provider,
			COUNT(*) as total_requests,
			COALESCE(SUM(ul.total_tokens), 0) as total_tokens,
			COALESCE(SUM(ul.prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(ul.completion_tokens), 0) as completion_tokens,
			COALESCE(AVG(ul.response_time_ms), 0) as avg_response_time
		FROM usage_logs ul
		JOIN models m ON ul.model_id = m.id
		WHERE ul.organization_id = $1
		AND ul.created_at >= NOW() - INTERVAL '%d days'
		GROUP BY ul.model_id, m.name, m.provider
		ORDER BY total_tokens DESC`

	rows, err := db.Query(fmt.Sprintf(query, days), orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usageByModel []ModelUsageStats
	for rows.Next() {
		var usage ModelUsageStats
		err := rows.Scan(
			&usage.ModelID, &usage.ModelName, &usage.Provider,
			&usage.TotalRequests, &usage.TotalTokens,
			&usage.PromptTokens, &usage.CompletionTokens,
			&usage.AvgResponseTime,
		)
		if err != nil {
			continue
		}
		usageByModel = append(usageByModel, usage)
	}

	return usageByModel, nil
}

// ModelUsageStats represents usage statistics for a specific model
type ModelUsageStats struct {
	ModelID          string  `json:"model_id"`
	ModelName        string  `json:"model_name"`
	Provider         string  `json:"provider"`
	TotalRequests    int64   `json:"total_requests"`
	TotalTokens      int64   `json:"total_tokens"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	AvgResponseTime  float64 `json:"avg_response_time_ms"`
}

// CheckOrganizationQuota checks if organization has exceeded their quota
func CheckOrganizationQuota(db *sql.DB, orgID string) (bool, int64, int64, error) {
	query := `
		SELECT total_quota, used_tokens
		FROM organization_quotas
		WHERE organization_id = $1`

	var totalQuota, usedTokens int64
	err := db.QueryRow(query, orgID).Scan(&totalQuota, &usedTokens)
	if err != nil {
		return false, 0, 0, err
	}

	hasQuota := usedTokens < totalQuota
	return hasQuota, usedTokens, totalQuota, nil
}

// GetUsersWithOrganizations fetches all users with their organization memberships
func GetUsersWithOrganizations(db *sql.DB) ([]models.UserWithOrganizations, error) {
	query := `
		SELECT
			u.id, u.azure_oid, u.email, u.name, u.is_active, u.last_login, u.created_at, u.updated_at,
			COALESCE(
				JSON_AGG(
					JSON_BUILD_OBJECT(
						'org_id', o.id,
						'org_name', o.name,
						'role_name', uo.role_name
					) ORDER BY o.name
				) FILTER (WHERE o.id IS NOT NULL),
				'[]'::json
			) as organizations
		FROM users u
		LEFT JOIN user_organizations uo ON u.id = uo.user_id
		LEFT JOIN organizations o ON uo.organization_id = o.id AND o.is_active = true
		GROUP BY u.id, u.azure_oid, u.email, u.name, u.is_active, u.last_login, u.created_at, u.updated_at
		ORDER BY u.name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.UserWithOrganizations
	for rows.Next() {
		var user models.UserWithOrganizations
		var orgsJSON string

		err := rows.Scan(
			&user.ID, &user.AzureOID, &user.Email, &user.Name,
			&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
			&orgsJSON,
		)
		if err != nil {
			return nil, err
		}

		// Parse organizations JSON
		var orgMemberships []models.UserOrgMembership
		if err := json.Unmarshal([]byte(orgsJSON), &orgMemberships); err != nil {
			return nil, err
		}
		user.Organizations = orgMemberships

		users = append(users, user)
	}

	return users, nil
}

// GetUsersByOrganization fetches users for a specific organization
func GetUsersByOrganization(db *sql.DB, orgID string) ([]models.UserWithOrganizations, error) {
	query := `
		SELECT
			u.id, u.azure_oid, u.email, u.name, u.is_active, u.last_login, u.created_at, u.updated_at,
			JSON_BUILD_OBJECT(
				'org_id', o.id,
				'org_name', o.name,
				'role_name', uo.role_name
			) as organization
		FROM users u
		JOIN user_organizations uo ON u.id = uo.user_id
		JOIN organizations o ON uo.organization_id = o.id
		WHERE o.id = $1 AND o.is_active = true
		ORDER BY u.name`

	rows, err := db.Query(query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.UserWithOrganizations
	for rows.Next() {
		var user models.UserWithOrganizations
		var orgJSON string

		err := rows.Scan(
			&user.ID, &user.AzureOID, &user.Email, &user.Name,
			&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
			&orgJSON,
		)
		if err != nil {
			return nil, err
		}

		// Parse single organization
		var orgMembership models.UserOrgMembership
		if err := json.Unmarshal([]byte(orgJSON), &orgMembership); err != nil {
			return nil, err
		}
		user.Organizations = []models.UserOrgMembership{orgMembership}

		users = append(users, user)
	}

	return users, nil
}

// GetAPIKeyByID fetches an API key by its ID
