DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'patient');

DELETE FROM roles
WHERE name = 'patient';
