-- Link user accounts to patient profile (used by company_worker role)

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS patient_id UUID REFERENCES patients(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_patient_id ON users(patient_id) WHERE deleted_at IS NULL;
