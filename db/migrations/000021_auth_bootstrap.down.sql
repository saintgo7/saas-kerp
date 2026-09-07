-- Reverse 000021 only.

DROP FUNCTION IF EXISTS auth_find_refresh_token(TEXT);
DROP FUNCTION IF EXISTS auth_find_users_by_email(TEXT);

DO $$
DECLARE tgt TEXT := current_schema();
BEGIN
    EXECUTE format('REVOKE ALL ON %I.users, %I.refresh_tokens FROM kerp_auth', tgt, tgt);
    EXECUTE format('REVOKE USAGE ON SCHEMA %I FROM kerp_auth', tgt);
EXCEPTION
    WHEN undefined_object THEN NULL;
END
$$;

DROP ROLE IF EXISTS kerp_auth;

-- Restore the PL/pgSQL is_admin_context() introduced by 000017.
CREATE OR REPLACE FUNCTION is_admin_context()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN pg_has_role(current_user, 'kerp_admin', 'MEMBER');
EXCEPTION
    WHEN undefined_object THEN
        RETURN FALSE;
    WHEN OTHERS THEN
        RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION is_admin_context() IS
    'True when the connected role is a member of kerp_admin. Deliberately NOT driven by a session variable.';
