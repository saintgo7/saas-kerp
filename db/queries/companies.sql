-- name: GetCompanyByID :one
SELECT * FROM companies
WHERE id = $1;

-- name: GetCompanyByBusinessNumber :one
SELECT * FROM companies
WHERE business_number = $1;

-- name: ListCompanies :many
SELECT * FROM companies
WHERE status = COALESCE(sqlc.narg('status'), status)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateCompany :one
INSERT INTO companies (
    business_number,
    company_name,
    company_name_en,
    representative,
    business_type,
    business_item,
    establishment_date,
    phone,
    fax,
    email,
    website,
    zip_code,
    address,
    address_detail,
    fiscal_year_start_month,
    currency_code,
    timezone,
    plan_type,
    plan_started_at,
    plan_expires_at,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
)
RETURNING *;

-- name: UpdateCompany :one
UPDATE companies SET
    company_name = COALESCE(sqlc.narg('company_name'), company_name),
    company_name_en = COALESCE(sqlc.narg('company_name_en'), company_name_en),
    representative = COALESCE(sqlc.narg('representative'), representative),
    business_type = COALESCE(sqlc.narg('business_type'), business_type),
    business_item = COALESCE(sqlc.narg('business_item'), business_item),
    phone = COALESCE(sqlc.narg('phone'), phone),
    fax = COALESCE(sqlc.narg('fax'), fax),
    email = COALESCE(sqlc.narg('email'), email),
    website = COALESCE(sqlc.narg('website'), website),
    zip_code = COALESCE(sqlc.narg('zip_code'), zip_code),
    address = COALESCE(sqlc.narg('address'), address),
    address_detail = COALESCE(sqlc.narg('address_detail'), address_detail),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCompanyPlan :one
UPDATE companies SET
    plan_type = $2,
    plan_started_at = $3,
    plan_expires_at = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCompanyStatus :exec
UPDATE companies SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;
