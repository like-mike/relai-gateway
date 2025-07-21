package admin

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
)

// SettingsHandler handles the main settings page
func SettingsHandler(c *gin.Context) {
	userData := GetUserContext(c)
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

	// Create organization
	orgID, err := createOrganization(sqlDB, name, description, isActive, quota)
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

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization name is required"})
		return
	}

	isActive := isActiveStr == "on" || isActiveStr == "true"

	err := updateOrganization(sqlDB, orgID, name, description, isActive)
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

func createOrganization(sqlDB *sql.DB, name, description string, isActive bool, quota int) (string, error) {
	tx, err := sqlDB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Create organization
	var orgID string
	err = tx.QueryRow(`
		INSERT INTO organizations (name, description, is_active)
		VALUES ($1, $2, $3)
		RETURNING id
	`, name, description, isActive).Scan(&orgID)
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

	return orgID, tx.Commit()
}

func updateOrganization(sqlDB *sql.DB, id, name, description string, isActive bool) error {
	_, err := sqlDB.Exec(`
		UPDATE organizations 
		SET name = $1, description = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4
	`, name, description, isActive, id)
	return err
}

func deleteOrganization(sqlDB *sql.DB, id string) error {
	// Note: This will cascade delete due to foreign key constraints
	_, err := sqlDB.Exec(`DELETE FROM organizations WHERE id = $1`, id)
	return err
}
