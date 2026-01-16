-- K-ERP v0.2 Migration: HR Tables
-- Positions, Employees, Salaries
-- Note: departments table created in 000005_accounting_tables.up.sql

-- ============================================
-- POSITIONS (Job Titles)
-- ============================================
CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Basic info
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100),

    -- Level (for sorting and hierarchy)
    rank_level INTEGER DEFAULT 0,

    -- Pay info
    min_salary DECIMAL(18, 0),
    max_salary DECIMAL(18, 0),

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, code)
);

CREATE INDEX idx_positions_company ON positions(company_id);
CREATE INDEX idx_positions_active ON positions(company_id) WHERE is_active = TRUE;

COMMENT ON TABLE positions IS 'Job positions/titles';

-- ============================================
-- EMPLOYEES
-- ============================================
CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),

    -- Employee info
    employee_no VARCHAR(20) NOT NULL,
    name VARCHAR(50) NOT NULL,
    name_en VARCHAR(100),

    -- Personal info (encrypted sensitive fields)
    resident_number_enc BYTEA,
    birth_date DATE,
    gender VARCHAR(10) CHECK (gender IN ('male', 'female', 'other')),
    nationality VARCHAR(50) DEFAULT 'KR',

    -- Contact
    phone VARCHAR(20),
    mobile VARCHAR(20),
    email VARCHAR(100),
    emergency_contact VARCHAR(100),
    emergency_phone VARCHAR(20),

    -- Address
    zip_code VARCHAR(10),
    address VARCHAR(200),
    address_detail VARCHAR(100),

    -- Organization
    department_id UUID REFERENCES departments(id),
    position_id UUID REFERENCES positions(id),
    manager_id UUID REFERENCES employees(id),

    -- Employment info
    hire_date DATE NOT NULL,
    probation_end_date DATE,
    resignation_date DATE,
    resignation_reason VARCHAR(200),

    -- Employment type
    employment_type VARCHAR(20) DEFAULT 'regular' CHECK (employment_type IN ('regular', 'contract', 'part_time', 'intern', 'dispatch')),
    contract_start_date DATE,
    contract_end_date DATE,

    -- Work info
    work_location VARCHAR(100),
    work_email VARCHAR(100),
    work_phone VARCHAR(20),

    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'on_leave', 'resigned', 'terminated')),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ,

    UNIQUE(company_id, employee_no)
);

CREATE INDEX idx_employees_company ON employees(company_id);
CREATE INDEX idx_employees_user ON employees(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_employees_department ON employees(company_id, department_id);
CREATE INDEX idx_employees_position ON employees(company_id, position_id);
CREATE INDEX idx_employees_manager ON employees(manager_id) WHERE manager_id IS NOT NULL;
CREATE INDEX idx_employees_status ON employees(company_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_employees_active ON employees(company_id) WHERE status = 'active' AND deleted_at IS NULL;

COMMENT ON TABLE employees IS 'Employee master data';
COMMENT ON COLUMN employees.resident_number_enc IS 'Encrypted Korean resident registration number';

-- ============================================
-- EMPLOYEE_SALARIES (Salary History)
-- ============================================
CREATE TABLE employee_salaries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Period
    effective_date DATE NOT NULL,
    end_date DATE,

    -- Salary info
    base_salary DECIMAL(18, 0) NOT NULL,
    payment_type VARCHAR(20) NOT NULL DEFAULT 'monthly' CHECK (payment_type IN ('monthly', 'hourly', 'daily', 'annual')),

    -- Allowances (JSONB for flexibility)
    allowances JSONB DEFAULT '{}'::JSONB,

    -- Bank info (encrypted)
    bank_code VARCHAR(10),
    account_number_enc BYTEA,
    account_holder VARCHAR(50),

    -- Tax info
    income_tax_type VARCHAR(20) DEFAULT 'class_A',
    dependents_count INTEGER DEFAULT 1,

    -- Notes
    notes VARCHAR(500),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(employee_id, effective_date)
);

CREATE INDEX idx_employee_salaries_employee ON employee_salaries(employee_id);
CREATE INDEX idx_employee_salaries_effective ON employee_salaries(employee_id, effective_date DESC);

COMMENT ON TABLE employee_salaries IS 'Employee salary history with effective dates';
COMMENT ON COLUMN employee_salaries.allowances IS 'JSON object with allowance types and amounts';

-- ============================================
-- EMPLOYEE_DOCUMENTS
-- ============================================
CREATE TABLE employee_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Document info
    document_type VARCHAR(50) NOT NULL,
    document_name VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size INTEGER NOT NULL,
    file_type VARCHAR(100),
    storage_path VARCHAR(500) NOT NULL,

    -- Expiration
    issue_date DATE,
    expiry_date DATE,

    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'expired', 'archived')),

    -- Audit
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by UUID REFERENCES users(id)
);

