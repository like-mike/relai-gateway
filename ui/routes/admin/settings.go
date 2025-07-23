package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
	"github.com/like-mike/relai-gateway/ui/auth"
)

// SettingsHandler handles the main settings page
func SettingsHandler(c *gin.Context) {
	userData := auth.GetUserContext(c)
	userData["activePage"] = "settings"
	userData["title"] = "Settings"

	c.HTML(http.StatusOK, "settings.html", userData)
}

// OrganizationsTableHandler returns the organizations table data
func OrganizationsTableHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		c.HTML(http.StatusInternalServerError, "organizations-table.html", gin.H{
			"error": "Database connection error",
		})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		c.HTML(http.StatusInternalServerError, "organizations-table.html", gin.H{
			"error": "Database connection error",
		})
		return
	}

	// Get organizations with quotas and user counts
	organizations, err := getOrganizationsWithDetails(sqlDB)
	if err != nil {
		log.Printf("Failed to get organizations: %v", err)
		c.HTML(http.StatusInternalServerError, "organizations-table.html", gin.H{
			"error": "Failed to load organizations",
		})
		return
	}

	// Render the organizations table template
	c.HTML(http.StatusOK, "organizations-table.html", gin.H{
		"organizations": organizations,
	})
}

// CreateOrganizationHandler creates a new organization
func CreateOrganizationHandler(c *gin.Context) {
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

	// Parse form data
	name := c.PostForm("name")
	description := c.PostForm("description")
	quotaStr := c.PostForm("quota")
	isActiveStr := c.PostForm("is_active")

	// Parse AD group fields
	adAdminGroupID := c.PostForm("ad_admin_group_id")
	adAdminGroupName := c.PostForm("ad_admin_group_name")
	adMemberGroupID := c.PostForm("ad_member_group_id")
	adMemberGroupName := c.PostForm("ad_member_group_name")

	log.Printf("Create form data - Admin Group ID: '%s', Name: '%s'", adAdminGroupID, adAdminGroupName)
	log.Printf("Create form data - Member Group ID: '%s', Name: '%s'", adMemberGroupID, adMemberGroupName)

	// Validate required fields
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization name is required"})
		return
	}

	// Parse quota
	quota := 100000 // default
	if quotaStr != "" {
		if q, err := strconv.Atoi(quotaStr); err == nil {
			quota = q
		}
	}

	// Parse is_active
	isActive := isActiveStr == "on" || isActiveStr == "true"

	// Create organization with AD groups
	orgID, err := createOrganizationWithADGroups(sqlDB, name, description, isActive, quota,
		adAdminGroupID, adAdminGroupName, adMemberGroupID, adMemberGroupName)
	if err != nil {
		log.Printf("Failed to create organization: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create organization"})
		return
	}

	log.Printf("Created organization: %s (ID: %s)", name, orgID)

	// Return updated organizations table
	organizations, err := getOrganizationsWithDetails(sqlDB)
	if err != nil {
		log.Printf("Failed to get updated organizations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh organizations"})
		return
	}

	c.HTML(http.StatusOK, "organizations-table.html", gin.H{
		"organizations": organizations,
	})
}

// GetOrganizationHandler returns a single organization's data
func GetOrganizationHandler(c *gin.Context) {
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

	orgID := c.Param("id")
	org, err := db.GetOrganizationByID(sqlDB, orgID)
	if err != nil {
		log.Printf("Failed to get organization: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	c.JSON(http.StatusOK, org)
}

// UpdateOrganizationHandler updates an organization
func UpdateOrganizationHandler(c *gin.Context) {
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

	orgID := c.Param("id")
	name := c.PostForm("name")
	description := c.PostForm("description")
	isActiveStr := c.PostForm("is_active")

	// Parse AD group fields
	adAdminGroupID := c.PostForm("ad_admin_group_id")
	adAdminGroupName := c.PostForm("ad_admin_group_name")
	adMemberGroupID := c.PostForm("ad_member_group_id")
	adMemberGroupName := c.PostForm("ad_member_group_name")

	log.Printf("Update form data - Admin Group ID: '%s', Name: '%s'", adAdminGroupID, adAdminGroupName)
	log.Printf("Update form data - Member Group ID: '%s', Name: '%s'", adMemberGroupID, adMemberGroupName)

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization name is required"})
		return
	}

	// Parse is_active
	isActive := isActiveStr == "on" || isActiveStr == "true"

	err := updateOrganizationWithADGroups(sqlDB, orgID, name, description, isActive,
		adAdminGroupID, adAdminGroupName, adMemberGroupID, adMemberGroupName)
	if err != nil {
		log.Printf("Failed to update organization: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update organization"})
		return
	}

	// Return updated organizations table
	organizations, err := getOrganizationsWithDetails(sqlDB)
	if err != nil {
		log.Printf("Failed to get updated organizations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh organizations"})
		return
	}

	c.HTML(http.StatusOK, "organizations-table.html", gin.H{
		"organizations": organizations,
	})
}

