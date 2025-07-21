package models

import (
	"fmt"
	"time"
)

type OrganizationQuota struct {
	ID             string    `json:"id" db:"id"`
	OrganizationID string    `json:"organization_id" db:"organization_id"`
	TotalQuota     int       `json:"total_quota" db:"total_quota"`
	UsedTokens     int       `json:"used_tokens" db:"used_tokens"`
	ResetDate      time.Time `json:"reset_date" db:"reset_date"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type QuotaStats struct {
	TotalUsage     string `json:"total_usage"`
	RemainingQuota string `json:"remaining_quota"`
	PercentUsed    string `json:"percent_used"`
}

type UsageLog struct {
	ID               string    `json:"id" db:"id"`
	APIKeyID         string    `json:"api_key_id" db:"api_key_id"`
	OrganizationID   string    `json:"organization_id" db:"organization_id"`
	ModelID          string    `json:"model_id" db:"model_id"`
	TokensUsed       int       `json:"tokens_used" db:"tokens_used"`
	RequestTimestamp time.Time `json:"request_timestamp" db:"request_timestamp"`
	ResponseStatus   *int      `json:"response_status" db:"response_status"`
	DurationMs       *int      `json:"duration_ms" db:"duration_ms"`
}

// CalculateQuotaStats calculates the quota statistics for display
func (q *OrganizationQuota) CalculateQuotaStats() QuotaStats {
	remaining := q.TotalQuota - q.UsedTokens
	if remaining < 0 {
		remaining = 0
	}

	var percentUsed float64
	if q.TotalQuota > 0 {
		percentUsed = (float64(q.UsedTokens) / float64(q.TotalQuota)) * 100
		if percentUsed > 100 {
			percentUsed = 100
		}
	}

	return QuotaStats{
		TotalUsage:     fmt.Sprintf("%s", formatNumber(q.UsedTokens)),
		RemainingQuota: fmt.Sprintf("%s", formatNumber(remaining)),
		PercentUsed:    fmt.Sprintf("%.1f%%", percentUsed),
	}
}

// formatNumber formats numbers with commas for better readability
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

type UpdateQuotaRequest struct {
	TotalQuota int `json:"total_quota" binding:"required"`
}
