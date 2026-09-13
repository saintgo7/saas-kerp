-- Refresh token indexes for JWT authentication.
--
-- The refresh_tokens TABLE is owned by 000003_core_tables; this migration only
-- adds indexes on top of it. Schema qualifiers are deliberately omitted so the
-- objects land in whatever schema search_path selects, exactly like 000001-000012
-- (a hardcoded "kerp." here previously aborted initialisation on any database
-- that had no kerp schema, e.g. docker-compose.test.yml and CI).
--
-- CREATE TABLE IF NOT EXISTS is kept as a safety net for databases where 000003
-- somehow did not create the table; the column list matches 000003 exactly.
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for token lookup
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token) WHERE revoked = FALSE;

-- Index for user cleanup
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Index for expired token cleanup
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at) WHERE revoked = FALSE;
