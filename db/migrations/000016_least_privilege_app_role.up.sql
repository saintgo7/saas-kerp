-- Least-privilege application DB roles.
--
-- The application previously connected as a SUPERUSER, which nullifies RLS and
-- means any SQL-injection or bug runs with full DBA power (read every table in
-- every database, COPY ... PROGRAM RCE, read server files, DROP, pg_authid, ...).
--
-- Two roles are created:
--   kerp_app   - the request-path role. NOSUPERUSER and, crucially, NOBYPASSRLS
--                so the policies from 000010/000017 are actually enforced.
--   kerp_admin - a separate role for cross-tenant maintenance. RLS policies
--                grant it tenant-wide visibility via pg_has_role() (000017),
--                which replaces the old `app.is_admin` session-variable switch
--                that any client could flip with a plain SET.
--
-- Neither role gets a password here (no secrets in git); passwords are assigned
-- at deploy time (ALTER ROLE kerp_app PASSWORD '...').
--
-- ORDER OF ROLLOUT: kerp_app cannot read anything until the application sets
-- app.current_tenant on the connection (see 000017). Point the app DSN at
-- kerp_app only after that code ships, or every query returns zero rows.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kerp_app') THEN
        CREATE ROLE kerp_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
    ELSE
        ALTER ROLE kerp_app NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kerp_admin') THEN
        CREATE ROLE kerp_admin LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
    ELSE
        ALTER ROLE kerp_admin NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
    END IF;
END
$$;

-- Grants are applied to the schema the migrations actually ran in
-- (current_schema()), not a hardcoded "kerp". Migrations 000001-000012 create
-- their objects unqualified, so on a database without a kerp schema they live
-- in public and a "GRANT ... IN SCHEMA kerp" would silently grant nothing.
DO $$
DECLARE
    tgt TEXT := current_schema();
BEGIN
    EXECUTE format('ALTER ROLE kerp_app SET search_path TO %I, public', tgt);
    EXECUTE format('ALTER ROLE kerp_admin SET search_path TO %I, public', tgt);

    EXECUTE format('GRANT USAGE ON SCHEMA %I TO kerp_app, kerp_admin', tgt);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO kerp_app, kerp_admin', tgt);
    EXECUTE format('GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA %I TO kerp_app, kerp_admin', tgt);
    EXECUTE format('GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA %I TO kerp_app, kerp_admin', tgt);

    -- Future objects created by the migration owner
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kerp_app, kerp_admin', tgt);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I GRANT USAGE, SELECT ON SEQUENCES TO kerp_app, kerp_admin', tgt);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I GRANT EXECUTE ON FUNCTIONS TO kerp_app, kerp_admin', tgt);

    IF tgt <> 'public' THEN
        EXECUTE 'GRANT USAGE ON SCHEMA public TO kerp_app, kerp_admin';
        EXECUTE 'GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO kerp_app, kerp_admin';
    END IF;
END
$$;
