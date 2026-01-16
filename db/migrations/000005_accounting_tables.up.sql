-- K-ERP v0.2 Migration: Accounting Tables
-- Chart of Accounts, Vouchers, Entries, Ledger Balances

-- ============================================
-- DEPARTMENTS (needed for voucher_entries FK)
-- ============================================
CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Basic info
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100),

    -- Hierarchy
    parent_id UUID REFERENCES departments(id),
    level INTEGER NOT NULL DEFAULT 1,
    path LTREE,

    -- Manager
    manager_id UUID REFERENCES users(id),

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, code)
);

CREATE INDEX idx_departments_company ON departments(company_id);
CREATE INDEX idx_departments_path ON departments USING GIST (path);
CREATE INDEX idx_departments_active ON departments(company_id) WHERE is_active = TRUE;

COMMENT ON TABLE departments IS 'Organizational departments';

-- ============================================
-- ACCOUNTS (Chart of Accounts)
-- ============================================
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Account info
    code VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100),

    -- Hierarchy
    parent_id UUID REFERENCES accounts(id),
    level INTEGER NOT NULL DEFAULT 1,
    path LTREE,

    -- Classification (K-IFRS)
    account_type VARCHAR(20) NOT NULL CHECK (account_type IN ('asset', 'liability', 'equity', 'revenue', 'expense')),
    account_nature VARCHAR(10) NOT NULL CHECK (account_nature IN ('debit', 'credit')),

    -- Sub-classification
    account_category VARCHAR(50),

    -- Settings
    is_active BOOLEAN DEFAULT TRUE,
    is_control_account BOOLEAN DEFAULT FALSE,
    allow_direct_posting BOOLEAN DEFAULT TRUE,

    -- Display order
    sort_order INTEGER DEFAULT 0,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, code)
);

CREATE INDEX idx_accounts_company ON accounts(company_id);
CREATE INDEX idx_accounts_parent ON accounts(parent_id);
CREATE INDEX idx_accounts_path ON accounts USING GIST (path);
CREATE INDEX idx_accounts_type ON accounts(company_id, account_type);
CREATE INDEX idx_accounts_active ON accounts(company_id) WHERE is_active = TRUE;

COMMENT ON TABLE accounts IS 'Chart of accounts following K-IFRS';
COMMENT ON COLUMN accounts.account_type IS 'Asset, Liability, Equity, Revenue, Expense';
COMMENT ON COLUMN accounts.account_nature IS 'Debit or Credit - determines normal balance side';
COMMENT ON COLUMN accounts.is_control_account IS 'Control accounts have sub-ledgers (e.g., AR, AP)';

-- ============================================
-- FISCAL_PERIODS
-- ============================================
CREATE TABLE fiscal_periods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Period info
    fiscal_year INTEGER NOT NULL,
    fiscal_month INTEGER NOT NULL CHECK (fiscal_month BETWEEN 1 AND 12),
    period_name VARCHAR(50),

    -- Dates
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,

    -- Status
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'closed', 'locked')),
    closed_at TIMESTAMPTZ,
    closed_by UUID REFERENCES users(id),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, fiscal_year, fiscal_month),
    CONSTRAINT chk_period_dates CHECK (start_date <= end_date)
);

CREATE INDEX idx_fiscal_periods_company ON fiscal_periods(company_id);
CREATE INDEX idx_fiscal_periods_year ON fiscal_periods(company_id, fiscal_year);
CREATE INDEX idx_fiscal_periods_open ON fiscal_periods(company_id) WHERE status = 'open';

COMMENT ON TABLE fiscal_periods IS 'Fiscal periods for accounting close process';

