-- name: CreateTaxInvoice :one
INSERT INTO tax_invoices (
    company_id, invoice_number, invoice_type, issue_date, status,
    supplier_business_number, supplier_name, supplier_ceo_name,
    supplier_address, supplier_business_type, supplier_business_item, supplier_email,
    buyer_business_number, buyer_name, buyer_ceo_name,
    buyer_address, buyer_business_type, buyer_business_item, buyer_email,
    supply_amount, tax_amount, total_amount,
    remarks, created_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10, $11, $12,
    $13, $14, $15, $16, $17, $18, $19,
    $20, $21, $22, $23, $24
) RETURNING *;

-- name: GetTaxInvoice :one
SELECT * FROM tax_invoices
WHERE id = $1 AND company_id = $2;

-- name: GetTaxInvoiceByNumber :one
SELECT * FROM tax_invoices
WHERE company_id = $1 AND invoice_number = $2 AND invoice_type = $3;

-- name: ListTaxInvoices :many
SELECT * FROM tax_invoices
WHERE company_id = $1
  AND ($2::DATE IS NULL OR issue_date >= $2)
  AND ($3::DATE IS NULL OR issue_date <= $3)
  AND ($4::VARCHAR IS NULL OR invoice_type = $4)
  AND ($5::VARCHAR IS NULL OR status = $5)
ORDER BY issue_date DESC, created_at DESC
LIMIT $6 OFFSET $7;

-- name: CountTaxInvoices :one
SELECT COUNT(*) FROM tax_invoices
WHERE company_id = $1
  AND ($2::DATE IS NULL OR issue_date >= $2)
  AND ($3::DATE IS NULL OR issue_date <= $3)
  AND ($4::VARCHAR IS NULL OR invoice_type = $4)
  AND ($5::VARCHAR IS NULL OR status = $5);

-- name: UpdateTaxInvoiceStatus :one
UPDATE tax_invoices
SET status = $3,
    nts_confirm_number = COALESCE($4, nts_confirm_number),
    nts_transmitted_at = COALESCE($5, nts_transmitted_at),
    nts_confirmed_at = COALESCE($6, nts_confirmed_at),
    updated_by = $7,
    updated_at = NOW()
WHERE id = $1 AND company_id = $2
RETURNING *;

-- name: UpdateTaxInvoiceASPInfo :one
UPDATE tax_invoices
SET asp_provider = $3,
    asp_invoice_id = $4,
    asp_response = $5,
    updated_by = $6,
    updated_at = NOW()
WHERE id = $1 AND company_id = $2
RETURNING *;

-- name: LinkTaxInvoiceToVoucher :one
UPDATE tax_invoices
SET voucher_id = $3,
    updated_by = $4,
    updated_at = NOW()
WHERE id = $1 AND company_id = $2
RETURNING *;

-- name: DeleteTaxInvoice :exec
DELETE FROM tax_invoices
WHERE id = $1 AND company_id = $2 AND status = 'draft';

-- name: CreateTaxInvoiceItem :one
INSERT INTO tax_invoice_items (
    tax_invoice_id, company_id, sequence_number, supply_date,
    description, specification, quantity, unit_price, amount, tax_amount, remarks
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: ListTaxInvoiceItems :many
SELECT * FROM tax_invoice_items
WHERE tax_invoice_id = $1 AND company_id = $2
ORDER BY sequence_number;

-- name: DeleteTaxInvoiceItems :exec
DELETE FROM tax_invoice_items
WHERE tax_invoice_id = $1 AND company_id = $2;

-- name: CreateTaxInvoiceHistory :one
INSERT INTO tax_invoice_history (
    tax_invoice_id, company_id, previous_status, new_status,
    changed_by, change_reason, api_response
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: ListTaxInvoiceHistory :many
SELECT * FROM tax_invoice_history
WHERE tax_invoice_id = $1 AND company_id = $2
ORDER BY created_at DESC;

-- name: CreateHometaxSession :one
INSERT INTO hometax_sessions (
    company_id, session_id, business_number, auth_type, expires_at, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetActiveHometaxSession :one
SELECT * FROM hometax_sessions
WHERE company_id = $1 AND is_active = true AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateHometaxSessionActivity :exec
UPDATE hometax_sessions
SET last_used_at = NOW()
WHERE session_id = $1 AND company_id = $2;

-- name: DeactivateHometaxSession :exec
UPDATE hometax_sessions
SET is_active = false
WHERE session_id = $1 AND company_id = $2;

-- name: GetPopbillConfig :one
SELECT * FROM popbill_configs
WHERE company_id = $1 AND is_active = true;

-- name: UpsertPopbillConfig :one
INSERT INTO popbill_configs (
    company_id, link_id, secret_key_encrypted, is_sandbox, monthly_quota, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (company_id) DO UPDATE SET
    link_id = EXCLUDED.link_id,
    secret_key_encrypted = EXCLUDED.secret_key_encrypted,
    is_sandbox = EXCLUDED.is_sandbox,
    monthly_quota = EXCLUDED.monthly_quota,
    updated_at = NOW()
RETURNING *;

-- name: IncrementPopbillUsage :exec
UPDATE popbill_configs
SET monthly_used = monthly_used + 1
WHERE company_id = $1;

-- name: ResetPopbillMonthlyUsage :exec
UPDATE popbill_configs
SET monthly_used = 0, quota_reset_at = $2
WHERE company_id = $1;

-- name: GetTaxInvoiceSummary :one
SELECT
    COUNT(*) FILTER (WHERE invoice_type = 'sales') AS sales_count,
    COUNT(*) FILTER (WHERE invoice_type = 'purchase') AS purchase_count,
    COALESCE(SUM(supply_amount) FILTER (WHERE invoice_type = 'sales'), 0) AS sales_supply_total,
    COALESCE(SUM(tax_amount) FILTER (WHERE invoice_type = 'sales'), 0) AS sales_tax_total,
    COALESCE(SUM(supply_amount) FILTER (WHERE invoice_type = 'purchase'), 0) AS purchase_supply_total,
    COALESCE(SUM(tax_amount) FILTER (WHERE invoice_type = 'purchase'), 0) AS purchase_tax_total
FROM tax_invoices
WHERE company_id = $1
  AND issue_date >= $2
  AND issue_date <= $3
  AND status NOT IN ('draft', 'cancelled');
