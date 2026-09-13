-- Revoke and drop the least-privilege roles created by this migration.
DO $$
DECLARE
    tgt TEXT := current_schema();
BEGIN
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM kerp_app, kerp_admin', tgt);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I REVOKE USAGE, SELECT ON SEQUENCES FROM kerp_app, kerp_admin', tgt);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I REVOKE EXECUTE ON FUNCTIONS FROM kerp_app, kerp_admin', tgt);

    EXECUTE format('REVOKE ALL ON ALL TABLES IN SCHEMA %I FROM kerp_app, kerp_admin', tgt);
    EXECUTE format('REVOKE ALL ON ALL SEQUENCES IN SCHEMA %I FROM kerp_app, kerp_admin', tgt);
    EXECUTE format('REVOKE ALL ON ALL FUNCTIONS IN SCHEMA %I FROM kerp_app, kerp_admin', tgt);
    EXECUTE format('REVOKE USAGE ON SCHEMA %I FROM kerp_app, kerp_admin', tgt);

    IF tgt <> 'public' THEN
        EXECUTE 'REVOKE ALL ON ALL FUNCTIONS IN SCHEMA public FROM kerp_app, kerp_admin';
        EXECUTE 'REVOKE USAGE ON SCHEMA public FROM kerp_app, kerp_admin';
    END IF;
END
$$;

DROP ROLE IF EXISTS kerp_app;
DROP ROLE IF EXISTS kerp_admin;
