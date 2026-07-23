-- Least-privilege application DB role.
--
-- The application previously connected as a SUPERUSER, which nullifies RLS and
-- means any SQL-injection or bug runs with full DBA power (read every table in
-- every database, COPY ... PROGRAM RCE, read server files, DROP, pg_authid,
-- ...). This creates a dedicated NOSUPERUSER / NOCREATEDB / NOCREATEROLE role
-- with only the table/sequence/function privileges the app needs.
--
-- NOTE ON RLS: kerp_app is created with BYPASSRLS so the app keeps working via
-- its existing application-level `company_id = ?` filtering (the verified
-- tenant boundary). Turning the RLS policies in 000010 into a hard DB-enforced
-- backstop additionally requires setting `app.current_tenant` per request on
-- the query's connection (a transaction-scoped change). Until then, dropping
-- SUPERUSER is the concrete, high-value hardening.
--
-- The role's password is intentionally NOT set here (no secrets in git); it is
-- assigned at deploy time (ALTER ROLE kerp_app PASSWORD '...').

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kerp_app') THEN
        CREATE ROLE kerp_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE BYPASSRLS;
    ELSE
        ALTER ROLE kerp_app NOSUPERUSER NOCREATEDB NOCREATEROLE BYPASSRLS;
    END IF;
END
$$;

ALTER ROLE kerp_app SET search_path TO kerp, public;

GRANT USAGE ON SCHEMA kerp, public TO kerp_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA kerp TO kerp_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA kerp TO kerp_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA kerp, public TO kerp_app;

-- Future objects created by the migration owner
ALTER DEFAULT PRIVILEGES IN SCHEMA kerp GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kerp_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA kerp GRANT USAGE, SELECT ON SEQUENCES TO kerp_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA kerp GRANT EXECUTE ON FUNCTIONS TO kerp_app;
