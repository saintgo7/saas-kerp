-- K-ERP v0.2 Migration: Row Level Security Policies (Rollback)

-- ============================================
-- DROP ALL RLS POLICIES
-- ============================================

-- EDI Jobs
DROP POLICY IF EXISTS tenant_insert_edi_jobs ON edi_jobs;
DROP POLICY IF EXISTS tenant_isolation_edi_jobs ON edi_jobs;

-- Insurance Credentials
DROP POLICY IF EXISTS tenant_insert_insurance_credentials ON insurance_credentials;
DROP POLICY IF EXISTS tenant_isolation_insurance_credentials ON insurance_credentials;

-- Insurance Monthly Contributions
DROP POLICY IF EXISTS tenant_insert_insurance_contributions ON insurance_monthly_contributions;
DROP POLICY IF EXISTS tenant_isolation_insurance_contributions ON insurance_monthly_contributions;

-- Insurance Report Items
DROP POLICY IF EXISTS tenant_isolation_insurance_report_items ON insurance_report_items;

-- Insurance Reports
DROP POLICY IF EXISTS tenant_insert_insurance_reports ON insurance_reports;
DROP POLICY IF EXISTS tenant_isolation_insurance_reports ON insurance_reports;

-- Employee Insurance
DROP POLICY IF EXISTS tenant_insert_employee_insurance ON employee_insurance;
DROP POLICY IF EXISTS tenant_isolation_employee_insurance ON employee_insurance;

-- Insurance Workplaces
DROP POLICY IF EXISTS tenant_insert_insurance_workplaces ON insurance_workplaces;
DROP POLICY IF EXISTS tenant_isolation_insurance_workplaces ON insurance_workplaces;

-- Invoice Sequences
DROP POLICY IF EXISTS tenant_insert_invoice_sequences ON invoice_sequences;
DROP POLICY IF EXISTS tenant_isolation_invoice_sequences ON invoice_sequences;

-- Hometax Credentials
DROP POLICY IF EXISTS tenant_insert_hometax_credentials ON hometax_credentials;
DROP POLICY IF EXISTS tenant_isolation_hometax_credentials ON hometax_credentials;

-- Scrape Job Logs
DROP POLICY IF EXISTS tenant_isolation_scrape_job_logs ON scrape_job_logs;

-- Scrape Jobs
DROP POLICY IF EXISTS tenant_insert_scrape_jobs ON scrape_jobs;
DROP POLICY IF EXISTS tenant_isolation_scrape_jobs ON scrape_jobs;

-- Invoice Items
DROP POLICY IF EXISTS tenant_insert_invoice_items ON invoice_items;
DROP POLICY IF EXISTS tenant_isolation_invoice_items ON invoice_items;

-- Invoices
DROP POLICY IF EXISTS tenant_insert_invoices ON invoices;
DROP POLICY IF EXISTS tenant_isolation_invoices ON invoices;

-- Payroll Transfer Details
DROP POLICY IF EXISTS tenant_isolation_payroll_transfer_details ON payroll_transfer_details;

-- Payroll Bank Transfers
DROP POLICY IF EXISTS tenant_insert_payroll_bank_transfers ON payroll_bank_transfers;
DROP POLICY IF EXISTS tenant_isolation_payroll_bank_transfers ON payroll_bank_transfers;

-- Payroll Items
DROP POLICY IF EXISTS tenant_insert_payroll_items ON payroll_items;
DROP POLICY IF EXISTS tenant_isolation_payroll_items ON payroll_items;

-- Payrolls
DROP POLICY IF EXISTS tenant_insert_payrolls ON payrolls;
DROP POLICY IF EXISTS tenant_isolation_payrolls ON payrolls;

-- Payroll Periods
DROP POLICY IF EXISTS tenant_insert_payroll_periods ON payroll_periods;
DROP POLICY IF EXISTS tenant_isolation_payroll_periods ON payroll_periods;

