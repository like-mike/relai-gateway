package email

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/like-mike/relai-gateway/shared/models"
)

// Service handles all email operations
type Service struct {
	db       *sql.DB
	smtp     *SMTPClient
	renderer *TemplateRenderer
}

// NewService creates a new email service instance
func NewService(db *sql.DB) *Service {
	return &Service{
		db:       db,
		smtp:     NewSMTPClient(),
		renderer: NewTemplateRenderer(),
	}
}

// GetEmailSettings retrieves the current email settings
func (s *Service) GetEmailSettings() (*models.EmailSettings, error) {
	query := `
		SELECT id, smtp_host, smtp_port, smtp_username, smtp_password, 
		       smtp_from_name, smtp_from_email, is_enabled, created_at, updated_at
		FROM email_settings 
		ORDER BY created_at DESC 
		LIMIT 1`

	var settings models.EmailSettings
	err := s.db.QueryRow(query).Scan(
		&settings.ID, &settings.SMTPHost, &settings.SMTPPort,
		&settings.SMTPUsername, &settings.SMTPPassword,
		&settings.SMTPFromName, &settings.SMTPFromEmail,
		&settings.IsEnabled, &settings.CreatedAt, &settings.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// UpdateEmailSettings updates the email configuration
func (s *Service) UpdateEmailSettings(req models.UpdateEmailSettingsRequest) error {
	// Get existing settings or create new ones
	settings, err := s.GetEmailSettings()
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if settings == nil {
		// Create new settings
		query := `
			INSERT INTO email_settings (smtp_host, smtp_port, smtp_username, smtp_password, 
			                           smtp_from_name, smtp_from_email, is_enabled)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`

		host := getStringOrDefault(req.SMTPHost, "smtp.gmail.com")
		port := 587
		if req.SMTPPort != nil {
			if p, err := strconv.Atoi(*req.SMTPPort); err == nil {
				port = p
			}
		}
		username := getStringOrDefault(req.SMTPUsername, "")
		password := getStringOrDefault(req.SMTPPassword, "")
		fromName := getStringOrDefault(req.SMTPFromName, "RelAI Gateway")
		fromEmail := getStringOrDefault(req.SMTPFromEmail, "")
		enabled := false
		if req.IsEnabled != nil {
			enabled = bool(*req.IsEnabled)
		}

		_, err = s.db.Exec(query, host, port, username, password, fromName, fromEmail, enabled)
		return err
	}

	// Update existing settings
	setParts := []string{}
	args := []interface{}{}
	argCount := 1

	if req.SMTPHost != nil {
		setParts = append(setParts, fmt.Sprintf("smtp_host = $%d", argCount))
		args = append(args, *req.SMTPHost)
		argCount++
	}

	if req.SMTPPort != nil {
		port, err := strconv.Atoi(*req.SMTPPort)
		if err != nil {
			return fmt.Errorf("invalid SMTP port: %v", err)
		}
		setParts = append(setParts, fmt.Sprintf("smtp_port = $%d", argCount))
		args = append(args, port)
		argCount++
	}

	if req.SMTPUsername != nil {
		setParts = append(setParts, fmt.Sprintf("smtp_username = $%d", argCount))
		args = append(args, *req.SMTPUsername)
		argCount++
	}

	if req.SMTPPassword != nil {
		setParts = append(setParts, fmt.Sprintf("smtp_password = $%d", argCount))
		args = append(args, *req.SMTPPassword)
		argCount++
	}

	if req.SMTPFromName != nil {
		setParts = append(setParts, fmt.Sprintf("smtp_from_name = $%d", argCount))
		args = append(args, *req.SMTPFromName)
		argCount++
	}

	if req.SMTPFromEmail != nil {
		setParts = append(setParts, fmt.Sprintf("smtp_from_email = $%d", argCount))
		args = append(args, *req.SMTPFromEmail)
		argCount++
	}

	if req.IsEnabled != nil {
		enabled := bool(*req.IsEnabled)
		setParts = append(setParts, fmt.Sprintf("is_enabled = $%d", argCount))
		args = append(args, enabled)
		argCount++
	}

	if len(setParts) == 0 {
		return nil // Nothing to update
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = NOW()"))

	query := fmt.Sprintf("UPDATE email_settings SET %s WHERE id = $%d",
		strings.Join(setParts, ", "), argCount)
	args = append(args, settings.ID)

	_, err = s.db.Exec(query, args...)
	return err
}

// SendTestEmail sends a test email using the specified template
func (s *Service) SendTestEmail(req models.SendTestEmailRequest) error {
	// Get email settings
	settings, err := s.GetEmailSettings()
	if err != nil {
		return fmt.Errorf("failed to get email settings: %v", err)
	}

	if !settings.IsEnabled {
		return fmt.Errorf("email service is disabled")
	}

	// Get template
	template, err := s.GetEmailTemplate(req.TemplateID)
	if err != nil {
		return fmt.Errorf("failed to get email template: %v", err)
	}

	// Use test data or default sample data
	variables := req.TestData
	if variables == nil {
		variables = &models.EmailTemplateVariables{
			UserName:            "Test User",
			APIKeyName:          "test-api-key",
			ExpirationDate:      "2024-01-15",
			OrganizationName:    "Test Organization",
			DaysUntilExpiration: 7,
			ManagementURL:       "https://your-gateway.com/admin",
		}
	}

	// Render email content
	subject, err := s.renderer.RenderText(template.Subject, variables)
	if err != nil {
		return fmt.Errorf("failed to render subject: %v", err)
	}

	htmlBody, err := s.renderer.RenderHTML(template.HTMLBody, variables)
	if err != nil {
		return fmt.Errorf("failed to render HTML body: %v", err)
	}

	// Send email
	err = s.smtp.SendEmail(SMTPConfig{
		Host:      settings.SMTPHost,
		Port:      settings.SMTPPort,
		Username:  settings.SMTPUsername.String,
		Password:  settings.SMTPPassword.String,
		FromName:  settings.SMTPFromName.String,
		FromEmail: settings.SMTPFromEmail.String,
	}, EmailMessage{
		To:      req.RecipientEmail,
		Subject: subject,
		Body:    htmlBody,
		IsHTML:  true,
	})

	// Log the email attempt
	s.logEmail(req.RecipientEmail, subject, &req.TemplateID, err)

	return err
}

// GetEmailTemplate retrieves an email template by ID
func (s *Service) GetEmailTemplate(id string) (*models.EmailTemplate, error) {
	query := `
		SELECT id, name, type, subject, html_body, text_body, is_active, created_at, updated_at
		FROM email_templates 
		WHERE id = $1`

	var template models.EmailTemplate
	err := s.db.QueryRow(query, id).Scan(
		&template.ID, &template.Name, &template.Type, &template.Subject,
		&template.HTMLBody, &template.TextBody, &template.IsActive,
		&template.CreatedAt, &template.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &template, nil
}

// GetAllEmailTemplates retrieves all email templates
func (s *Service) GetAllEmailTemplates() ([]models.EmailTemplate, error) {
	query := `
		SELECT id, name, type, subject, html_body, text_body, is_active, created_at, updated_at
		FROM email_templates 
		ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.EmailTemplate
	for rows.Next() {
		var template models.EmailTemplate
		err := rows.Scan(
			&template.ID, &template.Name, &template.Type, &template.Subject,
			&template.HTMLBody, &template.TextBody, &template.IsActive,
			&template.CreatedAt, &template.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// CreateEmailTemplate creates a new email template
func (s *Service) CreateEmailTemplate(req models.CreateEmailTemplateRequest) (*models.EmailTemplate, error) {
	query := `
		INSERT INTO email_templates (name, type, subject, html_body, text_body, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, type, subject, html_body, text_body, is_active, created_at, updated_at`

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var template models.EmailTemplate
	err := s.db.QueryRow(query, req.Name, req.Type, req.Subject, req.HTMLBody, req.TextBody, isActive).Scan(
		&template.ID, &template.Name, &template.Type, &template.Subject,
		&template.HTMLBody, &template.TextBody, &template.IsActive,
		&template.CreatedAt, &template.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &template, nil
}

// UpdateEmailTemplate updates an existing email template
func (s *Service) UpdateEmailTemplate(id string, req models.UpdateEmailTemplateRequest) (*models.EmailTemplate, error) {
	setParts := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argCount))
		args = append(args, *req.Name)
		argCount++
	}

	if req.Type != nil {
		setParts = append(setParts, fmt.Sprintf("type = $%d", argCount))
		args = append(args, *req.Type)
		argCount++
	}

	if req.Subject != nil {
		setParts = append(setParts, fmt.Sprintf("subject = $%d", argCount))
		args = append(args, *req.Subject)
		argCount++
	}

	if req.HTMLBody != nil {
		setParts = append(setParts, fmt.Sprintf("html_body = $%d", argCount))
		args = append(args, *req.HTMLBody)
		argCount++
	}

	if req.TextBody != nil {
		setParts = append(setParts, fmt.Sprintf("text_body = $%d", argCount))
		args = append(args, *req.TextBody)
		argCount++
	}

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argCount))
		args = append(args, *req.IsActive)
		argCount++
	}

	if len(setParts) == 0 {
		return s.GetEmailTemplate(id) // Nothing to update, return existing
	}

	setParts = append(setParts, "updated_at = NOW()")

	query := fmt.Sprintf(`
		UPDATE email_templates 
		SET %s 
		WHERE id = $%d
		RETURNING id, name, type, subject, html_body, text_body, is_active, created_at, updated_at`,
		strings.Join(setParts, ", "), argCount)
	args = append(args, id)

	var template models.EmailTemplate
	err := s.db.QueryRow(query, args...).Scan(
		&template.ID, &template.Name, &template.Type, &template.Subject,
		&template.HTMLBody, &template.TextBody, &template.IsActive,
		&template.CreatedAt, &template.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &template, nil
}

// logEmail records an email send attempt
func (s *Service) logEmail(recipient, subject string, templateID *string, sendErr error) {
	status := "sent"
	var errorMessage *string

	if sendErr != nil {
		status = "failed"
		errMsg := sendErr.Error()
		errorMessage = &errMsg
	}

	query := `
		INSERT INTO email_logs (recipient_email, subject, template_id, status, error_message, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	var sentAt interface{}
	if sendErr == nil {
		sentAt = "NOW()"
	}

	_, err := s.db.Exec(query, recipient, subject, templateID, status, errorMessage, sentAt)
	if err != nil {
		log.Printf("Failed to log email: %v", err)
	}
}

// Helper functions
func getStringOrDefault(ptr *string, defaultVal string) string {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}
