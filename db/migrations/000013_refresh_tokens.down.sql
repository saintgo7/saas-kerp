-- Roll back only what 000013 created: the indexes.
--
-- The refresh_tokens TABLE belongs to 000003_core_tables and must NOT be
-- dropped here. Dropping it used to destroy the table together with the RLS
-- ENABLE flag and the policies attached by 000010, so a single `migrate down`
-- silently removed tenant isolation from refresh tokens permanently.
DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP INDEX IF EXISTS idx_refresh_tokens_token;
