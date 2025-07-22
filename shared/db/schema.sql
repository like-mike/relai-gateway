-- RelAI Gateway Database Schema

-- Organizations table
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API Keys table (using raw API keys)
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    is_active BOOLEAN DEFAULT true,
    last_used TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Models table
CREATE TABLE IF NOT EXISTS models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    model_id VARCHAR(255) NOT NULL,
    provider VARCHAR(100) NOT NULL,
    api_endpoint VARCHAR(500),
    api_token VARCHAR(500),
    description TEXT,
    input_cost_per_1m DECIMAL(10,6) DEFAULT 0.0,
    output_cost_per_1m DECIMAL(10,6) DEFAULT 0.0,
    max_retries INTEGER DEFAULT 2 CHECK (max_retries >= 0 AND max_retries <= 3),
    timeout_seconds INTEGER DEFAULT 30 CHECK (timeout_seconds >= 5 AND timeout_seconds <= 300),
    retry_delay_ms INTEGER DEFAULT 1000 CHECK (retry_delay_ms >= 100 AND retry_delay_ms <= 10000),
    backoff_multiplier REAL DEFAULT 2.0 CHECK (backoff_multiplier >= 1.0 AND backoff_multiplier <= 5.0),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Model-Organization access table (many-to-many)
CREATE TABLE IF NOT EXISTS model_organization_access (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    granted_by UUID, -- Could reference a users table in the future
    UNIQUE(model_id, organization_id)
);

