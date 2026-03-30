-- Backfill professional users into amaur_workers profile table.
-- This keeps Users and Professionals modules consistent for existing data.

INSERT INTO amaur_workers (
    id,
    user_id,
    first_name,
    last_name,
    email,
    is_active,
    created_at
)
SELECT
    uuid_generate_v4(),
    u.id,
    u.first_name,
    u.last_name,
    u.email,
    u.is_active,
    NOW()
FROM users u
JOIN user_roles ur ON ur.user_id = u.id
JOIN roles r ON r.id = ur.role_id AND r.name = 'professional'
LEFT JOIN amaur_workers w ON w.user_id = u.id AND w.deleted_at IS NULL
WHERE u.deleted_at IS NULL
  AND w.id IS NULL;
