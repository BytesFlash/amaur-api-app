-- Visits to companies and individual care sessions
CREATE TABLE visits (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id          UUID NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
    branch_id           UUID REFERENCES company_branches(id) ON DELETE SET NULL,
    contract_id         UUID REFERENCES contracts(id) ON DELETE SET NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'scheduled'
                            CHECK (status IN ('scheduled','in_progress','completed','cancelled','no_show')),
    scheduled_date      DATE NOT NULL,
    scheduled_start     TIME,
    scheduled_end       TIME,
    actual_start        TIMESTAMPTZ,
    actual_end          TIMESTAMPTZ,
    coordinator_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    general_notes       TEXT,
    cancellation_reason TEXT,
    internal_report     TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by          UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_visits_company        ON visits(company_id);
CREATE INDEX idx_visits_scheduled_date ON visits(scheduled_date);
CREATE INDEX idx_visits_status         ON visits(status);
CREATE INDEX idx_visits_upcoming       ON visits(scheduled_date, company_id)
    WHERE status IN ('scheduled','in_progress');

CREATE TABLE visit_workers (
    visit_id        UUID NOT NULL REFERENCES visits(id) ON DELETE CASCADE,
    worker_id       UUID NOT NULL REFERENCES amaur_workers(id) ON DELETE CASCADE,
    role_in_visit   VARCHAR(50) NOT NULL DEFAULT 'profesional',
    PRIMARY KEY (visit_id, worker_id)
);

-- Group sessions for services like pausas activas (no individual records)
CREATE TABLE group_sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    visit_id        UUID NOT NULL REFERENCES visits(id) ON DELETE CASCADE,
    service_type_id UUID NOT NULL REFERENCES service_types(id),
    worker_id       UUID REFERENCES amaur_workers(id) ON DELETE SET NULL,
    attendee_count  INT NOT NULL DEFAULT 0,
    session_date    DATE NOT NULL,
    session_time    TIME,
    duration_minutes INT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Individual care sessions (company visit or particular)
CREATE TABLE care_sessions (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    visit_id            UUID REFERENCES visits(id) ON DELETE SET NULL,
    patient_id          UUID NOT NULL REFERENCES patients(id) ON DELETE RESTRICT,
    worker_id           UUID NOT NULL REFERENCES amaur_workers(id) ON DELETE RESTRICT,
    service_type_id     UUID NOT NULL REFERENCES service_types(id) ON DELETE RESTRICT,
    company_id          UUID REFERENCES companies(id) ON DELETE SET NULL,
    contract_service_id UUID REFERENCES contract_services(id) ON DELETE SET NULL,
    session_type        VARCHAR(20) NOT NULL
                            CHECK (session_type IN ('company_visit','particular')),
    session_date        DATE NOT NULL,
    session_time        TIME,
    duration_minutes    INT,
    status              VARCHAR(20) NOT NULL DEFAULT 'completed'
                            CHECK (status IN ('scheduled','completed','cancelled','no_show')),
    -- SOAP
    chief_complaint     TEXT,
    subjective          TEXT,
    objective           TEXT,
    assessment          TEXT,
    plan                TEXT,
    notes               TEXT,
    -- Follow-up workflow
    follow_up_required  BOOLEAN NOT NULL DEFAULT false,
    follow_up_status    VARCHAR(20) DEFAULT 'pending'
                            CHECK (follow_up_status IN ('pending','contacted','scheduled','resolved','discarded')),
    follow_up_date      DATE,
    follow_up_notes     TEXT,
    follow_up_contacted_at TIMESTAMPTZ,
    -- Audit
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by          UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_care_sessions_patient  ON care_sessions(patient_id);
CREATE INDEX idx_care_sessions_worker   ON care_sessions(worker_id);
CREATE INDEX idx_care_sessions_visit    ON care_sessions(visit_id) WHERE visit_id IS NOT NULL;
CREATE INDEX idx_care_sessions_company  ON care_sessions(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_care_sessions_date     ON care_sessions(session_date);
CREATE INDEX idx_care_sessions_followup ON care_sessions(follow_up_date)
    WHERE follow_up_required = true AND follow_up_status = 'pending';
CREATE INDEX idx_care_sessions_month    ON care_sessions(session_date)
    WHERE status = 'completed';

-- Add FK for progress_notes.care_session_id now that care_sessions exists
ALTER TABLE progress_notes
    ADD CONSTRAINT fk_progress_notes_session
    FOREIGN KEY (care_session_id) REFERENCES care_sessions(id) ON DELETE SET NULL;

-- Appointments (for patient self-booking / scheduling)
CREATE TABLE appointments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id      UUID NOT NULL REFERENCES patients(id) ON DELETE RESTRICT,
    worker_id       UUID REFERENCES amaur_workers(id) ON DELETE SET NULL,
    service_type_id UUID NOT NULL REFERENCES service_types(id),
    company_id      UUID REFERENCES companies(id) ON DELETE SET NULL,
    scheduled_at    TIMESTAMPTZ NOT NULL,
    duration_minutes INT,
    status          VARCHAR(20) NOT NULL DEFAULT 'requested'
                        CHECK (status IN ('requested','confirmed','completed','cancelled','no_show')),
    notes           TEXT,
    care_session_id UUID REFERENCES care_sessions(id) ON DELETE SET NULL, -- linked after completion
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_appointments_patient ON appointments(patient_id);
CREATE INDEX idx_appointments_worker  ON appointments(worker_id) WHERE worker_id IS NOT NULL;
CREATE INDEX idx_appointments_date    ON appointments(scheduled_at) WHERE status IN ('requested','confirmed');