CREATE INDEX idx_employee_documents_employee ON employee_documents(employee_id);
CREATE INDEX idx_employee_documents_type ON employee_documents(company_id, document_type);
CREATE INDEX idx_employee_documents_expiry ON employee_documents(expiry_date) WHERE expiry_date IS NOT NULL AND status = 'active';

COMMENT ON TABLE employee_documents IS 'Employee documents (contracts, certificates, etc.)';

-- ============================================
-- LEAVE_TYPES
-- ============================================
CREATE TABLE leave_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Basic info
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,

    -- Settings
    is_paid BOOLEAN DEFAULT TRUE,
    default_days DECIMAL(5, 1) DEFAULT 0,
    max_carryover_days DECIMAL(5, 1) DEFAULT 0,
    requires_approval BOOLEAN DEFAULT TRUE,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(company_id, code)
);

CREATE INDEX idx_leave_types_company ON leave_types(company_id);

COMMENT ON TABLE leave_types IS 'Types of employee leave (annual, sick, etc.)';

-- ============================================
-- EMPLOYEE_LEAVES
-- ============================================
CREATE TABLE employee_leaves (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    leave_type_id UUID NOT NULL REFERENCES leave_types(id),

    -- Leave period
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    days DECIMAL(5, 1) NOT NULL,

    -- Request info
    reason VARCHAR(500),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),

    -- Approval
    approved_at TIMESTAMPTZ,
    approved_by UUID REFERENCES users(id),
    rejection_reason VARCHAR(500),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_leave_dates CHECK (start_date <= end_date)
);

CREATE INDEX idx_employee_leaves_employee ON employee_leaves(employee_id);
CREATE INDEX idx_employee_leaves_dates ON employee_leaves(company_id, start_date, end_date);
CREATE INDEX idx_employee_leaves_status ON employee_leaves(company_id, status);

COMMENT ON TABLE employee_leaves IS 'Employee leave requests and records';

-- ============================================
-- EMPLOYEE_LEAVE_BALANCES
-- ============================================
CREATE TABLE employee_leave_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    leave_type_id UUID NOT NULL REFERENCES leave_types(id),

    -- Year
    fiscal_year INTEGER NOT NULL,

    -- Balances
    entitled_days DECIMAL(5, 1) NOT NULL DEFAULT 0,
    carryover_days DECIMAL(5, 1) NOT NULL DEFAULT 0,
    used_days DECIMAL(5, 1) NOT NULL DEFAULT 0,
    remaining_days DECIMAL(5, 1) GENERATED ALWAYS AS (entitled_days + carryover_days - used_days) STORED,

    -- Audit
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(employee_id, leave_type_id, fiscal_year)
);

CREATE INDEX idx_employee_leave_balances_employee ON employee_leave_balances(employee_id);
CREATE INDEX idx_employee_leave_balances_year ON employee_leave_balances(company_id, fiscal_year);

COMMENT ON TABLE employee_leave_balances IS 'Employee leave balance by year and type';

-- ============================================
-- TRIGGERS
-- ============================================
CREATE TRIGGER set_positions_updated_at
    BEFORE UPDATE ON positions
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_employees_updated_at
    BEFORE UPDATE ON employees
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_employee_salaries_updated_at
    BEFORE UPDATE ON employee_salaries
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_leave_types_updated_at
    BEFORE UPDATE ON leave_types
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_employee_leaves_updated_at
    BEFORE UPDATE ON employee_leaves
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_employee_leave_balances_updated_at
    BEFORE UPDATE ON employee_leave_balances
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
