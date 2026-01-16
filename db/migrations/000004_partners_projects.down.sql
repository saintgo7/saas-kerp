-- K-ERP v0.2 Migration: Partners and Projects (Rollback)

DROP TRIGGER IF EXISTS set_cost_centers_updated_at ON cost_centers;
DROP TRIGGER IF EXISTS set_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS set_partners_updated_at ON partners;

DROP TABLE IF EXISTS cost_centers;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS partners;
