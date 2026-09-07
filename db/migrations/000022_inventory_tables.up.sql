-- ============================================================================
-- INVENTORY: products, warehouses, stock, movements, purchase and sales orders
--
-- The React screens under web/src/pages/inventory have existed for a while
-- against web/src/types/inventory.ts, but no table in this schema ever backed
-- them. This migration creates that backing. The TypeScript interfaces are the
-- contract; column names are their snake_case form.
--
-- DESIGN NOTES
--
-- 1. THE MOVEMENT LEDGER IS THE SOURCE OF TRUTH.
--    stock_movements is append-only: one row per receipt, issue, adjustment or
--    transfer leg, carrying the before and after balance. `stocks` is a cached
--    running balance kept only so that "what is on hand right now" does not
--    have to sum the whole ledger. The service layer writes both inside ONE
--    transaction and updates the cache with an atomic
--        UPDATE stocks SET quantity = quantity + $delta ... RETURNING quantity
--    never a read-modify-write from Go. That is the same defect that was fixed
--    in voucher numbering (see GenerateVoucherNo): two concurrent requests that
--    each read 10 and each write 15 lose one movement.
--
-- 2. NEGATIVE STOCK IS FORBIDDEN, IN THE DATABASE.
--    chk_stocks_quantity_non_negative makes an over-issue fail even if it comes
--    from a script, an importer or manual SQL rather than from the REST path.
--    Combined with the atomic UPDATE above, the CHECK is what actually
--    serialises concurrent issues: the loser's statement violates the
--    constraint and its whole transaction (movement row included) rolls back.
--
-- 3. AMOUNT TYPES follow the tax-invoice precedent in 000012 / internal/domain:
--      quantity      NUMERIC(18,3)  - fractional units (kg, m, L) are real
--      unit price    NUMERIC(18,2)  - Korean ERPs allow sub-won unit prices
--      money amounts NUMERIC(18,0)  - KRW is booked in whole won, and integer
--                                     amounts make total = supply + tax an
--                                     exact identity that a CHECK can enforce
--    No column here is float/double: rounding a price into an amount happens
--    once, explicitly, in the service layer.
--
-- 4. UNIQUENESS IS PARTIAL WHERE SOFT DELETE APPLIES, and is therefore written
--    as CREATE UNIQUE INDEX ... WHERE, never as a table-level UNIQUE with a
--    WHERE clause - that is a syntax error and it has stopped this repository's
--    migrations from running end to end before.
--
-- 5. DELETES ARE RESTRICTED, not cascaded, wherever a row is evidence: a
--    product or warehouse that appears in the movement ledger cannot be
--    deleted out from under it. Same policy as 000019 took for payroll and
--    ledger history. Only the *_items children cascade from their own order.
-- ============================================================================

-- ============================================
-- PRODUCT CATEGORIES
-- ============================================
CREATE TABLE product_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,

    -- Self-referencing hierarchy. RESTRICT: deleting a parent that still has
    -- children would orphan them.
    parent_id UUID REFERENCES product_categories(id) ON DELETE RESTRICT,
    level INTEGER NOT NULL DEFAULT 1,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_product_categories_level CHECK (level >= 1),
    CONSTRAINT chk_product_categories_not_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);

-- Partial: a code freed by a soft delete may be reused.
CREATE UNIQUE INDEX uq_product_categories_code
    ON product_categories(company_id, code) WHERE deleted_at IS NULL;

-- The partial unique index above cannot serve the RLS predicate on its own
-- (a planner may only use it for queries that also assert deleted_at IS NULL),
-- and neither can it serve the companies-delete cascade check. Same reasoning
-- 000020 used to keep idx_companies_business_number.
CREATE INDEX idx_product_categories_company ON product_categories(company_id);
CREATE INDEX idx_product_categories_parent ON product_categories(parent_id) WHERE parent_id IS NOT NULL;

COMMENT ON TABLE product_categories IS 'Product classification tree (품목 분류)';

