package admin

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
	"github.com/like-mike/relai-gateway/ui/auth"
)

func GetQuotaHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		c.HTML(http.StatusInternalServerError, "quota-cards.html", gin.H{
			"error": "Database connection error",
		})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		c.HTML(http.StatusInternalServerError, "quota-cards.html", gin.H{
			"error": "Database connection error",
		})
		return
	}

	// Get organization ID from query parameter
	orgID := c.Query("org_id")

	// Add logging to debug the org_id parameter
	log.Printf("Quota request received with org_id: '%s'", orgID)

	// Get user context for RBAC
	userContext := auth.GetUserContext(c)
	userID, ok := userContext["id"].(string)
	if !ok || userID == "" {
		log.Printf("No user ID found in context for quota request")
		c.HTML(http.StatusUnauthorized, "quota-cards.html", gin.H{
			"error": "User authentication required",
		})
		return
	}

	// Get user's organization memberships for RBAC
	memberships, err := db.GetUserOrganizationMemberships(sqlDB, userID)
	if err != nil {
		log.Printf("Failed to get user memberships: %v", err)
		c.HTML(http.StatusInternalServerError, "quota-cards.html", gin.H{
			"error": "Failed to load user permissions",
		})
		return
	}

	var quotaStats *models.QuotaStats

	if orgID != "" && orgID != "null" && orgID != "undefined" {
		// Validate user has access to the requested organization
		if _, hasAccess := memberships[orgID]; !hasAccess {
			log.Printf("User %s denied access to organization %s", userID, orgID)
			c.HTML(http.StatusForbidden, "quota-cards.html", gin.H{
				"error": "Access denied to organization",
			})
			return
		}

		// Get quota for specific organization
		quota, err := db.GetOrganizationQuota(sqlDB, orgID)
		if err != nil {
			log.Printf("Failed to get quota for organization %s: %v", orgID, err)
			// Return default stats if no quota found for this organization
			quotaStats = &models.QuotaStats{
				TotalUsage:     "0",
				RemainingQuota: "100K",
				PercentUsed:    "0.0%",
			}
		} else {
			stats := quota.CalculateQuotaStats()
			quotaStats = &stats
		}

		log.Printf("Loaded quota stats for organization %s: %+v", orgID, quotaStats)
	} else {
		log.Printf("No valid org_id provided (got: '%s'), using fallback logic", orgID)
		// No organization specified, try to get first accessible organization
		if len(memberships) > 0 {
			// Get first organization the user has access to
			var firstAccessibleOrgID string
			for orgID := range memberships {
				firstAccessibleOrgID = orgID
				break
			}

			quota, err := db.GetOrganizationQuota(sqlDB, firstAccessibleOrgID)
			if err != nil {
				log.Printf("Failed to get quota for first accessible organization %s: %v", firstAccessibleOrgID, err)
				quotaStats = &models.QuotaStats{
					TotalUsage:     "0",
					RemainingQuota: "100K",
					PercentUsed:    "0.0%",
				}
			} else {
				stats := quota.CalculateQuotaStats()
				quotaStats = &stats
			}

			log.Printf("No org_id specified, loaded quota stats for first accessible organization %s", firstAccessibleOrgID)
		} else {
			// User has no accessible organizations, return default stats
			quotaStats = &models.QuotaStats{
				TotalUsage:     "0",
				RemainingQuota: "100K",
				PercentUsed:    "0.0%",
			}
			log.Printf("User has no accessible organizations, returning default stats")
		}
	}

	// Log the session for debugging (keeping the dummy-session log)
	log.Println("dummy-session")

	// Render the quota cards template with real data
	c.HTML(http.StatusOK, "quota-cards.html", gin.H{
		"TotalUsage":     quotaStats.TotalUsage,
		"RemainingQuota": quotaStats.RemainingQuota,
		"PercentUsed":    quotaStats.PercentUsed,
	})
}
