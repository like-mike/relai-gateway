package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromName  string
	FromEmail string
}

// EmailMessage represents an email to be sent
type EmailMessage struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

// SMTPClient handles sending emails via SMTP
type SMTPClient struct{}

// NewSMTPClient creates a new SMTP client
func NewSMTPClient() *SMTPClient {
	return &SMTPClient{}
}

// SendEmail sends an email using the provided SMTP configuration
func (c *SMTPClient) SendEmail(config SMTPConfig, message EmailMessage) error {
	// Create the email headers and body
	var body string
	if message.IsHTML {
		body = fmt.Sprintf("MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\nFrom: %s <%s>\nTo: %s\nSubject: %s\n\n%s",
			config.FromName, config.FromEmail, message.To, message.Subject, message.Body)
	} else {
		body = fmt.Sprintf("From: %s <%s>\nTo: %s\nSubject: %s\n\n%s",
			config.FromName, config.FromEmail, message.To, message.Subject, message.Body)
	}

	// Send using STARTTLS
	err := c.sendMailSTARTTLS(config, config.FromEmail, []string{message.To}, []byte(body), false)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

// sendMailSTARTTLS sends email using STARTTLS (proper method for Gmail)
func (c *SMTPClient) sendMailSTARTTLS(config SMTPConfig, from string, to []string, msg []byte, testOnly bool) error {
	// Set up authentication
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	// Gmail SMTP server address
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	// For Gmail, use the built-in smtp.SendMail which handles STARTTLS properly
	if !testOnly {
		return smtp.SendMail(addr, auth, from, to, msg)
	}

	// For connection testing, manually establish connection
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}
	defer client.Quit()

	// Start TLS if available
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: config.Host,
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %v", err)
		}
	}

	// Authenticate
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err = client.Auth(auth); err != nil {
				return fmt.Errorf("SMTP authentication failed: %v", err)
			}
		}
	}

	return nil
}

// TestConnection tests the SMTP connection with the provided configuration
func (c *SMTPClient) TestConnection(config SMTPConfig) error {
	return c.sendMailSTARTTLS(config, "", []string{}, []byte(""), true)
}
