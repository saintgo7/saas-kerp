-- K-ERP v0.2 Migration: Row Level Security Policies
-- Multi-tenancy isolation using RLS

-- ============================================
-- HELPER FUNCTION: Get Current Tenant
-- ============================================
CREATE OR REPLACE FUNCTION current_tenant_id()
RETURNS UUID AS $$
BEGIN
    RETURN NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION current_tenant_id() IS 'Get current tenant UUID from session variable';

-- ============================================
-- HELPER FUNCTION: Is Admin/Service Account
-- ============================================
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

-- ============================================
-- ENABLE RLS ON ALL TENANT TABLES
-- ============================================

-- Core tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens ENABLE ROW LEVEL SECURITY;

-- Partner/Project tables
ALTER TABLE partners ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE cost_centers ENABLE ROW LEVEL SECURITY;

-- Accounting tables
ALTER TABLE departments ENABLE ROW LEVEL SECURITY;
ALTER TABLE accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE fiscal_periods ENABLE ROW LEVEL SECURITY;
ALTER TABLE vouchers ENABLE ROW LEVEL SECURITY;
ALTER TABLE voucher_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE ledger_balances ENABLE ROW LEVEL SECURITY;
ALTER TABLE voucher_attachments ENABLE ROW LEVEL SECURITY;
ALTER TABLE voucher_sequences ENABLE ROW LEVEL SECURITY;

-- HR tables
ALTER TABLE positions ENABLE ROW LEVEL SECURITY;
ALTER TABLE employees ENABLE ROW LEVEL SECURITY;
ALTER TABLE employee_salaries ENABLE ROW LEVEL SECURITY;
ALTER TABLE employee_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE leave_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE employee_leaves ENABLE ROW LEVEL SECURITY;
ALTER TABLE employee_leave_balances ENABLE ROW LEVEL SECURITY;

-- Payroll tables
ALTER TABLE payroll_periods ENABLE ROW LEVEL SECURITY;
ALTER TABLE payrolls ENABLE ROW LEVEL SECURITY;
ALTER TABLE payroll_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE payroll_bank_transfers ENABLE ROW LEVEL SECURITY;
ALTER TABLE payroll_transfer_details ENABLE ROW LEVEL SECURITY;

-- Invoice tables
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoice_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE scrape_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE scrape_job_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE hometax_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoice_sequences ENABLE ROW LEVEL SECURITY;

-- Insurance tables
ALTER TABLE insurance_workplaces ENABLE ROW LEVEL SECURITY;
ALTER TABLE employee_insurance ENABLE ROW LEVEL SECURITY;
ALTER TABLE insurance_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE insurance_report_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE insurance_monthly_contributions ENABLE ROW LEVEL SECURITY;
ALTER TABLE insurance_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE edi_jobs ENABLE ROW LEVEL SECURITY;

-- ============================================
-- RLS POLICIES: Core Tables
-- ============================================

-- Users
CREATE POLICY tenant_isolation_users ON users
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_users ON users
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Roles
CREATE POLICY tenant_isolation_roles ON roles
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_roles ON roles
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- User Roles (join through user)
CREATE POLICY tenant_isolation_user_roles ON user_roles
    USING (
        EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = user_roles.user_id
            AND (u.company_id = current_tenant_id() OR is_admin_context())
        )
    );

-- Audit Logs
CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_audit_logs ON audit_logs
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Refresh Tokens (join through user)
CREATE POLICY tenant_isolation_refresh_tokens ON refresh_tokens
    USING (
        EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = refresh_tokens.user_id
            AND (u.company_id = current_tenant_id() OR is_admin_context())
        )
    );

-- ============================================
-- RLS POLICIES: Partner/Project Tables
-- ============================================

-- Partners
CREATE POLICY tenant_isolation_partners ON partners
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_partners ON partners
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Projects
CREATE POLICY tenant_isolation_projects ON projects
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_projects ON projects
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Cost Centers
CREATE POLICY tenant_isolation_cost_centers ON cost_centers
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_cost_centers ON cost_centers
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- ============================================
-- RLS POLICIES: Accounting Tables
-- ============================================

-- Departments
CREATE POLICY tenant_isolation_departments ON departments
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_departments ON departments
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Accounts
CREATE POLICY tenant_isolation_accounts ON accounts
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_accounts ON accounts
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Fiscal Periods
CREATE POLICY tenant_isolation_fiscal_periods ON fiscal_periods
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_fiscal_periods ON fiscal_periods
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Vouchers
CREATE POLICY tenant_isolation_vouchers ON vouchers
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_vouchers ON vouchers
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Voucher Entries
CREATE POLICY tenant_isolation_voucher_entries ON voucher_entries
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_voucher_entries ON voucher_entries
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Ledger Balances
CREATE POLICY tenant_isolation_ledger_balances ON ledger_balances
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_ledger_balances ON ledger_balances
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Voucher Attachments
CREATE POLICY tenant_isolation_voucher_attachments ON voucher_attachments
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_voucher_attachments ON voucher_attachments
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Voucher Sequences
CREATE POLICY tenant_isolation_voucher_sequences ON voucher_sequences
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_voucher_sequences ON voucher_sequences
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- ============================================
-- RLS POLICIES: HR Tables
-- ============================================

