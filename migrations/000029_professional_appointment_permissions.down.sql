-- Revoke appointments:edit and appointments:create from professional role.

DELETE FROM role_permissions
WHERE role_id  = (SELECT id FROM roles       WHERE name   = 'professional')
  AND permission_id IN (
      SELECT id FROM permissions
      WHERE (module, action) IN (
          ('appointments', 'edit'),
          ('appointments', 'create')
      )
  );
