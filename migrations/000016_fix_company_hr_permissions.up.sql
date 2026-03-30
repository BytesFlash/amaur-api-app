-- Elimina permisos demasiado amplios de company_hr
-- company_hr no debe ver la lista de profesionales AMAUR ni todos los pacientes
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'company_hr')
  AND permission_id IN (
    SELECT id FROM permissions
    WHERE (module, action) IN (
      ('workers',     'view'),
      ('workers',     'create'),
      ('workers',     'update'),
      ('workers',     'delete'),
      ('patients',    'view'),
      ('patients',    'create'),
      ('patients',    'update'),
      ('patients',    'delete')
    )
  );

-- Asegura que company_hr tenga permisos adecuados para su contexto
INSERT INTO permissions (module, action)
VALUES
  ('agendas',   'view'),
  ('agendas',   'create'),
  ('dashboard', 'view')
ON CONFLICT (module, action) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'company_hr'
  AND (p.module, p.action) IN (
    ('companies',    'view'),
    ('agendas',      'view'),
    ('visits',       'view'),
    ('care_sessions','view'),
    ('dashboard',    'view')
  )
ON CONFLICT DO NOTHING;
