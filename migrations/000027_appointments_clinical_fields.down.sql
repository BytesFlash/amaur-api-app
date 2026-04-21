-- Rollback clinical consultation fields on appointments

ALTER TABLE appointments
  DROP COLUMN IF EXISTS follow_up_date,
  DROP COLUMN IF EXISTS follow_up_notes,
  DROP COLUMN IF EXISTS follow_up_required,
  DROP COLUMN IF EXISTS plan,
  DROP COLUMN IF EXISTS assessment,
  DROP COLUMN IF EXISTS objective,
  DROP COLUMN IF EXISTS subjective,
  DROP COLUMN IF EXISTS chief_complaint;