-- Employee Leave Balances
DROP POLICY IF EXISTS tenant_insert_employee_leave_balances ON employee_leave_balances;
DROP POLICY IF EXISTS tenant_isolation_employee_leave_balances ON employee_leave_balances;

-- Employee Leaves
DROP POLICY IF EXISTS tenant_insert_employee_leaves ON employee_leaves;
DROP POLICY IF EXISTS tenant_isolation_employee_leaves ON employee_leaves;

-- Leave Types
DROP POLICY IF EXISTS tenant_insert_leave_types ON leave_types;
DROP POLICY IF EXISTS tenant_isolation_leave_types ON leave_types;

-- Employee Documents
DROP POLICY IF EXISTS tenant_insert_employee_documents ON employee_documents;
DROP POLICY IF EXISTS tenant_isolation_employee_documents ON employee_documents;

-- Employee Salaries
DROP POLICY IF EXISTS tenant_insert_employee_salaries ON employee_salaries;
DROP POLICY IF EXISTS tenant_isolation_employee_salaries ON employee_salaries;

-- Employees
DROP POLICY IF EXISTS tenant_insert_employees ON employees;
DROP POLICY IF EXISTS tenant_isolation_employees ON employees;

-- Positions
DROP POLICY IF EXISTS tenant_insert_positions ON positions;
DROP POLICY IF EXISTS tenant_isolation_positions ON positions;

-- Voucher Sequences
DROP POLICY IF EXISTS tenant_insert_voucher_sequences ON voucher_sequences;
DROP POLICY IF EXISTS tenant_isolation_voucher_sequences ON voucher_sequences;

-- Voucher Attachments
DROP POLICY IF EXISTS tenant_insert_voucher_attachments ON voucher_attachments;
DROP POLICY IF EXISTS tenant_isolation_voucher_attachments ON voucher_attachments;

-- Ledger Balances
DROP POLICY IF EXISTS tenant_insert_ledger_balances ON ledger_balances;
DROP POLICY IF EXISTS tenant_isolation_ledger_balances ON ledger_balances;

-- Voucher Entries
DROP POLICY IF EXISTS tenant_insert_voucher_entries ON voucher_entries;
DROP POLICY IF EXISTS tenant_isolation_voucher_entries ON voucher_entries;

-- Vouchers
DROP POLICY IF EXISTS tenant_insert_vouchers ON vouchers;
DROP POLICY IF EXISTS tenant_isolation_vouchers ON vouchers;

-- Fiscal Periods
DROP POLICY IF EXISTS tenant_insert_fiscal_periods ON fiscal_periods;
DROP POLICY IF EXISTS tenant_isolation_fiscal_periods ON fiscal_periods;

-- Accounts
DROP POLICY IF EXISTS tenant_insert_accounts ON accounts;
DROP POLICY IF EXISTS tenant_isolation_accounts ON accounts;

-- Departments
DROP POLICY IF EXISTS tenant_insert_departments ON departments;
DROP POLICY IF EXISTS tenant_isolation_departments ON departments;

-- Cost Centers
DROP POLICY IF EXISTS tenant_insert_cost_centers ON cost_centers;
DROP POLICY IF EXISTS tenant_isolation_cost_centers ON cost_centers;

-- Projects
DROP POLICY IF EXISTS tenant_insert_projects ON projects;
DROP POLICY IF EXISTS tenant_isolation_projects ON projects;

-- Partners
DROP POLICY IF EXISTS tenant_insert_partners ON partners;
DROP POLICY IF EXISTS tenant_isolation_partners ON partners;

-- Refresh Tokens
DROP POLICY IF EXISTS tenant_isolation_refresh_tokens ON refresh_tokens;

-- Audit Logs
DROP POLICY IF EXISTS tenant_insert_audit_logs ON audit_logs;
DROP POLICY IF EXISTS tenant_isolation_audit_logs ON audit_logs;

