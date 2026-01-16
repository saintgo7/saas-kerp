-- name: GetAccountByID :one
SELECT * FROM accounts
WHERE id = $1 AND company_id = $2;

-- name: GetAccountByCode :one
SELECT * FROM accounts
WHERE company_id = $1 AND code = $2;

-- name: ListAccounts :many
SELECT * FROM accounts
WHERE company_id = $1
    AND is_active = COALESCE(sqlc.narg('is_active'), is_active)
    AND account_type = COALESCE(sqlc.narg('account_type'), account_type)
ORDER BY code;

-- name: ListAccountsByType :many
SELECT * FROM accounts
WHERE company_id = $1
    AND account_type = $2
    AND is_active = TRUE
ORDER BY code;

-- name: ListAccountsHierarchy :many
SELECT * FROM accounts
WHERE company_id = $1
    AND is_active = TRUE
ORDER BY path, code;

-- name: ListPostableAccounts :many
SELECT * FROM accounts
WHERE company_id = $1
    AND is_active = TRUE
    AND allow_direct_posting = TRUE
ORDER BY code;

-- name: CreateAccount :one
INSERT INTO accounts (
    company_id,
    code,
    name,
    name_en,
    parent_id,
    level,
    path,
    account_type,
    account_nature,
    account_category,
    is_active,
    is_control_account,
    allow_direct_posting,
    sort_order
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: UpdateAccount :one
UPDATE accounts SET
    name = COALESCE(sqlc.narg('name'), name),
    name_en = COALESCE(sqlc.narg('name_en'), name_en),
    account_category = COALESCE(sqlc.narg('account_category'), account_category),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    allow_direct_posting = COALESCE(sqlc.narg('allow_direct_posting'), allow_direct_posting),
    sort_order = COALESCE(sqlc.narg('sort_order'), sort_order),
    updated_at = NOW()
WHERE id = $1 AND company_id = $2
RETURNING *;

-- name: DeactivateAccount :exec
UPDATE accounts SET
    is_active = FALSE,
    updated_at = NOW()
WHERE id = $1;

-- name: CountAccountUsage :one
SELECT COUNT(*) FROM voucher_entries
WHERE account_id = $1;

-- name: SearchAccounts :many
SELECT * FROM accounts
WHERE company_id = $1
    AND is_active = TRUE
    AND (code LIKE $2 OR name LIKE $2)
ORDER BY code
LIMIT 20;