-- ============================================
-- WAREHOUSES
-- ============================================
CREATE TABLE warehouses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,

    address VARCHAR(200),
    manager VARCHAR(50),
    phone VARCHAR(20),

    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_warehouses_code
    ON warehouses(company_id, code) WHERE deleted_at IS NULL;

-- At most one default warehouse per tenant. This is the partial-uniqueness case
-- that a table-level UNIQUE cannot express.
CREATE UNIQUE INDEX uq_warehouses_one_default
    ON warehouses(company_id) WHERE is_default AND deleted_at IS NULL;

CREATE INDEX idx_warehouses_company ON warehouses(company_id);

COMMENT ON TABLE warehouses IS 'Stock locations (창고)';
COMMENT ON INDEX uq_warehouses_one_default IS 'At most one is_default warehouse per company among the live rows';

-- ============================================
-- PRODUCTS
-- ============================================
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    code VARCHAR(50) NOT NULL,
    name VARCHAR(200) NOT NULL,

    category_id UUID REFERENCES product_categories(id) ON DELETE RESTRICT,

    specification VARCHAR(200),
    unit VARCHAR(20) NOT NULL DEFAULT 'EA',

    -- Sub-won unit prices are allowed; booked amounts are not (see header note 3).
    unit_price NUMERIC(18, 2) NOT NULL DEFAULT 0,
    cost_price NUMERIC(18, 2) NOT NULL DEFAULT 0,

    -- Reorder thresholds. NULL means "no threshold", which is not the same as 0.
    min_stock NUMERIC(18, 3),
    max_stock NUMERIC(18, 3),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,
    barcode VARCHAR(50),
    image_url VARCHAR(500),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_products_prices CHECK (unit_price >= 0 AND cost_price >= 0),
    CONSTRAINT chk_products_min_stock CHECK (min_stock IS NULL OR min_stock >= 0),
    CONSTRAINT chk_products_max_stock CHECK (max_stock IS NULL OR max_stock >= 0),
    CONSTRAINT chk_products_stock_range CHECK (
        min_stock IS NULL OR max_stock IS NULL OR max_stock >= min_stock
    )
);

CREATE UNIQUE INDEX uq_products_code
    ON products(company_id, code) WHERE deleted_at IS NULL;

-- A barcode identifies one physical article, so it must be unique where it is
-- present - but most rows have none, and NULLs would collide in a plain UNIQUE
-- only by accident of the standard. Stated explicitly instead.
CREATE UNIQUE INDEX uq_products_barcode
    ON products(company_id, barcode) WHERE barcode IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX idx_products_company ON products(company_id);
CREATE INDEX idx_products_category ON products(company_id, category_id) WHERE category_id IS NOT NULL;
CREATE INDEX idx_products_name ON products(company_id, name);
CREATE INDEX idx_products_active ON products(company_id) WHERE is_active AND deleted_at IS NULL;

COMMENT ON TABLE products IS 'Inventory items (품목)';
COMMENT ON COLUMN products.min_stock IS 'Reorder point; NULL disables the low-stock alert for this item';

-- ============================================
-- STOCKS (cached on-hand balance per product per warehouse)
-- ============================================
CREATE TABLE stocks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,

    quantity NUMERIC(18, 3) NOT NULL DEFAULT 0,
    reserved_quantity NUMERIC(18, 3) NOT NULL DEFAULT 0,

    -- Derived, never written. Keeping it in the table (rather than computing it
    -- in Go) means the invariant holds for every writer, and the frontend's
    -- Stock.availableQuantity is read straight out of the row.
    available_quantity NUMERIC(18, 3) GENERATED ALWAYS AS (quantity - reserved_quantity) STORED,

    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Negative stock is a policy decision, taken here: not allowed. See header.
    CONSTRAINT chk_stocks_quantity_non_negative CHECK (quantity >= 0),
    CONSTRAINT chk_stocks_reserved_non_negative CHECK (reserved_quantity >= 0),
    CONSTRAINT chk_stocks_reserved_within_quantity CHECK (reserved_quantity <= quantity)
);

