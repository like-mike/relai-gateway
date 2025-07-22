package models

import "time"

type DashboardMetrics struct {
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	TotalTokens        int64   `json:"total_tokens"`
	AvgCostPerRequest  float64 `json:"avg_cost_per_request"`
	TotalCost          float64 `json:"total_cost"`
	SuccessRate        float64 `json:"success_rate"`
}

type DailyCostData struct {
	Date         string  `json:"date"`
	Cost         float64 `json:"cost"`
	RequestCount int64   `json:"request_count"`
}

type TopModelData struct {
	Name         string  `json:"name"`
	ModelID      string  `json:"model_id"`
	TotalCost    float64 `json:"total_cost"`
	RequestCount int64   `json:"request_count"`
}

type TopAPIKeyData struct {
	Name         string  `json:"name"`
	KeyPrefix    string  `json:"key_prefix"`
	TotalCost    float64 `json:"total_cost"`
	RequestCount int64   `json:"request_count"`
}

type ProviderSpendData struct {
	Provider     string  `json:"provider"`
	TotalCost    float64 `json:"total_cost"`
	RequestCount int64   `json:"request_count"`
	Percentage   float64 `json:"percentage"`
}

type DashboardData struct {
	Metrics       DashboardMetrics    `json:"metrics"`
	DailyCosts    []DailyCostData     `json:"daily_costs"`
	TopModels     []TopModelData      `json:"top_models"`
	TopAPIKeys    []TopAPIKeyData     `json:"top_api_keys"`
	ProviderSpend []ProviderSpendData `json:"provider_spend"`
	TimeRange     string              `json:"time_range"`
	Organization  string              `json:"organization"`
	GeneratedAt   time.Time           `json:"generated_at"`
}

type AnalyticsFilter struct {
	TimeRange    string `json:"time_range"`
	StartDate    string `json:"start_date,omitempty"`
	EndDate      string `json:"end_date,omitempty"`
	Organization string `json:"organization,omitempty"`
}
