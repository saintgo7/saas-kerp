-- K-ERP v0.2 Migration: Accounting Tables (Rollback)

DROP TRIGGER IF EXISTS set_ledger_balances_updated_at ON ledger_balances;
DROP TRIGGER IF EXISTS set_vouchers_updated_at ON vouchers;
DROP TRIGGER IF EXISTS set_accounts_updated_at ON accounts;
DROP TRIGGER IF EXISTS set_departments_updated_at ON departments;

DROP FUNCTION IF EXISTS generate_voucher_number(UUID, VARCHAR, DATE);

DROP TABLE IF EXISTS voucher_sequences;
DROP TABLE IF EXISTS voucher_attachments;
DROP TABLE IF EXISTS ledger_balances;
DROP TABLE IF EXISTS voucher_entries;
DROP TABLE IF EXISTS vouchers;
DROP TABLE IF EXISTS fiscal_periods;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS departments;