-- One balance row per (tenant, product, warehouse). This is also the conflict
-- target of the upsert the service uses to create a balance on first receipt.
-- Not partial: stocks are never soft deleted.
ALTER TABLE stocks ADD CONSTRAINT uq_stocks_product_warehouse
    UNIQUE (company_id, product_id, warehouse_id);

CREATE INDEX idx_stocks_warehouse ON stocks(company_id, warehouse_id);
CREATE INDEX idx_stocks_product ON stocks(product_id);

COMMENT ON TABLE stocks IS 'Cached on-hand balance; stock_movements is the ledger of record';
COMMENT ON COLUMN stocks.available_quantity IS 'quantity - reserved_quantity, maintained by PostgreSQL';

-- ============================================
-- STOCK MOVEMENTS (append-only ledger)
-- ============================================
CREATE TABLE stock_movements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,

    movement_type VARCHAR(20) NOT NULL,

    -- Always positive. The direction lives in movement_type, so a sign error in
    -- one caller cannot silently invert a receipt into an issue.
    quantity NUMERIC(18, 3) NOT NULL,
    previous_quantity NUMERIC(18, 3) NOT NULL,
    current_quantity NUMERIC(18, 3) NOT NULL,

    -- Free-form link back to whatever caused the movement
    -- ('purchase_order', 'sales_order', 'stock_transfer', ...). Deliberately not
    -- a foreign key: the referent is polymorphic.
    reference_type VARCHAR(30),
    reference_id UUID,

    note TEXT,

    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_stock_movements_type CHECK (movement_type IN (
        'purchase_in', 'sales_out',
        'adjustment_in', 'adjustment_out',
        'transfer_in', 'transfer_out',
        'return_in', 'return_out'
    )),
    CONSTRAINT chk_stock_movements_quantity_positive CHECK (quantity > 0),
    CONSTRAINT chk_stock_movements_balances CHECK (
        previous_quantity >= 0 AND current_quantity >= 0
    ),
    -- The row must agree with its own arithmetic: an inbound movement raises the
    -- balance by exactly `quantity`, an outbound one lowers it by exactly that.
    CONSTRAINT chk_stock_movements_delta CHECK (
        CASE
            WHEN movement_type IN ('purchase_in', 'adjustment_in', 'transfer_in', 'return_in')
                THEN current_quantity = previous_quantity + quantity
            ELSE current_quantity = previous_quantity - quantity
        END
    )
);

CREATE INDEX idx_stock_movements_company_created ON stock_movements(company_id, created_at DESC);
CREATE INDEX idx_stock_movements_product ON stock_movements(company_id, product_id, created_at DESC);
CREATE INDEX idx_stock_movements_warehouse ON stock_movements(company_id, warehouse_id, created_at DESC);
CREATE INDEX idx_stock_movements_reference ON stock_movements(company_id, reference_type, reference_id)
    WHERE reference_id IS NOT NULL;

COMMENT ON TABLE stock_movements IS 'Append-only stock ledger (입출고 이력). Rows are never updated or deleted.';

-- ============================================
-- ORDER NUMBER SEQUENCES
-- ============================================
-- Same shape and the same reason as voucher_sequences: the number is reserved
-- with a single INSERT ... ON CONFLICT DO UPDATE ... RETURNING, so two
-- concurrent orders cannot reserve the same one and no caller can push the
-- counter backwards.
CREATE TABLE order_sequences (
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    order_type VARCHAR(20) NOT NULL,
    fiscal_year INTEGER NOT NULL,
    prefix VARCHAR(10) NOT NULL,
    last_number INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (company_id, order_type, fiscal_year),
    CONSTRAINT chk_order_sequences_type CHECK (order_type IN ('purchase', 'sales')),
    CONSTRAINT chk_order_sequences_last_number CHECK (last_number >= 0)
);

