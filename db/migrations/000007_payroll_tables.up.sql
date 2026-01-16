-- K-ERP v0.2 Migration: Payroll Tables
-- Payroll processing, tax tables, and payment records

-- ============================================
-- PAYROLL_PERIODS
-- ============================================
CREATE TABLE payroll_periods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Period info
    pay_year INTEGER NOT NULL,
    pay_month INTEGER NOT NULL CHECK (pay_month BETWEEN 1 AND 12),
    period_name VARCHAR(50),

    -- Dates
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    payment_date DATE,

    -- Status
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'processing', 'calculated', 'approved', 'paid', 'closed')),

    -- Processing info
    calculated_at TIMESTAMPTZ,
    calculated_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    approved_by UUID REFERENCES users(id),
    paid_at TIMESTAMPTZ,

    -- Totals
    total_employees INTEGER DEFAULT 0,
    total_earnings DECIMAL(18, 0) DEFAULT 0,
    total_deductions DECIMAL(18, 0) DEFAULT 0,
    total_net_pay DECIMAL(18, 0) DEFAULT 0,

    -- Voucher link
    voucher_id UUID REFERENCES vouchers(id),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, pay_year, pay_month),
    CONSTRAINT chk_period_dates CHECK (period_start <= period_end)
);

CREATE INDEX idx_payroll_periods_company ON payroll_periods(company_id);
CREATE INDEX idx_payroll_periods_status ON payroll_periods(company_id, status);

COMMENT ON TABLE payroll_periods IS 'Monthly payroll processing periods';

-- ============================================
-- PAYROLLS (Employee Payroll Records)
-- ============================================
CREATE TABLE payrolls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    payroll_period_id UUID REFERENCES payroll_periods(id),

    -- Period
    pay_year INTEGER NOT NULL,
    pay_month INTEGER NOT NULL CHECK (pay_month BETWEEN 1 AND 12),

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'calculated', 'approved', 'paid', 'cancelled')),

    -- Earnings
    base_salary DECIMAL(18, 0) NOT NULL DEFAULT 0,
    overtime_pay DECIMAL(18, 0) NOT NULL DEFAULT 0,
    night_pay DECIMAL(18, 0) NOT NULL DEFAULT 0,
    holiday_pay DECIMAL(18, 0) NOT NULL DEFAULT 0,
    bonus DECIMAL(18, 0) NOT NULL DEFAULT 0,
    allowances JSONB DEFAULT '{}'::JSONB,
    other_earnings JSONB DEFAULT '{}'::JSONB,
    total_earnings DECIMAL(18, 0) NOT NULL DEFAULT 0,

    -- Deductions - Taxes
    income_tax DECIMAL(18, 0) NOT NULL DEFAULT 0,
    local_income_tax DECIMAL(18, 0) NOT NULL DEFAULT 0,

    -- Deductions - Social Insurance (Employee portion)
    nps_employee DECIMAL(18, 0) NOT NULL DEFAULT 0,
    nhis_employee DECIMAL(18, 0) NOT NULL DEFAULT 0,
    nhis_ltc_employee DECIMAL(18, 0) NOT NULL DEFAULT 0,
    ei_employee DECIMAL(18, 0) NOT NULL DEFAULT 0,

    -- Deductions - Other
    other_deductions JSONB DEFAULT '{}'::JSONB,
    total_deductions DECIMAL(18, 0) NOT NULL DEFAULT 0,

    -- Net pay
    net_pay DECIMAL(18, 0) NOT NULL DEFAULT 0,

    -- Employer costs (for reference)
    nps_employer DECIMAL(18, 0) NOT NULL DEFAULT 0,
    nhis_employer DECIMAL(18, 0) NOT NULL DEFAULT 0,
    nhis_ltc_employer DECIMAL(18, 0) NOT NULL DEFAULT 0,
    ei_employer DECIMAL(18, 0) NOT NULL DEFAULT 0,
    wci_employer DECIMAL(18, 0) NOT NULL DEFAULT 0,
    total_employer_cost DECIMAL(18, 0) NOT NULL DEFAULT 0,

    -- Payment info
    payment_date DATE,
    paid_at TIMESTAMPTZ,
    bank_code VARCHAR(10),
    account_number_enc BYTEA,

    -- Work summary
    work_days INTEGER DEFAULT 0,
    overtime_hours DECIMAL(5, 1) DEFAULT 0,
    night_hours DECIMAL(5, 1) DEFAULT 0,
    holiday_hours DECIMAL(5, 1) DEFAULT 0,

    -- Notes
    notes VARCHAR(500),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, employee_id, pay_year, pay_month)
);

CREATE INDEX idx_payrolls_company_period ON payrolls(company_id, pay_year, pay_month);
CREATE INDEX idx_payrolls_employee ON payrolls(employee_id);
CREATE INDEX idx_payrolls_status ON payrolls(company_id, status);
CREATE INDEX idx_payrolls_period_id ON payrolls(payroll_period_id);

COMMENT ON TABLE payrolls IS 'Monthly payroll records per employee';
COMMENT ON COLUMN payrolls.nps_employee IS 'National Pension Service - employee portion';
COMMENT ON COLUMN payrolls.nhis_employee IS 'National Health Insurance - employee portion';
COMMENT ON COLUMN payrolls.nhis_ltc_employee IS 'Long-term Care Insurance - employee portion';
COMMENT ON COLUMN payrolls.ei_employee IS 'Employment Insurance - employee portion';
COMMENT ON COLUMN payrolls.wci_employer IS 'Workers Compensation Insurance - employer only';

