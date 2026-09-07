-- K-ERP v0.2 Seed: Demo Company Data
-- Creates a demo company for testing and development.
--
-- DEVELOPMENT / TEST ONLY. Do not run against a production database.
--
-- The demo administrator's password hash is NOT stored in this repository.
-- Supply it at run time as a database setting, e.g.
--
--   PGOPTIONS="-c kerp.demo_admin_password_hash=$DEMO_ADMIN_BCRYPT_HASH" \
--     psql -d kerp -f db/seed/004_demo_company.sql
--
-- If the setting is absent the seed aborts with
-- "unrecognized configuration parameter", which is intentional: it must not be
-- possible to create a login account with a credential that is public.

-- Create demo company
INSERT INTO companies (
    id,
    business_number,
    name,
    name_en,
    representative,
    business_type,
    business_item,
    establishment_date,
    phone,
    fax,
    email,
    zip_code,
    address,
    address_detail,
    fiscal_year_start_month,
    plan_type,
    plan_started_at,
    plan_expires_at,
    status
) VALUES (
    '019478e0-0000-7000-8000-000000000001'::UUID,
    '1234567890',
    '데모 주식회사',
    'Demo Corporation',
    '홍길동',
    '서비스업',
    '소프트웨어 개발, IT 컨설팅',
    '2020-01-01',
    '02-1234-5678',
    '02-1234-5679',
    'demo@demo.co.kr',
    '06164',
    '서울특별시 강남구 테헤란로 123',
    '데모빌딩 5층',
    1,
    'enterprise',
    NOW(),
    NOW() + INTERVAL '1 year',
    'active'
);

-- Create standard accounts for demo company
SELECT create_standard_accounts('019478e0-0000-7000-8000-000000000001'::UUID);

-- Create demo admin user. Password hash is injected at run time (see header).
INSERT INTO users (
    id,
    company_id,
    email,
    password_hash,
    name,
    phone,
    status,
    email_verified_at
) VALUES (
    '019478e0-0000-7000-8000-000000000002'::UUID,
    '019478e0-0000-7000-8000-000000000001'::UUID,
    'admin@demo.co.kr',
    current_setting('kerp.demo_admin_password_hash'),
    '관리자',
    '010-1234-5678',
    'active',
    NOW()
);

-- Create admin role
INSERT INTO roles (
    id,
    company_id,
    code,
    name,
    description,
    is_system,
    is_active
) VALUES (
    '019478e0-0000-7000-8000-000000000003'::UUID,
    '019478e0-0000-7000-8000-000000000001'::UUID,
    'SYSADMIN',
    '시스템관리자',
    'Full system access',
    TRUE,
    TRUE
);

-- Assign all permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT
    '019478e0-0000-7000-8000-000000000003'::UUID,
    id
FROM permissions;

-- Mirror the junction rows into roles.permissions.
-- domain.Role.HasPermission() reads ONLY the JSONB column, while this seed (and
-- the UI) populate the normalized role_permissions table. Leaving the JSONB at
-- its '[]' default meant the freshly seeded system administrator was denied
-- every permission. The junction table stays the source of truth; this keeps
-- the column the application actually reads consistent with it.
UPDATE roles r
SET permissions = COALESCE((
        SELECT jsonb_agg(
                   jsonb_build_object(
                       'code', p.code,
                       'name', p.name,
                       'description', p.description,
                       'module', p.module
                   ) ORDER BY p.code
               )
        FROM role_permissions rp
        JOIN permissions p ON p.id = rp.permission_id
        WHERE rp.role_id = r.id
    ), '[]'::jsonb)
WHERE r.company_id = '019478e0-0000-7000-8000-000000000001'::UUID;

-- Give the demo administrator the app-level admin role.
-- Scoped by company_id on purpose: users are unique per (company_id, email), so
-- matching on the address alone would promote a same-named user in every tenant.
UPDATE users
SET role = 'admin'
WHERE company_id = '019478e0-0000-7000-8000-000000000001'::UUID
  AND email = 'admin@demo.co.kr';

-- Assign admin role to admin user
INSERT INTO user_roles (user_id, role_id, assigned_by)
VALUES (
    '019478e0-0000-7000-8000-000000000002'::UUID,
    '019478e0-0000-7000-8000-000000000003'::UUID,
    '019478e0-0000-7000-8000-000000000002'::UUID
);