-- User Roles
DROP POLICY IF EXISTS tenant_isolation_user_roles ON user_roles;

-- Roles
DROP POLICY IF EXISTS tenant_insert_roles ON roles;
DROP POLICY IF EXISTS tenant_isolation_roles ON roles;

-- Users
DROP POLICY IF EXISTS tenant_insert_users ON users;
DROP POLICY IF EXISTS tenant_isolation_users ON users;

-- ============================================
-- DISABLE RLS ON ALL TABLES
-- ============================================

ALTER TABLE edi_jobs DISABLE ROW LEVEL SECURITY;
ALTER TABLE insurance_credentials DISABLE ROW LEVEL SECURITY;
ALTER TABLE insurance_monthly_contributions DISABLE ROW LEVEL SECURITY;
ALTER TABLE insurance_report_items DISABLE ROW LEVEL SECURITY;
ALTER TABLE insurance_reports DISABLE ROW LEVEL SECURITY;
ALTER TABLE employee_insurance DISABLE ROW LEVEL SECURITY;
ALTER TABLE insurance_workplaces DISABLE ROW LEVEL SECURITY;

ALTER TABLE invoice_sequences DISABLE ROW LEVEL SECURITY;
ALTER TABLE hometax_credentials DISABLE ROW LEVEL SECURITY;
ALTER TABLE scrape_job_logs DISABLE ROW LEVEL SECURITY;
ALTER TABLE scrape_jobs DISABLE ROW LEVEL SECURITY;
ALTER TABLE invoice_items DISABLE ROW LEVEL SECURITY;
ALTER TABLE invoices DISABLE ROW LEVEL SECURITY;

ALTER TABLE payroll_transfer_details DISABLE ROW LEVEL SECURITY;
ALTER TABLE payroll_bank_transfers DISABLE ROW LEVEL SECURITY;
ALTER TABLE payroll_items DISABLE ROW LEVEL SECURITY;
ALTER TABLE payrolls DISABLE ROW LEVEL SECURITY;
ALTER TABLE payroll_periods DISABLE ROW LEVEL SECURITY;

ALTER TABLE employee_leave_balances DISABLE ROW LEVEL SECURITY;
ALTER TABLE employee_leaves DISABLE ROW LEVEL SECURITY;
ALTER TABLE leave_types DISABLE ROW LEVEL SECURITY;
ALTER TABLE employee_documents DISABLE ROW LEVEL SECURITY;
ALTER TABLE employee_salaries DISABLE ROW LEVEL SECURITY;
ALTER TABLE employees DISABLE ROW LEVEL SECURITY;
ALTER TABLE positions DISABLE ROW LEVEL SECURITY;

ALTER TABLE voucher_sequences DISABLE ROW LEVEL SECURITY;
ALTER TABLE voucher_attachments DISABLE ROW LEVEL SECURITY;
ALTER TABLE ledger_balances DISABLE ROW LEVEL SECURITY;
ALTER TABLE voucher_entries DISABLE ROW LEVEL SECURITY;
ALTER TABLE vouchers DISABLE ROW LEVEL SECURITY;
ALTER TABLE fiscal_periods DISABLE ROW LEVEL SECURITY;
ALTER TABLE accounts DISABLE ROW LEVEL SECURITY;
ALTER TABLE departments DISABLE ROW LEVEL SECURITY;

ALTER TABLE cost_centers DISABLE ROW LEVEL SECURITY;
ALTER TABLE projects DISABLE ROW LEVEL SECURITY;
ALTER TABLE partners DISABLE ROW LEVEL SECURITY;

ALTER TABLE refresh_tokens DISABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
ALTER TABLE user_roles DISABLE ROW LEVEL SECURITY;
ALTER TABLE roles DISABLE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;

-- ============================================
-- DROP HELPER FUNCTIONS
-- ============================================

DROP FUNCTION IF EXISTS is_admin_context();
DROP FUNCTION IF EXISTS current_tenant_id();
