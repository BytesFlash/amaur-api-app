ALTER TABLE company_program_schedule_rules
    DROP CONSTRAINT IF EXISTS company_program_schedule_rules_weekday_check;

UPDATE company_program_schedule_rules
SET weekday = 7
WHERE weekday = 0;

ALTER TABLE company_program_schedule_rules
    ADD CONSTRAINT company_program_schedule_rules_weekday_check
    CHECK (weekday BETWEEN 1 AND 7);
