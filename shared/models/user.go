package models

import (
	"time"
)

type User struct {
	ID        string     `json:"id" db:"id"`
	AzureOID  string     `json:"azure_oid" db:"azure_oid"`
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	IsActive  bool       `json:"is_active" db:"is_active"`
	LastLogin *time.Time `json:"last_login" db:"last_login"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// Legacy User struct for backwards compatibility
type LegacyUser struct {
	ID           string     `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        *string    `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	LastLogin    *time.Time `json:"last_login" db:"last_login"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type LoginRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
}

type CreateUserRequest struct {
	AzureOID string `json:"azure_oid" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Name     string `json:"name" binding:"required"`
}

type UpdateUserRequest struct {
	Email    *string `json:"email"`
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

// UserOrgMembership represents a user's membership in an organization
type UserOrgMembership struct {
	OrgID    string `json:"org_id"`
	OrgName  string `json:"org_name"`
	RoleName string `json:"role_name"`
}

// UserWithOrganizations represents a user with their organization memberships
type UserWithOrganizations struct {
	User
	Organizations []UserOrgMembership `json:"organizations"`
}
