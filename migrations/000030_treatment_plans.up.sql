-- ============================================================
-- Planes de tratamiento, registros de sesión y tareas de seguimiento
-- ============================================================

-- ── Planes de tratamiento ────────────────────────────────────
CREATE TABLE treatment_plans (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id          UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    professional_id     UUID REFERENCES amaur_workers(id) ON DELETE SET NULL,
    service_type_id     UUID NOT NULL REFERENCES service_types(id),
    title               VARCHAR(255) NOT NULL,
    objective           TEXT,
    total_sessions      INT  NOT NULL DEFAULT 1,
    completed_sessions  INT  NOT NULL DEFAULT 0,
    frequency_type      VARCHAR(30) NOT NULL DEFAULT 'weekly',
    frequency_interval  INT  NOT NULL DEFAULT 7,   -- días entre sesiones
    start_date          DATE NOT NULL,
    estimated_end_date  DATE,
    status              VARCHAR(20) NOT NULL DEFAULT 'active',
    notes               TEXT,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ,

    CONSTRAINT treatment_plans_status_check
        CHECK (status IN ('active','paused','completed','cancelled')),
    CONSTRAINT treatment_plans_frequency_type_check
        CHECK (frequency_type IN ('weekly','twice_weekly','monthly','custom')),
    CONSTRAINT treatment_plans_sessions_check
        CHECK (total_sessions >= 1 AND completed_sessions >= 0),
    CONSTRAINT treatment_plans_completed_le_total
        CHECK (completed_sessions <= total_sessions)
);

CREATE INDEX idx_treatment_plans_patient      ON treatment_plans(patient_id);
CREATE INDEX idx_treatment_plans_professional ON treatment_plans(professional_id);
CREATE INDEX idx_treatment_plans_status       ON treatment_plans(status)
    WHERE status IN ('active','paused');

-- ── Extensión de appointments ────────────────────────────────
ALTER TABLE appointments
    ADD COLUMN IF NOT EXISTS treatment_plan_id UUID
        REFERENCES treatment_plans(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS session_number    INT,
    ADD COLUMN IF NOT EXISTS counts_as_session BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS idx_appointments_treatment_plan
    ON appointments(treatment_plan_id)
    WHERE treatment_plan_id IS NOT NULL;

-- ── Registros de sesión (evolución clínica por sesión) ───────
CREATE TABLE session_records (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    treatment_plan_id    UUID NOT NULL REFERENCES treatment_plans(id) ON DELETE CASCADE,
    appointment_id       UUID REFERENCES appointments(id) ON DELETE SET NULL,
    patient_id           UUID NOT NULL REFERENCES patients(id),
    professional_id      UUID NOT NULL REFERENCES amaur_workers(id),
    session_number       INT  NOT NULL,
    evolution_notes      TEXT,
    performed_treatment  TEXT,
    patient_instructions TEXT,
    pain_level           SMALLINT,
    next_action          TEXT,
    follow_up_required   BOOLEAN NOT NULL DEFAULT FALSE,
    follow_up_date       DATE,
    internal_notes       TEXT,
    created_by           UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ,

    CONSTRAINT session_records_pain_level_check
        CHECK (pain_level IS NULL OR (pain_level BETWEEN 0 AND 10))
);

CREATE INDEX idx_session_records_plan        ON session_records(treatment_plan_id);
CREATE INDEX idx_session_records_appointment ON session_records(appointment_id)
    WHERE appointment_id IS NOT NULL;
CREATE INDEX idx_session_records_patient     ON session_records(patient_id);

-- ── Tareas de seguimiento ────────────────────────────────────
CREATE TABLE follow_up_tasks (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id        UUID NOT NULL REFERENCES patients(id),
    treatment_plan_id UUID REFERENCES treatment_plans(id) ON DELETE SET NULL,
    appointment_id    UUID REFERENCES appointments(id) ON DELETE SET NULL,
    professional_id   UUID REFERENCES amaur_workers(id) ON DELETE SET NULL,
    title             VARCHAR(255) NOT NULL,
    description       TEXT,
    due_date          DATE NOT NULL,
    status            VARCHAR(20) NOT NULL DEFAULT 'pending',
    priority          VARCHAR(20) NOT NULL DEFAULT 'medium',
    created_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ,

    CONSTRAINT follow_up_tasks_status_check
        CHECK (status IN ('pending','in_progress','done','cancelled')),
    CONSTRAINT follow_up_tasks_priority_check
        CHECK (priority IN ('low','medium','high','urgent'))
);

CREATE INDEX idx_follow_up_tasks_patient      ON follow_up_tasks(patient_id);
CREATE INDEX idx_follow_up_tasks_professional ON follow_up_tasks(professional_id);
CREATE INDEX idx_follow_up_tasks_due_date     ON follow_up_tasks(due_date);
CREATE INDEX idx_follow_up_tasks_status       ON follow_up_tasks(status)
    WHERE status IN ('pending','in_progress');
