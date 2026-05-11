-- Permisos para planes de tratamiento, registros de sesión y seguimiento

INSERT INTO permissions (module, action, description) VALUES
    ('treatment_plans', 'view',   'Ver planes de tratamiento'),
    ('treatment_plans', 'create', 'Crear planes de tratamiento'),
    ('treatment_plans', 'edit',   'Editar planes de tratamiento'),
    ('treatment_plans', 'delete', 'Eliminar planes de tratamiento'),
    ('session_records', 'view',   'Ver registros de sesión'),
    ('session_records', 'create', 'Crear registros de sesión'),
    ('session_records', 'edit',   'Editar registros de sesión'),
    ('follow_up_tasks', 'view',   'Ver tareas de seguimiento'),
    ('follow_up_tasks', 'create', 'Crear tareas de seguimiento'),
    ('follow_up_tasks', 'edit',   'Editar tareas de seguimiento')
ON CONFLICT (module, action) DO NOTHING;

-- super_admin y admin: acceso total
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('super_admin', 'admin')
  AND p.module IN ('treatment_plans', 'session_records', 'follow_up_tasks')
ON CONFLICT DO NOTHING;

-- coordinator y receptionist: view + create en planes y seguimiento
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.module IN ('treatment_plans', 'session_records', 'follow_up_tasks')
                   AND p.action IN ('view', 'create', 'edit')
WHERE r.name IN ('coordinator', 'receptionist')
ON CONFLICT DO NOTHING;

-- professional/worker: view + create + edit en sus propios registros
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.module IN ('treatment_plans', 'session_records', 'follow_up_tasks')
                   AND p.action IN ('view', 'create', 'edit')
WHERE r.name IN ('professional')
ON CONFLICT DO NOTHING;
