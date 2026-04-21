-- Clinical consultation notes directly on appointments

ALTER TABLE appointments
  ADD COLUMN IF NOT EXISTS chief_complaint TEXT,
  ADD COLUMN IF NOT EXISTS subjective TEXT,
  ADD COLUMN IF NOT EXISTS objective TEXT,
  ADD COLUMN IF NOT EXISTS assessment TEXT,
  ADD COLUMN IF NOT EXISTS plan TEXT,
  ADD COLUMN IF NOT EXISTS follow_up_required BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS follow_up_notes TEXT,
  ADD COLUMN IF NOT EXISTS follow_up_date DATE;
