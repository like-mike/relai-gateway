-- RelAI Gateway Database Schema

-- Users table for Azure AD user persistence
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    azure_oid VARCHAR(255) UNIQUE NOT NULL, -- Azure Object ID
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Roles table
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    is_system_role BOOLEAN DEFAULT FALSE, -- True for System Admin
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Organizations table
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    ad_admin_group_id VARCHAR(255), -- AD group for org admins
    ad_admin_group_name VARCHAR(255),
    ad_member_group_id VARCHAR(255), -- AD group for org members
    ad_member_group_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User-Organization relationships (Org Admins and Members)
CREATE TABLE IF NOT EXISTS user_organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role_name VARCHAR(50) NOT NULL, -- Direct role name: 'admin' or 'member'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    UNIQUE(user_id, organization_id)
);

-- System-level user roles (System Admins)
CREATE TABLE IF NOT EXISTS user_system_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    UNIQUE(user_id, role_id)
);

-- Organization AD Group mappings
CREATE TABLE IF NOT EXISTS organization_ad_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    ad_group_id VARCHAR(255) NOT NULL,
    ad_group_name VARCHAR(255),
    role_type VARCHAR(50) NOT NULL, -- 'admin' or 'member'
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, ad_group_id, role_type)
);

-- System-level AD Group mappings (for System Admins)
CREATE TABLE IF NOT EXISTS system_ad_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ad_group_id VARCHAR(255) UNIQUE NOT NULL,
    ad_group_name VARCHAR(255),
    role_id UUID NOT NULL REFERENCES roles(id),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API Keys table (using raw API keys)
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    is_active BOOLEAN DEFAULT true,
    last_used TIMESTAMP WITH TIME ZONE,
    created_by_user_id UUID REFERENCES users(id), -- Link API keys to users
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Models table with all fields and constraints defined upfront
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

-- Indexes for performance
-- RBAC indexes
CREATE INDEX IF NOT EXISTS idx_users_azure_oid ON users(azure_oid);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_user_organizations_user_id ON user_organizations(user_id);
CREATE INDEX IF NOT EXISTS idx_user_organizations_org_id ON user_organizations(organization_id);
CREATE INDEX IF NOT EXISTS idx_user_system_roles_user_id ON user_system_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_organization_ad_groups_org_id ON organization_ad_groups(organization_id);
CREATE INDEX IF NOT EXISTS idx_organization_ad_groups_ad_group_id ON organization_ad_groups(ad_group_id);
CREATE INDEX IF NOT EXISTS idx_system_ad_groups_ad_group_id ON system_ad_groups(ad_group_id);
CREATE INDEX IF NOT EXISTS idx_organizations_ad_admin_group ON organizations(ad_admin_group_id);
CREATE INDEX IF NOT EXISTS idx_organizations_ad_member_group ON organizations(ad_member_group_id);

