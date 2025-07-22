package models

import (
	"time"
)

// UsageLog represents a single API usage event for tracking and billing
type UsageLog struct {
	ID               string                 `json:"id" db:"id"`
	OrganizationID   string                 `json:"organization_id" db:"organization_id"`
	APIKeyID         string                 `json:"api_key_id" db:"api_key_id"`
	ModelID          string                 `json:"model_id" db:"model_id"`
	Endpoint         string                 `json:"endpoint" db:"endpoint"`
	PromptTokens     int                    `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens" db:"completion_tokens"`
	TotalTokens      int                    `json:"total_tokens" db:"total_tokens"`
	RequestID        *string                `json:"request_id" db:"request_id"`
	ResponseStatus   int                    `json:"response_status" db:"response_status"`
	ResponseTimeMS   *int                   `json:"response_time_ms" db:"response_time_ms"`
	CostUSD          *float64               `json:"cost_usd" db:"cost_usd"`
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
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

// UsageStats represents aggregated usage statistics
type UsageStats struct {
	TotalRequests    int64   `json:"total_requests"`
	TotalTokens      int64   `json:"total_tokens"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalCostUSD     float64 `json:"total_cost_usd"`
	AvgResponseTime  float64 `json:"avg_response_time_ms"`
}

// UsageByModel represents usage statistics grouped by model
type UsageByModel struct {
	ModelID    string     `json:"model_id"`
	ModelName  string     `json:"model_name"`
	Provider   string     `json:"provider"`
	Stats      UsageStats `json:"stats"`
}

// UsageResponse represents the response for usage analytics endpoints
type UsageResponse struct {
	OrganizationID string         `json:"organization_id"`
	DateRange      string         `json:"date_range"`
	Overall        UsageStats     `json:"overall"`
	ByModel        []UsageByModel `json:"by_model"`
}

// AIProviderUsage represents usage information from AI provider responses
type AIProviderUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIUsageResponse represents the usage portion of OpenAI API responses
type OpenAIUsageResponse struct {
	Usage AIProviderUsage `json:"usage"`
}