COMMENT ON TABLE order_sequences IS 'Per-tenant, per-year counters for purchase and sales order numbers';

-- ============================================
-- PURCHASE ORDERS
-- ============================================
CREATE TABLE purchase_orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    order_number VARCHAR(30) NOT NULL,
    order_date DATE NOT NULL,
    expected_date DATE,

    supplier_id UUID NOT NULL REFERENCES partners(id) ON DELETE RESTRICT,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,

    total_amount NUMERIC(18, 0) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(18, 0) NOT NULL DEFAULT 0,
    grand_total NUMERIC(18, 0) NOT NULL DEFAULT 0,

    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    note TEXT,

    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approved_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    approved_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_purchase_orders_status CHECK (status IN (
        'draft', 'pending', 'approved', 'ordered', 'partial', 'completed', 'cancelled'
    )),
    CONSTRAINT chk_purchase_orders_amounts CHECK (
        total_amount >= 0 AND tax_amount >= 0 AND grand_total = total_amount + tax_amount
    ),
    CONSTRAINT chk_purchase_orders_dates CHECK (
        expected_date IS NULL OR expected_date >= order_date
    ),
    -- An approved order must say who approved it and when. This is the
    -- database's half of the state-transition guard in the service layer.
    CONSTRAINT chk_purchase_orders_approval CHECK (
        (status IN ('draft', 'pending', 'cancelled'))
        OR (approved_by IS NOT NULL AND approved_at IS NOT NULL)
    )
);

ALTER TABLE purchase_orders ADD CONSTRAINT uq_purchase_orders_number
    UNIQUE (company_id, order_number);

CREATE INDEX idx_purchase_orders_company_date ON purchase_orders(company_id, order_date DESC);
CREATE INDEX idx_purchase_orders_status ON purchase_orders(company_id, status);
CREATE INDEX idx_purchase_orders_supplier ON purchase_orders(company_id, supplier_id);
CREATE INDEX idx_purchase_orders_warehouse ON purchase_orders(company_id, warehouse_id);

COMMENT ON TABLE purchase_orders IS 'Purchase orders (발주서)';

CREATE TABLE purchase_order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    line_no INTEGER NOT NULL,

    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,

    quantity NUMERIC(18, 3) NOT NULL,
    unit_price NUMERIC(18, 2) NOT NULL,
    amount NUMERIC(18, 0) NOT NULL,
    tax_amount NUMERIC(18, 0) NOT NULL DEFAULT 0,
    received_quantity NUMERIC(18, 3) NOT NULL DEFAULT 0,

    note TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_purchase_order_items_quantity CHECK (quantity > 0),
    CONSTRAINT chk_purchase_order_items_amounts CHECK (
        unit_price >= 0 AND amount >= 0 AND tax_amount >= 0
    ),
    -- Over-receipt is what turns a "partial" order into a permanently
    -- inconsistent one, and it is the shape a double-clicked receive button
    -- takes. Rejected here as well as in the service.
    CONSTRAINT chk_purchase_order_items_received CHECK (
        received_quantity >= 0 AND received_quantity <= quantity
    ),
    CONSTRAINT chk_purchase_order_items_line_no CHECK (line_no >= 1)
);

ALTER TABLE purchase_order_items ADD CONSTRAINT uq_purchase_order_items_line
    UNIQUE (purchase_order_id, line_no);

CREATE INDEX idx_purchase_order_items_company ON purchase_order_items(company_id);
CREATE INDEX idx_purchase_order_items_product ON purchase_order_items(company_id, product_id);

COMMENT ON TABLE purchase_order_items IS 'Purchase order lines (발주 명세)';

