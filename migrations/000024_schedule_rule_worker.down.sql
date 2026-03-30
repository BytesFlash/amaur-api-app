ALTER TABLE company_program_schedule_rules DROP COLUMN IF EXISTS worker_id;
DROP INDEX IF EXISTS idx_program_rules_worker;
