package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/like-mike/relai-gateway/shared/models"
)

func GetDashboardMetrics(db *sql.DB, filter models.AnalyticsFilter) (*models.DashboardMetrics, error) {
	startTime, err := parseTimeRange(filter.TimeRange, filter.StartDate)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			COUNT(*) as total_requests,
			COUNT(CASE WHEN response_status >= 200 AND response_status < 400 THEN 1 END) as successful_requests,
			COUNT(CASE WHEN response_status >= 400 THEN 1 END) as failed_requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(AVG(cost_usd), 0) as avg_cost_per_request,
			COALESCE(SUM(cost_usd), 0) as total_cost
		FROM usage_logs
		WHERE created_at >= $1
		  AND ($2 = '' OR organization_id = $2::uuid)`

	var metrics models.DashboardMetrics
	err = db.QueryRow(query, startTime, filter.Organization).Scan(
		&metrics.TotalRequests,
		&metrics.SuccessfulRequests,
		&metrics.FailedRequests,
		&metrics.TotalTokens,
		&metrics.AvgCostPerRequest,
		&metrics.TotalCost,
	)

	if err != nil {
		return nil, err
	}

	// Calculate success rate
	if metrics.TotalRequests > 0 {
		metrics.SuccessRate = float64(metrics.SuccessfulRequests) / float64(metrics.TotalRequests) * 100
	}

	return &metrics, nil
}

func GetDailyCostTrend(db *sql.DB, filter models.AnalyticsFilter) ([]models.DailyCostData, error) {
	startTime, err := parseTimeRange(filter.TimeRange, filter.StartDate)
	if err != nil {
		return nil, err
	}

	// Determine if we should group by hour or day based on time range
	var query string
	switch filter.TimeRange {
	case "6h", "12h", "24h":
		// Use hourly grouping for shorter time ranges
		query = `
			SELECT
				TO_CHAR(DATE_TRUNC('hour', created_at), 'YYYY-MM-DD HH24:00') as date,
				COALESCE(SUM(cost_usd), 0) as daily_cost,
				COUNT(*) as daily_requests
			FROM usage_logs
			WHERE created_at >= $1
			  AND ($2 = '' OR organization_id = $2::uuid)
			GROUP BY DATE_TRUNC('hour', created_at)
			ORDER BY DATE_TRUNC('hour', created_at)`
	default:
		// Use daily grouping for longer time ranges
		query = `
			SELECT
				DATE(created_at)::text as date,
				COALESCE(SUM(cost_usd), 0) as daily_cost,
				COUNT(*) as daily_requests
			FROM usage_logs
			WHERE created_at >= $1
			  AND ($2 = '' OR organization_id = $2::uuid)
			GROUP BY DATE(created_at)
			ORDER BY DATE(created_at)`
	}

	rows, err := db.Query(query, startTime, filter.Organization)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailyCosts []models.DailyCostData
	for rows.Next() {
		var data models.DailyCostData
		err := rows.Scan(&data.Date, &data.Cost, &data.RequestCount)
		if err != nil {
			return nil, err
		}
		dailyCosts = append(dailyCosts, data)
	}

	return dailyCosts, nil
}

func GetTopModelsBySpend(db *sql.DB, filter models.AnalyticsFilter, limit int) ([]models.TopModelData, error) {
	startTime, err := parseTimeRange(filter.TimeRange, filter.StartDate)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			m.name,
			m.model_id,
			COALESCE(SUM(ul.cost_usd), 0) as total_cost,
			COUNT(ul.id) as request_count
		FROM usage_logs ul
		JOIN models m ON ul.model_id = m.id
		WHERE ul.created_at >= $1
		  AND ($2 = '' OR ul.organization_id = $2::uuid)
		GROUP BY m.id, m.name, m.model_id
		ORDER BY total_cost DESC
		LIMIT $3`

	rows, err := db.Query(query, startTime, filter.Organization, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topModels []models.TopModelData
	for rows.Next() {
		var model models.TopModelData
		err := rows.Scan(&model.Name, &model.ModelID, &model.TotalCost, &model.RequestCount)
		if err != nil {
			return nil, err
		}
		topModels = append(topModels, model)
	}

	return topModels, nil
}

func GetTopAPIKeysBySpend(db *sql.DB, filter models.AnalyticsFilter, limit int) ([]models.TopAPIKeyData, error) {
	startTime, err := parseTimeRange(filter.TimeRange, filter.StartDate)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			ak.name,
			CONCAT('sk-', SUBSTRING(ak.id::text, 1, 8), '...') as key_prefix,
			COALESCE(SUM(ul.cost_usd), 0) as total_cost,
			COUNT(ul.id) as request_count
		FROM usage_logs ul
		JOIN api_keys ak ON ul.api_key_id = ak.id
		WHERE ul.created_at >= $1
		  AND ($2 = '' OR ul.organization_id = $2::uuid)
		GROUP BY ak.id, ak.name
		ORDER BY total_cost DESC
		LIMIT $3`

	rows, err := db.Query(query, startTime, filter.Organization, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topKeys []models.TopAPIKeyData
	for rows.Next() {
		var key models.TopAPIKeyData
		err := rows.Scan(&key.Name, &key.KeyPrefix, &key.TotalCost, &key.RequestCount)
		if err != nil {
			return nil, err
		}
		topKeys = append(topKeys, key)
	}

	return topKeys, nil
}

func GetProviderSpendBreakdown(db *sql.DB, filter models.AnalyticsFilter) ([]models.ProviderSpendData, error) {
	startTime, err := parseTimeRange(filter.TimeRange, filter.StartDate)
	if err != nil {
		return nil, err
	}

	// First get total spend for percentage calculation
	var totalSpend float64
	totalQuery := `
		SELECT COALESCE(SUM(ul.cost_usd), 0)
		FROM usage_logs ul
		JOIN models m ON ul.model_id = m.id
		WHERE ul.created_at >= $1
		  AND ($2 = '' OR ul.organization_id = $2::uuid)`

	err = db.QueryRow(totalQuery, startTime, filter.Organization).Scan(&totalSpend)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			m.provider,
			COALESCE(SUM(ul.cost_usd), 0) as total_cost,
			COUNT(ul.id) as request_count
		FROM usage_logs ul
		JOIN models m ON ul.model_id = m.id
		WHERE ul.created_at >= $1
		  AND ($2 = '' OR ul.organization_id = $2::uuid)
		GROUP BY m.provider
		ORDER BY total_cost DESC`

	rows, err := db.Query(query, startTime, filter.Organization)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providerSpend []models.ProviderSpendData
	for rows.Next() {
		var provider models.ProviderSpendData
		err := rows.Scan(&provider.Provider, &provider.TotalCost, &provider.RequestCount)
		if err != nil {
			return nil, err
		}

		// Calculate percentage
		if totalSpend > 0 {
			provider.Percentage = (provider.TotalCost / totalSpend) * 100
		}

		providerSpend = append(providerSpend, provider)
	}

	return providerSpend, nil
}

func parseTimeRange(timeRange, startDate string) (time.Time, error) {
	now := time.Now()

	switch timeRange {
	case "6h":
		return now.Add(-6 * time.Hour), nil
	case "12h":
		return now.Add(-12 * time.Hour), nil
	case "24h":
		return now.Add(-24 * time.Hour), nil
	case "7d":
		return now.Add(-7 * 24 * time.Hour), nil
	case "30d":
		return now.Add(-30 * 24 * time.Hour), nil
	case "custom":
		if startDate == "" {
			return time.Time{}, fmt.Errorf("start_date required for custom range")
		}
		return time.Parse("2006-01-02", startDate)
	default:
		return now.Add(-7 * 24 * time.Hour), nil // Default to 7 days
	}
}