-- ============================================
-- SALES ORDERS
-- ============================================
CREATE TABLE sales_orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    order_number VARCHAR(30) NOT NULL,
    order_date DATE NOT NULL,
    expected_date DATE,

    customer_id UUID NOT NULL REFERENCES partners(id) ON DELETE RESTRICT,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,

    total_amount NUMERIC(18, 0) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(18, 0) NOT NULL DEFAULT 0,
    grand_total NUMERIC(18, 0) NOT NULL DEFAULT 0,

    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    note TEXT,

    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approved_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    approved_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_sales_orders_status CHECK (status IN (
        'draft', 'pending', 'approved', 'confirmed', 'partial', 'completed', 'cancelled'
    )),
    CONSTRAINT chk_sales_orders_amounts CHECK (
        total_amount >= 0 AND tax_amount >= 0 AND grand_total = total_amount + tax_amount
    ),
    CONSTRAINT chk_sales_orders_dates CHECK (
        expected_date IS NULL OR expected_date >= order_date
    ),
    CONSTRAINT chk_sales_orders_approval CHECK (
        (status IN ('draft', 'pending', 'cancelled'))
        OR (approved_by IS NOT NULL AND approved_at IS NOT NULL)
    )
);

ALTER TABLE sales_orders ADD CONSTRAINT uq_sales_orders_number
    UNIQUE (company_id, order_number);

CREATE INDEX idx_sales_orders_company_date ON sales_orders(company_id, order_date DESC);
CREATE INDEX idx_sales_orders_status ON sales_orders(company_id, status);
CREATE INDEX idx_sales_orders_customer ON sales_orders(company_id, customer_id);
CREATE INDEX idx_sales_orders_warehouse ON sales_orders(company_id, warehouse_id);

COMMENT ON TABLE sales_orders IS 'Sales orders (수주)';

CREATE TABLE sales_order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    sales_order_id UUID NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    line_no INTEGER NOT NULL,

    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,

    quantity NUMERIC(18, 3) NOT NULL,
    unit_price NUMERIC(18, 2) NOT NULL,
    amount NUMERIC(18, 0) NOT NULL,
    tax_amount NUMERIC(18, 0) NOT NULL DEFAULT 0,
    shipped_quantity NUMERIC(18, 3) NOT NULL DEFAULT 0,

    note TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_sales_order_items_quantity CHECK (quantity > 0),
    CONSTRAINT chk_sales_order_items_amounts CHECK (
        unit_price >= 0 AND amount >= 0 AND tax_amount >= 0
    ),
    CONSTRAINT chk_sales_order_items_shipped CHECK (
        shipped_quantity >= 0 AND shipped_quantity <= quantity
    ),
    CONSTRAINT chk_sales_order_items_line_no CHECK (line_no >= 1)
);

ALTER TABLE sales_order_items ADD CONSTRAINT uq_sales_order_items_line
    UNIQUE (sales_order_id, line_no);

CREATE INDEX idx_sales_order_items_company ON sales_order_items(company_id);
CREATE INDEX idx_sales_order_items_product ON sales_order_items(company_id, product_id);

COMMENT ON TABLE sales_order_items IS 'Sales order lines (수주 명세)';

-- ============================================
-- updated_at TRIGGERS
-- ============================================
-- trigger_set_updated_at() is created by 000003 and is NOT dropped by this
-- migration's rollback: it is 000003's object.
CREATE TRIGGER set_product_categories_updated_at BEFORE UPDATE ON product_categories
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_warehouses_updated_at BEFORE UPDATE ON warehouses
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_products_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_stocks_updated_at BEFORE UPDATE ON stocks
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_purchase_orders_updated_at BEFORE UPDATE ON purchase_orders
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_purchase_order_items_updated_at BEFORE UPDATE ON purchase_order_items
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_sales_orders_updated_at BEFORE UPDATE ON sales_orders
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_sales_order_items_updated_at BEFORE UPDATE ON sales_order_items
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- stock_movements gets no updated_at trigger and no updated_at column: it is
-- append-only.

