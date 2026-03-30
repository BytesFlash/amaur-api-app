ALTER TABLE patients DROP CONSTRAINT IF EXISTS chk_patient_not_own_tutor;
DROP INDEX IF EXISTS idx_patients_tutor_id;
ALTER TABLE patients DROP COLUMN IF EXISTS tutor_id;
