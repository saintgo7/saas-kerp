-- K-ERP v0.2 Migration: Insurance Tables (Rollback)

DROP TRIGGER IF EXISTS set_insurance_credentials_updated_at ON insurance_credentials;
DROP TRIGGER IF EXISTS set_insurance_contributions_updated_at ON insurance_monthly_contributions;
DROP TRIGGER IF EXISTS set_insurance_reports_updated_at ON insurance_reports;
DROP TRIGGER IF EXISTS set_employee_insurance_updated_at ON employee_insurance;
DROP TRIGGER IF EXISTS set_insurance_workplaces_updated_at ON insurance_workplaces;

DROP TABLE IF EXISTS edi_jobs;
DROP TABLE IF EXISTS insurance_credentials;
DROP TABLE IF EXISTS insurance_monthly_contributions;
DROP TABLE IF EXISTS insurance_report_items;
DROP TABLE IF EXISTS insurance_reports;
DROP TABLE IF EXISTS employee_insurance;
DROP TABLE IF EXISTS insurance_workplaces;
