# sqlc Query Generator

복잡한 SQL 쿼리를 위한 sqlc 쿼리 파일을 생성합니다.
GORM으로 처리하기 어려운 집계, 조인, 분석 쿼리에 사용합니다.

## Trigger
`/sqlc <module> <query-name>` 또는 `/sqlc`

## Arguments
- `module`: 모듈명 (account, invoice, payroll, etc.)
- `query-name`: 쿼리명 (kebab-case)

## Process

### 1. Query 파일 생성
```
db/queries/${module}.sql
```

### 2. 쿼리 작성 패턴

#### 기본 CRUD
```sql
-- name: Get${Resource}ByID :one
SELECT * FROM ${table}
WHERE company_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: List${Resource}s :many
SELECT * FROM ${table}
WHERE company_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: Create${Resource} :one
INSERT INTO ${table} (company_id, ...)
VALUES ($1, ...)
RETURNING *;

-- name: Update${Resource} :one
UPDATE ${table}
SET field = $3, updated_at = NOW()
WHERE company_id = $1 AND id = $2
RETURNING *;

-- name: Delete${Resource} :exec
UPDATE ${table}
SET deleted_at = NOW()
WHERE company_id = $1 AND id = $2;
```

#### 집계 쿼리
```sql
-- name: GetAccountBalance :one
SELECT
    account_id,
    SUM(debit_amount) as total_debit,
    SUM(credit_amount) as total_credit,
    SUM(debit_amount) - SUM(credit_amount) as balance
FROM voucher_lines vl
JOIN vouchers v ON v.id = vl.voucher_id
WHERE v.company_id = $1
  AND vl.account_id = $2
  AND v.voucher_date BETWEEN $3 AND $4
  AND v.status = 'posted'
GROUP BY account_id;

-- name: GetTrialBalance :many
SELECT
    a.code,
    a.name,
    a.account_type,
    COALESCE(SUM(vl.debit_amount), 0) as debit,
    COALESCE(SUM(vl.credit_amount), 0) as credit
FROM accounts a
LEFT JOIN voucher_lines vl ON a.id = vl.account_id
LEFT JOIN vouchers v ON v.id = vl.voucher_id
    AND v.voucher_date BETWEEN $2 AND $3
    AND v.status = 'posted'
WHERE a.company_id = $1
GROUP BY a.id, a.code, a.name, a.account_type
ORDER BY a.code;
```

#### 재무제표 쿼리
```sql
-- name: GetIncomeStatement :many
SELECT
    a.account_type,
    a.code,
    a.name,
    CASE
        WHEN a.account_type IN ('revenue') THEN
            COALESCE(SUM(vl.credit_amount), 0) - COALESCE(SUM(vl.debit_amount), 0)
        ELSE
            COALESCE(SUM(vl.debit_amount), 0) - COALESCE(SUM(vl.credit_amount), 0)
    END as amount
FROM accounts a
LEFT JOIN voucher_lines vl ON a.id = vl.account_id
LEFT JOIN vouchers v ON v.id = vl.voucher_id
    AND v.voucher_date BETWEEN $2 AND $3
    AND v.status = 'posted'
WHERE a.company_id = $1
  AND a.account_type IN ('revenue', 'expense')
GROUP BY a.id
HAVING COALESCE(SUM(vl.debit_amount), 0) + COALESCE(SUM(vl.credit_amount), 0) > 0
ORDER BY a.code;
```

### 3. sqlc.yaml 설정
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/migrations/"
    gen:
      go:
        package: "sqlc"
        out: "internal/repository/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: true
        emit_interface: true
```

### 4. 코드 생성
```bash
sqlc generate
```

## Output Structure
```
db/queries/${module}.sql
internal/repository/sqlc/
├── db.go
├── models.go
├── ${module}.sql.go
└── querier.go
```

## Rules
- 모든 쿼리에 `company_id` 조건 필수
- `deleted_at IS NULL` 조건 포함
- 인덱스 활용 가능한 쿼리 작성
- 복잡한 집계는 CTE 활용