-- Create departments
INSERT INTO departments (id, company_id, code, name, level, path, is_active) VALUES
('019478e0-0000-7000-8000-000000000010'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'CEO', '대표이사', 1, 'CEO', TRUE),
('019478e0-0000-7000-8000-000000000011'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'DEV', '개발팀', 2, 'CEO.DEV', TRUE),
('019478e0-0000-7000-8000-000000000012'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'SALES', '영업팀', 2, 'CEO.SALES', TRUE),
('019478e0-0000-7000-8000-000000000013'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'HR', '인사팀', 2, 'CEO.HR', TRUE),
('019478e0-0000-7000-8000-000000000014'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'FIN', '재무팀', 2, 'CEO.FIN', TRUE);

-- Update department parent_id
UPDATE departments SET parent_id = '019478e0-0000-7000-8000-000000000010'::UUID
WHERE company_id = '019478e0-0000-7000-8000-000000000001'::UUID AND level = 2;

-- Create positions
INSERT INTO positions (id, company_id, code, name, rank_level, is_active) VALUES
('019478e0-0000-7000-8000-000000000020'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'CEO', '대표이사', 1, TRUE),
('019478e0-0000-7000-8000-000000000021'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'DIR', '이사', 2, TRUE),
('019478e0-0000-7000-8000-000000000022'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'MGR', '부장', 3, TRUE),
('019478e0-0000-7000-8000-000000000023'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'SMGR', '차장', 4, TRUE),
('019478e0-0000-7000-8000-000000000024'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'ASST', '과장', 5, TRUE),
('019478e0-0000-7000-8000-000000000025'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'STAFF', '대리', 6, TRUE),
('019478e0-0000-7000-8000-000000000026'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'ENTRY', '사원', 7, TRUE);

-- Create leave types
INSERT INTO leave_types (id, company_id, code, name, is_paid, default_days, max_carryover_days, is_active) VALUES
('019478e0-0000-7000-8000-000000000030'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'ANNUAL', '연차휴가', TRUE, 15, 10, TRUE),
('019478e0-0000-7000-8000-000000000031'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'SICK', '병가', TRUE, 0, 0, TRUE),
('019478e0-0000-7000-8000-000000000032'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'SPECIAL', '경조휴가', TRUE, 0, 0, TRUE),
('019478e0-0000-7000-8000-000000000033'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'UNPAID', '무급휴가', FALSE, 0, 0, TRUE);

-- Create fiscal periods for 2026
INSERT INTO fiscal_periods (company_id, fiscal_year, fiscal_month, period_name, start_date, end_date, status)
SELECT
    '019478e0-0000-7000-8000-000000000001'::UUID,
    2026,
    m,
    2026 || '년 ' || m || '월',
    make_date(2026, m, 1),
    (make_date(2026, m, 1) + INTERVAL '1 month - 1 day')::DATE,
    CASE WHEN m = 1 THEN 'open' ELSE 'open' END
FROM generate_series(1, 12) AS m;

-- Create demo partners
INSERT INTO partners (id, company_id, partner_type, business_number, partner_name, representative, phone, email, is_active) VALUES
('019478e0-0000-7000-8000-000000000040'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'customer', '2345678901', '(주)고객사', '김철수', '02-2345-6789', 'customer@example.com', TRUE),
('019478e0-0000-7000-8000-000000000041'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'vendor', '3456789012', '(주)공급사', '이영희', '02-3456-7890', 'vendor@example.com', TRUE),
('019478e0-0000-7000-8000-000000000042'::UUID, '019478e0-0000-7000-8000-000000000001'::UUID, 'both', '4567890123', '(주)거래처', '박민수', '02-4567-8901', 'partner@example.com', TRUE);

-- Create insurance workplace
INSERT INTO insurance_workplaces (
    id, company_id, business_number, workplace_name,
    nps_registered, nhis_registered, ei_registered, wci_registered,
    representative, is_active
) VALUES (
    '019478e0-0000-7000-8000-000000000050'::UUID,
    '019478e0-0000-7000-8000-000000000001'::UUID,
    '1234567890',
    '데모 주식회사',
    TRUE, TRUE, TRUE, TRUE,
    '홍길동',
    TRUE
);
