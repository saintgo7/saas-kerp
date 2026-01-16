-- K-ERP v0.2 Migration: Insurance Tables
-- 4대보험 (National Pension, Health Insurance, Employment Insurance, Workers Comp)

-- ============================================
-- INSURANCE_WORKPLACES (사업장)
-- ============================================
CREATE TABLE insurance_workplaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Workplace info
    business_number VARCHAR(12) NOT NULL,
    workplace_name VARCHAR(100) NOT NULL,

    -- Insurance registration numbers
    nps_workplace_number VARCHAR(20),
    nhis_workplace_number VARCHAR(20),
    ei_workplace_number VARCHAR(20),
    wci_workplace_number VARCHAR(20),

    -- Registration status
    nps_registered BOOLEAN DEFAULT FALSE,
    nhis_registered BOOLEAN DEFAULT FALSE,
    ei_registered BOOLEAN DEFAULT FALSE,
    wci_registered BOOLEAN DEFAULT FALSE,

    -- Representative
    representative VARCHAR(50),
    representative_resident_enc BYTEA,

    -- Contact
    phone VARCHAR(20),
    fax VARCHAR(20),
    email VARCHAR(100),

    -- Address
    zip_code VARCHAR(10),
    address VARCHAR(200),
    address_detail VARCHAR(100),

    -- Business info
    business_type_code VARCHAR(10),
    business_type_name VARCHAR(100),

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, business_number)
);

CREATE INDEX idx_insurance_workplaces_company ON insurance_workplaces(company_id);

COMMENT ON TABLE insurance_workplaces IS 'Insurance workplace registrations';

-- ============================================
-- EMPLOYEE_INSURANCE (Employee Insurance Info)
-- ============================================
CREATE TABLE employee_insurance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Registration status
    nps_qualified BOOLEAN DEFAULT TRUE,
    nhis_qualified BOOLEAN DEFAULT TRUE,
    ei_qualified BOOLEAN DEFAULT TRUE,
    wci_qualified BOOLEAN DEFAULT TRUE,

    -- Acquisition dates
    nps_acquisition_date DATE,
    nhis_acquisition_date DATE,
    ei_acquisition_date DATE,

    -- Loss dates
    nps_loss_date DATE,
    nhis_loss_date DATE,
    ei_loss_date DATE,

    -- Standard monthly remuneration (기준소득월액)
    nps_standard_remuneration DECIMAL(18, 0),
    nhis_standard_remuneration DECIMAL(18, 0),

    -- Grades (등급)
    nhis_grade VARCHAR(10),

    -- Dependent info (for NHIS)
    dependents_count INTEGER DEFAULT 0,

    -- Reduction flags
    nps_reduced BOOLEAN DEFAULT FALSE,
    nhis_reduced BOOLEAN DEFAULT FALSE,
    nps_reduction_type VARCHAR(20),
    nhis_reduction_type VARCHAR(20),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(employee_id)
);

CREATE INDEX idx_employee_insurance_employee ON employee_insurance(employee_id);
CREATE INDEX idx_employee_insurance_company ON employee_insurance(company_id);

COMMENT ON TABLE employee_insurance IS 'Employee social insurance qualifications';

-- ============================================
-- INSURANCE_REPORTS (신고서)
-- ============================================
CREATE TABLE insurance_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    workplace_id UUID REFERENCES insurance_workplaces(id),

    -- Report info
    report_type VARCHAR(30) NOT NULL,
    agency_type VARCHAR(10) NOT NULL CHECK (agency_type IN ('nps', 'nhis', 'ei', 'wci')),

    -- Target (single employee or batch)
    employee_id UUID REFERENCES employees(id),
    is_batch BOOLEAN DEFAULT FALSE,

    -- Period
    report_year INTEGER,
    report_month INTEGER CHECK (report_month BETWEEN 1 AND 12),
    effective_date DATE,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending', 'submitted', 'accepted', 'rejected', 'cancelled')),

    -- Report data (JSON for flexibility across different report types)
    report_data JSONB NOT NULL,

    -- Submission info
    receipt_number VARCHAR(50),
    submission_method VARCHAR(20),
    submitted_at TIMESTAMPTZ,
    submitted_by UUID REFERENCES users(id),

    -- Result
    result_code VARCHAR(20),
    result_message TEXT,
    accepted_at TIMESTAMPTZ,
    rejection_reason TEXT,

    -- EDI info
    edi_standard VARCHAR(20),
    edi_message TEXT,
    edi_response TEXT,
    edi_file_path VARCHAR(500),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_insurance_reports_company ON insurance_reports(company_id);
