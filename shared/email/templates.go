package email

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/like-mike/relai-gateway/shared/models"
)

// TemplateRenderer handles rendering email templates with variables
type TemplateRenderer struct{}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{}
}

// RenderHTML renders an HTML template with the provided variables
func (r *TemplateRenderer) RenderHTML(templateStr string, variables *models.EmailTemplateVariables) (string, error) {
	tmpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, variables)
	if err != nil {
		return "", fmt.Errorf("failed to execute HTML template: %v", err)
	}

	return buf.String(), nil
}

// RenderText renders a text template with the provided variables (using html/template for simplicity)
func (r *TemplateRenderer) RenderText(templateStr string, variables *models.EmailTemplateVariables) (string, error) {
	tmpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse text template: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, variables)
	if err != nil {
		return "", fmt.Errorf("failed to execute text template: %v", err)
	}

	return buf.String(), nil
}

// ValidateTemplate validates that a template string is syntactically correct
func (r *TemplateRenderer) ValidateTemplate(templateStr string) error {
	_, err := template.New("validation").Parse(templateStr)
	if err != nil {
		return fmt.Errorf("template validation failed: %v", err)
	}
	return nil
}

// GetSampleVariables returns sample data for template preview
func (r *TemplateRenderer) GetSampleVariables() models.EmailTemplateVariables {
	return models.EmailTemplateVariables{
		UserName:            "John Doe",
		APIKeyName:          "production-api-key",
		ExpirationDate:      "January 15, 2024",
		OrganizationName:    "Acme Corporation",
		DaysUntilExpiration: 7,
		ManagementURL:       "https://your-gateway.com/admin",
	}
}

// PreviewTemplate renders a template with sample data for preview purposes
func (r *TemplateRenderer) PreviewTemplate(subject, htmlBody string) (string, string, error) {
	sampleVars := r.GetSampleVariables()

	renderedSubject, err := r.RenderText(subject, &sampleVars)
	if err != nil {
		return "", "", fmt.Errorf("failed to render subject: %v", err)
	}

	renderedHTML, err := r.RenderHTML(htmlBody, &sampleVars)
	if err != nil {
		return "", "", fmt.Errorf("failed to render HTML: %v", err)
	}

	return renderedSubject, renderedHTML, nil
}

// GetAvailableVariables returns a list of available template variables with descriptions
func (r *TemplateRenderer) GetAvailableVariables() map[string]string {
	return map[string]string{
		"{{.UserName}}":            "The name of the user who owns the API key",
		"{{.APIKeyName}}":          "The name/identifier of the API key",
		"{{.ExpirationDate}}":      "The date when the API key expires",
		"{{.OrganizationName}}":    "The name of the organization",
		"{{.DaysUntilExpiration}}": "Number of days until the API key expires",
		"{{.ManagementURL}}":       "URL to the API key management interface",
	}
}
