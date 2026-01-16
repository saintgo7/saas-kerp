-- name: GetVoucherByID :one
SELECT * FROM vouchers
WHERE id = $1 AND company_id = $2;

-- name: GetVoucherByNo :one
SELECT * FROM vouchers
WHERE company_id = $1 AND voucher_no = $2;

-- name: GetVoucherWithEntries :one
SELECT
    v.*,
    COALESCE(
        json_agg(
            json_build_object(
                'id', e.id,
                'line_no', e.line_no,
                'account_id', e.account_id,
                'account_code', a.code,
                'account_name', a.name,
                'debit_amount', e.debit_amount,
                'credit_amount', e.credit_amount,
                'description', e.description,
                'partner_id', e.partner_id,
                'department_id', e.department_id,
                'project_id', e.project_id
            ) ORDER BY e.line_no
        ) FILTER (WHERE e.id IS NOT NULL),
        '[]'
    ) AS entries
FROM vouchers v
LEFT JOIN voucher_entries e ON v.id = e.voucher_id
LEFT JOIN accounts a ON e.account_id = a.id
WHERE v.id = $1 AND v.company_id = $2
GROUP BY v.id;

-- name: ListVouchers :many
SELECT * FROM vouchers
WHERE company_id = $1
    AND voucher_date BETWEEN $2 AND $3
    AND status = COALESCE(sqlc.narg('status'), status)
    AND voucher_type = COALESCE(sqlc.narg('voucher_type'), voucher_type)
ORDER BY voucher_date DESC, voucher_no DESC
LIMIT $4 OFFSET $5;

-- name: ListVouchersPendingApproval :many
SELECT * FROM vouchers
WHERE company_id = $1
    AND status = 'pending'
ORDER BY created_at ASC
LIMIT $2 OFFSET $3;

-- name: CreateVoucher :one
INSERT INTO vouchers (
    company_id,
    voucher_no,
    voucher_date,
    voucher_type,
    status,
    total_debit,
    total_credit,
    description,
    reference_type,
    reference_id,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: UpdateVoucher :one
UPDATE vouchers SET
    voucher_date = COALESCE(sqlc.narg('voucher_date'), voucher_date),
    description = COALESCE(sqlc.narg('description'), description),
    updated_at = NOW(),
    updated_by = $2
WHERE id = $1 AND status = 'draft'
RETURNING *;

-- name: UpdateVoucherTotals :exec
UPDATE vouchers SET
    total_debit = $2,
    total_credit = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: SubmitVoucher :exec
UPDATE vouchers SET
    status = 'pending',
    submitted_at = NOW(),
    submitted_by = $2,
    updated_at = NOW()
WHERE id = $1 AND status = 'draft';

-- name: ApproveVoucher :exec
UPDATE vouchers SET
    status = 'approved',
    approved_at = NOW(),
    approved_by = $2,
    updated_at = NOW()
WHERE id = $1 AND status = 'pending';

-- name: RejectVoucher :exec
UPDATE vouchers SET
    status = 'rejected',
    rejected_at = NOW(),
    rejected_by = $2,
    rejection_reason = $3,
    updated_at = NOW()
WHERE id = $1 AND status = 'pending';

-- name: PostVoucher :exec
UPDATE vouchers SET
    status = 'posted',
    posted_at = NOW(),
    posted_by = $2,
    updated_at = NOW()
WHERE id = $1 AND status = 'approved';

-- name: CancelVoucher :exec
UPDATE vouchers SET
    status = 'cancelled',
    updated_at = NOW(),
    updated_by = $2
WHERE id = $1 AND status IN ('draft', 'pending');

-- name: GenerateVoucherNumber :one
SELECT generate_voucher_number($1, $2, $3);

-- name: CountVouchersByPeriod :one
SELECT COUNT(*) FROM vouchers
WHERE company_id = $1
    AND voucher_date BETWEEN $2 AND $3
    AND status = COALESCE(sqlc.narg('status'), status);

-- Voucher Entries

-- name: CreateVoucherEntry :one
INSERT INTO voucher_entries (
    voucher_id,
    company_id,
    line_no,
    account_id,
    debit_amount,
    credit_amount,
    description,
    partner_id,
    department_id,
    project_id,
    cost_center_id,
    tags
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: ListVoucherEntries :many
SELECT
    e.*,
    a.code AS account_code,
    a.name AS account_name
FROM voucher_entries e
JOIN accounts a ON e.account_id = a.id
WHERE e.voucher_id = $1
ORDER BY e.line_no;

-- name: DeleteVoucherEntries :exec
DELETE FROM voucher_entries
WHERE voucher_id = $1;

-- name: GetNextLineNo :one
SELECT COALESCE(MAX(line_no), 0) + 1 AS next_line_no
FROM voucher_entries
WHERE voucher_id = $1;
