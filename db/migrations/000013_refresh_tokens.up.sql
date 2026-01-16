-- Refresh tokens table for JWT authentication
CREATE TABLE IF NOT EXISTS kerp.refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES kerp.users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for token lookup
CREATE INDEX idx_refresh_tokens_token ON kerp.refresh_tokens(token) WHERE revoked = FALSE;

-- Index for user cleanup
CREATE INDEX idx_refresh_tokens_user_id ON kerp.refresh_tokens(user_id);

-- Index for expired token cleanup
CREATE INDEX idx_refresh_tokens_expires_at ON kerp.refresh_tokens(expires_at) WHERE revoked = FALSE;
