-- K-ERP Database Initialization
-- Run on first startup

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create schema
CREATE SCHEMA IF NOT EXISTS kerp;

-- Grant permissions
GRANT ALL ON SCHEMA kerp TO kerp;
GRANT ALL ON ALL TABLES IN SCHEMA kerp TO kerp;
GRANT ALL ON ALL SEQUENCES IN SCHEMA kerp TO kerp;

-- Set default search path
ALTER DATABASE kerp SET search_path TO kerp, public;

-- Companies table (tenant)
CREATE TABLE IF NOT EXISTS kerp.companies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    business_number VARCHAR(20) UNIQUE,
    representative VARCHAR(100),
    address TEXT,
    phone VARCHAR(20),
    email VARCHAR(255),
    status VARCHAR(20) DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Users table
CREATE TABLE IF NOT EXISTS kerp.users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES kerp.companies(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    status VARCHAR(20) DEFAULT 'active',
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(company_id, email)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_users_company_id ON kerp.users(company_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON kerp.users(email);

-- Row Level Security
ALTER TABLE kerp.companies ENABLE ROW LEVEL SECURITY;
ALTER TABLE kerp.users ENABLE ROW LEVEL SECURITY;

-- RLS Policies (will be applied via application)
-- CREATE POLICY company_isolation ON kerp.users
--     USING (company_id = current_setting('app.current_tenant')::uuid);

-- NOTE: demo/seed data lives in init-db.dev.sql and is mounted ONLY by
-- docker-compose.dev.yml. Never seed known-credential accounts into an
-- environment reachable from the internet.

-- Log initialization
DO $$
BEGIN
    RAISE NOTICE 'K-ERP Database initialized successfully';
END $$;
