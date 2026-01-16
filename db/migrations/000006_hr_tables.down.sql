-- K-ERP v0.2 Migration: HR Tables (Rollback)

DROP TRIGGER IF EXISTS set_employee_leave_balances_updated_at ON employee_leave_balances;
DROP TRIGGER IF EXISTS set_employee_leaves_updated_at ON employee_leaves;
DROP TRIGGER IF EXISTS set_leave_types_updated_at ON leave_types;
DROP TRIGGER IF EXISTS set_employee_salaries_updated_at ON employee_salaries;
DROP TRIGGER IF EXISTS set_employees_updated_at ON employees;
DROP TRIGGER IF EXISTS set_positions_updated_at ON positions;

DROP TABLE IF EXISTS employee_leave_balances;
DROP TABLE IF EXISTS employee_leaves;
DROP TABLE IF EXISTS leave_types;
DROP TABLE IF EXISTS employee_documents;
DROP TABLE IF EXISTS employee_salaries;
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS positions;
