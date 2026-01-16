-- K-ERP v0.2 Migration: Additional Performance Indexes (Rollback)

-- Drop Statistics
DROP STATISTICS IF EXISTS stat_employees_company_dept_status;
DROP STATISTICS IF EXISTS stat_invoices_company_type_date;
DROP STATISTICS IF EXISTS stat_vouchers_company_date_status;

-- Drop BRIN Indexes
DROP INDEX IF EXISTS idx_scrape_job_logs_brin;
DROP INDEX IF EXISTS idx_audit_logs_brin;

-- Drop Partial Indexes
DROP INDEX IF EXISTS idx_projects_not_deleted;
DROP INDEX IF EXISTS idx_partners_not_deleted;

-- Drop GIN Indexes
DROP INDEX IF EXISTS idx_voucher_entries_tags_gin;
DROP INDEX IF EXISTS idx_invoices_provider_response_gin;
DROP INDEX IF EXISTS idx_insurance_reports_data_gin;
DROP INDEX IF EXISTS idx_payrolls_other_deductions_gin;
DROP INDEX IF EXISTS idx_payrolls_allowances_gin;

-- Drop Insurance Indexes
DROP INDEX IF EXISTS idx_insurance_contributions_summary;
DROP INDEX IF EXISTS idx_insurance_reports_pending;
DROP INDEX IF EXISTS idx_insurance_reports_agency_period;

-- Drop Ledger Indexes
DROP INDEX IF EXISTS idx_ledger_account_history;
DROP INDEX IF EXISTS idx_ledger_trial_balance;

-- Drop Employee Indexes
DROP INDEX IF EXISTS idx_employees_hire_date;
DROP INDEX IF EXISTS idx_employees_name_search;
DROP INDEX IF EXISTS idx_employees_dept_active;

-- Drop Payroll Indexes
DROP INDEX IF EXISTS idx_payrolls_unpaid;
DROP INDEX IF EXISTS idx_payrolls_employee_period;
DROP INDEX IF EXISTS idx_payrolls_company_period_desc;

-- Drop Invoice Indexes
DROP INDEX IF EXISTS idx_invoices_pending_submit;
DROP INDEX IF EXISTS idx_invoices_purchase_vat;
DROP INDEX IF EXISTS idx_invoices_sales_vat;
DROP INDEX IF EXISTS idx_invoices_company_type_date;

-- Drop Voucher Indexes
DROP INDEX IF EXISTS idx_vouchers_posted_period;
DROP INDEX IF EXISTS idx_vouchers_approval_queue;
DROP INDEX IF EXISTS idx_vouchers_company_date_status;
