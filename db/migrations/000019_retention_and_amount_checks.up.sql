-- ============================================================================
-- RETENTION-CRITICAL FOREIGN KEYS + AMOUNT INTEGRITY
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. Payroll and insurance history must survive an employee row deletion
-- ----------------------------------------------------------------------------
-- Deleting one employee cascaded away that person's payroll register, salary
-- history, withholding records and social-insurance contributions. The Labor
-- Standards Act requires the wage ledger to be retained for three years, and
-- the application deletes hard (no gorm.DeletedAt anywhere), so a mistaken
-- delete of a same-named employee destroyed the records irrecoverably.
--
-- RESTRICT makes the database refuse the delete instead. Note that
-- voucher_entries.account_id already relies on the same protection, so this
-- also makes the policy consistent across the schema.
ALTER TABLE payrolls DROP CONSTRAINT IF EXISTS payrolls_employee_id_fkey;
ALTER TABLE payrolls ADD CONSTRAINT payrolls_employee_id_fkey
    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE RESTRICT;

ALTER TABLE employee_salaries DROP CONSTRAINT IF EXISTS employee_salaries_employee_id_fkey;
ALTER TABLE employee_salaries ADD CONSTRAINT employee_salaries_employee_id_fkey
    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE RESTRICT;

ALTER TABLE employee_insurance DROP CONSTRAINT IF EXISTS employee_insurance_employee_id_fkey;
ALTER TABLE employee_insurance ADD CONSTRAINT employee_insurance_employee_id_fkey
    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE RESTRICT;

ALTER TABLE insurance_monthly_contributions DROP CONSTRAINT IF EXISTS insurance_monthly_contributions_employee_id_fkey;
ALTER TABLE insurance_monthly_contributions ADD CONSTRAINT insurance_monthly_contributions_employee_id_fkey
    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE RESTRICT;

-- ----------------------------------------------------------------------------
-- 2. Closed period balances must survive a chart-of-accounts deletion
-- ----------------------------------------------------------------------------
-- ledger_balances.account_id was the only FK to accounts(id) with CASCADE,
-- while voucher_entries.account_id was already protected. Deleting an account
-- that happened to have no journal lines silently erased its closed monthly
-- balances, so the brought-forward figures - and the trial balance - broke.
ALTER TABLE ledger_balances DROP CONSTRAINT IF EXISTS ledger_balances_account_id_fkey;
ALTER TABLE ledger_balances ADD CONSTRAINT ledger_balances_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE RESTRICT;

-- ----------------------------------------------------------------------------
-- 3. Amount sign and total identity constraints
-- ----------------------------------------------------------------------------
-- Double-entry balance (000005) and the debit/credit exclusion (000005) are
-- enforced in SQL, but nothing stopped a negative or internally inconsistent
-- amount. The Go checks only cover the REST path; the scrapers, EDI importers
-- and manual SQL bypass them entirely, so a parse error like
-- supply=1,000,000 tax=100,000 total=110,000 was stored and only discovered at
-- VAT filing time.
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS chk_invoices_amounts;
ALTER TABLE invoices ADD CONSTRAINT chk_invoices_amounts
    CHECK (supply_amount >= 0 AND tax_amount >= 0 AND total_amount = supply_amount + tax_amount);

ALTER TABLE tax_invoices DROP CONSTRAINT IF EXISTS chk_tax_invoices_amounts;
ALTER TABLE tax_invoices ADD CONSTRAINT chk_tax_invoices_amounts
    CHECK (supply_amount >= 0 AND tax_amount >= 0 AND total_amount = supply_amount + tax_amount);

-- Payroll: gross, deductions and employer cost are never negative. net_pay is
-- deliberately NOT constrained to >= 0 - a month whose deductions exceed gross
-- pay (mid-month joiner with a year-end tax settlement clawback) is legitimate.
ALTER TABLE payrolls DROP CONSTRAINT IF EXISTS chk_payrolls_amounts;
ALTER TABLE payrolls ADD CONSTRAINT chk_payrolls_amounts
    CHECK (
        base_salary >= 0 AND overtime_pay >= 0 AND night_pay >= 0 AND holiday_pay >= 0
        AND bonus >= 0 AND total_earnings >= 0 AND total_deductions >= 0
        AND income_tax >= 0 AND local_income_tax >= 0
        AND total_employer_cost >= 0
    );

ALTER TABLE partners DROP CONSTRAINT IF EXISTS chk_partners_credit_limit;
ALTER TABLE partners ADD CONSTRAINT chk_partners_credit_limit
    CHECK (credit_limit >= 0);
