-- name: GetPayrollByID :one
SELECT * FROM payrolls
WHERE id = $1 AND company_id = $2;

-- name: GetPayrollByEmployeePeriod :one
SELECT * FROM payrolls
WHERE company_id = $1
    AND employee_id = $2
    AND pay_year = $3
    AND pay_month = $4;

-- name: ListPayrolls :many
SELECT
    p.*,
    e.employee_no,
    e.name AS employee_name,
    d.name AS department_name
FROM payrolls p
JOIN employees e ON p.employee_id = e.id
LEFT JOIN departments d ON e.department_id = d.id
WHERE p.company_id = $1
    AND p.pay_year = $2
    AND p.pay_month = $3
    AND p.status = COALESCE(sqlc.narg('status'), p.status)
ORDER BY e.name;

-- name: GetPayrollPeriod :one
SELECT * FROM payroll_periods
WHERE company_id = $1
    AND pay_year = $2
    AND pay_month = $3;

-- name: CreatePayrollPeriod :one
INSERT INTO payroll_periods (
    company_id,
    pay_year,
    pay_month,
    period_name,
    period_start,
    period_end,
    payment_date,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'draft'
)
RETURNING *;

-- name: CreatePayroll :one
INSERT INTO payrolls (
    company_id,
    employee_id,
    payroll_period_id,
    pay_year,
    pay_month,
    status,
    base_salary,
    overtime_pay,
    night_pay,
    holiday_pay,
    bonus,
    allowances,
    other_earnings,
    total_earnings,
    income_tax,
    local_income_tax,
    nps_employee,
    nhis_employee,
    nhis_ltc_employee,
    ei_employee,
    other_deductions,
    total_deductions,
    net_pay,
    nps_employer,
    nhis_employer,
    nhis_ltc_employer,
    ei_employer,
    wci_employer,
    total_employer_cost,
    work_days,
    overtime_hours,
    night_hours,
    holiday_hours,
    bank_code,
    account_number_enc
) VALUES (
    $1, $2, $3, $4, $5, 'draft',
    $6, $7, $8, $9, $10, $11, $12, $13,
    $14, $15, $16, $17, $18, $19, $20, $21, $22,
    $23, $24, $25, $26, $27, $28,
    $29, $30, $31, $32, $33, $34
)
RETURNING *;

-- name: UpdatePayroll :one
UPDATE payrolls SET
    base_salary = $2,
    overtime_pay = $3,
    night_pay = $4,
    holiday_pay = $5,
    bonus = $6,
    allowances = $7,
    other_earnings = $8,
    total_earnings = $9,
    income_tax = $10,
    local_income_tax = $11,
    nps_employee = $12,
    nhis_employee = $13,
    nhis_ltc_employee = $14,
    ei_employee = $15,
    other_deductions = $16,
    total_deductions = $17,
    net_pay = $18,
    nps_employer = $19,
    nhis_employer = $20,
    nhis_ltc_employer = $21,
    ei_employer = $22,
    wci_employer = $23,
    total_employer_cost = $24,
    work_days = $25,
    overtime_hours = $26,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdatePayrollStatus :exec
UPDATE payrolls SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: MarkPayrollPaid :exec
UPDATE payrolls SET
    status = 'paid',
    payment_date = $2,
    paid_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: UpdatePayrollPeriodStatus :exec
UPDATE payroll_periods SET
    status = $2,
    calculated_at = CASE WHEN $2 = 'calculated' THEN NOW() ELSE calculated_at END,
    calculated_by = CASE WHEN $2 = 'calculated' THEN $3 ELSE calculated_by END,
    approved_at = CASE WHEN $2 = 'approved' THEN NOW() ELSE approved_at END,
    approved_by = CASE WHEN $2 = 'approved' THEN $3 ELSE approved_by END,
    paid_at = CASE WHEN $2 = 'paid' THEN NOW() ELSE paid_at END,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdatePayrollPeriodTotals :exec
UPDATE payroll_periods SET
    total_employees = $2,
    total_earnings = $3,
    total_deductions = $4,
    total_net_pay = $5,
    updated_at = NOW()
WHERE id = $1;

-- name: GetPayrollSummary :one
SELECT
    COUNT(*) AS employee_count,
    COALESCE(SUM(total_earnings), 0) AS total_earnings,
    COALESCE(SUM(total_deductions), 0) AS total_deductions,
    COALESCE(SUM(net_pay), 0) AS total_net_pay,
    COALESCE(SUM(total_employer_cost), 0) AS total_employer_cost,
    COALESCE(SUM(income_tax + local_income_tax), 0) AS total_tax,
    COALESCE(SUM(nps_employee + nps_employer), 0) AS total_nps,
    COALESCE(SUM(nhis_employee + nhis_employer + nhis_ltc_employee + nhis_ltc_employer), 0) AS total_nhis,
    COALESCE(SUM(ei_employee + ei_employer), 0) AS total_ei,
    COALESCE(SUM(wci_employer), 0) AS total_wci
FROM payrolls
WHERE company_id = $1
    AND pay_year = $2
    AND pay_month = $3
    AND status != 'cancelled';

-- name: GetEmployeePayrollHistory :many
SELECT * FROM payrolls
WHERE employee_id = $1
ORDER BY pay_year DESC, pay_month DESC
LIMIT $2 OFFSET $3;

-- name: GetSocialInsuranceRates :one
SELECT * FROM social_insurance_rates
WHERE effective_year = $1
    AND effective_month <= $2
    AND insurance_type = $3
ORDER BY effective_month DESC
LIMIT 1;

-- name: GetIncomeTaxBracket :one
SELECT * FROM income_tax_tables
WHERE effective_year = $1
    AND min_amount <= $2
    AND (max_amount IS NULL OR max_amount > $2);
