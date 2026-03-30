-- Seed: permissions catalogue and system roles

-- ────────────────────────────────────────────────────────────────────────────
-- PERMISSIONS
-- ────────────────────────────────────────────────────────────────────────────
INSERT INTO permissions (module, action, description) VALUES
    ('dashboard',           'view',     'Ver dashboard'),
    ('patients',            'view',     'Ver pacientes'),
    ('patients',            'create',   'Crear pacientes'),
    ('patients',            'edit',     'Editar pacientes'),
    ('patients',            'delete',   'Eliminar pacientes'),
    ('patients',            'export',   'Exportar pacientes'),
    ('clinical_records',    'view',     'Ver ficha clínica'),
    ('clinical_records',    'create',   'Crear ficha clínica'),
    ('clinical_records',    'edit',     'Editar ficha clínica'),
    ('companies',           'view',     'Ver empresas'),
    ('companies',           'create',   'Crear empresas'),
    ('companies',           'edit',     'Editar empresas'),
    ('companies',           'delete',   'Eliminar empresas'),
    ('companies',           'export',   'Exportar empresas'),
    ('visits',              'view',     'Ver visitas'),
    ('visits',              'create',   'Crear visitas'),
    ('visits',              'edit',     'Editar visitas'),
    ('visits',              'delete',   'Cancelar visitas'),
    ('visits',              'export',   'Exportar visitas'),
    ('care_sessions',       'view',     'Ver atenciones'),
    ('care_sessions',       'create',   'Registrar atenciones'),
    ('care_sessions',       'edit',     'Editar atenciones'),
    ('care_sessions',       'delete',   'Eliminar atenciones'),
    ('workers',             'view',     'Ver trabajadores'),
    ('workers',             'create',   'Crear trabajadores'),
    ('workers',             'edit',     'Editar trabajadores'),
    ('workers',             'delete',   'Desactivar trabajadores'),
    ('contracts',           'view',     'Ver contratos'),
    ('contracts',           'create',   'Crear contratos'),
    ('contracts',           'edit',     'Editar contratos'),
    ('contracts',           'delete',   'Terminar contratos'),
    ('contracts',           'export',   'Exportar contratos'),
    ('users',               'view',     'Ver usuarios'),
    ('users',               'create',   'Crear usuarios'),
    ('users',               'edit',     'Editar usuarios'),
    ('users',               'delete',   'Desactivar usuarios'),
    ('roles',               'view',     'Ver roles'),
    ('roles',               'edit',     'Modificar roles'),
    ('reports',             'view',     'Ver reportes'),
    ('reports',             'export',   'Exportar reportes'),
    ('audit_logs',          'view',     'Ver auditoría'),
    ('settings',            'view',     'Ver configuración'),
    ('settings',            'edit',     'Editar configuración'),
    ('files',               'view',     'Ver archivos'),
    ('files',               'upload',   'Subir archivos'),
    ('files',               'delete',   'Eliminar archivos'),
    ('appointments',        'view',     'Ver agenda'),
    ('appointments',        'create',   'Crear citas'),
    ('appointments',        'edit',     'Editar citas');

-- ────────────────────────────────────────────────────────────────────────────
-- SYSTEM ROLES
-- ────────────────────────────────────────────────────────────────────────────
INSERT INTO roles (name, description, is_system) VALUES
    ('super_admin',  'Acceso total al sistema',                      true),
    ('admin',        'Administrador operacional',                     true),
    ('coordinator',  'Coordinador de visitas y operaciones',          true),
    ('professional', 'Profesional AMAUR (terapeuta/kinesiólogo)',     true),
    ('receptionist', 'Recepción y apoyo administrativo',              true),
    ('read_only',    'Solo lectura',                                  true);

-- ────────────────────────────────────────────────────────────────────────────
-- SUPER ADMIN → all permissions
-- ────────────────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'super_admin';

-- ────────────────────────────────────────────────────────────────────────────
-- ADMIN permissions
-- ────────────────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND (p.module, p.action) IN (
    ('dashboard','view'),
    ('patients','view'),('patients','create'),('patients','edit'),('patients','delete'),('patients','export'),
    ('clinical_records','view'),('clinical_records','create'),('clinical_records','edit'),
    ('companies','view'),('companies','create'),('companies','edit'),('companies','delete'),('companies','export'),
    ('visits','view'),('visits','create'),('visits','edit'),('visits','delete'),('visits','export'),
    ('care_sessions','view'),('care_sessions','create'),('care_sessions','edit'),('care_sessions','delete'),
    ('workers','view'),('workers','create'),('workers','edit'),('workers','delete'),
    ('contracts','view'),('contracts','create'),('contracts','edit'),('contracts','delete'),('contracts','export'),
    ('reports','view'),('reports','export'),
    ('settings','view'),('settings','edit'),
    ('files','view'),('files','upload'),('files','delete'),
    ('appointments','view'),('appointments','create'),('appointments','edit')
);

-- ────────────────────────────────────────────────────────────────────────────
-- COORDINATOR permissions
-- ────────────────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'coordinator'
  AND (p.module, p.action) IN (
    ('dashboard','view'),
    ('patients','view'),('patients','create'),('patients','edit'),('patients','export'),
    ('clinical_records','view'),('clinical_records','create'),('clinical_records','edit'),
    ('companies','view'),('companies','create'),('companies','edit'),
    ('visits','view'),('visits','create'),('visits','edit'),('visits','export'),
    ('care_sessions','view'),('care_sessions','create'),('care_sessions','edit'),
    ('workers','view'),
    ('contracts','view'),
    ('reports','view'),('reports','export'),
    ('files','view'),('files','upload'),
    ('appointments','view'),('appointments','create'),('appointments','edit')
);

-- ────────────────────────────────────────────────────────────────────────────
-- PROFESSIONAL permissions
-- ────────────────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'professional'
  AND (p.module, p.action) IN (
    ('dashboard','view'),
    ('patients','view'),
    ('clinical_records','view'),('clinical_records','edit'),
    ('visits','view'),
    ('care_sessions','view'),('care_sessions','create'),('care_sessions','edit'),
    ('workers','view'),
    ('appointments','view'),
    ('files','view'),('files','upload')
);

-- ────────────────────────────────────────────────────────────────────────────
-- RECEPTIONIST permissions
-- ────────────────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'receptionist'
  AND (p.module, p.action) IN (
    ('dashboard','view'),
    ('patients','view'),('patients','create'),('patients','edit'),
    ('companies','view'),
    ('visits','view'),
    ('care_sessions','view'),
    ('workers','view'),
    ('appointments','view'),('appointments','create'),('appointments','edit')
);

-- ────────────────────────────────────────────────────────────────────────────
-- READ ONLY permissions
-- ────────────────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'read_only'
  AND p.action = 'view';
