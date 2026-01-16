-- K-ERP v0.2 Migration: Additional Performance Indexes
-- Composite and specialized indexes for common query patterns

-- ============================================
-- COMPOSITE INDEXES: Vouchers
-- ============================================

-- Voucher list query: company + date range + status
CREATE INDEX idx_vouchers_company_date_status
    ON vouchers(company_id, voucher_date DESC, status)
    WHERE status != 'cancelled';

-- Voucher approval queue
CREATE INDEX idx_vouchers_approval_queue
    ON vouchers(company_id, created_at DESC)
    WHERE status = 'pending';

-- Posted vouchers by period (for financial reports)
CREATE INDEX idx_vouchers_posted_period
    ON vouchers(company_id, voucher_date)
    WHERE status = 'posted';

-- ============================================
-- COMPOSITE INDEXES: Invoices
-- ============================================

-- Invoice list: company + type + date
CREATE INDEX idx_invoices_company_type_date
    ON invoices(company_id, invoice_type, issue_date DESC);

-- Sales invoices for VAT reporting
CREATE INDEX idx_invoices_sales_vat
    ON invoices(company_id, issue_date)
    WHERE invoice_type = 'sales' AND status IN ('issued', 'sent');

-- Purchase invoices for VAT reporting
CREATE INDEX idx_invoices_purchase_vat
    ON invoices(company_id, issue_date)
    WHERE invoice_type = 'purchase' AND status IN ('received');

-- Pending invoices to submit
CREATE INDEX idx_invoices_pending_submit
    ON invoices(company_id, created_at)
    WHERE status = 'draft' AND nts_confirm_num IS NULL;

-- ============================================
-- COMPOSITE INDEXES: Payroll
-- ============================================

-- Payroll by period (descending for recent first)
CREATE INDEX idx_payrolls_company_period_desc
    ON payrolls(company_id, pay_year DESC, pay_month DESC);

-- Active employee payrolls
CREATE INDEX idx_payrolls_employee_period
    ON payrolls(employee_id, pay_year DESC, pay_month DESC);

-- Unpaid payrolls
CREATE INDEX idx_payrolls_unpaid
    ON payrolls(company_id, pay_year, pay_month)
    WHERE status = 'approved' AND paid_at IS NULL;

-- ============================================
-- COMPOSITE INDEXES: Employees
-- ============================================

-- Active employees by department (for org charts, lists)
CREATE INDEX idx_employees_dept_active
    ON employees(company_id, department_id, name)
    WHERE status = 'active' AND deleted_at IS NULL;

-- Employee search by name
CREATE INDEX idx_employees_name_search
    ON employees(company_id, name varchar_pattern_ops)
    WHERE deleted_at IS NULL;

-- Employees by hire date (for anniversary, tenure reports)
CREATE INDEX idx_employees_hire_date
    ON employees(company_id, hire_date)
    WHERE status = 'active' AND deleted_at IS NULL;

-- ============================================
-- COMPOSITE INDEXES: Ledger
-- ============================================

-- Ledger balances for trial balance
CREATE INDEX idx_ledger_trial_balance
    ON ledger_balances(company_id, fiscal_year, fiscal_month, account_id);

-- Account balances over time
CREATE INDEX idx_ledger_account_history
    ON ledger_balances(company_id, account_id, fiscal_year DESC, fiscal_month DESC);

-- ============================================
-- COMPOSITE INDEXES: Insurance
-- ============================================

-- Insurance reports by agency and period
CREATE INDEX idx_insurance_reports_agency_period
    ON insurance_reports(company_id, agency_type, report_year DESC, report_month DESC);

-- Pending insurance reports
CREATE INDEX idx_insurance_reports_pending
    ON insurance_reports(company_id, agency_type, created_at)
    WHERE status IN ('draft', 'pending');

-- Monthly contributions summary
CREATE INDEX idx_insurance_contributions_summary
    ON insurance_monthly_contributions(company_id, contribution_year, contribution_month);

-- ============================================
-- GIN INDEXES: JSONB Fields
-- ============================================

-- Payroll allowances search
CREATE INDEX idx_payrolls_allowances_gin
    ON payrolls USING GIN (allowances jsonb_path_ops);

-- Payroll other deductions
CREATE INDEX idx_payrolls_other_deductions_gin
    ON payrolls USING GIN (other_deductions jsonb_path_ops);

-- Insurance report data
CREATE INDEX idx_insurance_reports_data_gin
    ON insurance_reports USING GIN (report_data jsonb_path_ops);

-- Invoice provider response
CREATE INDEX idx_invoices_provider_response_gin
    ON invoices USING GIN (provider_response jsonb_path_ops)
    WHERE provider_response IS NOT NULL;

-- Voucher entry tags
CREATE INDEX idx_voucher_entries_tags_gin
    ON voucher_entries USING GIN (tags jsonb_path_ops)
    WHERE tags != '[]'::JSONB;

-- ============================================
-- PARTIAL INDEXES: Soft Deletes
-- ============================================

-- Partners not deleted
CREATE INDEX idx_partners_not_deleted
    ON partners(company_id, partner_name)
    WHERE deleted_at IS NULL;

-- Projects not deleted
CREATE INDEX idx_projects_not_deleted
    ON projects(company_id, code)
    WHERE deleted_at IS NULL;

-- ============================================
-- BRIN INDEXES: Time-Series Data
-- ============================================

-- Audit logs (time-series, high volume)
CREATE INDEX idx_audit_logs_brin
    ON audit_logs USING BRIN (created_at)
    WITH (pages_per_range = 128);

-- Scrape job logs (time-series)
CREATE INDEX idx_scrape_job_logs_brin
    ON scrape_job_logs USING BRIN (created_at)
    WITH (pages_per_range = 128);

-- ============================================
-- TEXT SEARCH INDEXES (Korean support requires pg_bigm or similar)
-- ============================================

-- Note: For Korean full-text search, consider:
-- 1. pg_bigm extension for bi-gram indexing
-- 2. Meilisearch for external search
-- These are placeholder indexes using trigram for basic search

-- CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Partner name search (if pg_trgm available)
-- CREATE INDEX idx_partners_name_trgm
--     ON partners USING GIN (partner_name gin_trgm_ops);

-- Employee name search (if pg_trgm available)
-- CREATE INDEX idx_employees_name_trgm
--     ON employees USING GIN (name gin_trgm_ops);

-- ============================================
-- STATISTICS: Extended Statistics
-- ============================================

-- Multi-column statistics for voucher queries
CREATE STATISTICS stat_vouchers_company_date_status
    ON company_id, voucher_date, status
    FROM vouchers;

-- Multi-column statistics for invoice queries
CREATE STATISTICS stat_invoices_company_type_date
    ON company_id, invoice_type, issue_date
    FROM invoices;

-- Multi-column statistics for employee queries
CREATE STATISTICS stat_employees_company_dept_status
    ON company_id, department_id, status
    FROM employees;

-- ============================================
-- ANALYZE: Update Statistics
-- ============================================

ANALYZE companies;
ANALYZE users;
ANALYZE vouchers;
ANALYZE voucher_entries;
ANALYZE invoices;
ANALYZE employees;
ANALYZE payrolls;
ANALYZE insurance_reports;
