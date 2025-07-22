package admin

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
)

func AnalyticsDashboardHandler(c *gin.Context) {
	// Get database connection
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

	// Parse query parameters
	filter := models.AnalyticsFilter{
		TimeRange:    c.DefaultQuery("range", "7d"),
		StartDate:    c.Query("start_date"),
		EndDate:      c.Query("end_date"),
		Organization: c.Query("org_id"),
	}

	// Fetch dashboard data
	dashboardData := &models.DashboardData{
		TimeRange:    filter.TimeRange,
		Organization: filter.Organization,
		GeneratedAt:  time.Now(),
	}

	// Get metrics
	metrics, err := db.GetDashboardMetrics(sqlDB, filter)
	if err != nil {
		log.Printf("Failed to get dashboard metrics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch metrics"})
		return
	}
	dashboardData.Metrics = *metrics

	// Get daily cost trend
	dailyCosts, err := db.GetDailyCostTrend(sqlDB, filter)
	if err != nil {
		log.Printf("Failed to get daily cost trend: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cost trend"})
		return
	}
	dashboardData.DailyCosts = dailyCosts

	// Get top models
	topModels, err := db.GetTopModelsBySpend(sqlDB, filter, 10)
	if err != nil {
		log.Printf("Failed to get top models: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top models"})
		return
	}
	dashboardData.TopModels = topModels

	// Get top API keys
	topAPIKeys, err := db.GetTopAPIKeysBySpend(sqlDB, filter, 10)
	if err != nil {
		log.Printf("Failed to get top API keys: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top API keys"})
		return
	}
	dashboardData.TopAPIKeys = topAPIKeys

	// Get provider spend breakdown
	providerSpend, err := db.GetProviderSpendBreakdown(sqlDB, filter)
	if err != nil {
		log.Printf("Failed to get provider spend: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch provider spend"})
		return
	}
	dashboardData.ProviderSpend = providerSpend

	c.JSON(http.StatusOK, dashboardData)
}

func AnalyticsPageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "analytics.html", gin.H{
		"title": "Usage Analytics",
	})
}
