-- K-ERP v0.2 Migration: Core Tables (Rollback)

DROP TRIGGER IF EXISTS set_roles_updated_at ON roles;
DROP TRIGGER IF EXISTS set_users_updated_at ON users;
DROP TRIGGER IF EXISTS set_companies_updated_at ON companies;
DROP FUNCTION IF EXISTS trigger_set_updated_at();

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS companies;