-- ============================================
-- PAYROLL_ITEMS (Detailed breakdown)
-- ============================================
CREATE TABLE payroll_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    payroll_id UUID NOT NULL REFERENCES payrolls(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Item info
    item_type VARCHAR(20) NOT NULL CHECK (item_type IN ('earning', 'deduction')),
    item_code VARCHAR(50) NOT NULL,
    item_name VARCHAR(100) NOT NULL,

    -- Amount
    amount DECIMAL(18, 0) NOT NULL,

    -- Flags
    is_taxable BOOLEAN DEFAULT TRUE,
    is_fixed BOOLEAN DEFAULT FALSE,

    -- Order for display
    sort_order INTEGER DEFAULT 0,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payroll_items_payroll ON payroll_items(payroll_id);
CREATE INDEX idx_payroll_items_type ON payroll_items(payroll_id, item_type);

COMMENT ON TABLE payroll_items IS 'Detailed earning and deduction items per payroll';

-- ============================================
-- INCOME_TAX_TABLES (Korean tax brackets)
-- ============================================
CREATE TABLE income_tax_tables (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),

    -- Effective period
    effective_year INTEGER NOT NULL,

    -- Bracket info
    min_amount DECIMAL(18, 0) NOT NULL,
    max_amount DECIMAL(18, 0),
    base_tax DECIMAL(18, 0) NOT NULL DEFAULT 0,
    tax_rate DECIMAL(5, 4) NOT NULL,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(effective_year, min_amount)
);

CREATE INDEX idx_income_tax_tables_year ON income_tax_tables(effective_year);

COMMENT ON TABLE income_tax_tables IS 'Korean income tax brackets by year';

-- ============================================
-- SOCIAL_INSURANCE_RATES
-- ============================================
CREATE TABLE social_insurance_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),

    -- Effective period
    effective_year INTEGER NOT NULL,
    effective_month INTEGER NOT NULL CHECK (effective_month BETWEEN 1 AND 12),

    -- Insurance type
    insurance_type VARCHAR(10) NOT NULL CHECK (insurance_type IN ('nps', 'nhis', 'nhis_ltc', 'ei', 'wci')),

    -- Rates (as decimal, e.g., 0.045 for 4.5%)
    employee_rate DECIMAL(7, 5),
    employer_rate DECIMAL(7, 5),

    -- Caps
    min_base DECIMAL(18, 0),
    max_base DECIMAL(18, 0),

    -- Notes
    notes VARCHAR(200),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(effective_year, effective_month, insurance_type)
);

CREATE INDEX idx_social_insurance_rates_period ON social_insurance_rates(effective_year, effective_month);

COMMENT ON TABLE social_insurance_rates IS 'Social insurance contribution rates';
COMMENT ON COLUMN social_insurance_rates.nps IS 'National Pension Service rates';

-- ============================================
-- PAYROLL_BANK_TRANSFERS
-- ============================================
CREATE TABLE payroll_bank_transfers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    payroll_period_id UUID NOT NULL REFERENCES payroll_periods(id),

    -- Transfer info
    transfer_date DATE NOT NULL,
    bank_code VARCHAR(10) NOT NULL,
    total_amount DECIMAL(18, 0) NOT NULL,
    total_count INTEGER NOT NULL,

    -- File info
    file_name VARCHAR(255),
    file_path VARCHAR(500),

    -- Status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'submitted', 'processing', 'completed', 'failed', 'partial')),

    -- Result
    success_count INTEGER DEFAULT 0,
    fail_count INTEGER DEFAULT 0,
    result_message TEXT,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    submitted_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_payroll_bank_transfers_period ON payroll_bank_transfers(payroll_period_id);
CREATE INDEX idx_payroll_bank_transfers_status ON payroll_bank_transfers(company_id, status);

COMMENT ON TABLE payroll_bank_transfers IS 'Bulk bank transfer records for payroll';

-- ============================================
-- PAYROLL_TRANSFER_DETAILS
-- ============================================
CREATE TABLE payroll_transfer_details (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    transfer_id UUID NOT NULL REFERENCES payroll_bank_transfers(id) ON DELETE CASCADE,
    payroll_id UUID NOT NULL REFERENCES payrolls(id),

    -- Recipient info
    employee_id UUID NOT NULL REFERENCES employees(id),
    bank_code VARCHAR(10) NOT NULL,
    account_number_enc BYTEA NOT NULL,
    account_holder VARCHAR(50) NOT NULL,
    amount DECIMAL(18, 0) NOT NULL,

    -- Status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failed')),
    error_code VARCHAR(20),
    error_message VARCHAR(500),

    -- Audit
    processed_at TIMESTAMPTZ
);

CREATE INDEX idx_payroll_transfer_details_transfer ON payroll_transfer_details(transfer_id);
CREATE INDEX idx_payroll_transfer_details_employee ON payroll_transfer_details(employee_id);

COMMENT ON TABLE payroll_transfer_details IS 'Individual transfer records within a batch';

-- ============================================
-- TRIGGERS
-- ============================================
CREATE TRIGGER set_payroll_periods_updated_at
    BEFORE UPDATE ON payroll_periods
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_payrolls_updated_at
    BEFORE UPDATE ON payrolls
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
