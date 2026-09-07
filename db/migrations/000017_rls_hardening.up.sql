-- ============================================================================
-- RLS HARDENING
--
-- Fixes four independent defects that together left row-level security with no
-- runtime effect at all:
--
--   1. GUC NAME SPLIT. 000010 reads app.current_tenant; 000012 read
--      app.current_company_id. Setting one left the other six tables broken.
--      >>> THE SINGLE CANONICAL GUC IS NOW  app.current_tenant  <<<
--      Every policy in the database now resolves the tenant through
--      current_tenant_id(), which reads app.current_tenant with missing_ok.
--   2. current_setting() WITHOUT missing_ok. 000012 called
--      current_setting('app.current_company_id') with no second argument, so an
--      unset variable raised "unrecognized configuration parameter" instead of
--      returning NULL - every tax-invoice query became a 500.
--   3. companies AND role_permissions HAD NO RLS AT ALL. The tenant root table
--      and the role->permission mapping were readable and writable across
--      tenants even with every other policy in place.
--   4. NO FORCE ROW LEVEL SECURITY ANYWHERE. Policies do not apply to a table's
--      owner unless FORCE is set, and the application connects as the owner.
--
-- Also replaces the app.is_admin escape hatch (see is_admin_context below).
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. Admin context: DB role membership instead of a client-settable variable
-- ----------------------------------------------------------------------------
-- Any client could run `SET app.is_admin = 'true'` - the GUC is in the
-- user-definable "app." namespace and needs no special privilege - and every
-- one of the 39 policies ended in `OR is_admin_context()`. One SQL-injection
-- point or one multi-statement path was enough to read every tenant's data.
--
-- The function keeps its name and signature so the existing policies stay
-- valid, but the answer now comes from PostgreSQL role membership, which a
-- session cannot grant itself.
CREATE OR REPLACE FUNCTION is_admin_context()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN pg_has_role(current_user, 'kerp_admin', 'MEMBER');
EXCEPTION
    WHEN undefined_object THEN
        -- kerp_admin does not exist (e.g. a bare test database): deny.
        RETURN FALSE;
    WHEN OTHERS THEN
        RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION is_admin_context() IS
    'True when the connected role is a member of kerp_admin. Deliberately NOT driven by a session variable.';

COMMENT ON FUNCTION current_tenant_id() IS
    'Current tenant UUID, read from the app.current_tenant GUC (the one canonical name). Returns NULL when unset so policies fail closed.';

-- ----------------------------------------------------------------------------
-- 2. Rebuild the 000012 tax-invoice policies onto the canonical GUC
-- ----------------------------------------------------------------------------
DROP POLICY IF EXISTS tax_invoices_tenant_policy ON tax_invoices;
DROP POLICY IF EXISTS tax_invoice_items_tenant_policy ON tax_invoice_items;
DROP POLICY IF EXISTS tax_invoice_attachments_tenant_policy ON tax_invoice_attachments;
DROP POLICY IF EXISTS tax_invoice_history_tenant_policy ON tax_invoice_history;
DROP POLICY IF EXISTS hometax_sessions_tenant_policy ON hometax_sessions;
DROP POLICY IF EXISTS popbill_configs_tenant_policy ON popbill_configs;

CREATE POLICY tenant_isolation_tax_invoices ON tax_invoices
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context());
CREATE POLICY tenant_isolation_tax_invoice_items ON tax_invoice_items
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context());
CREATE POLICY tenant_isolation_tax_invoice_attachments ON tax_invoice_attachments
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context());
CREATE POLICY tenant_isolation_tax_invoice_history ON tax_invoice_history
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context());
CREATE POLICY tenant_isolation_hometax_sessions ON hometax_sessions
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context());
CREATE POLICY tenant_isolation_popbill_configs ON popbill_configs
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context());

-- ----------------------------------------------------------------------------
-- 3. Cover the two tables that had no RLS at all
-- ----------------------------------------------------------------------------
-- companies: the tenant root. Without this any authenticated tenant could list
-- every customer's trade name, business registration number, CEO and plan.
ALTER TABLE companies ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_companies ON companies;
CREATE POLICY tenant_isolation_companies ON companies
    FOR ALL USING (id = current_tenant_id() OR is_admin_context());

-- role_permissions: has no company_id of its own, so it is scoped through the
-- owning role. Without this a tenant could grant arbitrary permissions to
-- another tenant's roles.
ALTER TABLE role_permissions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_role_permissions ON role_permissions;
CREATE POLICY tenant_isolation_role_permissions ON role_permissions
    FOR ALL USING (
        EXISTS (
            SELECT 1 FROM roles r
            WHERE r.id = role_permissions.role_id
              AND (r.company_id = current_tenant_id() OR is_admin_context())
        )
    );

-- ----------------------------------------------------------------------------
-- 4. FORCE row level security on every RLS-enabled table
-- ----------------------------------------------------------------------------
-- Without FORCE, the table owner - which is exactly who the application
-- connects as today - ignores every policy. Applied dynamically so the set
-- always matches the tables that actually have RLS enabled in this schema.
--
-- NOTE: superusers still bypass RLS entirely; that is a property of the role,
-- not of the table. Seed scripts therefore keep working when run as the
-- bootstrap superuser. A non-superuser owner running db/seed/* must either be
-- granted kerp_admin or set app.current_tenant first.
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT c.oid::regclass AS tbl
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relkind = 'r'
          AND c.relrowsecurity
          AND n.nspname = current_schema()
    LOOP
        EXECUTE format('ALTER TABLE %s FORCE ROW LEVEL SECURITY', r.tbl);
    END LOOP;
END
$$;
