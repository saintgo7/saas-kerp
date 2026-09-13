-- Roll back exactly what 000017 changed, and nothing else.

-- 4. Drop FORCE again (leave ENABLE alone - that belongs to 000010/000012).
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT c.oid::regclass AS tbl
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relkind = 'r'
          AND c.relforcerowsecurity
          AND n.nspname = current_schema()
    LOOP
        EXECUTE format('ALTER TABLE %s NO FORCE ROW LEVEL SECURITY', r.tbl);
    END LOOP;
END
$$;

-- 3. Remove the two policies this migration introduced, and disable RLS on
--    those two tables again (000010 never enabled it for them).
DROP POLICY IF EXISTS tenant_isolation_role_permissions ON role_permissions;
ALTER TABLE role_permissions DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_companies ON companies;
ALTER TABLE companies DISABLE ROW LEVEL SECURITY;

-- 2. Restore the original 000012 policy names and predicates so that migrating
--    down to 12 and back up again is a no-op.
DROP POLICY IF EXISTS tenant_isolation_popbill_configs ON popbill_configs;
DROP POLICY IF EXISTS tenant_isolation_hometax_sessions ON hometax_sessions;
DROP POLICY IF EXISTS tenant_isolation_tax_invoice_history ON tax_invoice_history;
DROP POLICY IF EXISTS tenant_isolation_tax_invoice_attachments ON tax_invoice_attachments;
DROP POLICY IF EXISTS tenant_isolation_tax_invoice_items ON tax_invoice_items;
DROP POLICY IF EXISTS tenant_isolation_tax_invoices ON tax_invoices;

CREATE POLICY tax_invoices_tenant_policy ON tax_invoices
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);
CREATE POLICY tax_invoice_items_tenant_policy ON tax_invoice_items
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);
CREATE POLICY tax_invoice_attachments_tenant_policy ON tax_invoice_attachments
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);
CREATE POLICY tax_invoice_history_tenant_policy ON tax_invoice_history
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);
CREATE POLICY hometax_sessions_tenant_policy ON hometax_sessions
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);
CREATE POLICY popbill_configs_tenant_policy ON popbill_configs
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);

-- 1. Restore the original session-variable admin check from 000010.
CREATE OR REPLACE FUNCTION is_admin_context()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN COALESCE(current_setting('app.is_admin', TRUE), 'false')::BOOLEAN;
EXCEPTION
    WHEN OTHERS THEN
        RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION is_admin_context() IS 'Check if current context is admin/service account';
