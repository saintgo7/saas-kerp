-- K-ERP Development Seed Data
-- Mounted ONLY by docker-compose.dev.yml (local development).
-- Contains a well-known demo credential and must never reach staging/production.

-- Insert demo company
INSERT INTO kerp.companies (id, name, business_number, representative, email)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Demo Company',
    '123-45-67890',
    'Demo Admin',
    'admin@demo.com'
) ON CONFLICT DO NOTHING;

-- Insert demo admin user (password: admin123)
INSERT INTO kerp.users (company_id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@demo.com',
    '$2a$10$b.KnLPr0/OTgyx2F7y7v2uXcH/ZHTAE7tTK9Auw2t15MZdOQ8cdz.',
    'Demo Admin',
    'admin'
) ON CONFLICT DO NOTHING;

-- Log seed
DO $$
BEGIN
    RAISE NOTICE 'K-ERP development seed data loaded (DEV ONLY)';
END $$;