// UsersTableHandler returns the users table data
func UsersTableHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		c.HTML(http.StatusInternalServerError, "users-table.html", gin.H{
			"error": "Database connection error",
		})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		c.HTML(http.StatusInternalServerError, "users-table.html", gin.H{
			"error": "Database connection error",
		})
		return
	}

	// Check if org filter is provided
	orgID := c.Query("org_id")

	var users []models.UserWithOrganizations
	var err error

	if orgID != "" {
		users, err = db.GetUsersByOrganization(sqlDB, orgID)
	} else {
		users, err = db.GetUsersWithOrganizations(sqlDB)
	}

	if err != nil {
		log.Printf("Failed to get users: %v", err)
		c.HTML(http.StatusInternalServerError, "users-table.html", gin.H{
			"error": "Failed to load users",
		})
		return
	}

	// Render the users table template
	c.HTML(http.StatusOK, "users-table.html", gin.H{
		"users":     users,
		"orgFilter": orgID,
	})
}

// GetADGroupsHandler returns available Azure AD groups
func GetADGroupsHandler(c *gin.Context) {
	// Get Azure AD configuration
	config := auth.LoadConfig()
	if !config.EnableAzureAD {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Azure AD integration is disabled"})
		return
	}

	// Get access token for Microsoft Graph
	accessToken, err := auth.GetAccessToken(config.AzureTenantID, config.AzureClientID, config.AzureClientSecret)
	if err != nil {
		log.Printf("Failed to get access token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate with Azure AD"})
		return
	}

	// Get all groups from Azure AD
	groups, err := getAllADGroups(accessToken)
	if err != nil {
		log.Printf("Failed to get AD groups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Azure AD groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// DeleteOrganizationHandler deletes an organization
func DeleteOrganizationHandler(c *gin.Context) {
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

	orgID := c.Param("id")

	err := deleteOrganization(sqlDB, orgID)
	if err != nil {
		log.Printf("Failed to delete organization: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete organization"})
		return
	}

	// Return updated organizations table
	organizations, err := getOrganizationsWithDetails(sqlDB)
	if err != nil {
		log.Printf("Failed to get updated organizations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh organizations"})
		return
	}

	c.HTML(http.StatusOK, "organizations-table.html", gin.H{
		"organizations": organizations,
	})
}

// Helper functions

func getOrganizationsWithDetails(sqlDB *sql.DB) ([]models.OrganizationWithDetails, error) {
	query := `
		SELECT
			o.id, o.name, o.description, o.is_active, o.created_at, o.updated_at,
			o.ad_admin_group_id, o.ad_admin_group_name, o.ad_member_group_id, o.ad_member_group_name,
			COALESCE(oq.total_quota, 100000) as total_quota,
			COALESCE(oq.used_tokens, 0) as used_tokens
		FROM organizations o
		LEFT JOIN organization_quotas oq ON o.id = oq.organization_id
		ORDER BY o.created_at DESC`

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var organizations []models.OrganizationWithDetails
	for rows.Next() {
		var org models.OrganizationWithDetails
		var quota models.OrganizationQuota

		err := rows.Scan(
			&org.ID, &org.Name, &org.Description, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
			&org.AdAdminGroupID, &org.AdAdminGroupName, &org.AdMemberGroupID, &org.AdMemberGroupName,
			&quota.TotalQuota, &quota.UsedTokens,
		)
		if err != nil {
			return nil, err
		}

		// Set a default user count since users aren't linked to organizations yet
		org.UserCount = 1

		if quota.TotalQuota > 0 {
			org.Quota = &quota
		}

		organizations = append(organizations, org)
	}

	return organizations, nil
}

