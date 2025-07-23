package db

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
)

func InitDB() (*sql.DB, error) {
	// Get database connection string from POSTGRES_DSN environment variable
	connStr := os.Getenv("POSTGRES_DSN")

	// If POSTGRES_DSN is not set, fall back to individual environment variables
	if connStr == "" {
		// Database connection parameters
		dbHost := os.Getenv("DB_HOST")
		dbPort := os.Getenv("DB_PORT")
		dbUser := os.Getenv("DB_USER")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbName := os.Getenv("DB_NAME")
		dbSSLMode := os.Getenv("DB_SSLMODE")

		// Set defaults if not provided
		if dbHost == "" {
			dbHost = "localhost"
		}
		if dbPort == "" {
			dbPort = "5432"
		}
		if dbUser == "" {
			dbUser = "postgres"
		}
		if dbPassword == "" {
			dbPassword = "postgres"
		}
		if dbName == "" {
			dbName = "relai_gateway"
		}
		if dbSSLMode == "" {
			dbSSLMode = "disable"
		}

		// Build connection string from individual components
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	}

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize schema if needed
	if err := initializeSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Printf("Successfully connected to database using POSTGRES_DSN")
	return db, nil
}

func initializeSchema(db *sql.DB) error {
	// Check if the organizations table exists
	var exists bool
	query := `SELECT EXISTS (
		SELECT FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name = 'organizations'
	);`

	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if schema exists: %w", err)
	}

	// If tables don't exist, create them
	if !exists {
		log.Println("Database schema not found, initializing...")
		err = createSchema(db)
		if err != nil {
			return err
		}
		log.Println("Database schema initialized successfully")
	} else {
		log.Println("Database schema already exists, checking for updates...")
		err = updateSchema(db)
		if err != nil {
			return err
		}
	}

	return nil
}