-- API and models indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_api_key ON api_keys(api_key);
CREATE INDEX IF NOT EXISTS idx_api_keys_organization_id ON api_keys(organization_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys(is_active);
CREATE INDEX IF NOT EXISTS idx_api_keys_created_by_user_id ON api_keys(created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_models_model_id ON models(model_id);
CREATE INDEX IF NOT EXISTS idx_models_is_active ON models(is_active);
CREATE INDEX IF NOT EXISTS idx_model_org_access_model_id ON model_organization_access(model_id);
CREATE INDEX IF NOT EXISTS idx_model_org_access_org_id ON model_organization_access(organization_id);

-- Usage tracking indexes
CREATE INDEX IF NOT EXISTS idx_usage_logs_organization_id ON usage_logs(organization_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_api_key_id ON usage_logs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_model_id ON usage_logs(model_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_logs_endpoint ON usage_logs(endpoint);
CREATE INDEX IF NOT EXISTS idx_organization_quotas_org_id ON organization_quotas(organization_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at_org_id ON usage_logs(created_at, organization_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_model_id_created_at ON usage_logs(model_id, created_at);
CREATE INDEX IF NOT EXISTS idx_usage_logs_api_key_created_at ON usage_logs(api_key_id, created_at);

-- Email system indexes
CREATE INDEX IF NOT EXISTS idx_email_templates_type ON email_templates(type);
CREATE INDEX IF NOT EXISTS idx_email_schedules_org_id ON email_schedules(organization_id);
CREATE INDEX IF NOT EXISTS idx_email_schedules_type ON email_schedules(schedule_type);
CREATE INDEX IF NOT EXISTS idx_email_logs_recipient ON email_logs(recipient_email);
CREATE INDEX IF NOT EXISTS idx_email_logs_status ON email_logs(status);
CREATE INDEX IF NOT EXISTS idx_email_logs_sent_at ON email_logs(sent_at);

-- Insert default roles
INSERT INTO roles (id, name, description, is_system_role) VALUES
('00000000-0000-0000-0000-000000000001', 'System Admin', 'Global administrator with access to all organizations', true),
('00000000-0000-0000-0000-000000000002', 'Org Admin', 'Organization administrator with full access within their organization', false),
('00000000-0000-0000-0000-000000000003', 'Org Member', 'Organization member with limited access to own resources', false)
ON CONFLICT (id) DO NOTHING;

-- Insert default organization
INSERT INTO organizations (id, name, description)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Organization', 'Default organization for initial setup')
ON CONFLICT (id) DO NOTHING;

-- Insert OpenAI models only (with proper costs and constraints)
INSERT INTO models (id, name, model_id, provider, api_endpoint, api_token, description, input_cost_per_1m, output_cost_per_1m, max_retries, timeout_seconds, retry_delay_ms, backoff_multiplier) VALUES
('00000000-0000-0000-0000-000000000001', 'GPT-3.5 Turbo', 'gpt-3.5-turbo', 'openai', 'https://api.openai.com/v1', 'your-openai-api-key', 'OpenAI GPT-3.5 Turbo model', 1.5, 2.0, 2, 30, 1000, 2.0),
('00000000-0000-0000-0000-000000000002', 'GPT-4', 'gpt-4', 'openai', 'https://api.openai.com/v1', 'your-openai-api-key', 'OpenAI GPT-4 model', 30.0, 60.0, 2, 60, 1000, 2.0),
('00000000-0000-0000-0000-000000000003', 'GPT-4 Turbo', 'gpt-4-turbo-preview', 'openai', 'https://api.openai.com/v1', 'your-openai-api-key', 'OpenAI GPT-4 Turbo model', 10.0, 30.0, 2, 45, 1000, 2.0),
('00000000-0000-0000-0000-000000000004', 'Text Embedding Ada 002', 'text-embedding-ada-002', 'openai', 'https://api.openai.com/v1', 'your-openai-api-key', 'OpenAI text embedding model', 0.1, 0.1, 2, 20, 1000, 2.0)
-- Non-OpenAI models commented out for cleaner default setup
-- ('00000000-0000-0000-0000-000000000005', 'Claude 3 Haiku', 'claude-3-haiku-20240307', 'anthropic', 'https://api.anthropic.com', 'your-anthropic-api-key', 'Anthropic Claude 3 Haiku model', 0.25, 1.25, 2, 30, 1000, 2.0),
-- ('00000000-0000-0000-0000-000000000006', 'Claude 3 Sonnet', 'claude-3-sonnet-20240229', 'anthropic', 'https://api.anthropic.com', 'your-anthropic-api-key', 'Anthropic Claude 3 Sonnet model', 3.0, 15.0, 2, 30, 1000, 2.0),
-- ('00000000-0000-0000-0000-000000000007', 'Claude 3 Opus', 'claude-3-opus-20240229', 'anthropic', 'https://api.anthropic.com', 'your-anthropic-api-key', 'Anthropic Claude 3 Opus model', 15.0, 75.0, 2, 30, 1000, 2.0)
ON CONFLICT (id) DO NOTHING;

-- Grant default organization access to OpenAI models only
INSERT INTO model_organization_access (model_id, organization_id) 
SELECT m.id, '00000000-0000-0000-0000-000000000001'
FROM models m 
WHERE m.id IN (
    '00000000-0000-0000-0000-000000000001', -- GPT-3.5 Turbo
    '00000000-0000-0000-0000-000000000002', -- GPT-4
    '00000000-0000-0000-0000-000000000003', -- GPT-4 Turbo
    '00000000-0000-0000-0000-000000000004'  -- Text Embedding Ada 002
)
ON CONFLICT (model_id, organization_id) DO NOTHING;

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