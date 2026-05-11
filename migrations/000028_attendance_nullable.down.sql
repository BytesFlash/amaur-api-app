-- Revert: treat NULL as false before re-adding NOT NULL constraint.
UPDATE agenda_service_participants SET attended = false WHERE attended IS NULL;

ALTER TABLE agenda_service_participants
  ALTER COLUMN attended SET NOT NULL,
  ALTER COLUMN attended SET DEFAULT false;
