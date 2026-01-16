-- Tax Invoice Tables for K-ERP SaaS
-- Phase 5: Hometax Integration
-- Supports both issued (sales) and received (purchase) tax invoices

-- Tax Invoice main table (세금계산서)
CREATE TABLE tax_invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Invoice identification
    invoice_number VARCHAR(50) NOT NULL,
    invoice_type VARCHAR(20) NOT NULL CHECK (invoice_type IN ('sales', 'purchase')),
    issue_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'issued', 'transmitted', 'confirmed', 'cancelled', 'rejected')),

    -- Supplier (seller) information
    supplier_business_number VARCHAR(10) NOT NULL,
    supplier_name VARCHAR(200) NOT NULL,
    supplier_ceo_name VARCHAR(100),
    supplier_address TEXT,
    supplier_business_type VARCHAR(200),
    supplier_business_item VARCHAR(200),
    supplier_email VARCHAR(255),

    -- Buyer (purchaser) information
    buyer_business_number VARCHAR(10) NOT NULL,
    buyer_name VARCHAR(200) NOT NULL,
    buyer_ceo_name VARCHAR(100),
    buyer_address TEXT,
    buyer_business_type VARCHAR(200),
    buyer_business_item VARCHAR(200),
    buyer_email VARCHAR(255),

    -- Amount information (in KRW)
    supply_amount BIGINT NOT NULL DEFAULT 0,
    tax_amount BIGINT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL DEFAULT 0,

    -- NTS (National Tax Service) information
    nts_confirm_number VARCHAR(50),
    nts_transmitted_at TIMESTAMPTZ,
    nts_confirmed_at TIMESTAMPTZ,

    -- Popbill/ASP information
    asp_provider VARCHAR(50),  -- 'popbill', 'hometax', etc.
    asp_invoice_id VARCHAR(100),
    asp_response JSONB,

    -- Linked voucher
    voucher_id UUID REFERENCES vouchers(id) ON DELETE SET NULL,

    -- Metadata
    remarks TEXT,
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint per company
    CONSTRAINT uq_tax_invoices_number UNIQUE (company_id, invoice_number, invoice_type)
);

-- Tax invoice items table (세금계산서 품목)
CREATE TABLE tax_invoice_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tax_invoice_id UUID NOT NULL REFERENCES tax_invoices(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Item details
    sequence_number INT NOT NULL DEFAULT 1,
    supply_date DATE,
    description VARCHAR(500) NOT NULL,
    specification VARCHAR(200),
    quantity DECIMAL(18, 4) NOT NULL DEFAULT 1,
    unit_price DECIMAL(18, 2) NOT NULL DEFAULT 0,
    amount BIGINT NOT NULL DEFAULT 0,
    tax_amount BIGINT NOT NULL DEFAULT 0,
    remarks VARCHAR(500),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tax_invoice_items_seq UNIQUE (tax_invoice_id, sequence_number)
);

