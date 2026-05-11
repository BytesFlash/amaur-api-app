-- Make attended nullable so NULL = "not yet recorded",
-- true = attended, false = explicitly absent.
ALTER TABLE agenda_service_participants
  ALTER COLUMN attended DROP NOT NULL,
  ALTER COLUMN attended DROP DEFAULT;
