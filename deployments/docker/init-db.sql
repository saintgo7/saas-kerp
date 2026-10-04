-- K-ERP Database Initialization
--
-- Mounted at /docker-entrypoint-initdb.d/init.sql, so this runs exactly once:
-- when the postgres data volume is created empty. It must therefore do only
-- what has to exist *before* the migration tool can run, and nothing else.
--
-- SCOPE (deliberately narrow):
--   1. the kerp schema
--   2. the extensions that migration 000001 expects to already be there
--   3. a database-level search_path so every later session sees kerp first
--
-- EVERYTHING ELSE BELONGS TO db/migrations + db/seed. Run them with:
--     make migrate-up            # applies db/migrations/*.up.sql, recorded in
--                                # kerp.schema_migrations
--     (in the api image:  /app/migrate up)
-- Until that has run the stack cannot serve traffic: there are no tables.
--
-- WHY THIS FILE NO LONGER CREATES TABLES
-- It used to create kerp.companies and kerp.users (plus a demo tenant and a
-- demo admin) with a shape that predates the migrations. Because
-- db/migrations/000003_core_tables.up.sql says `CREATE TABLE companies`
-- without IF NOT EXISTS, that made every fresh volume unmigratable:
--     migration 000003_core_tables failed: relation "companies" already exists
-- The migrations own the schema; this file must not compete with them.
--
-- WHY THIS FILE DOES NOT CREATE THE kerp_app ROLE
-- kerp_app is created by db/migrations/000016_least_privilege_app_role.up.sql,
-- and its password is intentionally not in git. This container only receives
-- POSTGRES_DB / POSTGRES_USER / POSTGRES_PASSWORD (see docker-compose.yml), so
-- APP_DB_PASSWORD is not reachable from here and a role created here could not
-- be given a usable password anyway. Duplicating the CREATE ROLE would only
-- add a second definition to keep in sync. Giving it a password is a separate
-- deploy step (docs/RUNBOOK_least_privilege_db_role.md); until then the api
-- connects as the owner role.

-- Create the application schema first, so that what follows is deterministic.
CREATE SCHEMA IF NOT EXISTS kerp;

-- Extensions are pinned to `public` on purpose.
-- Without WITH SCHEMA they land in the first schema of the current search_path,
-- which differs depending on whether kerp already exists -- and the running
-- staging database has uuid-ossp/pgcrypto in public. Pinning keeps a rebuilt
-- volume comparable with it. (ltree is created by migration 000001 and lands in
-- kerp; that is existing behaviour and is left alone.)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION IF NOT EXISTS "pgcrypto" WITH SCHEMA public;

-- Grants: current_user is the POSTGRES_USER that owns this database, so it is
-- already the schema owner; the GRANT is kept as an explicit statement of
-- intent, and using current_user keeps it correct if POSTGRES_USER is renamed.
-- The previous "GRANT ALL ON ALL TABLES/SEQUENCES IN SCHEMA kerp" lines were
-- removed: during initdb the schema is empty, so they could never grant
-- anything. Privileges for the application role are handled by migration
-- 000016.
DO $$
BEGIN
    EXECUTE format('GRANT ALL ON SCHEMA kerp TO %I', current_user);
END
$$;

-- Set the default search path for every future session on this database.
-- Required because the migrations create unqualified objects (CREATE TABLE
-- companies) and the Go/Python services query unqualified names too.
DO $$
BEGIN
    EXECUTE format('ALTER DATABASE %I SET search_path TO kerp, public', current_database());
END
$$;

DO $$
BEGIN
    RAISE NOTICE 'K-ERP: schema "kerp" and extensions are ready.';
    RAISE NOTICE 'K-ERP: the database is EMPTY. Next steps, in order:';
    RAISE NOTICE 'K-ERP:   make migrate-up            (create the schema)';
    RAISE NOTICE 'K-ERP:   db/seed/00*.sql            (reference + demo data)';
    RAISE NOTICE 'K-ERP: api/worker cannot start before those have run.';
END $$;
