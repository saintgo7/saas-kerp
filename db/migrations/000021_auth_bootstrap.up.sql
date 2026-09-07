-- ============================================================================
-- AUTHENTICATION BOOTSTRAP UNDER RLS
--
-- THE PROBLEM
-- /auth/login, /auth/refresh and /auth/forgot-password must read users and
-- refresh_tokens BEFORE the tenant is known - the whole point of those lookups
-- is to discover which company_id the caller belongs to. At that moment
-- app.current_tenant cannot be set, so once the application role loses
-- BYPASSRLS (000016) every authentication query returns zero rows and login
-- breaks. Today the symptom is hidden only because the DSN is still a
-- superuser.
--
-- THE CHOICE
-- Three options were on the table:
--   (a) exempt users + refresh_tokens from RLS entirely
--   (b) a separate auth login role holding BYPASSRLS
--   (c) let the policy pass when app.current_tenant is unset
--
-- (a) is rejected: it reopens every tenant's user list - e-mail addresses,
--     names, bcrypt hashes - to any authenticated tenant. That is a bigger
--     hole than the one being closed, and it is exactly the table the audit
--     already flagged for a global `WHERE email = ?` lookup.
-- (c) is rejected outright: "allow when the variable is not set" is the same
--     shape as the app.is_admin switch removed in 000017. It converts a
--     fail-closed design into a fail-open one, so any code path that merely
--     FORGETS to set the tenant silently gets unrestricted global reads.
-- (b) is the right direction but does not need a second login role, a second
--     credential or a second connection pool.
--
-- THE IMPLEMENTATION - a narrowed form of (b)
-- A NOLOGIN role (kerp_auth) holds BYPASSRLS and owns two SECURITY DEFINER
-- functions. Because it cannot log in, its privileges are reachable ONLY by
-- executing those two functions, whose result shape is fixed here. The tables
-- keep RLS + FORCE; no table-level exemption exists; the application keeps one
-- pool and one credential.
--
-- Privileged surface, in full: "look up a user by e-mail" and "look up a
-- refresh token by its value". Both are inherent to authenticating someone
-- whose tenant is not yet known.
--
-- CALLING CONTRACT (application side)
--   1. call auth_find_users_by_email() / auth_find_refresh_token()
--   2. verify the credential
--   3. set app.current_tenant to the company_id that came back
--   4. do everything else - refresh-token insert, last-login update - through
--      the ordinary RLS-enforced path
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 0. Make is_admin_context() inlinable
-- ----------------------------------------------------------------------------
-- The 000017 version is PL/pgSQL with an EXCEPTION block. An exception block
-- opens a subtransaction on every call, and this function is evaluated inside
-- the qualifier of all 40+ policies. Rewritten as a plain SQL function it can
-- be inlined by the planner and cannot raise: the scalar subquery yields NULL
-- when kerp_admin does not exist, and COALESCE turns that into false.
CREATE OR REPLACE FUNCTION is_admin_context()
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
    SELECT COALESCE(
        (SELECT pg_has_role(current_user, r.oid, 'MEMBER')
           FROM pg_roles r WHERE r.rolname = 'kerp_admin'),
        FALSE)
$$;

COMMENT ON FUNCTION is_admin_context() IS
    'True when the connected role is a member of kerp_admin. Deliberately NOT driven by a session variable.';

-- ----------------------------------------------------------------------------
-- 1. The authentication owner role: privileged but unable to log in
-- ----------------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kerp_auth') THEN
        CREATE ROLE kerp_auth NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE BYPASSRLS;
    ELSE
        ALTER ROLE kerp_auth NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE BYPASSRLS;
    END IF;
END
$$;

COMMENT ON ROLE kerp_auth IS
    'Owns the SECURITY DEFINER authentication lookups. NOLOGIN on purpose: its BYPASSRLS is reachable only through those functions.';

-- The migration role must be able to hand ownership over. A superuser already
-- can; anyone else needs membership.
DO $$
BEGIN
    IF NOT (SELECT rolsuper FROM pg_roles WHERE rolname = current_user) THEN
        EXECUTE format('GRANT kerp_auth TO %I', current_user);
    END IF;
END
$$;

-- BYPASSRLS waives row-level security, not table privileges.
DO $$
DECLARE tgt TEXT := current_schema();
BEGIN
    EXECUTE format('GRANT USAGE ON SCHEMA %I TO kerp_auth', tgt);
    EXECUTE format('GRANT SELECT ON %I.users, %I.refresh_tokens TO kerp_auth', tgt, tgt);
END
$$;

-- ----------------------------------------------------------------------------
-- 2. The two bootstrap lookups
-- ----------------------------------------------------------------------------
-- search_path is pinned at definition time. A SECURITY DEFINER function that
-- inherits the caller's search_path can be hijacked by a same-named table in a
-- schema the caller controls.
DO $$
DECLARE tgt TEXT := current_schema();
BEGIN
EXECUTE format($fn$
CREATE OR REPLACE FUNCTION auth_find_users_by_email(p_email TEXT)
RETURNS TABLE (
    id            UUID,
    company_id    UUID,
    email         TEXT,
    password_hash TEXT,
    name          TEXT,
    role          TEXT,
    status        TEXT
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = %I, public
AS $body$
    SELECT u.id,
           u.company_id,
           u.email::TEXT,
           u.password_hash::TEXT,
           u.name::TEXT,
           u.role::TEXT,
           u.status::TEXT
    FROM users u
    WHERE u.email = p_email
      AND u.deleted_at IS NULL
    ORDER BY u.created_at
$body$;
$fn$, tgt);

EXECUTE format($fn$
CREATE OR REPLACE FUNCTION auth_find_refresh_token(p_token TEXT)
RETURNS TABLE (
    id         UUID,
    user_id    UUID,
    company_id UUID,
    expires_at TIMESTAMPTZ,
    revoked    BOOLEAN
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = %I, public
AS $body$
    SELECT rt.id,
           rt.user_id,
           u.company_id,
           rt.expires_at,
           rt.revoked
    FROM refresh_tokens rt
    JOIN users u ON u.id = rt.user_id
    WHERE rt.token = p_token
      AND u.deleted_at IS NULL
$body$;
$fn$, tgt);
END
$$;

ALTER FUNCTION auth_find_users_by_email(TEXT)  OWNER TO kerp_auth;
ALTER FUNCTION auth_find_refresh_token(TEXT)   OWNER TO kerp_auth;

-- Functions are EXECUTE-to-PUBLIC by default; that would hand the bypass to
-- every role in the cluster.
REVOKE ALL ON FUNCTION auth_find_users_by_email(TEXT) FROM PUBLIC;
REVOKE ALL ON FUNCTION auth_find_refresh_token(TEXT)  FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_find_users_by_email(TEXT) TO kerp_app, kerp_admin;
GRANT EXECUTE ON FUNCTION auth_find_refresh_token(TEXT)  TO kerp_app, kerp_admin;

COMMENT ON FUNCTION auth_find_users_by_email(TEXT) IS
    'Pre-tenant login lookup. Returns EVERY tenant match for the address - users are unique per (company_id, email), so the caller must handle more than one row rather than taking the first. Soft-deleted users are excluded.';
COMMENT ON FUNCTION auth_find_refresh_token(TEXT) IS
    'Pre-tenant refresh-token lookup; also returns the owning company_id so the caller can set app.current_tenant before doing anything else.';
