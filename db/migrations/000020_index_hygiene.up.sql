-- ============================================================================
-- INDEX HYGIENE
--   1. drop indexes fully covered by another index (write amplification only)
--   2. add indexes on the columns RLS now filters by on every single query
--   3. make uniqueness compatible with soft delete
--   4. add the unique indexes the GORM models declare
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. Redundant indexes
-- ----------------------------------------------------------------------------
-- Exact duplicates: identical column list AND identical predicate, so every
-- INSERT/UPDATE maintained the same B-tree twice.
DROP INDEX IF EXISTS idx_insurance_contributions_summary;  -- = idx_insurance_contributions_period
DROP INDEX IF EXISTS idx_payrolls_company_period_desc;     -- = idx_payrolls_company_period (B-trees scan both ways)
DROP INDEX IF EXISTS idx_ledger_account_history;           -- = ledger_balances unique constraint index
DROP INDEX IF EXISTS idx_employee_salaries_effective;      -- = employee_salaries unique constraint index

-- Leading-prefix duplicates: the listed columns are a prefix of another index,
-- which already serves equality lookups on them.
DROP INDEX IF EXISTS idx_users_company;                    -- prefix of users_company_id_email_key
DROP INDEX IF EXISTS idx_roles_company;                    -- prefix of roles_company_id_name_key
DROP INDEX IF EXISTS idx_partners_company;                 -- prefix of partners_company_id_business_number_key
DROP INDEX IF EXISTS idx_partners_business_number;         -- covered by the same unique constraint
DROP INDEX IF EXISTS idx_projects_company;                 -- prefix of projects_company_id_code_key
DROP INDEX IF EXISTS idx_accounts_company;                 -- prefix of accounts_company_id_code_key
DROP INDEX IF EXISTS idx_fiscal_periods_company;           -- prefix of the fiscal_periods unique constraint
DROP INDEX IF EXISTS idx_fiscal_periods_year;              -- prefix of the fiscal_periods unique constraint
DROP INDEX IF EXISTS idx_voucher_entries_voucher;          -- prefix of voucher_entries_voucher_id_line_no_key
DROP INDEX IF EXISTS idx_ledger_balances_period;           -- prefix of idx_ledger_trial_balance
DROP INDEX IF EXISTS idx_ledger_balances_account;          -- prefix of the ledger_balances unique constraint
DROP INDEX IF EXISTS idx_payroll_periods_company;          -- prefix of payroll_periods unique constraint
DROP INDEX IF EXISTS idx_invoice_items_invoice;            -- prefix of invoice_items_invoice_id_line_no_key
DROP INDEX IF EXISTS idx_employee_insurance_employee;      -- = employee_insurance_employee_id_key
DROP INDEX IF EXISTS idx_insurance_report_items_report;    -- prefix of insurance_report_items_report_id_line_no_key
DROP INDEX IF EXISTS idx_tax_invoice_items_invoice;        -- prefix of uq_tax_invoice_items_seq

-- NOTE: idx_companies_business_number is deliberately KEPT. 000018 replaced the
-- companies_business_number_key constraint with a partial unique index, so this
-- plain index is no longer redundant.

-- ----------------------------------------------------------------------------
-- 2. Indexes for the RLS predicate
-- ----------------------------------------------------------------------------
-- These five tables carry company_id and are filtered by it in an RLS policy,
-- but had no index with company_id in the leading position. Now that the
-- policies are actually enforced (000017), every read of them added a
-- company_id predicate that could only be answered by a sequential scan; the
-- same columns are also what a companies-row delete must check for CASCADE.
CREATE INDEX IF NOT EXISTS idx_voucher_attachments_company     ON voucher_attachments(company_id);
CREATE INDEX IF NOT EXISTS idx_payroll_items_company           ON payroll_items(company_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_company           ON invoice_items(company_id);
CREATE INDEX IF NOT EXISTS idx_employee_salaries_company       ON employee_salaries(company_id);
CREATE INDEX IF NOT EXISTS idx_tax_invoice_attachments_company ON tax_invoice_attachments(company_id);

-- ----------------------------------------------------------------------------
-- 3. Uniqueness that does not collide with soft delete
-- ----------------------------------------------------------------------------
-- These tables all carry deleted_at and partial indexes elsewhere in the schema
-- assume it is honoured, but the unique constraints covered deleted rows too.
-- Re-hiring an employee with their old employee_no, or reusing the code of a
-- deleted project, failed with duplicate key forever.
ALTER TABLE users     DROP CONSTRAINT IF EXISTS users_company_id_email_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_company_email
    ON users(company_id, email) WHERE deleted_at IS NULL;

ALTER TABLE projects  DROP CONSTRAINT IF EXISTS projects_company_id_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_projects_company_code
    ON projects(company_id, code) WHERE deleted_at IS NULL;

ALTER TABLE employees DROP CONSTRAINT IF EXISTS employees_company_id_employee_no_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_employees_company_employee_no
    ON employees(company_id, employee_no) WHERE deleted_at IS NULL;

-- partners additionally excludes the empty string: domain.Partner.BusinessNumber
-- is a plain string, so partners without a registration number all store ''.
ALTER TABLE partners  DROP CONSTRAINT IF EXISTS partners_company_id_business_number_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_partners_company_business_number
    ON partners(company_id, business_number)
    WHERE deleted_at IS NULL AND business_number IS NOT NULL AND business_number <> '';

-- ----------------------------------------------------------------------------
-- 4. Unique indexes declared by the GORM models
-- ----------------------------------------------------------------------------
-- domain.Company.Code carries `uniqueIndex`, domain.Role.Code carries
-- `uniqueIndex:idx_roles_company_code`. Both columns were added in 000014
-- without their uniqueness. Partial, because existing rows have NULL/''.
CREATE UNIQUE INDEX IF NOT EXISTS uq_companies_code
    ON companies(code) WHERE code IS NOT NULL AND code <> '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_roles_company_code
    ON roles(company_id, code) WHERE code IS NOT NULL AND code <> '';
