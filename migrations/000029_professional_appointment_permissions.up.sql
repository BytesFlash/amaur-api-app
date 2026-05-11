-- Grant professional role the ability to edit and create appointments.
-- Ownership is enforced server-side: a professional can only modify
-- appointments where they are the assigned worker (worker_id = claims.WorkerID).

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.module, p.action) IN (
    ('appointments', 'edit'),
    ('appointments', 'create')
)
WHERE r.name = 'professional'
ON CONFLICT DO NOTHING;
