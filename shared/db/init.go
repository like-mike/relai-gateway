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

		log.Println("Database schema initialized successfully")
	} else {
		log.Println("Database schema already exists")
	}

	return nil
}

// GetDB is a helper function to get database connection from context
func GetDB(c interface{}) (*sql.DB, bool) {
	// This will be implemented based on how the DB is stored in context
	// For now, return nil, false
	return nil, false
}
