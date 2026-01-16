-- K-ERP v0.2 Migration: Partners and Projects
-- Trading partners and cost tracking projects

-- ============================================
-- PARTNERS (Trading Partners)
-- ============================================
CREATE TABLE partners (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Basic info
    partner_type VARCHAR(20) NOT NULL CHECK (partner_type IN ('customer', 'vendor', 'both')),
    business_number VARCHAR(12),
    partner_name VARCHAR(100) NOT NULL,
    partner_name_en VARCHAR(100),
    representative VARCHAR(50),

    -- Business info
    business_type VARCHAR(100),
    business_item VARCHAR(200),

    -- Contact
    phone VARCHAR(20),
    fax VARCHAR(20),
    email VARCHAR(100),
    website VARCHAR(200),

    -- Address
    zip_code VARCHAR(10),
    address VARCHAR(200),
    address_detail VARCHAR(100),

    -- Bank info (encrypted)
    bank_code VARCHAR(10),
    account_number_enc BYTEA,
    account_holder VARCHAR(50),

    -- Credit
    credit_limit DECIMAL(18, 0) DEFAULT 0,
    payment_terms INTEGER DEFAULT 30,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    -- Notes
    notes TEXT,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ,

    UNIQUE(company_id, business_number) WHERE business_number IS NOT NULL
);

CREATE INDEX idx_partners_company ON partners(company_id);
CREATE INDEX idx_partners_type ON partners(company_id, partner_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_partners_business_number ON partners(company_id, business_number) WHERE business_number IS NOT NULL;
CREATE INDEX idx_partners_name ON partners(company_id, partner_name);
CREATE INDEX idx_partners_active ON partners(company_id) WHERE is_active = TRUE AND deleted_at IS NULL;

COMMENT ON TABLE partners IS 'Trading partners (customers and vendors)';
COMMENT ON COLUMN partners.account_number_enc IS 'Encrypted bank account number';

-- ============================================
-- PROJECTS (Cost Centers)
-- ============================================
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Basic info
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Classification
    project_type VARCHAR(20) DEFAULT 'general' CHECK (project_type IN ('general', 'internal', 'external', 'capex')),

    -- Period
    start_date DATE,
    end_date DATE,

    -- Budget
    budget_amount DECIMAL(18, 0) DEFAULT 0,
    actual_amount DECIMAL(18, 0) DEFAULT 0,

    -- Manager
    manager_id UUID REFERENCES users(id),

    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('draft', 'active', 'on_hold', 'completed', 'cancelled')),

    -- Hierarchy
    parent_id UUID REFERENCES projects(id),
    level INTEGER NOT NULL DEFAULT 1,
    path LTREE,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ,

    UNIQUE(company_id, code)
);

CREATE INDEX idx_projects_company ON projects(company_id);
CREATE INDEX idx_projects_status ON projects(company_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_path ON projects USING GIST (path);
CREATE INDEX idx_projects_manager ON projects(manager_id) WHERE manager_id IS NOT NULL;
CREATE INDEX idx_projects_active ON projects(company_id) WHERE status = 'active' AND deleted_at IS NULL;

COMMENT ON TABLE projects IS 'Projects for cost tracking and budgeting';

-- ============================================
-- COST_CENTERS (Alternative to Projects for simpler tracking)
-- ============================================
CREATE TABLE cost_centers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Basic info
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500),

    -- Hierarchy
    parent_id UUID REFERENCES cost_centers(id),
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

CREATE INDEX idx_cost_centers_company ON cost_centers(company_id);
CREATE INDEX idx_cost_centers_path ON cost_centers USING GIST (path);
CREATE INDEX idx_cost_centers_active ON cost_centers(company_id) WHERE is_active = TRUE;

COMMENT ON TABLE cost_centers IS 'Cost centers for departmental expense tracking';

-- ============================================
-- TRIGGERS
-- ============================================
CREATE TRIGGER set_partners_updated_at
    BEFORE UPDATE ON partners
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_cost_centers_updated_at
    BEFORE UPDATE ON cost_centers
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
