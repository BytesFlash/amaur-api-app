-- Step 2: Programs, specialties, recurring scheduling, and visit service execution.
-- Non-disruptive approach: create new tables instead of altering existing core tables
-- that are currently consumed with SELECT * mappings.

-- 1) Specialties catalog
CREATE TABLE specialties (
    code        VARCHAR(50) PRIMARY KEY,
    name        VARCHAR(120) NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO specialties (code, name) VALUES
    ('KINESIOLOGIA', 'Kinesiologia'),
    ('MASOTERAPIA', 'Masoterapia'),
    ('TERAPIA_OCUPACIONAL', 'Terapia ocupacional'),
    ('ERGONOMIA', 'Ergonomia'),
    ('EVALUACION_MOVILIDAD', 'Evaluacion de movilidad'),
    ('EVALUACION_FUNCIONAL', 'Evaluacion funcional'),
    ('EJERCICIO_TERAPEUTICO', 'Ejercicio terapeutico'),
    ('PREVENCION_LESIONES', 'Prevencion de lesiones laborales'),
    ('READAPTACION_LABORAL', 'Readaptacion laboral');

-- 2) Worker-specialty many-to-many
CREATE TABLE worker_specialties (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    worker_id       UUID NOT NULL REFERENCES amaur_workers(id) ON DELETE CASCADE,
    specialty_code  VARCHAR(50) NOT NULL REFERENCES specialties(code) ON DELETE RESTRICT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE (worker_id, specialty_code)
);

CREATE INDEX idx_worker_specialties_worker ON worker_specialties(worker_id);
CREATE INDEX idx_worker_specialties_specialty ON worker_specialties(specialty_code);

-- 3) Service type - specialty requirements
CREATE TABLE service_type_specialties (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_type_id UUID NOT NULL REFERENCES service_types(id) ON DELETE CASCADE,
    specialty_code  VARCHAR(50) NOT NULL REFERENCES specialties(code) ON DELETE RESTRICT,
    UNIQUE (service_type_id, specialty_code)
);

CREATE INDEX idx_service_type_specialties_service ON service_type_specialties(service_type_id);
CREATE INDEX idx_service_type_specialties_specialty ON service_type_specialties(specialty_code);

-- 4) Service catalog extension (new entries requested by product)
INSERT INTO service_types (name, category, description, default_duration_minutes, is_group_service, requires_clinical_record, is_active)
SELECT 'Masaje descontracturante', 'terapia', 'Masaje focalizado para disminuir contracturas musculares', 45, false, true, true
WHERE NOT EXISTS (SELECT 1 FROM service_types WHERE lower(name) = lower('Masaje descontracturante'));

INSERT INTO service_types (name, category, description, default_duration_minutes, is_group_service, requires_clinical_record, is_active)
SELECT 'Masaje relajante', 'terapia', 'Masaje orientado a relajacion general y bienestar', 45, false, true, true
WHERE NOT EXISTS (SELECT 1 FROM service_types WHERE lower(name) = lower('Masaje relajante'));

INSERT INTO service_types (name, category, description, default_duration_minutes, is_group_service, requires_clinical_record, is_active)
SELECT 'Masaje postdeportivo', 'terapia', 'Masaje de recuperacion posterior a actividad fisica', 45, false, true, true
WHERE NOT EXISTS (SELECT 1 FROM service_types WHERE lower(name) = lower('Masaje postdeportivo'));

INSERT INTO service_types (name, category, description, default_duration_minutes, is_group_service, requires_clinical_record, is_active)
SELECT 'Evaluacion de movilidad', 'evaluacion', 'Evaluacion funcional del rango y calidad del movimiento', 45, false, true, true
WHERE NOT EXISTS (SELECT 1 FROM service_types WHERE lower(name) = lower('Evaluacion de movilidad'));

INSERT INTO service_types (name, category, description, default_duration_minutes, is_group_service, requires_clinical_record, is_active)
SELECT 'Ejercicio terapeutico guiado', 'terapia', 'Sesion de ejercicios terapeuticos guiados por profesional', 45, false, true, true
WHERE NOT EXISTS (SELECT 1 FROM service_types WHERE lower(name) = lower('Ejercicio terapeutico guiado'));

-- 5) Default service-specialty mappings
INSERT INTO service_type_specialties (service_type_id, specialty_code)
SELECT st.id, 'MASOTERAPIA'
FROM service_types st
WHERE lower(st.name) IN (
    lower('Masaje descontracturante'),
    lower('Masaje relajante'),
    lower('Masaje postdeportivo')
)
ON CONFLICT (service_type_id, specialty_code) DO NOTHING;

INSERT INTO service_type_specialties (service_type_id, specialty_code)
SELECT st.id, 'KINESIOLOGIA'
FROM service_types st
WHERE lower(st.name) IN (
    lower('Sesion de Kinesiterapia'),
    lower('Evaluacion Kinesica'),
    lower('Ejercicio terapeutico guiado')
)
ON CONFLICT (service_type_id, specialty_code) DO NOTHING;

INSERT INTO service_type_specialties (service_type_id, specialty_code)
SELECT st.id, 'TERAPIA_OCUPACIONAL'
FROM service_types st
WHERE lower(st.name) IN (
    lower('Terapia ocupacional')
)
ON CONFLICT (service_type_id, specialty_code) DO NOTHING;

INSERT INTO service_type_specialties (service_type_id, specialty_code)
SELECT st.id, 'ERGONOMIA'
FROM service_types st
WHERE lower(st.name) IN (
    lower('Taller de Ergonomia'),
    lower('Evaluacion de Riesgo Ergonomico')
)
ON CONFLICT (service_type_id, specialty_code) DO NOTHING;

