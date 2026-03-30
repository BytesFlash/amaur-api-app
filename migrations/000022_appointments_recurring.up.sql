-- Track recurring batches on appointments
ALTER TABLE appointments ADD COLUMN IF NOT EXISTS recurring_group_id UUID;
CREATE INDEX IF NOT EXISTS idx_appointments_group
    ON appointments(recurring_group_id) WHERE recurring_group_id IS NOT NULL;
