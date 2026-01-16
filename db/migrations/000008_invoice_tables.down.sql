-- K-ERP v0.2 Migration: Invoice Tables (Rollback)

DROP TRIGGER IF EXISTS set_hometax_credentials_updated_at ON hometax_credentials;
DROP TRIGGER IF EXISTS set_invoices_updated_at ON invoices;

DROP FUNCTION IF EXISTS generate_invoice_number(UUID, VARCHAR, DATE);

DROP TABLE IF EXISTS invoice_sequences;
DROP TABLE IF EXISTS hometax_credentials;
DROP TABLE IF EXISTS scrape_job_logs;
DROP TABLE IF EXISTS scrape_jobs;
DROP TABLE IF EXISTS invoice_items;
DROP TABLE IF EXISTS invoices;
