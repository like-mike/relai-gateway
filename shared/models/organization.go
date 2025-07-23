package models

import (
	"time"
)

type Organization struct {
	ID                string    `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	Description       *string   `json:"description" db:"description"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	AdAdminGroupID    *string   `json:"ad_admin_group_id" db:"ad_admin_group_id"`
	AdAdminGroupName  *string   `json:"ad_admin_group_name" db:"ad_admin_group_name"`
	AdMemberGroupID   *string   `json:"ad_member_group_id" db:"ad_member_group_id"`
	AdMemberGroupName *string   `json:"ad_member_group_name" db:"ad_member_group_name"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type CreateOrganizationRequest struct {
	Name              string  `json:"name" binding:"required"`
	Description       *string `json:"description"`
	AdAdminGroupID    *string `json:"ad_admin_group_id"`
	AdAdminGroupName  *string `json:"ad_admin_group_name"`
	AdMemberGroupID   *string `json:"ad_member_group_id"`
	AdMemberGroupName *string `json:"ad_member_group_name"`
}

type UpdateOrganizationRequest struct {
	Name              *string `json:"name"`
	Description       *string `json:"description"`
	IsActive          *bool   `json:"is_active"`
	AdAdminGroupID    *string `json:"ad_admin_group_id"`
	AdAdminGroupName  *string `json:"ad_admin_group_name"`
	AdMemberGroupID   *string `json:"ad_member_group_id"`
	AdMemberGroupName *string `json:"ad_member_group_name"`
}

// OrganizationWithDetails extends Organization with additional details
type OrganizationWithDetails struct {
	Organization
	Quota     *OrganizationQuota `json:"quota,omitempty"`
	UserCount int                `json:"user_count"`
}
