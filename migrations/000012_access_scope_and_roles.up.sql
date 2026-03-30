-- Access scope and strict profile flows

-- 1) Users can be scoped to a company (for company portal accounts)
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_company_id ON users(company_id) WHERE deleted_at IS NULL;

-- 2) Professional profile must be linked to a login user
ALTER TABLE amaur_workers
    ALTER COLUMN user_id SET NOT NULL;

-- 3) Company-facing roles
INSERT INTO roles (name, description, is_system) VALUES
    ('company_hr', 'Portal empresa RRHH (solo datos de su empresa)', true),
    ('company_worker', 'Trabajador empresa (solo sus propias atenciones/participaciones)', true)
ON CONFLICT (name) DO NOTHING;

-- Company HR permissions (base permissions, data scope enforced in handlers)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.module, p.action) IN (
    ('dashboard','view'),
    ('patients','view'),
    ('visits','view'),
    ('care_sessions','view'),
    ('workers','view'),
    ('appointments','view')
)
WHERE r.name = 'company_hr'
ON CONFLICT DO NOTHING;

-- Company Worker permissions (base permissions, own-data scope to be enforced incrementally)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.module, p.action) IN (
    ('dashboard','view'),
    ('care_sessions','view'),
    ('appointments','view'),
    ('visits','view')
)
WHERE r.name = 'company_worker'
ON CONFLICT DO NOTHING;