CREATE INDEX idx_insurance_reports_workplace ON insurance_reports(workplace_id);
CREATE INDEX idx_insurance_reports_employee ON insurance_reports(employee_id) WHERE employee_id IS NOT NULL;
CREATE INDEX idx_insurance_reports_status ON insurance_reports(company_id, status);
CREATE INDEX idx_insurance_reports_agency ON insurance_reports(company_id, agency_type);
CREATE INDEX idx_insurance_reports_period ON insurance_reports(company_id, report_year, report_month);
CREATE INDEX idx_insurance_reports_type ON insurance_reports(company_id, report_type, status);

COMMENT ON TABLE insurance_reports IS 'Social insurance reports to government agencies';
COMMENT ON COLUMN insurance_reports.report_type IS 'acquisition, loss, change, monthly, annual, etc.';

-- ============================================
-- INSURANCE_REPORT_ITEMS (Report Details)
-- ============================================
CREATE TABLE insurance_report_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    report_id UUID NOT NULL REFERENCES insurance_reports(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id),

    -- Line info
    line_no INTEGER NOT NULL,

    -- Employee snapshot
    employee_name VARCHAR(50) NOT NULL,
    resident_number_masked VARCHAR(14),

    -- Amounts
    base_amount DECIMAL(18, 0),
    employee_amount DECIMAL(18, 0),
    employer_amount DECIMAL(18, 0),
    total_amount DECIMAL(18, 0),

    -- Additional data
    item_data JSONB,

    -- Status
    status VARCHAR(20) DEFAULT 'pending',
    error_message VARCHAR(500),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(report_id, line_no)
);

CREATE INDEX idx_insurance_report_items_report ON insurance_report_items(report_id);
CREATE INDEX idx_insurance_report_items_employee ON insurance_report_items(employee_id);

COMMENT ON TABLE insurance_report_items IS 'Individual employee records within a batch report';

-- ============================================
-- INSURANCE_MONTHLY_CONTRIBUTIONS (월별 보험료)
-- ============================================
CREATE TABLE insurance_monthly_contributions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,

    -- Period
    contribution_year INTEGER NOT NULL,
    contribution_month INTEGER NOT NULL CHECK (contribution_month BETWEEN 1 AND 12),

    -- NPS (국민연금)
    nps_base DECIMAL(18, 0) DEFAULT 0,
    nps_employee DECIMAL(18, 0) DEFAULT 0,
    nps_employer DECIMAL(18, 0) DEFAULT 0,

    -- NHIS (건강보험)
    nhis_base DECIMAL(18, 0) DEFAULT 0,
    nhis_employee DECIMAL(18, 0) DEFAULT 0,
    nhis_employer DECIMAL(18, 0) DEFAULT 0,

    -- NHIS LTC (장기요양)
    nhis_ltc_employee DECIMAL(18, 0) DEFAULT 0,
    nhis_ltc_employer DECIMAL(18, 0) DEFAULT 0,

    -- EI (고용보험)
    ei_base DECIMAL(18, 0) DEFAULT 0,
    ei_employee DECIMAL(18, 0) DEFAULT 0,
    ei_employer DECIMAL(18, 0) DEFAULT 0,

    -- WCI (산재보험) - employer only
    wci_base DECIMAL(18, 0) DEFAULT 0,
    wci_employer DECIMAL(18, 0) DEFAULT 0,

    -- Totals
    total_employee DECIMAL(18, 0) GENERATED ALWAYS AS (
        nps_employee + nhis_employee + nhis_ltc_employee + ei_employee
    ) STORED,
    total_employer DECIMAL(18, 0) GENERATED ALWAYS AS (
        nps_employer + nhis_employer + nhis_ltc_employer + ei_employer + wci_employer
    ) STORED,

    -- Link to payroll
    payroll_id UUID REFERENCES payrolls(id),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, employee_id, contribution_year, contribution_month)
);

