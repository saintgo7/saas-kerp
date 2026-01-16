-- name: GetLedgerBalance :one
SELECT * FROM ledger_balances
WHERE company_id = $1
    AND account_id = $2
    AND fiscal_year = $3
    AND fiscal_month = $4;

-- name: ListLedgerBalances :many
SELECT
    lb.*,
    a.code AS account_code,
    a.name AS account_name,
    a.account_type,
    a.account_nature
FROM ledger_balances lb
JOIN accounts a ON lb.account_id = a.id
WHERE lb.company_id = $1
    AND lb.fiscal_year = $2
    AND lb.fiscal_month = $3
ORDER BY a.code;

-- name: GetTrialBalance :many
SELECT
    a.id AS account_id,
    a.code,
    a.name,
    a.account_type,
    a.account_nature,
    COALESCE(lb.closing_debit, 0) AS debit_balance,
    COALESCE(lb.closing_credit, 0) AS credit_balance,
    COALESCE(lb.balance, 0) AS balance
FROM accounts a
LEFT JOIN ledger_balances lb ON a.id = lb.account_id
    AND lb.fiscal_year = $2
    AND lb.fiscal_month = $3
WHERE a.company_id = $1
    AND a.is_active = TRUE
    AND (lb.id IS NOT NULL OR COALESCE(sqlc.narg('include_zero'), FALSE) = TRUE)
ORDER BY a.code;

-- name: GetAccountLedger :many
SELECT
    v.id AS voucher_id,
    v.voucher_no,
    v.voucher_date,
    v.voucher_type,
    v.description AS voucher_description,
    e.line_no,
    e.debit_amount,
    e.credit_amount,
    e.description AS entry_description,
    SUM(
        CASE WHEN v2.voucher_date < v.voucher_date
            OR (v2.voucher_date = v.voucher_date AND v2.voucher_no < v.voucher_no)
            OR (v2.voucher_date = v.voucher_date AND v2.voucher_no = v.voucher_no AND e2.line_no < e.line_no)
        THEN e2.debit_amount - e2.credit_amount
        ELSE 0
        END
    ) OVER (ORDER BY v.voucher_date, v.voucher_no, e.line_no) AS running_balance
FROM voucher_entries e
JOIN vouchers v ON e.voucher_id = v.id
LEFT JOIN voucher_entries e2 ON e2.account_id = e.account_id
LEFT JOIN vouchers v2 ON e2.voucher_id = v2.id AND v2.status = 'posted'
WHERE e.account_id = $1
    AND e.company_id = $2
    AND v.status = 'posted'
    AND v.voucher_date BETWEEN $3 AND $4
GROUP BY v.id, v.voucher_no, v.voucher_date, v.voucher_type, v.description,
         e.id, e.line_no, e.debit_amount, e.credit_amount, e.description
ORDER BY v.voucher_date, v.voucher_no, e.line_no;

-- name: UpsertLedgerBalance :exec
INSERT INTO ledger_balances (
    company_id,
    account_id,
    fiscal_year,
    fiscal_month,
    opening_debit,
    opening_credit,
    period_debit,
    period_credit,
    closing_debit,
    closing_credit
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
ON CONFLICT (company_id, account_id, fiscal_year, fiscal_month)
DO UPDATE SET
    opening_debit = EXCLUDED.opening_debit,
    opening_credit = EXCLUDED.opening_credit,
    period_debit = EXCLUDED.period_debit,
    period_credit = EXCLUDED.period_credit,
    closing_debit = EXCLUDED.closing_debit,
    closing_credit = EXCLUDED.closing_credit,
    updated_at = NOW();

-- name: CalculatePeriodTotals :one
SELECT
    COALESCE(SUM(e.debit_amount), 0) AS total_debit,
    COALESCE(SUM(e.credit_amount), 0) AS total_credit
FROM voucher_entries e
JOIN vouchers v ON e.voucher_id = v.id
WHERE e.company_id = $1
    AND e.account_id = $2
    AND v.status = 'posted'
    AND v.voucher_date BETWEEN $3 AND $4;

-- name: GetIncomeStatement :many
SELECT
    a.account_type,
    a.code,
    a.name,
    COALESCE(SUM(
        CASE WHEN a.account_nature = 'debit'
            THEN e.debit_amount - e.credit_amount
            ELSE e.credit_amount - e.debit_amount
        END
    ), 0) AS amount
FROM accounts a
LEFT JOIN voucher_entries e ON a.id = e.account_id
LEFT JOIN vouchers v ON e.voucher_id = v.id
    AND v.status = 'posted'
    AND v.voucher_date BETWEEN $2 AND $3
WHERE a.company_id = $1
    AND a.account_type IN ('revenue', 'expense')
    AND a.is_active = TRUE
GROUP BY a.id, a.account_type, a.code, a.name
HAVING COALESCE(SUM(
    CASE WHEN a.account_nature = 'debit'
        THEN e.debit_amount - e.credit_amount
        ELSE e.credit_amount - e.debit_amount
    END
), 0) != 0
ORDER BY a.code;

-- name: GetBalanceSheet :many
SELECT
    a.account_type,
    a.code,
    a.name,
    COALESCE(lb.balance, 0) AS balance
FROM accounts a
LEFT JOIN ledger_balances lb ON a.id = lb.account_id
    AND lb.fiscal_year = $2
    AND lb.fiscal_month = $3
WHERE a.company_id = $1
    AND a.account_type IN ('asset', 'liability', 'equity')
    AND a.is_active = TRUE
    AND lb.balance != 0
ORDER BY a.code;
