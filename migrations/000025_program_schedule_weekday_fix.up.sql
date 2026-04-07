-- Normalise company program schedule rules to the same weekday convention used
-- across worker availability and the frontend: 0=Sunday .. 6=Saturday.
UPDATE company_program_schedule_rules
SET weekday = 0
WHERE weekday = 7;

ALTER TABLE company_program_schedule_rules
    DROP CONSTRAINT IF EXISTS company_program_schedule_rules_weekday_check;

ALTER TABLE company_program_schedule_rules
    ADD CONSTRAINT company_program_schedule_rules_weekday_check
    CHECK (weekday BETWEEN 0 AND 6);
