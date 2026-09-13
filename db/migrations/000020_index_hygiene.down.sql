-- Reverse 000020 only, restoring the original index and constraint set.

-- 4. model-declared unique indexes
DROP INDEX IF EXISTS uq_roles_company_code;
DROP INDEX IF EXISTS uq_companies_code;

-- 3. soft-delete-aware uniqueness -> original full unique constraints
DROP INDEX IF EXISTS uq_partners_company_business_number;
ALTER TABLE partners ADD CONSTRAINT partners_company_id_business_number_key UNIQUE (company_id, business_number);

DROP INDEX IF EXISTS uq_employees_company_employee_no;
ALTER TABLE employees ADD CONSTRAINT employees_company_id_employee_no_key UNIQUE (company_id, employee_no);

DROP INDEX IF EXISTS uq_projects_company_code;
ALTER TABLE projects ADD CONSTRAINT projects_company_id_code_key UNIQUE (company_id, code);

DROP INDEX IF EXISTS uq_users_company_email;
ALTER TABLE users ADD CONSTRAINT users_company_id_email_key UNIQUE (company_id, email);

-- 2. RLS predicate indexes
DROP INDEX IF EXISTS idx_tax_invoice_attachments_company;
DROP INDEX IF EXISTS idx_employee_salaries_company;
DROP INDEX IF EXISTS idx_invoice_items_company;
DROP INDEX IF EXISTS idx_payroll_items_company;
DROP INDEX IF EXISTS idx_voucher_attachments_company;

-- 1. recreate the redundant indexes exactly as their original migrations had them
CREATE INDEX IF NOT EXISTS idx_tax_invoice_items_invoice ON tax_invoice_items(tax_invoice_id);
CREATE INDEX IF NOT EXISTS idx_insurance_report_items_report ON insurance_report_items(report_id);
CREATE INDEX IF NOT EXISTS idx_employee_insurance_employee ON employee_insurance(employee_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice ON invoice_items(invoice_id);
CREATE INDEX IF NOT EXISTS idx_payroll_periods_company ON payroll_periods(company_id);
CREATE INDEX IF NOT EXISTS idx_ledger_balances_account ON ledger_balances(company_id, account_id);
CREATE INDEX IF NOT EXISTS idx_ledger_balances_period ON ledger_balances(company_id, fiscal_year, fiscal_month);
CREATE INDEX IF NOT EXISTS idx_voucher_entries_voucher ON voucher_entries(voucher_id);
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_year ON fiscal_periods(company_id, fiscal_year);
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_company ON fiscal_periods(company_id);
CREATE INDEX IF NOT EXISTS idx_accounts_company ON accounts(company_id);
CREATE INDEX IF NOT EXISTS idx_projects_company ON projects(company_id);
CREATE INDEX IF NOT EXISTS idx_partners_business_number ON partners(company_id, business_number) WHERE business_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_partners_company ON partners(company_id);
CREATE INDEX IF NOT EXISTS idx_roles_company ON roles(company_id);
CREATE INDEX IF NOT EXISTS idx_users_company ON users(company_id);
CREATE INDEX IF NOT EXISTS idx_employee_salaries_effective ON employee_salaries(employee_id, effective_date);
CREATE INDEX IF NOT EXISTS idx_ledger_account_history ON ledger_balances(company_id, account_id, fiscal_year, fiscal_month);
CREATE INDEX IF NOT EXISTS idx_payrolls_company_period_desc ON payrolls(company_id, pay_year DESC, pay_month DESC);
CREATE INDEX IF NOT EXISTS idx_insurance_contributions_summary ON insurance_monthly_contributions(company_id, contribution_year, contribution_month);