-- Tax invoice attachments
CREATE TABLE tax_invoice_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tax_invoice_id UUID NOT NULL REFERENCES tax_invoices(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    mime_type VARCHAR(100) NOT NULL,
    attachment_type VARCHAR(50) NOT NULL DEFAULT 'general'
        CHECK (attachment_type IN ('general', 'original', 'signed', 'nts_receipt')),

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tax invoice history (상태 변경 이력)
CREATE TABLE tax_invoice_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tax_invoice_id UUID NOT NULL REFERENCES tax_invoices(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    previous_status VARCHAR(20),
    new_status VARCHAR(20) NOT NULL,
    changed_by UUID REFERENCES users(id),
    change_reason TEXT,
    api_response JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hometax session table (for scraper sessions)
CREATE TABLE hometax_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    session_id VARCHAR(100) NOT NULL UNIQUE,
    business_number VARCHAR(10) NOT NULL,
    auth_type VARCHAR(20) NOT NULL CHECK (auth_type IN ('certificate', 'simple_auth', 'id_password')),

    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Popbill API configuration per company
CREATE TABLE popbill_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    link_id VARCHAR(100) NOT NULL,
    secret_key_encrypted BYTEA NOT NULL,  -- SEED encrypted
    is_sandbox BOOLEAN NOT NULL DEFAULT true,
    is_active BOOLEAN NOT NULL DEFAULT true,

    -- API quotas and usage
    monthly_quota INT NOT NULL DEFAULT 0,
    monthly_used INT NOT NULL DEFAULT 0,
    quota_reset_at DATE,

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_popbill_config_company UNIQUE (company_id)
);

-- Indexes for tax_invoices
CREATE INDEX idx_tax_invoices_company ON tax_invoices(company_id);
CREATE INDEX idx_tax_invoices_issue_date ON tax_invoices(company_id, issue_date);
CREATE INDEX idx_tax_invoices_status ON tax_invoices(company_id, status);
CREATE INDEX idx_tax_invoices_supplier ON tax_invoices(company_id, supplier_business_number);
CREATE INDEX idx_tax_invoices_buyer ON tax_invoices(company_id, buyer_business_number);
CREATE INDEX idx_tax_invoices_voucher ON tax_invoices(voucher_id) WHERE voucher_id IS NOT NULL;
CREATE INDEX idx_tax_invoices_nts_confirm ON tax_invoices(nts_confirm_number) WHERE nts_confirm_number IS NOT NULL;

-- Indexes for tax_invoice_items
CREATE INDEX idx_tax_invoice_items_invoice ON tax_invoice_items(tax_invoice_id);
CREATE INDEX idx_tax_invoice_items_company ON tax_invoice_items(company_id);

-- Indexes for attachments
CREATE INDEX idx_tax_invoice_attachments_invoice ON tax_invoice_attachments(tax_invoice_id);

-- Indexes for history
CREATE INDEX idx_tax_invoice_history_invoice ON tax_invoice_history(tax_invoice_id);
CREATE INDEX idx_tax_invoice_history_created ON tax_invoice_history(company_id, created_at);

-- Indexes for sessions
CREATE INDEX idx_hometax_sessions_company ON hometax_sessions(company_id);
CREATE INDEX idx_hometax_sessions_active ON hometax_sessions(company_id, is_active, expires_at);

-- RLS Policies
ALTER TABLE tax_invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_invoice_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_invoice_attachments ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_invoice_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE hometax_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE popbill_configs ENABLE ROW LEVEL SECURITY;

-- Create RLS policies (tenant isolation)
CREATE POLICY tax_invoices_tenant_policy ON tax_invoices
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);

CREATE POLICY tax_invoice_items_tenant_policy ON tax_invoice_items
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);

CREATE POLICY tax_invoice_attachments_tenant_policy ON tax_invoice_attachments
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);

CREATE POLICY tax_invoice_history_tenant_policy ON tax_invoice_history
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);

CREATE POLICY hometax_sessions_tenant_policy ON hometax_sessions
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);

CREATE POLICY popbill_configs_tenant_policy ON popbill_configs
    FOR ALL USING (company_id = current_setting('app.current_company_id')::UUID);

-- Triggers for updated_at
CREATE TRIGGER tax_invoices_updated_at
    BEFORE UPDATE ON tax_invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER tax_invoice_items_updated_at
    BEFORE UPDATE ON tax_invoice_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER popbill_configs_updated_at
    BEFORE UPDATE ON popbill_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Comments
COMMENT ON TABLE tax_invoices IS 'Tax invoices (세금계산서) for sales and purchases';
COMMENT ON TABLE tax_invoice_items IS 'Line items for tax invoices';
COMMENT ON TABLE tax_invoice_attachments IS 'Attached files for tax invoices';
COMMENT ON TABLE tax_invoice_history IS 'Status change history for tax invoices';
COMMENT ON TABLE hometax_sessions IS 'Hometax scraper session management';
COMMENT ON TABLE popbill_configs IS 'Popbill API configuration per company';