func createSchema(db *sql.DB) error {
	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Try different possible paths for the schema file
	schemaPaths := []string{
		filepath.Join(wd, "shared", "db", "schema.sql"),
		filepath.Join(wd, "..", "shared", "db", "schema.sql"),
		filepath.Join(wd, "..", "..", "shared", "db", "schema.sql"),
		"shared/db/schema.sql",
	}

	var schemaContent []byte
	var schemaPath string

	for _, path := range schemaPaths {
		if _, err := os.Stat(path); err == nil {
			schemaContent, err = ioutil.ReadFile(path)
			if err == nil {
				schemaPath = path
				break
			}
		}
	}

	if schemaContent == nil {
		return fmt.Errorf("schema.sql file not found in any of the expected locations: %v", schemaPaths)
	}

	log.Printf("Loading schema from: %s", schemaPath)

	// Execute the schema
	_, err = db.Exec(string(schemaContent))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

func updateSchema(db *sql.DB) error {
	// Check if models table has api_endpoint and api_token columns
	var hasAPIEndpoint bool
	var hasAPIToken bool

	checkColumnQuery := `SELECT EXISTS (
		SELECT FROM information_schema.columns
		WHERE table_schema = 'public'
		AND table_name = 'models'
		AND column_name = $1
	);`

	err := db.QueryRow(checkColumnQuery, "api_endpoint").Scan(&hasAPIEndpoint)
	if err != nil {
		return fmt.Errorf("failed to check api_endpoint column: %w", err)
	}

	err = db.QueryRow(checkColumnQuery, "api_token").Scan(&hasAPIToken)
	if err != nil {
		return fmt.Errorf("failed to check api_token column: %w", err)
	}

	// Add missing columns
	if !hasAPIEndpoint {
		log.Println("Adding api_endpoint column to models table...")
		_, err = db.Exec("ALTER TABLE models ADD COLUMN api_endpoint VARCHAR(500)")
		if err != nil {
			return fmt.Errorf("failed to add api_endpoint column: %w", err)
		}
	}

	if !hasAPIToken {
		log.Println("Adding api_token column to models table...")
		_, err = db.Exec("ALTER TABLE models ADD COLUMN api_token VARCHAR(500)")
		if err != nil {
			return fmt.Errorf("failed to add api_token column: %w", err)
		}
	}

	// Remove unique constraint on model_id if it exists
	var hasUniqueConstraint bool
	constraintQuery := `SELECT EXISTS (
		SELECT FROM information_schema.table_constraints
		WHERE table_schema = 'public'
		AND table_name = 'models'
		AND constraint_type = 'UNIQUE'
		AND constraint_name = 'models_model_id_key'
	);`

	err = db.QueryRow(constraintQuery).Scan(&hasUniqueConstraint)
	if err != nil {
		return fmt.Errorf("failed to check unique constraint: %w", err)
	}

	if hasUniqueConstraint {
		log.Println("Removing unique constraint on model_id...")
		_, err = db.Exec("ALTER TABLE models DROP CONSTRAINT models_model_id_key")
		if err != nil {
			return fmt.Errorf("failed to drop unique constraint: %w", err)
		}
	}

	// Check if email_settings table exists
	var emailTablesExist bool
	checkEmailTableQuery := `SELECT EXISTS (
		SELECT FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name = 'email_settings'
	);`

	err = db.QueryRow(checkEmailTableQuery).Scan(&emailTablesExist)
	if err != nil {
		return fmt.Errorf("failed to check email_settings table: %w", err)
	}

	if !emailTablesExist {
		log.Println("Email tables not found, creating them...")
		emailSQL := `
		-- Email system tables for notifications and reminders
		-- Email settings table for SMTP configuration
		CREATE TABLE IF NOT EXISTS email_settings (
		    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		    smtp_host VARCHAR(255) DEFAULT 'smtp.gmail.com',
		    smtp_port INTEGER DEFAULT 587,
		    smtp_username VARCHAR(255),
		    smtp_password VARCHAR(255), -- Encrypted
		    smtp_from_name VARCHAR(255),
		    smtp_from_email VARCHAR(255),
		    is_enabled BOOLEAN DEFAULT false,
		    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Email templates for different notification types
		CREATE TABLE IF NOT EXISTS email_templates (
		    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		    name VARCHAR(255) NOT NULL,
		    type VARCHAR(100) NOT NULL, -- 'warning', 'expiration', 'usage'
		    subject VARCHAR(500) NOT NULL,
		    html_body TEXT NOT NULL,
		    text_body TEXT,
		    is_active BOOLEAN DEFAULT true,
		    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Email reminder schedules
		CREATE TABLE IF NOT EXISTS email_schedules (
		    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		    organization_id UUID REFERENCES organizations(id),
		    schedule_type VARCHAR(100) NOT NULL, -- 'api_key_warning', 'api_key_expiration'
		    days_before INTEGER, -- For warnings (7, 3, 1 days before)
		    is_enabled BOOLEAN DEFAULT true,
		    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Email logs for tracking sent emails
		CREATE TABLE IF NOT EXISTS email_logs (
		    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		    recipient_email VARCHAR(255) NOT NULL,
		    subject VARCHAR(500),
		    template_id UUID REFERENCES email_templates(id),
		    status VARCHAR(50) NOT NULL, -- 'sent', 'failed', 'pending'
		    error_message TEXT,
		    sent_at TIMESTAMP WITH TIME ZONE,
		    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Indexes for email tables
		CREATE INDEX IF NOT EXISTS idx_email_templates_type ON email_templates(type);
		CREATE INDEX IF NOT EXISTS idx_email_schedules_org_id ON email_schedules(organization_id);
		CREATE INDEX IF NOT EXISTS idx_email_schedules_type ON email_schedules(schedule_type);
		CREATE INDEX IF NOT EXISTS idx_email_logs_recipient ON email_logs(recipient_email);
		CREATE INDEX IF NOT EXISTS idx_email_logs_status ON email_logs(status);
		CREATE INDEX IF NOT EXISTS idx_email_logs_sent_at ON email_logs(sent_at);

		-- Insert default email templates
		INSERT INTO email_templates (id, name, type, subject, html_body, text_body) VALUES
		('10000000-0000-0000-0000-000000000001', 'API Key Warning - 7 Days', 'warning', 
		 'Your API Key expires in {{.DaysUntilExpiration}} days', 
		 '<!DOCTYPE html><html><head><style>body{font-family:Arial,sans-serif;margin:40px;color:#333}.header{background:#f8f9fa;padding:20px;border-radius:8px;margin-bottom:20px}.warning{background:#fff3cd;border:1px solid #ffeaa7;padding:15px;border-radius:5px;margin:20px 0}.button{display:inline-block;background:#007bff;color:white;padding:10px 20px;text-decoration:none;border-radius:5px;margin:10px 0}</style></head><body><div class="header"><h2>🔑 API Key Expiration Warning</h2></div><p>Hello {{.UserName}},</p><p>This is a friendly reminder that your API key will expire soon:</p><div class="warning"><strong>API Key:</strong> {{.APIKeyName}}<br><strong>Organization:</strong> {{.OrganizationName}}<br><strong>Expires in:</strong> {{.DaysUntilExpiration}} days<br><strong>Expiration Date:</strong> {{.ExpirationDate}}</div><p>To avoid service interruption, please renew your API key before it expires.</p><a href="{{.ManagementURL}}" class="button">Manage API Keys</a><p>Best regards,<br>RelAI Gateway Team</p></body></html>',
		 'Hello {{.UserName}}, Your API key "{{.APIKeyName}}" for organization "{{.OrganizationName}}" will expire in {{.DaysUntilExpiration}} days on {{.ExpirationDate}}. Please renew it at: {{.ManagementURL}}'),

		('10000000-0000-0000-0000-000000000002', 'API Key Expired', 'expiration',
		 'Your API Key has expired',
		 '<!DOCTYPE html><html><head><style>body{font-family:Arial,sans-serif;margin:40px;color:#333}.header{background:#f8f9fa;padding:20px;border-radius:8px;margin-bottom:20px}.alert{background:#f8d7da;border:1px solid #f5c6cb;padding:15px;border-radius:5px;margin:20px 0}.button{display:inline-block;background:#dc3545;color:white;padding:10px 20px;text-decoration:none;border-radius:5px;margin:10px 0}</style></head><body><div class="header"><h2>🚨 API Key Expired</h2></div><p>Hello {{.UserName}},</p><p>Your API key has expired and is no longer active:</p><div class="alert"><strong>API Key:</strong> {{.APIKeyName}}<br><strong>Organization:</strong> {{.OrganizationName}}<br><strong>Expired on:</strong> {{.ExpirationDate}}</div><p>Please create a new API key to restore service.</p><a href="{{.ManagementURL}}" class="button">Create New API Key</a><p>Best regards,<br>RelAI Gateway Team</p></body></html>',
		 'Hello {{.UserName}}, Your API key "{{.APIKeyName}}" for organization "{{.OrganizationName}}" has expired on {{.ExpirationDate}}. Please create a new one at: {{.ManagementURL}}')

		ON CONFLICT (id) DO NOTHING;

		-- Insert default email settings (disabled by default)
		INSERT INTO email_settings (id, smtp_from_name, smtp_from_email, is_enabled) VALUES
		('20000000-0000-0000-0000-000000000001', 'RelAI Gateway', 'noreply@relai-gateway.com', false)
		ON CONFLICT (id) DO NOTHING;
		`

		_, err = db.Exec(emailSQL)
		if err != nil {
			return fmt.Errorf("failed to create email tables: %w", err)
		}

		log.Println("Email tables created successfully")
	}

	if !hasAPIEndpoint || !hasAPIToken || hasUniqueConstraint || !emailTablesExist {
		log.Println("Schema updated successfully")
	}

	return nil

}

// GetDB is a helper function to get database connection from context
func GetDB(c interface{}) (*sql.DB, bool) {
	// This will be implemented based on how the DB is stored in context
	// For now, return nil, false
	return nil, false
}
