package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// EmailSettings represents the SMTP configuration for sending emails
type EmailSettings struct {
	ID            string         `json:"id" db:"id"`
	SMTPHost      string         `json:"smtp_host" db:"smtp_host"`
	SMTPPort      int            `json:"smtp_port" db:"smtp_port"`
	SMTPUsername  sql.NullString `json:"-" db:"smtp_username"`
	SMTPPassword  sql.NullString `json:"-" db:"smtp_password"` // Encrypted
	SMTPFromName  sql.NullString `json:"-" db:"smtp_from_name"`
	SMTPFromEmail sql.NullString `json:"-" db:"smtp_from_email"`
	IsEnabled     bool           `json:"is_enabled" db:"is_enabled"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
}

// Custom JSON marshaling to handle sql.NullString properly
func (e EmailSettings) MarshalJSON() ([]byte, error) {
	type Alias EmailSettings
	return json.Marshal(&struct {
		SMTPUsername  string `json:"smtp_username"`
		SMTPPassword  string `json:"smtp_password"`
		SMTPFromName  string `json:"smtp_from_name"`
		SMTPFromEmail string `json:"smtp_from_email"`
		*Alias
	}{
		SMTPUsername:  e.SMTPUsername.String,
		SMTPPassword:  e.SMTPPassword.String,
		SMTPFromName:  e.SMTPFromName.String,
		SMTPFromEmail: e.SMTPFromEmail.String,
		Alias:         (*Alias)(&e),
	})
}

// EmailTemplate represents an email template for notifications
type EmailTemplate struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"` // 'warning', 'expiration', 'usage'
	Subject   string    `json:"subject" db:"subject"`
	HTMLBody  string    `json:"html_body" db:"html_body"`
	TextBody  *string   `json:"text_body" db:"text_body"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// EmailSchedule represents scheduled email reminders
type EmailSchedule struct {
	ID             string    `json:"id" db:"id"`
	OrganizationID *string   `json:"organization_id" db:"organization_id"`
	ScheduleType   string    `json:"schedule_type" db:"schedule_type"` // 'api_key_warning', 'api_key_expiration'
	DaysBefore     *int      `json:"days_before" db:"days_before"`     // For warnings (7, 3, 1 days before)
	IsEnabled      bool      `json:"is_enabled" db:"is_enabled"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// EmailLog represents a record of sent emails
type EmailLog struct {
	ID             string     `json:"id" db:"id"`
	RecipientEmail string     `json:"recipient_email" db:"recipient_email"`
	Subject        *string    `json:"subject" db:"subject"`
	TemplateID     *string    `json:"template_id" db:"template_id"`
	Status         string     `json:"status" db:"status"` // 'sent', 'failed', 'pending'
	ErrorMessage   *string    `json:"error_message" db:"error_message"`
	SentAt         *time.Time `json:"sent_at" db:"sent_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// EmailTemplateVariables represents the variables available for email templates
type EmailTemplateVariables struct {
	UserName            string `json:"user_name"`
	APIKeyName          string `json:"api_key_name"`
	ExpirationDate      string `json:"expiration_date"`
	OrganizationName    string `json:"organization_name"`
	DaysUntilExpiration int    `json:"days_until_expiration"`
	ManagementURL       string `json:"management_url"`
}

// CreateEmailTemplateRequest represents a request to create a new email template
type CreateEmailTemplateRequest struct {
	Name     string  `json:"name" binding:"required"`
	Type     string  `json:"type" binding:"required"`
	Subject  string  `json:"subject" binding:"required"`
	HTMLBody string  `json:"html_body" binding:"required"`
	TextBody *string `json:"text_body"`
	IsActive *bool   `json:"is_active"`
}

// UpdateEmailTemplateRequest represents a request to update an email template
type UpdateEmailTemplateRequest struct {
	Name     *string `json:"name"`
	Type     *string `json:"type"`
	Subject  *string `json:"subject"`
	HTMLBody *string `json:"html_body"`
	TextBody *string `json:"text_body"`
	IsActive *bool   `json:"is_active"`
}

// FlexibleBool is a custom type that can unmarshal from both bool and string
type FlexibleBool bool

func (fb *FlexibleBool) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as bool first
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*fb = FlexibleBool(b)
		return nil
	}

	// Try to unmarshal as string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		switch s {
		case "true", "1", "on":
			*fb = FlexibleBool(true)
		case "false", "0", "off", "":
			*fb = FlexibleBool(false)
		default:
			return fmt.Errorf("invalid boolean string: %s", s)
		}
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleBool", data)
}

// UpdateEmailSettingsRequest represents a request to update email settings
type UpdateEmailSettingsRequest struct {
	SMTPHost      *string       `json:"smtp_host"`
	SMTPPort      *string       `json:"smtp_port"` // Accept as string and convert in handler
	SMTPUsername  *string       `json:"smtp_username"`
	SMTPPassword  *string       `json:"smtp_password"`
	SMTPFromName  *string       `json:"smtp_from_name"`
	SMTPFromEmail *string       `json:"smtp_from_email"`
	IsEnabled     *FlexibleBool `json:"is_enabled"` // Can handle both bool and string
}

// SendTestEmailRequest represents a request to send a test email
type SendTestEmailRequest struct {
	RecipientEmail string                  `json:"recipient_email" binding:"required,email"`
	TemplateID     string                  `json:"template_id" binding:"required"`
	TestData       *EmailTemplateVariables `json:"test_data"`
}

// EmailTemplateWithVariables includes template and sample variables for preview
type EmailTemplateWithVariables struct {
	EmailTemplate
	SampleVariables EmailTemplateVariables `json:"sample_variables"`
}
