DROP TABLE IF EXISTS follow_up_tasks;
DROP TABLE IF EXISTS session_records;

ALTER TABLE appointments
    DROP COLUMN IF EXISTS counts_as_session,
    DROP COLUMN IF EXISTS session_number,
    DROP COLUMN IF EXISTS treatment_plan_id;

DROP TABLE IF EXISTS treatment_plans;
