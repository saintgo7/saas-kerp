-- K-ERP v0.2 Migration: Extensions (Rollback)
-- Note: Dropping extensions may fail if objects depend on them

DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "ltree";
DROP EXTENSION IF EXISTS "uuid-ossp";
