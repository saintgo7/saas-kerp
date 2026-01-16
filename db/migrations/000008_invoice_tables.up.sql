-- K-ERP v0.2 Migration: Invoice Tables
-- Tax invoices, items, and scraping jobs

-- ============================================
-- INVOICES (Tax Invoices / 세금계산서)
-- ============================================
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Invoice info
    invoice_no VARCHAR(30) NOT NULL,
    invoice_type VARCHAR(10) NOT NULL CHECK (invoice_type IN ('sales', 'purchase')),
    issue_type VARCHAR(20) NOT NULL DEFAULT 'normal' CHECK (issue_type IN ('normal', 'modified', 'cancelled')),
    issue_date DATE NOT NULL,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'issued', 'sent', 'received', 'cancelled', 'rejected')),

    -- NTS (National Tax Service) info
    nts_confirm_num VARCHAR(24),
    nts_issue_id VARCHAR(50),
    nts_result_code VARCHAR(10),
    nts_result_message VARCHAR(500),
    nts_submitted_at TIMESTAMPTZ,

    -- Supplier (공급자)
    supplier_business_number VARCHAR(12) NOT NULL,
    supplier_sub_number VARCHAR(4),
    supplier_company_name VARCHAR(100) NOT NULL,
    supplier_representative VARCHAR(50),
    supplier_address VARCHAR(200),
    supplier_business_type VARCHAR(100),
    supplier_business_item VARCHAR(200),
    supplier_email VARCHAR(100),
    supplier_phone VARCHAR(20),

    -- Buyer (공급받는자)
    buyer_business_number VARCHAR(12) NOT NULL,
    buyer_sub_number VARCHAR(4),
    buyer_company_name VARCHAR(100) NOT NULL,
    buyer_representative VARCHAR(50),
    buyer_address VARCHAR(200),
    buyer_business_type VARCHAR(100),
    buyer_business_item VARCHAR(200),
    buyer_email VARCHAR(100),
    buyer_email2 VARCHAR(100),
    buyer_phone VARCHAR(20),

    -- Amounts
    supply_amount DECIMAL(18, 0) NOT NULL,
    tax_amount DECIMAL(18, 0) NOT NULL,
    total_amount DECIMAL(18, 0) NOT NULL,

    -- Tax type
    tax_type VARCHAR(20) DEFAULT 'taxable' CHECK (tax_type IN ('taxable', 'zero_rate', 'exempt')),

    -- Payment info
    cash_amount DECIMAL(18, 0) DEFAULT 0,
    check_amount DECIMAL(18, 0) DEFAULT 0,
    note_amount DECIMAL(18, 0) DEFAULT 0,
    receivable_amount DECIMAL(18, 0) DEFAULT 0,

    -- Notes
    notes VARCHAR(500),
    remark1 VARCHAR(100),
    remark2 VARCHAR(100),
    remark3 VARCHAR(100),

    -- Provider info (Popbill, Scraper, NTS API)
    provider_used VARCHAR(20),
    provider_request_id VARCHAR(100),
    provider_response JSONB,

    -- Partner link
    partner_id UUID REFERENCES partners(id),

    -- Voucher link
    voucher_id UUID REFERENCES vouchers(id),

    -- Original invoice (for modified/cancelled)
    original_invoice_id UUID REFERENCES invoices(id),
    modification_code VARCHAR(10),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id),

    UNIQUE(company_id, invoice_no)
);

CREATE INDEX idx_invoices_company_date ON invoices(company_id, issue_date);
CREATE INDEX idx_invoices_type ON invoices(company_id, invoice_type);
CREATE INDEX idx_invoices_status ON invoices(company_id, status);
CREATE INDEX idx_invoices_nts_confirm ON invoices(nts_confirm_num) WHERE nts_confirm_num IS NOT NULL;
CREATE INDEX idx_invoices_supplier ON invoices(supplier_business_number);
CREATE INDEX idx_invoices_buyer ON invoices(buyer_business_number);
CREATE INDEX idx_invoices_partner ON invoices(company_id, partner_id) WHERE partner_id IS NOT NULL;
CREATE INDEX idx_invoices_voucher ON invoices(voucher_id) WHERE voucher_id IS NOT NULL;

COMMENT ON TABLE invoices IS 'Korean electronic tax invoices (전자세금계산서)';
COMMENT ON COLUMN invoices.nts_confirm_num IS 'NTS confirmation number (24 digits)';
COMMENT ON COLUMN invoices.provider_used IS 'Service provider: scraper, nts_api, popbill';

-- ============================================
-- INVOICE_ITEMS (Tax Invoice Line Items)
-- ============================================
CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Item info
    line_no INTEGER NOT NULL,
    item_date DATE NOT NULL,
    description VARCHAR(200) NOT NULL,

    -- Specifications
    specification VARCHAR(100),
    unit VARCHAR(20),
    quantity DECIMAL(18, 4) NOT NULL DEFAULT 1,
    unit_price DECIMAL(18, 2) NOT NULL,

    -- Amounts
    supply_amount DECIMAL(18, 0) NOT NULL,
    tax_amount DECIMAL(18, 0) NOT NULL,

    -- Notes
    notes VARCHAR(100),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(invoice_id, line_no)
);

CREATE INDEX idx_invoice_items_invoice ON invoice_items(invoice_id);

COMMENT ON TABLE invoice_items IS 'Line items within a tax invoice';

