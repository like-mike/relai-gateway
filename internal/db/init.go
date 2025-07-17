package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func InitDB() (*sql.DB, error) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN environment variable is not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	schema := `
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS orgs (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name TEXT NOT NULL UNIQUE,
	quota_tokens BIGINT NOT NULL DEFAULT 0,
	created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	org_id UUID NOT NULL REFERENCES orgs(id),
	name TEXT,
	email TEXT UNIQUE,
	role TEXT NOT NULL DEFAULT 'user',
	created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	org_id UUID NOT NULL REFERENCES orgs(id),
	user_id UUID NOT NULL REFERENCES users(id),
	key TEXT NOT NULL UNIQUE,
	expires_at TIMESTAMP,
	active BOOLEAN DEFAULT true,
	max_tokens BIGINT,
	created_at TIMESTAMP DEFAULT now()
);
`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}
