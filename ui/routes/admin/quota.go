package admin

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
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

	// Get quota stats from database
	quotaStats, err := db.GetQuotaStatsForFirstOrg(sqlDB)
	if err != nil {
		log.Printf("Failed to get quota stats: %v", err)
		// Return default stats on error
		quotaStats = &models.QuotaStats{
			TotalUsage:     "0",
			RemainingQuota: "100K",
			PercentUsed:    "0.0%",
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
