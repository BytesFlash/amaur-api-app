-- Patient Tutor/Guardian Relationship
--
-- Enables a patient to designate another patient as their guardian/tutor.
-- Primarily used when the patient is a minor (< 18 years old) who needs an
-- adult representative to manage their care and, optionally, log in on their
-- behalf while the minor has no independent login.
--
-- Rules enforced at DB level:
--   - A patient cannot be their own tutor  (CHECK constraint)
-- Rules enforced at application level:
--   - A tutor must be an adult (>= 18 years old)
--   - A tutor is itself a patient record in this table

ALTER TABLE patients
    ADD COLUMN IF NOT EXISTS tutor_id UUID REFERENCES patients(id) ON DELETE SET NULL;

ALTER TABLE patients
    ADD CONSTRAINT chk_patient_not_own_tutor
    CHECK (tutor_id IS NULL OR tutor_id <> id);

CREATE INDEX IF NOT EXISTS idx_patients_tutor_id
    ON patients(tutor_id)
    WHERE tutor_id IS NOT NULL;