func createOrganizationWithADGroups(sqlDB *sql.DB, name, description string, isActive bool, quota int,
	adAdminGroupID, adAdminGroupName, adMemberGroupID, adMemberGroupName string) (string, error) {
	tx, err := sqlDB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Create organization with AD group fields
	var orgID string
	err = tx.QueryRow(`
		INSERT INTO organizations (name, description, is_active, ad_admin_group_id, ad_admin_group_name, ad_member_group_id, ad_member_group_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, name, nullIfEmpty(description), isActive,
		nullIfEmpty(adAdminGroupID), nullIfEmpty(adAdminGroupName),
		nullIfEmpty(adMemberGroupID), nullIfEmpty(adMemberGroupName)).Scan(&orgID)
	if err != nil {
		return "", err
	}

	// Create quota for organization
	_, err = tx.Exec(`
		INSERT INTO organization_quotas (organization_id, total_quota, used_tokens)
		VALUES ($1, $2, 0)
	`, orgID, quota)
	if err != nil {
		return "", err
	}

	// Create AD group mappings if provided
	if adAdminGroupID != "" {
		err = createOrgADGroupMapping(tx, orgID, adAdminGroupID, adAdminGroupName, "admin")
		if err != nil {
			return "", err
		}
	}

	if adMemberGroupID != "" {
		err = createOrgADGroupMapping(tx, orgID, adMemberGroupID, adMemberGroupName, "member")
		if err != nil {
			return "", err
		}
	}

	return orgID, tx.Commit()
}

func updateOrganizationWithADGroups(sqlDB *sql.DB, id, name, description string, isActive bool,
	adAdminGroupID, adAdminGroupName, adMemberGroupID, adMemberGroupName string) error {
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update organization with AD group fields
	_, err = tx.Exec(`
		UPDATE organizations 
		SET name = $1, description = $2, is_active = $3, updated_at = NOW(),
		    ad_admin_group_id = $4, ad_admin_group_name = $5, 
		    ad_member_group_id = $6, ad_member_group_name = $7
		WHERE id = $8
	`, name, nullIfEmpty(description), isActive,
		nullIfEmpty(adAdminGroupID), nullIfEmpty(adAdminGroupName),
		nullIfEmpty(adMemberGroupID), nullIfEmpty(adMemberGroupName), id)
	if err != nil {
		return err
	}

	// Update AD group mappings
	// First, deactivate existing mappings
	_, err = tx.Exec(`
		UPDATE organization_ad_groups 
		SET is_active = false 
		WHERE organization_id = $1
	`, id)
	if err != nil {
		return err
	}

	// Create/update admin group mapping if provided
	if adAdminGroupID != "" {
		err = createOrgADGroupMapping(tx, id, adAdminGroupID, adAdminGroupName, "admin")
		if err != nil {
			return err
		}
	}

	// Create/update member group mapping if provided
	if adMemberGroupID != "" {
		err = createOrgADGroupMapping(tx, id, adMemberGroupID, adMemberGroupName, "member")
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func deleteOrganization(sqlDB *sql.DB, id string) error {
	// Note: This will cascade delete due to foreign key constraints
	_, err := sqlDB.Exec(`DELETE FROM organizations WHERE id = $1`, id)
	return err
}

// Helper function to convert empty string to null for database
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// Helper function to create AD group mappings
func createOrgADGroupMapping(tx *sql.Tx, orgID, adGroupID, adGroupName, roleType string) error {
	_, err := tx.Exec(`
		INSERT INTO organization_ad_groups (organization_id, ad_group_id, ad_group_name, role_type, is_active)
		VALUES ($1, $2, $3, $4, true)
		ON CONFLICT (organization_id, ad_group_id, role_type) DO UPDATE SET
			ad_group_name = EXCLUDED.ad_group_name,
			is_active = true
	`, orgID, adGroupID, nullIfEmpty(adGroupName), roleType)
	return err
}

// ADGroup represents an Azure AD group
type ADGroup struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
}

// getAllADGroups fetches all Azure AD groups
func getAllADGroups(accessToken string) ([]ADGroup, error) {
	var groups []ADGroup

	url := "https://graph.microsoft.com/v1.0/groups"

	for url != "" {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return groups, err
		}
		req.Header.Add("Authorization", "Bearer "+accessToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return groups, err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			return groups, fmt.Errorf("graph request failed: %s", string(body))
		}

		var result struct {
			Value    []ADGroup `json:"value"`
			NextLink string    `json:"@odata.nextLink,omitempty"`
		}

		err = json.Unmarshal(body, &result)
		if err != nil {
			return groups, err
		}

		groups = append(groups, result.Value...)
		url = result.NextLink // Handle pagination
	}

	return groups, nil
}