CREATE INDEX idx_insurance_contributions_company ON insurance_monthly_contributions(company_id);
CREATE INDEX idx_insurance_contributions_employee ON insurance_monthly_contributions(employee_id);
CREATE INDEX idx_insurance_contributions_period ON insurance_monthly_contributions(company_id, contribution_year, contribution_month);

COMMENT ON TABLE insurance_monthly_contributions IS 'Monthly social insurance contributions per employee';

-- ============================================
-- INSURANCE_CREDENTIALS (EDI Credentials)
-- ============================================
CREATE TABLE insurance_credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    workplace_id UUID REFERENCES insurance_workplaces(id),

    -- Agency
    agency_type VARCHAR(10) NOT NULL CHECK (agency_type IN ('nps', 'nhis', 'ei', 'wci', 'comwel')),

    -- Credential type
    credential_type VARCHAR(20) NOT NULL CHECK (credential_type IN ('cert', 'id_password', 'api_key')),

    -- Certificate (ARIA encrypted)
    cert_data_enc BYTEA,
    cert_password_enc BYTEA,
    cert_serial_number VARCHAR(100),
    cert_expires_at TIMESTAMPTZ,

    -- ID/Password
    login_id_enc BYTEA,
    login_password_enc BYTEA,

    -- API credentials
    api_key_enc BYTEA,
    api_secret_enc BYTEA,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    last_verified_at TIMESTAMPTZ,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, agency_type, credential_type)
);

CREATE INDEX idx_insurance_credentials_company ON insurance_credentials(company_id);
CREATE INDEX idx_insurance_credentials_active ON insurance_credentials(company_id) WHERE is_active = TRUE;

COMMENT ON TABLE insurance_credentials IS 'Encrypted credentials for insurance agency EDI';
COMMENT ON COLUMN insurance_credentials.cert_data_enc IS 'ARIA-encrypted certificate data';

-- ============================================
-- EDI_JOBS (Insurance EDI Jobs)
-- ============================================
CREATE TABLE edi_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    report_id UUID REFERENCES insurance_reports(id),

    -- Job info
    job_type VARCHAR(30) NOT NULL,
    agency_type VARCHAR(10) NOT NULL CHECK (agency_type IN ('nps', 'nhis', 'ei', 'wci', 'comwel')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'queued', 'processing', 'completed', 'failed', 'cancelled')),

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

    -- Retry
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,

    -- Timing
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Worker
    worker_id VARCHAR(100),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_edi_jobs_company ON edi_jobs(company_id);
CREATE INDEX idx_edi_jobs_report ON edi_jobs(report_id) WHERE report_id IS NOT NULL;
CREATE INDEX idx_edi_jobs_status ON edi_jobs(status);
CREATE INDEX idx_edi_jobs_pending ON edi_jobs(created_at) WHERE status IN ('pending', 'queued');

COMMENT ON TABLE edi_jobs IS 'Async jobs for insurance EDI operations';

-- ============================================
-- TRIGGERS
-- ============================================
CREATE TRIGGER set_insurance_workplaces_updated_at
    BEFORE UPDATE ON insurance_workplaces
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_employee_insurance_updated_at
    BEFORE UPDATE ON employee_insurance
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_insurance_reports_updated_at
    BEFORE UPDATE ON insurance_reports
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_insurance_contributions_updated_at
    BEFORE UPDATE ON insurance_monthly_contributions
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_insurance_credentials_updated_at
    BEFORE UPDATE ON insurance_credentials
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
