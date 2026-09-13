-- Reverse 000018. Width reductions are best-effort: rows written while the
-- wider definition was in force can legitimately block them, which is expected
-- for a narrowing rollback.

-- 8. varchar widths back to the 000003/000004/000014 definitions
ALTER TABLE projects ALTER COLUMN name TYPE VARCHAR(100);
ALTER TABLE projects ALTER COLUMN code TYPE VARCHAR(20);

ALTER TABLE roles ALTER COLUMN name        TYPE VARCHAR(50);
ALTER TABLE roles ALTER COLUMN description TYPE VARCHAR(200);

ALTER TABLE users ALTER COLUMN role  TYPE VARCHAR(30);
ALTER TABLE users ALTER COLUMN name  TYPE VARCHAR(50);
ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(100);

ALTER TABLE companies ALTER COLUMN address_detail TYPE VARCHAR(100);
ALTER TABLE companies ALTER COLUMN address        TYPE VARCHAR(200);
ALTER TABLE companies ALTER COLUMN email          TYPE VARCHAR(100);
ALTER TABLE companies ALTER COLUMN representative TYPE VARCHAR(50);
ALTER TABLE companies ALTER COLUMN name_en        TYPE VARCHAR(100);
ALTER TABLE companies ALTER COLUMN name           TYPE VARCHAR(100);

-- 7. insurance rate precision
ALTER TABLE social_insurance_rates ALTER COLUMN employer_rate TYPE DECIMAL(7, 5);
ALTER TABLE social_insurance_rates ALTER COLUMN employee_rate TYPE DECIMAL(7, 5);

-- 6. money precision
ALTER TABLE projects  ALTER COLUMN actual_cost TYPE DECIMAL(18, 0);
ALTER TABLE projects  ALTER COLUMN budget      TYPE DECIMAL(18, 0);
ALTER TABLE partners ALTER COLUMN credit_limit TYPE DECIMAL(18, 0);

-- 5. project money column names
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'projects'
                 AND column_name = 'actual_cost') THEN
        ALTER TABLE projects RENAME COLUMN actual_cost TO actual_amount;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'projects'
                 AND column_name = 'budget') THEN
        ALTER TABLE projects RENAME COLUMN budget TO budget_amount;
    END IF;
END
$$;

-- 4. status / role CHECK constraints
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;

ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_status_check;
ALTER TABLE projects ADD CONSTRAINT projects_status_check
    CHECK (status IN ('draft', 'active', 'on_hold', 'completed', 'cancelled'));

ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_status_check;
ALTER TABLE companies ADD CONSTRAINT companies_status_check
    CHECK (status IN ('active', 'suspended', 'cancelled'));

-- 3. business_number back to NOT NULL UNIQUE
DROP INDEX IF EXISTS uq_companies_business_number;
ALTER TABLE companies ADD CONSTRAINT companies_business_number_key UNIQUE (business_number);
ALTER TABLE companies ALTER COLUMN business_number SET NOT NULL;

-- 2. company name columns
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'companies'
                 AND column_name = 'name_en') THEN
        ALTER TABLE companies RENAME COLUMN name_en TO company_name_en;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'companies'
                 AND column_name = 'name') THEN
        ALTER TABLE companies RENAME COLUMN name TO company_name;
    END IF;
END
$$;

-- 1. ledger_balances.created_at
ALTER TABLE ledger_balances DROP COLUMN IF EXISTS created_at;
