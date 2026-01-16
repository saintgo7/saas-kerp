-- name: GetEmployeeByID :one
SELECT * FROM employees
WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL;

-- name: GetEmployeeByNo :one
SELECT * FROM employees
WHERE company_id = $1 AND employee_no = $2 AND deleted_at IS NULL;

-- name: GetEmployeeByUserID :one
SELECT * FROM employees
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: ListEmployees :many
SELECT
    e.*,
    d.name AS department_name,
    p.name AS position_name
FROM employees e
LEFT JOIN departments d ON e.department_id = d.id
LEFT JOIN positions p ON e.position_id = p.id
WHERE e.company_id = $1
    AND e.deleted_at IS NULL
    AND e.status = COALESCE(sqlc.narg('status'), e.status)
    AND e.department_id = COALESCE(sqlc.narg('department_id'), e.department_id)
ORDER BY e.name
LIMIT $2 OFFSET $3;

-- name: ListActiveEmployees :many
SELECT
    e.*,
    d.name AS department_name,
    p.name AS position_name
FROM employees e
LEFT JOIN departments d ON e.department_id = d.id
LEFT JOIN positions p ON e.position_id = p.id
WHERE e.company_id = $1
    AND e.status = 'active'
    AND e.deleted_at IS NULL
ORDER BY d.name, e.name;

-- name: CreateEmployee :one
INSERT INTO employees (
    company_id,
    user_id,
    employee_no,
    name,
    name_en,
    resident_number_enc,
    birth_date,
    gender,
    phone,
    mobile,
    email,
    zip_code,
    address,
    address_detail,
    department_id,
    position_id,
    manager_id,
    hire_date,
    probation_end_date,
    employment_type,
    contract_start_date,
    contract_end_date,
    work_location,
    work_email,
    status,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
    $21, $22, $23, $24, $25, $26
)
RETURNING *;

-- name: UpdateEmployee :one
UPDATE employees SET
    name = COALESCE(sqlc.narg('name'), name),
    name_en = COALESCE(sqlc.narg('name_en'), name_en),
    phone = COALESCE(sqlc.narg('phone'), phone),
    mobile = COALESCE(sqlc.narg('mobile'), mobile),
    email = COALESCE(sqlc.narg('email'), email),
    zip_code = COALESCE(sqlc.narg('zip_code'), zip_code),
    address = COALESCE(sqlc.narg('address'), address),
    address_detail = COALESCE(sqlc.narg('address_detail'), address_detail),
    department_id = COALESCE(sqlc.narg('department_id'), department_id),
    position_id = COALESCE(sqlc.narg('position_id'), position_id),
    manager_id = COALESCE(sqlc.narg('manager_id'), manager_id),
    work_location = COALESCE(sqlc.narg('work_location'), work_location),
    work_email = COALESCE(sqlc.narg('work_email'), work_email),
    updated_at = NOW(),
    updated_by = $2
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ResignEmployee :exec
UPDATE employees SET
    status = 'resigned',
    resignation_date = $2,
    resignation_reason = $3,
    updated_at = NOW(),
    updated_by = $4
WHERE id = $1;

-- name: SoftDeleteEmployee :exec
UPDATE employees SET
    deleted_at = NOW(),
    status = 'terminated'
WHERE id = $1;

-- name: CountEmployeesByDepartment :many
SELECT
    d.id,
    d.name,
    COUNT(e.id) AS employee_count
FROM departments d
LEFT JOIN employees e ON d.id = e.department_id
    AND e.status = 'active'
    AND e.deleted_at IS NULL
WHERE d.company_id = $1
    AND d.is_active = TRUE
GROUP BY d.id, d.name
ORDER BY d.name;

-- name: GetEmployeesForPayroll :many
SELECT
    e.*,
    es.base_salary,
    es.payment_type,
    es.allowances,
    es.bank_code,
    es.account_number_enc,
    es.account_holder,
    ei.nps_qualified,
    ei.nhis_qualified,
    ei.ei_qualified,
    ei.nps_standard_remuneration,
    ei.nhis_standard_remuneration
FROM employees e
LEFT JOIN employee_salaries es ON e.id = es.employee_id
    AND es.effective_date <= $2
    AND (es.end_date IS NULL OR es.end_date >= $2)
LEFT JOIN employee_insurance ei ON e.id = ei.employee_id
WHERE e.company_id = $1
    AND e.status = 'active'
    AND e.hire_date <= $2
    AND (e.resignation_date IS NULL OR e.resignation_date >= $2)
    AND e.deleted_at IS NULL
ORDER BY e.name;

-- name: SearchEmployees :many
SELECT * FROM employees
WHERE company_id = $1
    AND deleted_at IS NULL
    AND (name LIKE $2 OR employee_no LIKE $2)
ORDER BY name
LIMIT 20;
