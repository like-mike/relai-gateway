package models

import (
	"time"
)

type Endpoint struct {
	ID               string    `json:"id" db:"id"`
	OrganizationID   string    `json:"organization_id" db:"organization_id"`
	Name             string    `json:"name" db:"name"`
	PathPrefix       string    `json:"path_prefix" db:"path_prefix"`
	Description      *string   `json:"description" db:"description"`
	PrimaryModelID   *string   `json:"primary_model_id" db:"primary_model_id"`
	FallbackModelID  *string   `json:"fallback_model_id" db:"fallback_model_id"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
	
	// Joined fields for display
	PrimaryModelName   *string `json:"primary_model_name,omitempty" db:"primary_model_name"`
	FallbackModelName  *string `json:"fallback_model_name,omitempty" db:"fallback_model_name"`
}

type EndpointCreate struct {
	Name            string  `json:"name" validate:"required,min=1,max=255"`
	PathPrefix      string  `json:"path_prefix" validate:"required,min=1,max=255,alphanum"`
	Description     *string `json:"description" validate:"omitempty,max=1000"`
	PrimaryModelID  *string `json:"primary_model_id" validate:"omitempty,uuid"`
	FallbackModelID *string `json:"fallback_model_id" validate:"omitempty,uuid"`
	IsActive        *bool   `json:"is_active"`
}

type EndpointUpdate struct {
	Name            *string `json:"name" validate:"omitempty,min=1,max=255"`
	PathPrefix      *string `json:"path_prefix" validate:"omitempty,min=1,max=255,alphanum"`
	Description     *string `json:"description" validate:"omitempty,max=1000"`
	PrimaryModelID  *string `json:"primary_model_id" validate:"omitempty,uuid"`
	FallbackModelID *string `json:"fallback_model_id" validate:"omitempty,uuid"`
	IsActive        *bool   `json:"is_active"`
}