INSERT INTO service_type_specialties (service_type_id, specialty_code)
SELECT st.id, 'EVALUACION_MOVILIDAD'
FROM service_types st
WHERE lower(st.name) IN (
    lower('Evaluacion de movilidad')
)
ON CONFLICT (service_type_id, specialty_code) DO NOTHING;

INSERT INTO service_type_specialties (service_type_id, specialty_code)
SELECT st.id, 'EJERCICIO_TERAPEUTICO'
FROM service_types st
WHERE lower(st.name) IN (
    lower('Ejercicio terapeutico guiado')
)
ON CONFLICT (service_type_id, specialty_code) DO NOTHING;

INSERT INTO service_type_specialties (service_type_id, specialty_code)
SELECT st.id, 'PREVENCION_LESIONES'
FROM service_types st
WHERE lower(st.name) IN (
    lower('Pausa Activa'),
    lower('Charla de Salud Ocupacional')
)
ON CONFLICT (service_type_id, specialty_code) DO NOTHING;

-- 6) Contract schedule policy (days and period rules)
CREATE TABLE contract_schedule_policies (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contract_id             UUID NOT NULL UNIQUE REFERENCES contracts(id) ON DELETE CASCADE,
    allowed_weekdays        SMALLINT[] NOT NULL DEFAULT '{}',
    allowed_start_time      TIME,
    allowed_end_time        TIME,
    execution_period_start  DATE,
    execution_period_end    DATE,
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ
);

ALTER TABLE contract_schedule_policies
    ADD CONSTRAINT chk_contract_schedule_weekdays_valid
    CHECK (allowed_weekdays <@ ARRAY[1,2,3,4,5,6,7]::SMALLINT[]);

ALTER TABLE contract_schedule_policies
    ADD CONSTRAINT chk_contract_schedule_time_range
    CHECK (
        allowed_start_time IS NULL OR
        allowed_end_time IS NULL OR
        allowed_start_time < allowed_end_time
    );

-- 7) Company programs
CREATE TABLE company_programs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id      UUID NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
    contract_id     UUID NOT NULL REFERENCES contracts(id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    start_date      DATE NOT NULL,
    end_date        DATE,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft','active','completed','cancelled')),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_company_program_date_range CHECK (end_date IS NULL OR start_date <= end_date)
);

CREATE INDEX idx_company_programs_company ON company_programs(company_id);
CREATE INDEX idx_company_programs_contract ON company_programs(contract_id);
CREATE INDEX idx_company_programs_status ON company_programs(status);

-- 8) Program recurrence rules
CREATE TABLE company_program_schedule_rules (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    program_id                  UUID NOT NULL REFERENCES company_programs(id) ON DELETE CASCADE,
    weekday                     SMALLINT NOT NULL CHECK (weekday BETWEEN 1 AND 7),
    start_time                  TIME NOT NULL,
    duration_minutes            INT NOT NULL CHECK (duration_minutes > 0),
    frequency_interval_weeks    INT NOT NULL DEFAULT 1 CHECK (frequency_interval_weeks >= 1),
    max_occurrences             INT CHECK (max_occurrences IS NULL OR max_occurrences > 0),
    service_type_id             UUID REFERENCES service_types(id) ON DELETE SET NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                  UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_program_rules_program ON company_program_schedule_rules(program_id);

-- 9) Link generated agendas (visits table was renamed to agendas in 000014)
CREATE TABLE company_program_agendas (
    program_id   UUID NOT NULL REFERENCES company_programs(id) ON DELETE CASCADE,
    agenda_id    UUID NOT NULL REFERENCES agendas(id) ON DELETE CASCADE,
    PRIMARY KEY (program_id, agenda_id),
    UNIQUE (agenda_id)
);

-- 10) Services planned/executed inside an agenda
CREATE TABLE agenda_services (
    id                        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agenda_id                 UUID NOT NULL REFERENCES agendas(id) ON DELETE CASCADE,
    service_type_id           UUID NOT NULL REFERENCES service_types(id) ON DELETE RESTRICT,
    worker_id                 UUID REFERENCES amaur_workers(id) ON DELETE SET NULL,
    planned_start_time        TIME,
    planned_duration_minutes  INT CHECK (planned_duration_minutes IS NULL OR planned_duration_minutes > 0),
    status                    VARCHAR(20) NOT NULL DEFAULT 'planned'
                                  CHECK (status IN ('planned','completed','cancelled')),
    notes                     TEXT,
    completed_at              TIMESTAMPTZ,
    completed_by              UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ
);

CREATE INDEX idx_agenda_services_agenda ON agenda_services(agenda_id);
CREATE INDEX idx_agenda_services_status ON agenda_services(status);

-- 11) Participants per agenda service
CREATE TABLE agenda_service_participants (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agenda_service_id   UUID NOT NULL REFERENCES agenda_services(id) ON DELETE CASCADE,
    patient_id          UUID NOT NULL REFERENCES patients(id) ON DELETE RESTRICT,
    attended            BOOLEAN NOT NULL DEFAULT false,
    attended_at         TIMESTAMPTZ,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE (agenda_service_id, patient_id)
);

CREATE INDEX idx_agenda_service_participants_service ON agenda_service_participants(agenda_service_id);
CREATE INDEX idx_agenda_service_participants_patient ON agenda_service_participants(patient_id);

-- 12) Extend care_sessions.session_type values to include company_program
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
    CHECK (session_type IN ('company_visit', 'particular', 'company_program'));
