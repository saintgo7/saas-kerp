-- name: GetInvoiceByID :one
SELECT * FROM invoices
WHERE id = $1 AND company_id = $2;

-- name: GetInvoiceByNo :one
SELECT * FROM invoices
WHERE company_id = $1 AND invoice_no = $2;

-- name: GetInvoiceByNTSConfirm :one
SELECT * FROM invoices
WHERE nts_confirm_num = $1;

-- name: GetInvoiceWithItems :one
SELECT
    i.*,
    COALESCE(
        json_agg(
            json_build_object(
                'id', it.id,
                'line_no', it.line_no,
                'item_date', it.item_date,
                'description', it.description,
                'specification', it.specification,
                'quantity', it.quantity,
                'unit_price', it.unit_price,
                'supply_amount', it.supply_amount,
                'tax_amount', it.tax_amount,
                'notes', it.notes
            ) ORDER BY it.line_no
        ) FILTER (WHERE it.id IS NOT NULL),
        '[]'
    ) AS items
FROM invoices i
LEFT JOIN invoice_items it ON i.id = it.invoice_id
WHERE i.id = $1 AND i.company_id = $2
GROUP BY i.id;

-- name: ListInvoices :many
SELECT * FROM invoices
WHERE company_id = $1
    AND issue_date BETWEEN $2 AND $3
    AND invoice_type = COALESCE(sqlc.narg('invoice_type'), invoice_type)
    AND status = COALESCE(sqlc.narg('status'), status)
ORDER BY issue_date DESC, invoice_no DESC
LIMIT $4 OFFSET $5;

-- name: ListInvoicesByPartner :many
SELECT * FROM invoices
WHERE company_id = $1
    AND (supplier_business_number = $2 OR buyer_business_number = $2)
    AND issue_date BETWEEN $3 AND $4
ORDER BY issue_date DESC
LIMIT $5 OFFSET $6;

-- name: CreateInvoice :one
INSERT INTO invoices (
    company_id,
    invoice_no,
    invoice_type,
    issue_type,
    issue_date,
    status,
    supplier_business_number,
    supplier_sub_number,
    supplier_company_name,
    supplier_representative,
    supplier_address,
    supplier_business_type,
    supplier_business_item,
    supplier_email,
    buyer_business_number,
    buyer_sub_number,
    buyer_company_name,
    buyer_representative,
    buyer_address,
    buyer_business_type,
    buyer_business_item,
    buyer_email,
    supply_amount,
    tax_amount,
    total_amount,
    tax_type,
    notes,
    partner_id,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
    $21, $22, $23, $24, $25, $26, $27, $28, $29
)
RETURNING *;

-- name: UpdateInvoiceStatus :exec
UPDATE invoices SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateInvoiceNTSResult :exec
UPDATE invoices SET
    nts_confirm_num = $2,
    nts_issue_id = $3,
    nts_result_code = $4,
    nts_result_message = $5,
    nts_submitted_at = NOW(),
    status = CASE WHEN $4 = 'SUC' THEN 'issued' ELSE status END,
    updated_at = NOW()
WHERE id = $1;

-- name: LinkInvoiceToVoucher :exec
UPDATE invoices SET
    voucher_id = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: GenerateInvoiceNumber :one
SELECT generate_invoice_number($1, $2, $3);

-- name: CreateInvoiceItem :one
INSERT INTO invoice_items (
    invoice_id,
    company_id,
    line_no,
    item_date,
    description,
    specification,
    unit,
    quantity,
    unit_price,
    supply_amount,
    tax_amount,
    notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: ListInvoiceItems :many
SELECT * FROM invoice_items
WHERE invoice_id = $1
ORDER BY line_no;

-- name: DeleteInvoiceItems :exec
DELETE FROM invoice_items
WHERE invoice_id = $1;

-- name: GetVATSummary :one
SELECT
    COUNT(*) AS total_count,
    COALESCE(SUM(supply_amount), 0) AS total_supply,
    COALESCE(SUM(tax_amount), 0) AS total_tax,
    COALESCE(SUM(total_amount), 0) AS total_amount
FROM invoices
WHERE company_id = $1
    AND invoice_type = $2
    AND issue_date BETWEEN $3 AND $4
    AND status IN ('issued', 'sent', 'received');

-- name: GetVATReport :many
SELECT
    DATE_TRUNC('month', issue_date) AS period,
    invoice_type,
    COUNT(*) AS invoice_count,
    SUM(supply_amount) AS supply_total,
    SUM(tax_amount) AS tax_total
FROM invoices
WHERE company_id = $1
    AND issue_date BETWEEN $2 AND $3
    AND status IN ('issued', 'sent', 'received')
GROUP BY DATE_TRUNC('month', issue_date), invoice_type
ORDER BY period, invoice_type;