-- ============================================
-- SCRAPE_JOBS (Tax Invoice Scraping Tasks)
-- ============================================
CREATE TABLE scrape_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Job info
    job_type VARCHAR(30) NOT NULL CHECK (job_type IN ('invoice_scrape', 'invoice_issue', 'invoice_send', 'certificate_check')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'queued', 'processing', 'completed', 'failed', 'cancelled')),
    priority INTEGER DEFAULT 0,

    -- Parameters
    parameters JSONB NOT NULL,

    -- Progress
    total_count INTEGER DEFAULT 0,
    processed_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    fail_count INTEGER DEFAULT 0,

    -- Result
    result JSONB,
    error_code VARCHAR(50),
    error_message TEXT,
    error_details JSONB,

    -- Retry
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at TIMESTAMPTZ,

    -- Timing
    scheduled_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Worker info
    worker_id VARCHAR(100),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_scrape_jobs_company ON scrape_jobs(company_id);
CREATE INDEX idx_scrape_jobs_status ON scrape_jobs(status);
CREATE INDEX idx_scrape_jobs_type_status ON scrape_jobs(company_id, job_type, status);
CREATE INDEX idx_scrape_jobs_pending ON scrape_jobs(priority DESC, created_at) WHERE status IN ('pending', 'queued');
CREATE INDEX idx_scrape_jobs_retry ON scrape_jobs(next_retry_at) WHERE status = 'failed' AND retry_count < max_retries;

COMMENT ON TABLE scrape_jobs IS 'Async jobs for Hometax scraping and invoice operations';

-- ============================================
-- SCRAPE_JOB_LOGS
-- ============================================
CREATE TABLE scrape_job_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    job_id UUID NOT NULL REFERENCES scrape_jobs(id) ON DELETE CASCADE,

    -- Log info
    log_level VARCHAR(10) NOT NULL DEFAULT 'info' CHECK (log_level IN ('debug', 'info', 'warn', 'error')),
    message TEXT NOT NULL,
    details JSONB,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_scrape_job_logs_job ON scrape_job_logs(job_id);
CREATE INDEX idx_scrape_job_logs_level ON scrape_job_logs(job_id, log_level);

COMMENT ON TABLE scrape_job_logs IS 'Execution logs for scrape jobs';

-- ============================================
-- HOMETAX_CREDENTIALS (Encrypted)
-- ============================================
CREATE TABLE hometax_credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Credential type
    credential_type VARCHAR(20) NOT NULL CHECK (credential_type IN ('cert', 'simple_auth', 'bio')),

    -- Business number (for cert matching)
    business_number VARCHAR(12) NOT NULL,

    -- Encrypted certificate data
    cert_data_enc BYTEA,
    cert_password_enc BYTEA,
    cert_serial_number VARCHAR(100),
    cert_subject_dn VARCHAR(500),
    cert_expires_at TIMESTAMPTZ,

    -- Simple auth
    simple_auth_id_enc BYTEA,
    simple_auth_password_enc BYTEA,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    last_verified_at TIMESTAMPTZ,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, credential_type, business_number)
);

CREATE INDEX idx_hometax_credentials_company ON hometax_credentials(company_id);
CREATE INDEX idx_hometax_credentials_active ON hometax_credentials(company_id) WHERE is_active = TRUE;

COMMENT ON TABLE hometax_credentials IS 'Encrypted Hometax login credentials';
COMMENT ON COLUMN hometax_credentials.cert_data_enc IS 'SEED-encrypted certificate PFX/P12 data';

-- ============================================
-- INVOICE_SEQUENCES
-- ============================================
CREATE TABLE invoice_sequences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Sequence info
    fiscal_year INTEGER NOT NULL,
    invoice_type VARCHAR(10) NOT NULL,
    prefix VARCHAR(10) DEFAULT '',
    last_number INTEGER NOT NULL DEFAULT 0,

    -- Audit
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, fiscal_year, invoice_type)
);

COMMENT ON TABLE invoice_sequences IS 'Invoice number sequences per company/year/type';

-- ============================================
-- FUNCTION: Generate Invoice Number
-- ============================================
CREATE OR REPLACE FUNCTION generate_invoice_number(
    p_company_id UUID,
    p_invoice_type VARCHAR(10),
    p_issue_date DATE
)
RETURNS VARCHAR(30) AS $$
DECLARE
    v_year INTEGER;
    v_prefix VARCHAR(10);
    v_next_number INTEGER;
    v_invoice_no VARCHAR(30);
BEGIN
    v_year := EXTRACT(YEAR FROM p_issue_date);

    -- Get type prefix
    v_prefix := CASE p_invoice_type
        WHEN 'sales' THEN 'S'
        WHEN 'purchase' THEN 'P'
        ELSE 'X'
    END;

    -- Get and increment sequence (with lock)
    INSERT INTO invoice_sequences (company_id, fiscal_year, invoice_type, prefix, last_number)
    VALUES (p_company_id, v_year, p_invoice_type, v_prefix, 1)
    ON CONFLICT (company_id, fiscal_year, invoice_type)
    DO UPDATE SET
        last_number = invoice_sequences.last_number + 1,
        updated_at = NOW()
    RETURNING last_number INTO v_next_number;

    -- Format: PREFIX-YYYY-NNNNNNNN (e.g., S-2026-00000001)
    v_invoice_no := v_prefix || '-' || v_year::TEXT || '-' || LPAD(v_next_number::TEXT, 8, '0');

    RETURN v_invoice_no;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION generate_invoice_number IS 'Generate sequential invoice number by company/type/year';

-- ============================================
-- TRIGGERS
-- ============================================
CREATE TRIGGER set_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_hometax_credentials_updated_at
    BEFORE UPDATE ON hometax_credentials
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
