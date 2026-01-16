-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE company_id = $1 AND email = $2 AND deleted_at IS NULL;

-- name: GetUserWithRoles :one
SELECT
    u.*,
    COALESCE(
        json_agg(
            json_build_object('id', r.id, 'name', r.name)
        ) FILTER (WHERE r.id IS NOT NULL),
        '[]'
    ) AS roles
FROM users u
LEFT JOIN user_roles ur ON u.id = ur.user_id
LEFT JOIN roles r ON ur.role_id = r.id
WHERE u.id = $1 AND u.deleted_at IS NULL
GROUP BY u.id;

-- name: ListUsers :many
SELECT * FROM users
WHERE company_id = $1
    AND deleted_at IS NULL
    AND status = COALESCE(sqlc.narg('status'), status)
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: CreateUser :one
INSERT INTO users (
    company_id,
    email,
    password_hash,
    name,
    phone,
    avatar_url,
    status,
    mfa_enabled
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET
    name = COALESCE(sqlc.narg('name'), name),
    phone = COALESCE(sqlc.narg('phone'), phone),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users SET
    password_hash = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserStatus :exec
UPDATE users SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserLastLogin :exec
UPDATE users SET
    last_login_at = NOW(),
    failed_login_attempts = 0,
    locked_until = NULL
WHERE id = $1;

-- name: IncrementFailedLogin :exec
UPDATE users SET
    failed_login_attempts = failed_login_attempts + 1,
    locked_until = CASE
        WHEN failed_login_attempts >= 4 THEN NOW() + INTERVAL '30 minutes'
        ELSE locked_until
    END
WHERE id = $1;

-- name: VerifyUserEmail :exec
UPDATE users SET
    email_verified_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: EnableUserMFA :exec
UPDATE users SET
    mfa_enabled = TRUE,
    mfa_secret = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: DisableUserMFA :exec
UPDATE users SET
    mfa_enabled = FALSE,
    mfa_secret = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: SoftDeleteUser :exec
UPDATE users SET
    deleted_at = NOW(),
    status = 'inactive'
WHERE id = $1;

-- name: CountUsersByCompany :one
SELECT COUNT(*) FROM users
WHERE company_id = $1 AND deleted_at IS NULL;
