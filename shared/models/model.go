package models

import (
	"time"
)

type Model struct {
	ID            string         `json:"id" db:"id"`
	Name          string         `json:"name" db:"name"`
	Description   *string        `json:"description" db:"description"`
	Provider      string         `json:"provider" db:"provider"`
	ModelID       string         `json:"model_id" db:"model_id"`
	IsActive      bool           `json:"active" db:"is_active"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
	Organizations []Organization `json:"organizations,omitempty"`
}

type CreateModelRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description *string  `json:"description"`
	Provider    string   `json:"provider" binding:"required"`
	ModelID     string   `json:"model_id" binding:"required"`
	OrgIDs      []string `json:"organization_ids"`
}

type UpdateModelRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Provider    *string  `json:"provider"`
	ModelID     *string  `json:"model_id"`
	IsActive    *bool    `json:"is_active"`
	OrgIDs      []string `json:"organization_ids"`
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
