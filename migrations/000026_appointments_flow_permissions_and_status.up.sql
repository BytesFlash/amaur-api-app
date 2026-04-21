-- Add missing delete permission for appointments and enable in_progress status

INSERT INTO permissions (module, action, description)
VALUES ('appointments', 'delete', 'Eliminar citas')
ON CONFLICT (module, action) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.module = 'appointments' AND p.action = 'delete'
WHERE r.name IN ('super_admin', 'admin', 'coordinator', 'receptionist')
ON CONFLICT DO NOTHING;

ALTER TABLE appointments DROP CONSTRAINT IF EXISTS appointments_status_check;
ALTER TABLE appointments
ADD CONSTRAINT appointments_status_check
CHECK (status IN ('requested','confirmed','in_progress','completed','cancelled','no_show'));

DROP INDEX IF EXISTS idx_appointments_date;
CREATE INDEX IF NOT EXISTS idx_appointments_date
ON appointments(scheduled_at)
WHERE status IN ('requested','confirmed','in_progress');
