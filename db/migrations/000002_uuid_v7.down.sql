-- K-ERP v0.2 Migration: UUID v7 Function (Rollback)

DROP FUNCTION IF EXISTS uuid_v7_to_timestamp(UUID);
DROP FUNCTION IF EXISTS uuid_generate_v7();
