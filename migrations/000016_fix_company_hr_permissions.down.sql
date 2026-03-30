-- Revert: restaura permisos originales de company_hr (desde 000012)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'company_hr'
  AND (p.module, p.action) IN (
    ('workers',      'view'),
    ('patients',     'view'),
    ('visits',       'view'),
    ('care_sessions','view')
  )
ON CONFLICT DO NOTHING;

DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'company_hr')
  AND permission_id IN (
    SELECT id FROM permissions
    WHERE (module, action) IN (
      ('agendas',   'view'),
      ('agendas',   'create'),
      ('dashboard', 'view')
    )
  );
