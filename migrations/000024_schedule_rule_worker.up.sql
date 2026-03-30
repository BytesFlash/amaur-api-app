-- Add worker_id to program schedule rules so that the planned professional
-- is persisted per recurrence rule and automatically propagated to generated
-- agenda_services when GenerateAgendas runs.
ALTER TABLE company_program_schedule_rules
    ADD COLUMN IF NOT EXISTS worker_id UUID REFERENCES amaur_workers(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_program_rules_worker ON company_program_schedule_rules(worker_id);
