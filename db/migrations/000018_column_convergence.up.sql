-- ============================================================================
-- COLUMN CONVERGENCE
-- Aligns the physical schema with what the Go/GORM domain models actually read
-- and write. Every item here is a live 42703/23514/22001/23505 failure today.
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. ledger_balances.created_at  (period close / trial balance / year end)
-- ----------------------------------------------------------------------------
-- domain.LedgerBalance embeds BaseModel, so GORM puts created_at in the INSERT
-- column list, but 000005 only gave the table updated_at. Every UpsertBalance /
-- CalculatePeriodBalances write failed with
--   ERROR: column "created_at" of relation "ledger_balances" does not exist
-- taking monthly close, the trial balance and the financial statements with it.
ALTER TABLE ledger_balances ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- ----------------------------------------------------------------------------
-- 2. companies: column names the model expects
-- ----------------------------------------------------------------------------
-- domain.Company declares Name/NameEn with no `column:` tag, so GORM addresses
-- them as name/name_en while the table had company_name/company_name_en.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'companies'
                 AND column_name = 'company_name') THEN
        ALTER TABLE companies RENAME COLUMN company_name TO name;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'companies'
                 AND column_name = 'company_name_en') THEN
        ALTER TABLE companies RENAME COLUMN company_name_en TO name_en;
    END IF;
END
$$;

-- ----------------------------------------------------------------------------
-- 3. companies.business_number: stop blocking the second tenant
-- ----------------------------------------------------------------------------
-- The column was NOT NULL UNIQUE, but domain.Company.BusinessNumber is a plain
-- string (zero value '') and NewCompany() never sets it. '' is an ordinary
-- value to a unique index, so onboarding tenant #2 without a registration
-- number failed with duplicate key on companies_business_number_key.
ALTER TABLE companies ALTER COLUMN business_number DROP NOT NULL;
ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_business_number_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_companies_business_number
    ON companies(business_number)
    WHERE business_number IS NOT NULL AND business_number <> '';

-- ----------------------------------------------------------------------------
-- 4. Status CHECK constraints that reject the models' own default values
-- ----------------------------------------------------------------------------
-- domain.CompanyStatusTrial = 'trial'; the CHECK listed only active/suspended/
-- cancelled, so any trial signup violated companies_status_check.
ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_status_check;
ALTER TABLE companies ADD CONSTRAINT companies_status_check
    CHECK (status IN ('active', 'suspended', 'cancelled', 'trial'));

-- domain.ProjectStatusPlanning = 'planning' is what NewProject() assigns, and
-- it was absent from the CHECK, so project creation failed 100% of the time.
-- 'draft' is retained: it exists in the DDL and may already be stored.
ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_status_check;
ALTER TABLE projects ADD CONSTRAINT projects_status_check
    CHECK (status IN ('draft', 'planning', 'active', 'on_hold', 'completed', 'cancelled'));

-- users.role was free text while every other status column is constrained; a
-- typo silently produced a user whose UserRole.IsValid() is false.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'user', 'viewer'));

-- ----------------------------------------------------------------------------
-- 5. projects: money column names + precision expected by the model
-- ----------------------------------------------------------------------------
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'projects'
                 AND column_name = 'budget_amount') THEN
        ALTER TABLE projects RENAME COLUMN budget_amount TO budget;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'projects'
                 AND column_name = 'actual_amount') THEN
        ALTER TABLE projects RENAME COLUMN actual_amount TO actual_cost;
    END IF;
END
$$;

-- ----------------------------------------------------------------------------
-- 6. Money precision: columns the models tag decimal(18,2) but that were (18,0)
-- ----------------------------------------------------------------------------
-- GORM tags do not alter existing columns, so these silently rounded to whole
-- won: a credit limit of 1,000,000.50 was stored as 1,000,001.
-- Scope is deliberately limited to the three columns whose model tag says
-- (18,2). Payroll/insurance amounts stay (18,0) - Korean payroll settles in
-- whole won and widening them would change reconciliation semantics.
ALTER TABLE partners ALTER COLUMN credit_limit TYPE DECIMAL(18, 2);
ALTER TABLE projects  ALTER COLUMN budget      TYPE DECIMAL(18, 2);
ALTER TABLE projects  ALTER COLUMN actual_cost TYPE DECIMAL(18, 2);

-- ----------------------------------------------------------------------------
-- 7. Social insurance rate precision
-- ----------------------------------------------------------------------------
-- DECIMAL(7,5) truncated the seeded long-term-care rate 0.004591 to 0.00459.
-- On a 4,000,000 KRW monthly base that is 18,360 instead of 18,364 - 4 KRW per
-- employee per month, which breaks reconciliation against the NHIS assessment.
ALTER TABLE social_insurance_rates ALTER COLUMN employee_rate TYPE DECIMAL(9, 7);
ALTER TABLE social_insurance_rates ALTER COLUMN employer_rate TYPE DECIMAL(9, 7);

-- ----------------------------------------------------------------------------
-- 8. varchar width drift: columns narrower than the model (and the UI) allow
-- ----------------------------------------------------------------------------
-- The frontend validates against the model tags, so input that passes the UI
-- hit SQLSTATE 22001 "value too long" and surfaced as a 500.
ALTER TABLE companies ALTER COLUMN name           TYPE VARCHAR(200);
ALTER TABLE companies ALTER COLUMN name_en        TYPE VARCHAR(200);
ALTER TABLE companies ALTER COLUMN representative TYPE VARCHAR(100);
ALTER TABLE companies ALTER COLUMN email          TYPE VARCHAR(255);
ALTER TABLE companies ALTER COLUMN address        TYPE VARCHAR(300);
ALTER TABLE companies ALTER COLUMN address_detail TYPE VARCHAR(200);

ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(255);
ALTER TABLE users ALTER COLUMN name  TYPE VARCHAR(100);
ALTER TABLE users ALTER COLUMN role  TYPE VARCHAR(50);

ALTER TABLE roles ALTER COLUMN description TYPE VARCHAR(500);
ALTER TABLE roles ALTER COLUMN name        TYPE VARCHAR(100);

ALTER TABLE projects ALTER COLUMN code        TYPE VARCHAR(50);
ALTER TABLE projects ALTER COLUMN name        TYPE VARCHAR(200);
-- projects.description stays TEXT: it is already wider than the model's
-- varchar(1000) tag, and narrowing it could truncate existing rows.
