-- Revert 000021_programs_and_agenda_services.up.sql

-- 1) Restore original care_sessions session_type check
DO $$
DECLARE
    c RECORD;
BEGIN
    FOR c IN
        SELECT conname
        FROM pg_constraint
        WHERE conrelid = 'care_sessions'::regclass
          AND contype = 'c'
          AND pg_get_constraintdef(oid) ILIKE '%session_type%'
    LOOP
        EXECUTE format('ALTER TABLE care_sessions DROP CONSTRAINT %I', c.conname);
    END LOOP;
END $$;

ALTER TABLE care_sessions
    ADD CONSTRAINT care_sessions_session_type_check
    CHECK (session_type IN ('company_visit', 'particular'));

-- 2) Drop new program/scheduling/execution tables
DROP TABLE IF EXISTS agenda_service_participants;
DROP TABLE IF EXISTS agenda_services;
DROP TABLE IF EXISTS company_program_agendas;
DROP TABLE IF EXISTS company_program_schedule_rules;
DROP TABLE IF EXISTS company_programs;
DROP TABLE IF EXISTS contract_schedule_policies;
DROP TABLE IF EXISTS service_type_specialties;
DROP TABLE IF EXISTS worker_specialties;
DROP TABLE IF EXISTS specialties;

-- 3) Remove service catalog entries added by this migration
DELETE FROM service_types WHERE lower(name) IN (
    lower('Masaje descontracturante'),
    lower('Masaje relajante'),
    lower('Masaje postdeportivo'),
    lower('Evaluacion de movilidad'),
    lower('Ejercicio terapeutico guiado')
);
