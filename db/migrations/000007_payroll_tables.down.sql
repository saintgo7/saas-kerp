-- K-ERP v0.2 Migration: Payroll Tables (Rollback)

DROP TRIGGER IF EXISTS set_payrolls_updated_at ON payrolls;
DROP TRIGGER IF EXISTS set_payroll_periods_updated_at ON payroll_periods;

DROP TABLE IF EXISTS payroll_transfer_details;
DROP TABLE IF EXISTS payroll_bank_transfers;
DROP TABLE IF EXISTS social_insurance_rates;
DROP TABLE IF EXISTS income_tax_tables;
DROP TABLE IF EXISTS payroll_items;
DROP TABLE IF EXISTS payrolls;
DROP TABLE IF EXISTS payroll_periods;