-- ============================================
-- ROW LEVEL SECURITY
-- ============================================
-- The canonical GUC is app.current_tenant, read through current_tenant_id()
-- (000010, hardened by 000017). FORCE is mandatory: without it the table owner
-- - which is who the application connects as - ignores every policy.
--
-- Each policy states WITH CHECK explicitly as well as USING. For a FOR ALL
-- policy PostgreSQL would default WITH CHECK to the USING expression, but
-- writing it out is what makes "an INSERT for another tenant is rejected"
-- a property you can read off the schema instead of infer.

ALTER TABLE product_categories  ENABLE ROW LEVEL SECURITY;
ALTER TABLE warehouses          ENABLE ROW LEVEL SECURITY;
ALTER TABLE products            ENABLE ROW LEVEL SECURITY;
ALTER TABLE stocks              ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_movements     ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_sequences     ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_orders     ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_orders        ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_order_items   ENABLE ROW LEVEL SECURITY;

ALTER TABLE product_categories  FORCE ROW LEVEL SECURITY;
ALTER TABLE warehouses          FORCE ROW LEVEL SECURITY;
ALTER TABLE products            FORCE ROW LEVEL SECURITY;
ALTER TABLE stocks              FORCE ROW LEVEL SECURITY;
ALTER TABLE stock_movements     FORCE ROW LEVEL SECURITY;
ALTER TABLE order_sequences     FORCE ROW LEVEL SECURITY;
ALTER TABLE purchase_orders     FORCE ROW LEVEL SECURITY;
ALTER TABLE purchase_order_items FORCE ROW LEVEL SECURITY;
ALTER TABLE sales_orders        FORCE ROW LEVEL SECURITY;
ALTER TABLE sales_order_items   FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_product_categories ON product_categories
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context())
    WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_isolation_warehouses ON warehouses
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context())
    WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_isolation_products ON products
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context())
    WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_isolation_stocks ON stocks
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context())
    WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_isolation_stock_movements ON stock_movements
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context())
    WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_isolation_order_sequences ON order_sequences
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context())
    WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_isolation_purchase_orders ON purchase_orders
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context())
    WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_isolation_purchase_order_items ON purchase_order_items
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context())
    WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_isolation_sales_orders ON sales_orders
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context())
    WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

CREATE POLICY tenant_isolation_sales_order_items ON sales_order_items
    FOR ALL USING (company_id = current_tenant_id() OR is_admin_context())
    WITH CHECK (company_id = current_tenant_id() OR is_admin_context());

-- ============================================
-- GRANTS TO THE APPLICATION ROLES
-- ============================================
-- 000016 set ALTER DEFAULT PRIVILEGES for the role that ran it, so tables
-- created by that same role are granted automatically. This block does not rely
-- on that: a migration applied by a different role (a fresh test database, a
-- deploy account) would otherwise leave kerp_app unable to touch any of the ten
-- tables above, and the failure would surface as a runtime 500, not here.
DO $$
DECLARE
    tgt TEXT := current_schema();
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kerp_app') THEN
        EXECUTE format(
            'GRANT SELECT, INSERT, UPDATE, DELETE ON '
            || '%1$I.product_categories, %1$I.warehouses, %1$I.products, '
            || '%1$I.stocks, %1$I.stock_movements, %1$I.order_sequences, '
            || '%1$I.purchase_orders, %1$I.purchase_order_items, '
            || '%1$I.sales_orders, %1$I.sales_order_items TO kerp_app', tgt);
    END IF;

    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kerp_admin') THEN
        EXECUTE format(
            'GRANT SELECT, INSERT, UPDATE, DELETE ON '
            || '%1$I.product_categories, %1$I.warehouses, %1$I.products, '
            || '%1$I.stocks, %1$I.stock_movements, %1$I.order_sequences, '
            || '%1$I.purchase_orders, %1$I.purchase_order_items, '
            || '%1$I.sales_orders, %1$I.sales_order_items TO kerp_admin', tgt);
    END IF;
END
$$;
