DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'patient')
  AND permission_id IN (
      SELECT id FROM permissions
      WHERE (module, action) IN (
          ('treatment_plans', 'view'),
          ('session_records',  'view'),
          ('follow_up_tasks',  'view')
      )
  );
