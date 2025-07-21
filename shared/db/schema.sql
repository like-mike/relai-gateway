-- RelAI Gateway Database Schema
-- Generated from DATABASE_SCHEMA_DESIGN.md

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Users table for admin authentication
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    last_login TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- AI Models table
CREATE TABLE models (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    provider VARCHAR(100) NOT NULL, -- openai, anthropic, etc.
    model_id VARCHAR(255) NOT NULL, -- gpt-4, claude-3, etc.
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- API Keys table
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    key_hash VARCHAR(255) NOT NULL UNIQUE, -- hashed version of the key
    key_prefix VARCHAR(10) NOT NULL, -- first few chars for display (sk-...)
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id VARCHAR(255),
    type VARCHAR(50) NOT NULL, -- production, development, testing
    max_tokens INTEGER DEFAULT 10000,
    permissions TEXT[] DEFAULT ARRAY[]::TEXT[], -- read, write, admin, analytics
    is_active BOOLEAN DEFAULT true,
    last_used TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Model Organization Access (many-to-many)
CREATE TABLE model_organization_access (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(model_id, organization_id)
);

-- Usage tracking for quota management
CREATE TABLE usage_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    tokens_used INTEGER NOT NULL,
    request_timestamp TIMESTAMP DEFAULT NOW(),
    response_status INTEGER,
    duration_ms INTEGER
);

-- Organization quotas
CREATE TABLE organization_quotas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    total_quota INTEGER NOT NULL DEFAULT 100000,
    used_tokens INTEGER DEFAULT 0,
    reset_date TIMESTAMP DEFAULT NOW() + INTERVAL '1 month',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(organization_id)
);

-- Indexes for performance
CREATE INDEX idx_api_keys_organization_id ON api_keys(organization_id);
CREATE INDEX idx_api_keys_active ON api_keys(is_active);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);

CREATE INDEX idx_models_active ON models(is_active);
CREATE INDEX idx_models_provider ON models(provider);

CREATE INDEX idx_model_org_access_model_id ON model_organization_access(model_id);
CREATE INDEX idx_model_org_access_org_id ON model_organization_access(organization_id);

CREATE INDEX idx_usage_logs_api_key_id ON usage_logs(api_key_id);
CREATE INDEX idx_usage_logs_org_id ON usage_logs(organization_id);
CREATE INDEX idx_usage_logs_timestamp ON usage_logs(request_timestamp);

CREATE INDEX idx_organizations_active ON organizations(is_active);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_active ON users(is_active);

-- Sample data for testing
INSERT INTO organizations (id, name, description) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Acme Corp', 'Main enterprise client'),
    ('22222222-2222-2222-2222-222222222222', 'TechStart', 'Startup company'),
    ('33333333-3333-3333-3333-333333333333', 'Global Inc', 'Large corporation');

-- Sample admin user (password: admin)
INSERT INTO users (username, email, password_hash) VALUES
    ('admin', 'admin@relai.dev', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi');

-- Sample models
INSERT INTO models (id, name, description, provider, model_id) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'GPT-4', 'OpenAI GPT-4 model', 'openai', 'gpt-4'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Claude 3', 'Anthropic Claude 3 model', 'anthropic', 'claude-3-sonnet'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', 'GPT-3.5 Turbo', 'OpenAI GPT-3.5 model', 'openai', 'gpt-3.5-turbo');

-- Sample model access (give all organizations access to all models for testing)
INSERT INTO model_organization_access (model_id, organization_id) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '22222222-2222-2222-2222-222222222222'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '33333333-3333-3333-3333-333333333333'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '22222222-2222-2222-2222-222222222222'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '33333333-3333-3333-3333-333333333333');

-- Sample quotas
INSERT INTO organization_quotas (organization_id, total_quota, used_tokens) VALUES
    ('11111111-1111-1111-1111-111111111111', 100000, 25000),
    ('22222222-2222-2222-2222-222222222222', 50000, 12000),
    ('33333333-3333-3333-3333-333333333333', 200000, 75000);

-- Sample API keys
INSERT INTO api_keys (id, name, description, key_hash, key_prefix, organization_id, type, max_tokens, permissions) VALUES
    ('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Production API Key', 'Main production key for Acme Corp', 'hash1', 'sk-1234', '11111111-1111-1111-1111-111111111111', 'production', 50000, ARRAY['read', 'write']),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'Development Key', 'Development testing key', 'hash2', 'sk-5678', '22222222-2222-2222-2222-222222222222', 'development', 10000, ARRAY['read']),
    ('ffffffff-ffff-ffff-ffff-ffffffffffff', 'Analytics Key', 'Analytics and monitoring', 'hash3', 'sk-9999', '33333333-3333-3333-3333-333333333333', 'production', 25000, ARRAY['read', 'analytics']);