-- Custom endpoints table
CREATE TABLE IF NOT EXISTS endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    path_prefix VARCHAR(255) NOT NULL,
    primary_model_id UUID REFERENCES models(id),
    fallback_model_id UUID REFERENCES models(id),
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, path_prefix)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_api_keys_api_key ON api_keys(api_key);
CREATE INDEX IF NOT EXISTS idx_api_keys_organization_id ON api_keys(organization_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys(is_active);
CREATE INDEX IF NOT EXISTS idx_models_model_id ON models(model_id);
CREATE INDEX IF NOT EXISTS idx_models_is_active ON models(is_active);
CREATE INDEX IF NOT EXISTS idx_model_org_access_model_id ON model_organization_access(model_id);
CREATE INDEX IF NOT EXISTS idx_model_org_access_org_id ON model_organization_access(organization_id);
CREATE INDEX IF NOT EXISTS idx_endpoints_organization_id ON endpoints(organization_id);
CREATE INDEX IF NOT EXISTS idx_endpoints_path_prefix ON endpoints(path_prefix);
CREATE INDEX IF NOT EXISTS idx_endpoints_is_active ON endpoints(is_active);

-- Insert default organization
INSERT INTO organizations (id, name, description) 
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Organization', 'Default organization for initial setup')
ON CONFLICT (id) DO NOTHING;

-- Insert some default models (handle existing schemas gracefully)
INSERT INTO models (id, name, model_id, provider, api_endpoint, api_token, description, input_cost_per_1m, output_cost_per_1m) VALUES
('00000000-0000-0000-0000-000000000001', 'GPT-3.5 Turbo', 'gpt-3.5-turbo', 'openai', 'https://api.openai.com/v1', 'your-openai-api-key', 'OpenAI GPT-3.5 Turbo model', 1.5, 2.0),
('00000000-0000-0000-0000-000000000002', 'GPT-4', 'gpt-4', 'openai', 'https://api.openai.com/v1', 'your-openai-api-key', 'OpenAI GPT-4 model', 30.0, 60.0),
('00000000-0000-0000-0000-000000000003', 'GPT-4 Turbo', 'gpt-4-turbo-preview', 'openai', 'https://api.openai.com/v1', 'your-openai-api-key', 'OpenAI GPT-4 Turbo model', 10.0, 30.0),
('00000000-0000-0000-0000-000000000004', 'Text Embedding Ada 002', 'text-embedding-ada-002', 'openai', 'https://api.openai.com/v1', 'your-openai-api-key', 'OpenAI text embedding model', 0.1, 0.1),
('00000000-0000-0000-0000-000000000005', 'Claude 3 Haiku', 'claude-3-haiku-20240307', 'anthropic', 'https://api.anthropic.com', 'your-anthropic-api-key', 'Anthropic Claude 3 Haiku model', 0.25, 1.25),
('00000000-0000-0000-0000-000000000006', 'Claude 3 Sonnet', 'claude-3-sonnet-20240229', 'anthropic', 'https://api.anthropic.com', 'your-anthropic-api-key', 'Anthropic Claude 3 Sonnet model', 3.0, 15.0),
('00000000-0000-0000-0000-000000000007', 'Claude 3 Opus', 'claude-3-opus-20240229', 'anthropic', 'https://api.anthropic.com', 'your-anthropic-api-key', 'Anthropic Claude 3 Opus model', 15.0, 75.0)
ON CONFLICT (id) DO NOTHING;

-- Grant default organization access to all default models
INSERT INTO model_organization_access (model_id, organization_id) 
SELECT m.id, '00000000-0000-0000-0000-000000000001'
FROM models m 
WHERE m.id IN (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000005',
    '00000000-0000-0000-0000-000000000006',
    '00000000-0000-0000-0000-000000000007'
)
ON CONFLICT (model_id, organization_id) DO NOTHING;

-- Usage tracking table for token consumption analytics and billing
CREATE TABLE IF NOT EXISTS usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    endpoint VARCHAR(255) NOT NULL, -- e.g., "/v1/chat/completions"
    prompt_tokens INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    request_id VARCHAR(255), -- Provider's request ID if available
    response_status INTEGER NOT NULL, -- HTTP status code
    response_time_ms INTEGER, -- Response time in milliseconds
    cost_usd DECIMAL(10,6), -- Calculated cost in USD
    metadata JSONB, -- Additional provider-specific data
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Organization quota tracking table
CREATE TABLE IF NOT EXISTS organization_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE UNIQUE,
    total_quota BIGINT NOT NULL DEFAULT 1000000, -- Total tokens allowed
    used_tokens BIGINT NOT NULL DEFAULT 0, -- Tokens consumed
    reset_date TIMESTAMP WITH TIME ZONE DEFAULT (NOW() + INTERVAL '1 month'),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for usage tracking performance
CREATE INDEX IF NOT EXISTS idx_usage_logs_organization_id ON usage_logs(organization_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_api_key_id ON usage_logs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_model_id ON usage_logs(model_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_logs_endpoint ON usage_logs(endpoint);
CREATE INDEX IF NOT EXISTS idx_organization_quotas_org_id ON organization_quotas(organization_id);

-- Additional indexes for analytics dashboard performance
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at_org_id ON usage_logs(created_at, organization_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_model_id_created_at ON usage_logs(model_id, created_at);
CREATE INDEX IF NOT EXISTS idx_usage_logs_api_key_created_at ON usage_logs(api_key_id, created_at);

-- Insert default quota for default organization
INSERT INTO organization_quotas (organization_id, total_quota, used_tokens)
VALUES ('00000000-0000-0000-0000-000000000001', 1000000, 0)
ON CONFLICT (organization_id) DO NOTHING;

-- Create a default API key for testing (you should change this in production)
INSERT INTO api_keys (id, organization_id, name, api_key)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'Default API Key',
    'sk-c145fad9c61c729b46357481e31b2fa524ec52737f73bcfd86dd3fdb7309bf29'
)
ON CONFLICT (api_key) DO NOTHING;

-- Add cost fields to models table (Simple Model Costs Implementation)
ALTER TABLE models ADD COLUMN IF NOT EXISTS input_cost_per_1m DECIMAL(10,6) DEFAULT 0.0;
ALTER TABLE models ADD COLUMN IF NOT EXISTS output_cost_per_1m DECIMAL(10,6) DEFAULT 0.0;

-- Add retry and timeout configuration fields to models table
ALTER TABLE models ADD COLUMN IF NOT EXISTS max_retries INTEGER DEFAULT 2;
ALTER TABLE models ADD COLUMN IF NOT EXISTS timeout_seconds INTEGER DEFAULT 30;
ALTER TABLE models ADD COLUMN IF NOT EXISTS retry_delay_ms INTEGER DEFAULT 1000;
ALTER TABLE models ADD COLUMN IF NOT EXISTS backoff_multiplier REAL DEFAULT 2.0;

-- Add production safety constraints for retry/timeout fields
DO $$ BEGIN
    ALTER TABLE models ADD CONSTRAINT check_max_retries CHECK (max_retries >= 0 AND max_retries <= 3);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE models ADD CONSTRAINT check_timeout_seconds CHECK (timeout_seconds >= 5 AND timeout_seconds <= 180);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE models ADD CONSTRAINT check_retry_delay_ms CHECK (retry_delay_ms >= 100 AND retry_delay_ms <= 10000);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE models ADD CONSTRAINT check_backoff_multiplier CHECK (backoff_multiplier >= 1.0 AND backoff_multiplier <= 5.0);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Update existing models with default costs (per 1M tokens)
UPDATE models SET
    input_cost_per_1m = CASE
        WHEN model_id = 'gpt-3.5-turbo' THEN 1.5
        WHEN model_id = 'gpt-4' THEN 30.0
        WHEN model_id = 'gpt-4-turbo-preview' THEN 10.0
        WHEN model_id = 'text-embedding-ada-002' THEN 0.1
        WHEN model_id = 'claude-3-haiku-20240307' THEN 0.25
        WHEN model_id = 'claude-3-sonnet-20240229' THEN 3.0
        WHEN model_id = 'claude-3-opus-20240229' THEN 15.0
        ELSE 0.0
    END,
    output_cost_per_1m = CASE
        WHEN model_id = 'gpt-3.5-turbo' THEN 2.0
        WHEN model_id = 'gpt-4' THEN 60.0
        WHEN model_id = 'gpt-4-turbo-preview' THEN 30.0
        WHEN model_id = 'text-embedding-ada-002' THEN 0.1
        WHEN model_id = 'claude-3-haiku-20240307' THEN 1.25
        WHEN model_id = 'claude-3-sonnet-20240229' THEN 15.0
        WHEN model_id = 'claude-3-opus-20240229' THEN 75.0
        ELSE 0.0
    END
WHERE input_cost_per_1m = 0.0 OR input_cost_per_1m IS NULL;