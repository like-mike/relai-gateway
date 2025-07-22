package models

import (
	"time"
)

type Model struct {
	ID                string         `json:"id" db:"id"`
	Name              string         `json:"name" db:"name"`
	Description       *string        `json:"description" db:"description"`
	Provider          string         `json:"provider" db:"provider"`
	ModelID           string         `json:"model_id" db:"model_id"`
	APIEndpoint       *string        `json:"api_endpoint" db:"api_endpoint"`
	APIToken          *string        `json:"api_token" db:"api_token"`
	InputCostPer1M    *float64       `json:"input_cost_per_1m" db:"input_cost_per_1m"`
	OutputCostPer1M   *float64       `json:"output_cost_per_1m" db:"output_cost_per_1m"`
	MaxRetries        *int           `json:"max_retries" db:"max_retries"`
	TimeoutSeconds    *int           `json:"timeout_seconds" db:"timeout_seconds"`
	RetryDelayMs      *int           `json:"retry_delay_ms" db:"retry_delay_ms"`
	BackoffMultiplier *float64       `json:"backoff_multiplier" db:"backoff_multiplier"`
	IsActive          bool           `json:"active" db:"is_active"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" db:"updated_at"`
	Organizations     []Organization `json:"organizations,omitempty"`
}

type CreateModelRequest struct {
	Name              string   `json:"name" binding:"required"`
	Description       *string  `json:"description"`
	Provider          string   `json:"provider" binding:"required"`
	ModelID           string   `json:"model_id" binding:"required"`
	APIEndpoint       *string  `json:"api_endpoint"`
	APIToken          *string  `json:"api_token"`
	InputCostPer1M    *string  `json:"input_cost_per_1m"`
	OutputCostPer1M   *string  `json:"output_cost_per_1m"`
	MaxRetries        *string  `json:"max_retries"`
	TimeoutSeconds    *string  `json:"timeout_seconds"`
	RetryDelayMs      *string  `json:"retry_delay_ms"`
	BackoffMultiplier *string  `json:"backoff_multiplier"`
	OrgIDs            []string `json:"organization_ids"`
}

type UpdateModelRequest struct {
	Name              *string  `json:"name"`
	Description       *string  `json:"description"`
	Provider          *string  `json:"provider"`
	ModelID           *string  `json:"model_id"`
	APIEndpoint       *string  `json:"api_endpoint"`
	APIToken          *string  `json:"api_token"`
	InputCostPer1M    *string  `json:"input_cost_per_1m"`
	OutputCostPer1M   *string  `json:"output_cost_per_1m"`
	MaxRetries        *string  `json:"max_retries"`
	TimeoutSeconds    *string  `json:"timeout_seconds"`
	RetryDelayMs      *string  `json:"retry_delay_ms"`
	BackoffMultiplier *string  `json:"backoff_multiplier"`
	IsActive          *bool    `json:"is_active"`
	OrgIDs            []string `json:"organization_ids"`
}

type ModelOrganizationAccess struct {
	ID             string    `json:"id" db:"id"`
	ModelID        string    `json:"model_id" db:"model_id"`
	OrganizationID string    `json:"organization_id" db:"organization_id"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ModelsResponse represents the JSON response for the models API
type ModelsResponse struct {
	Models []Model `json:"models"`
}
