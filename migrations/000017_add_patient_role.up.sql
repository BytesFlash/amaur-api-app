-- Add independent patient portal role (separate from company_worker)
INSERT INTO roles (name, description, is_system)
VALUES ('patient', 'Paciente con acceso a sus propias atenciones e historial', true)
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.module, p.action) IN (
  ('dashboard','view'),
  ('care_sessions','view'),
  ('appointments','view'),
  ('visits','view')
)
WHERE r.name = 'patient'
ON CONFLICT DO NOTHING;
