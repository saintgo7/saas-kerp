-- Schema drift fixes: add columns the Go/GORM models expect but that were
-- missing from the migration schema. Additive only (no renames) so existing
-- data, seeds and RLS policies are unaffected.

-- companies: model expects code/settings/trial_ends_at/logo
ALTER TABLE companies ADD COLUMN IF NOT EXISTS code VARCHAR(50);
ALTER TABLE companies ADD COLUMN IF NOT EXISTS settings JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS logo VARCHAR(500);

-- partners: model expects a code
ALTER TABLE partners ADD COLUMN IF NOT EXISTS code VARCHAR(20);

-- projects: model expects is_active (in addition to status)
ALTER TABLE projects ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- roles: model expects code + is_active
ALTER TABLE roles ADD COLUMN IF NOT EXISTS code VARCHAR(50);
ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- users: model expects a single role column (app-level role)
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(30) NOT NULL DEFAULT 'user';

-- voucher_entries / fiscal_periods: GORM writes updated_at on Save/Create
ALTER TABLE voucher_entries ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE fiscal_periods ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Seed admin gets an admin role so authorization can be wired later
UPDATE users SET role = 'admin' WHERE email = 'admin@demo.co.kr';