-- Positions
CREATE POLICY tenant_isolation_positions ON positions
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_positions ON positions
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Employees
CREATE POLICY tenant_isolation_employees ON employees
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_employees ON employees
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Employee Salaries
CREATE POLICY tenant_isolation_employee_salaries ON employee_salaries
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_employee_salaries ON employee_salaries
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Employee Documents
CREATE POLICY tenant_isolation_employee_documents ON employee_documents
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_employee_documents ON employee_documents
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Leave Types
CREATE POLICY tenant_isolation_leave_types ON leave_types
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_leave_types ON leave_types
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Employee Leaves
CREATE POLICY tenant_isolation_employee_leaves ON employee_leaves
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_employee_leaves ON employee_leaves
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Employee Leave Balances
CREATE POLICY tenant_isolation_employee_leave_balances ON employee_leave_balances
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_employee_leave_balances ON employee_leave_balances
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- ============================================
-- RLS POLICIES: Payroll Tables
-- ============================================

-- Payroll Periods
CREATE POLICY tenant_isolation_payroll_periods ON payroll_periods
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_payroll_periods ON payroll_periods
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Payrolls
CREATE POLICY tenant_isolation_payrolls ON payrolls
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_payrolls ON payrolls
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Payroll Items
CREATE POLICY tenant_isolation_payroll_items ON payroll_items
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_payroll_items ON payroll_items
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Payroll Bank Transfers
CREATE POLICY tenant_isolation_payroll_bank_transfers ON payroll_bank_transfers
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_payroll_bank_transfers ON payroll_bank_transfers
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Payroll Transfer Details (join through transfer)
CREATE POLICY tenant_isolation_payroll_transfer_details ON payroll_transfer_details
    USING (
        EXISTS (
            SELECT 1 FROM payroll_bank_transfers t
            WHERE t.id = payroll_transfer_details.transfer_id
            AND (t.company_id = current_tenant_id() OR is_admin_context())
        )
    );

-- ============================================
-- RLS POLICIES: Invoice Tables
-- ============================================

-- Invoices
CREATE POLICY tenant_isolation_invoices ON invoices
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_invoices ON invoices
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Invoice Items
CREATE POLICY tenant_isolation_invoice_items ON invoice_items
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_invoice_items ON invoice_items
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Scrape Jobs
CREATE POLICY tenant_isolation_scrape_jobs ON scrape_jobs
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_scrape_jobs ON scrape_jobs
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Scrape Job Logs (join through job)
CREATE POLICY tenant_isolation_scrape_job_logs ON scrape_job_logs
    USING (
        EXISTS (
            SELECT 1 FROM scrape_jobs j
            WHERE j.id = scrape_job_logs.job_id
            AND (j.company_id = current_tenant_id() OR is_admin_context())
        )
    );

-- Hometax Credentials
CREATE POLICY tenant_isolation_hometax_credentials ON hometax_credentials
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_hometax_credentials ON hometax_credentials
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Invoice Sequences
CREATE POLICY tenant_isolation_invoice_sequences ON invoice_sequences
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_invoice_sequences ON invoice_sequences
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- ============================================
-- RLS POLICIES: Insurance Tables
-- ============================================

-- Insurance Workplaces
CREATE POLICY tenant_isolation_insurance_workplaces ON insurance_workplaces
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_insurance_workplaces ON insurance_workplaces
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Employee Insurance
CREATE POLICY tenant_isolation_employee_insurance ON employee_insurance
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_employee_insurance ON employee_insurance
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Insurance Reports
CREATE POLICY tenant_isolation_insurance_reports ON insurance_reports
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_insurance_reports ON insurance_reports
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Insurance Report Items (join through report)
CREATE POLICY tenant_isolation_insurance_report_items ON insurance_report_items
    USING (
        EXISTS (
            SELECT 1 FROM insurance_reports r
            WHERE r.id = insurance_report_items.report_id
            AND (r.company_id = current_tenant_id() OR is_admin_context())
        )
    );

-- Insurance Monthly Contributions
CREATE POLICY tenant_isolation_insurance_contributions ON insurance_monthly_contributions
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_insurance_contributions ON insurance_monthly_contributions
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- Insurance Credentials
CREATE POLICY tenant_isolation_insurance_credentials ON insurance_credentials
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_insurance_credentials ON insurance_credentials
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- EDI Jobs
CREATE POLICY tenant_isolation_edi_jobs ON edi_jobs
    USING (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_insert_edi_jobs ON edi_jobs
    FOR INSERT WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- ============================================
-- GRANT PERMISSIONS TO APP ROLE
-- ============================================
-- Note: Create app role and grant permissions in deployment script
-- This is just a placeholder showing the pattern

-- Example:
-- CREATE ROLE kerp_app LOGIN PASSWORD 'secure_password';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO kerp_app;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kerp_app;
