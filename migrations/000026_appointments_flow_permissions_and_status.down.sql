-- Revert appointment delete permission and in_progress status support

UPDATE appointments
SET status = 'confirmed'
WHERE status = 'in_progress';

ALTER TABLE appointments DROP CONSTRAINT IF EXISTS appointments_status_check;
ALTER TABLE appointments
ADD CONSTRAINT appointments_status_check
CHECK (status IN ('requested','confirmed','completed','cancelled','no_show'));

DROP INDEX IF EXISTS idx_appointments_date;
CREATE INDEX IF NOT EXISTS idx_appointments_date
ON appointments(scheduled_at)
WHERE status IN ('requested','confirmed');

DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND p.module = 'appointments'
  AND p.action = 'delete'
  AND r.name IN ('super_admin', 'admin', 'coordinator', 'receptionist');

DELETE FROM permissions
WHERE module = 'appointments'
  AND action = 'delete';