-- ============================================
-- VOUCHERS (Journal Entries)
-- ============================================
CREATE TABLE vouchers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Voucher info
    voucher_no VARCHAR(20) NOT NULL,
    voucher_date DATE NOT NULL,
    voucher_type VARCHAR(20) NOT NULL CHECK (voucher_type IN ('general', 'sales', 'purchase', 'payment', 'receipt', 'adjustment', 'closing')),

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending', 'approved', 'posted', 'rejected', 'cancelled')),

    -- Amounts (for validation)
    total_debit DECIMAL(18, 2) NOT NULL DEFAULT 0,
    total_credit DECIMAL(18, 2) NOT NULL DEFAULT 0,

    -- Description
    description VARCHAR(500),
    reference_type VARCHAR(50),
    reference_id UUID,

    -- Attachments
    attachment_count INTEGER DEFAULT 0,

    -- Approval workflow
    submitted_at TIMESTAMPTZ,
    submitted_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    approved_by UUID REFERENCES users(id),
    rejected_at TIMESTAMPTZ,
    rejected_by UUID REFERENCES users(id),
    rejection_reason VARCHAR(500),

    -- Posting
    posted_at TIMESTAMPTZ,
    posted_by UUID REFERENCES users(id),

    -- Reversal
    is_reversal BOOLEAN DEFAULT FALSE,
    reversal_of_id UUID REFERENCES vouchers(id),
    reversed_by_id UUID REFERENCES vouchers(id),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id),

    UNIQUE(company_id, voucher_no),
    CONSTRAINT chk_voucher_balance CHECK (total_debit = total_credit)
);

CREATE INDEX idx_vouchers_company_date ON vouchers(company_id, voucher_date);
CREATE INDEX idx_vouchers_status ON vouchers(company_id, status);
CREATE INDEX idx_vouchers_type ON vouchers(company_id, voucher_type);
CREATE INDEX idx_vouchers_reference ON vouchers(company_id, reference_type, reference_id) WHERE reference_id IS NOT NULL;
CREATE INDEX idx_vouchers_pending ON vouchers(company_id, created_at) WHERE status IN ('draft', 'pending');

COMMENT ON TABLE vouchers IS 'Journal vouchers (double-entry bookkeeping)';
COMMENT ON COLUMN vouchers.voucher_no IS 'Auto-generated voucher number per company';
COMMENT ON CONSTRAINT chk_voucher_balance ON vouchers IS 'Debit must equal Credit (double-entry principle)';

-- ============================================
-- VOUCHER_ENTRIES (Journal Entry Lines)
-- ============================================
CREATE TABLE voucher_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    voucher_id UUID NOT NULL REFERENCES vouchers(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Entry info
    line_no INTEGER NOT NULL,
    account_id UUID NOT NULL REFERENCES accounts(id),

    -- Amounts (one must be zero)
    debit_amount DECIMAL(18, 2) NOT NULL DEFAULT 0,
    credit_amount DECIMAL(18, 2) NOT NULL DEFAULT 0,

    -- Description
    description VARCHAR(200),

    -- Dimensions
    partner_id UUID REFERENCES partners(id),
    department_id UUID REFERENCES departments(id),
    project_id UUID REFERENCES projects(id),
    cost_center_id UUID REFERENCES cost_centers(id),

    -- Tags for analysis
    tags JSONB DEFAULT '[]',

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(voucher_id, line_no),
    CONSTRAINT chk_entry_amount CHECK (
        (debit_amount > 0 AND credit_amount = 0) OR
        (debit_amount = 0 AND credit_amount > 0)
    )
);

CREATE INDEX idx_voucher_entries_voucher ON voucher_entries(voucher_id);
CREATE INDEX idx_voucher_entries_account ON voucher_entries(company_id, account_id);
CREATE INDEX idx_voucher_entries_partner ON voucher_entries(company_id, partner_id) WHERE partner_id IS NOT NULL;
CREATE INDEX idx_voucher_entries_department ON voucher_entries(company_id, department_id) WHERE department_id IS NOT NULL;
CREATE INDEX idx_voucher_entries_project ON voucher_entries(company_id, project_id) WHERE project_id IS NOT NULL;

COMMENT ON TABLE voucher_entries IS 'Individual debit/credit entries within a voucher';
COMMENT ON CONSTRAINT chk_entry_amount ON voucher_entries IS 'Each line must be either debit or credit, not both';

-- ============================================
-- LEDGER_BALANCES (Period-End Balances)
-- ============================================
CREATE TABLE ledger_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,

    -- Period
    fiscal_year INTEGER NOT NULL,
    fiscal_month INTEGER NOT NULL CHECK (fiscal_month BETWEEN 1 AND 12),

    -- Balances
    opening_debit DECIMAL(18, 2) NOT NULL DEFAULT 0,
    opening_credit DECIMAL(18, 2) NOT NULL DEFAULT 0,
    period_debit DECIMAL(18, 2) NOT NULL DEFAULT 0,
    period_credit DECIMAL(18, 2) NOT NULL DEFAULT 0,
    closing_debit DECIMAL(18, 2) NOT NULL DEFAULT 0,
    closing_credit DECIMAL(18, 2) NOT NULL DEFAULT 0,

    -- Computed balance
    balance DECIMAL(18, 2) GENERATED ALWAYS AS (
        (opening_debit - opening_credit) + (period_debit - period_credit)
    ) STORED,

    -- Audit
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, account_id, fiscal_year, fiscal_month)
);

