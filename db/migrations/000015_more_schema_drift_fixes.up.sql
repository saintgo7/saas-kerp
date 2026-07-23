-- Additional additive columns expected by GORM models.

-- partners: AR/AP account links
ALTER TABLE partners ADD COLUMN IF NOT EXISTS ar_account_id UUID REFERENCES accounts(id);
ALTER TABLE partners ADD COLUMN IF NOT EXISTS ap_account_id UUID REFERENCES accounts(id);

-- roles: model carries an embedded permissions JSON array (alongside the
-- normalized role_permissions junction table)
ALTER TABLE roles ADD COLUMN IF NOT EXISTS permissions JSONB NOT NULL DEFAULT '[]'::jsonb;
