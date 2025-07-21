package models

import (
	"time"

	"github.com/lib/pq"
)

type APIKey struct {
	ID             string         `json:"id" db:"id"`
	Name           string         `json:"name" db:"name"`
	Description    *string        `json:"description" db:"description"`
	KeyHash        string         `json:"-" db:"key_hash"`
	KeyPrefix      string         `json:"key" db:"key_prefix"`
	OrganizationID string         `json:"organization_id" db:"organization_id"`
	UserID         *string        `json:"user_id" db:"user_id"`
	Type           string         `json:"type" db:"type"`
	MaxTokens      int            `json:"max_tokens" db:"max_tokens"`
	Permissions    pq.StringArray `json:"permissions" db:"permissions"`
	IsActive       bool           `json:"active" db:"is_active"`
	LastUsed       *time.Time     `json:"last_used" db:"last_used"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
	Organization   *Organization  `json:"organization,omitempty"`
	User           *User          `json:"user,omitempty"`
}

type CreateAPIKeyRequest struct {
	Name           string   `json:"name" form:"name" binding:"required"`
	Description    *string  `json:"description" form:"description"`
	Type           string   `json:"type" form:"type" binding:"required"`
	MaxTokens      int      `json:"max_tokens" form:"max_tokens"`
	Permissions    []string `json:"permissions" form:"permissions[]"`
	OrganizationID string   `json:"organization_id" form:"organization_id"`
	UserID         *string  `json:"user_id" form:"user_id"`
}

type CreateAPIKeyResponse struct {
	APIKey  APIKey `json:"api_key"`
	FullKey string `json:"full_key"` // Only returned once during creation
	Message string `json:"message"`
}

// APIKeyTableData represents the data structure for the HTMX table response
type APIKeyTableData struct {
	APIKeys []APIKey `json:"api_keys"`
}