CREATE INDEX idx_ledger_balances_period ON ledger_balances(company_id, fiscal_year, fiscal_month);
CREATE INDEX idx_ledger_balances_account ON ledger_balances(company_id, account_id);

COMMENT ON TABLE ledger_balances IS 'Pre-aggregated account balances by period';
COMMENT ON COLUMN ledger_balances.balance IS 'Computed as (opening_debit - opening_credit) + (period_debit - period_credit)';

-- ============================================
-- VOUCHER_ATTACHMENTS
-- ============================================
CREATE TABLE voucher_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    voucher_id UUID NOT NULL REFERENCES vouchers(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- File info
    file_name VARCHAR(255) NOT NULL,
    file_size INTEGER NOT NULL,
    file_type VARCHAR(100),
    storage_path VARCHAR(500) NOT NULL,

    -- Audit
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by UUID REFERENCES users(id)
);

CREATE INDEX idx_voucher_attachments_voucher ON voucher_attachments(voucher_id);

COMMENT ON TABLE voucher_attachments IS 'Supporting documents for vouchers';

-- ============================================
-- VOUCHER_NUMBER_SEQUENCE
-- ============================================
CREATE TABLE voucher_sequences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Sequence info
    fiscal_year INTEGER NOT NULL,
    voucher_type VARCHAR(20) NOT NULL,
    prefix VARCHAR(10) DEFAULT '',
    last_number INTEGER NOT NULL DEFAULT 0,

    -- Audit
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, fiscal_year, voucher_type)
);

COMMENT ON TABLE voucher_sequences IS 'Voucher number sequences per company/year/type';

-- ============================================
-- FUNCTION: Generate Voucher Number
-- ============================================
CREATE OR REPLACE FUNCTION generate_voucher_number(
    p_company_id UUID,
    p_voucher_type VARCHAR(20),
    p_voucher_date DATE
)
RETURNS VARCHAR(20) AS $$
DECLARE
    v_year INTEGER;
    v_prefix VARCHAR(10);
    v_next_number INTEGER;
    v_voucher_no VARCHAR(20);
BEGIN
    v_year := EXTRACT(YEAR FROM p_voucher_date);

    -- Get type prefix
    v_prefix := CASE p_voucher_type
        WHEN 'general' THEN 'GJ'
        WHEN 'sales' THEN 'SJ'
        WHEN 'purchase' THEN 'PJ'
        WHEN 'payment' THEN 'PM'
        WHEN 'receipt' THEN 'RC'
        WHEN 'adjustment' THEN 'AJ'
        WHEN 'closing' THEN 'CL'
        ELSE 'XX'
    END;

    -- Get and increment sequence (with lock)
    INSERT INTO voucher_sequences (company_id, fiscal_year, voucher_type, prefix, last_number)
    VALUES (p_company_id, v_year, p_voucher_type, v_prefix, 1)
    ON CONFLICT (company_id, fiscal_year, voucher_type)
    DO UPDATE SET
        last_number = voucher_sequences.last_number + 1,
        updated_at = NOW()
    RETURNING last_number INTO v_next_number;

    -- Format: PREFIX-YYYY-NNNNNN (e.g., GJ-2026-000001)
    v_voucher_no := v_prefix || '-' || v_year::TEXT || '-' || LPAD(v_next_number::TEXT, 6, '0');

    RETURN v_voucher_no;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION generate_voucher_number IS 'Generate sequential voucher number by company/type/year';

-- ============================================
-- TRIGGERS
-- ============================================
CREATE TRIGGER set_departments_updated_at
    BEFORE UPDATE ON departments
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_vouchers_updated_at
    BEFORE UPDATE ON vouchers
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_ledger_balances_updated_at
    BEFORE UPDATE ON ledger_balances
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
