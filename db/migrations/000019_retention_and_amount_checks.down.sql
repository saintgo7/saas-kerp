-- Reverse 000019 only.

-- 3. amount checks
ALTER TABLE partners     DROP CONSTRAINT IF EXISTS chk_partners_credit_limit;
ALTER TABLE payrolls     DROP CONSTRAINT IF EXISTS chk_payrolls_amounts;
ALTER TABLE tax_invoices DROP CONSTRAINT IF EXISTS chk_tax_invoices_amounts;
ALTER TABLE invoices     DROP CONSTRAINT IF EXISTS chk_invoices_amounts;

-- 2. ledger_balances.account_id back to CASCADE
ALTER TABLE ledger_balances DROP CONSTRAINT IF EXISTS ledger_balances_account_id_fkey;
ALTER TABLE ledger_balances ADD CONSTRAINT ledger_balances_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;

-- 1. employee FKs back to CASCADE
ALTER TABLE insurance_monthly_contributions DROP CONSTRAINT IF EXISTS insurance_monthly_contributions_employee_id_fkey;
ALTER TABLE insurance_monthly_contributions ADD CONSTRAINT insurance_monthly_contributions_employee_id_fkey
    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE;

ALTER TABLE employee_insurance DROP CONSTRAINT IF EXISTS employee_insurance_employee_id_fkey;
ALTER TABLE employee_insurance ADD CONSTRAINT employee_insurance_employee_id_fkey
    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE;

ALTER TABLE employee_salaries DROP CONSTRAINT IF EXISTS employee_salaries_employee_id_fkey;
ALTER TABLE employee_salaries ADD CONSTRAINT employee_salaries_employee_id_fkey
    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE;

ALTER TABLE payrolls DROP CONSTRAINT IF EXISTS payrolls_employee_id_fkey;
ALTER TABLE payrolls ADD CONSTRAINT payrolls_employee_id_fkey
    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE;
