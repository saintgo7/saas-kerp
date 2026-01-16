-- Rollback Tax Invoice Tables

-- Drop triggers
DROP TRIGGER IF EXISTS popbill_configs_updated_at ON popbill_configs;
DROP TRIGGER IF EXISTS tax_invoice_items_updated_at ON tax_invoice_items;
DROP TRIGGER IF EXISTS tax_invoices_updated_at ON tax_invoices;

-- Drop policies
DROP POLICY IF EXISTS popbill_configs_tenant_policy ON popbill_configs;
DROP POLICY IF EXISTS hometax_sessions_tenant_policy ON hometax_sessions;
DROP POLICY IF EXISTS tax_invoice_history_tenant_policy ON tax_invoice_history;
DROP POLICY IF EXISTS tax_invoice_attachments_tenant_policy ON tax_invoice_attachments;
DROP POLICY IF EXISTS tax_invoice_items_tenant_policy ON tax_invoice_items;
DROP POLICY IF EXISTS tax_invoices_tenant_policy ON tax_invoices;

-- Drop tables (in dependency order)
DROP TABLE IF EXISTS popbill_configs;
DROP TABLE IF EXISTS hometax_sessions;
DROP TABLE IF EXISTS tax_invoice_history;
DROP TABLE IF EXISTS tax_invoice_attachments;
DROP TABLE IF EXISTS tax_invoice_items;
DROP TABLE IF EXISTS tax_invoices;
