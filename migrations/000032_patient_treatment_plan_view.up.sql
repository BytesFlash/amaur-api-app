-- Permiso de lectura de planes de tratamiento y registros de sesión para el rol paciente.
-- El paciente sólo accede a sus propios datos (filtrado en aplicación por patient_id).

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.module, p.action) IN (
    ('treatment_plans', 'view'),
    ('session_records',  'view'),
    ('follow_up_tasks',  'view')
)
WHERE r.name = 'patient'
ON CONFLICT DO NOTHING;
