-- Patients and clinical data
CREATE TABLE patients (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rut                         VARCHAR(12) UNIQUE,
    first_name                  VARCHAR(100) NOT NULL,
    last_name                   VARCHAR(100) NOT NULL,
    birth_date                  DATE,
    gender                      VARCHAR(30) CHECK (gender IN ('masculino','femenino','otro','prefiero_no_decir')),
    email                       VARCHAR(255),
    phone                       VARCHAR(20),
    address                     TEXT,
    city                        VARCHAR(100),
    region                      VARCHAR(100),
    emergency_contact_name      VARCHAR(200),
    emergency_contact_phone     VARCHAR(20),
    general_notes               TEXT,
    patient_type                VARCHAR(20) NOT NULL DEFAULT 'company'
                                    CHECK (patient_type IN ('particular','company','both')),
    status                      VARCHAR(20) NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','inactive','discharged')),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ,
    created_by                  UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by                  UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted_at                  TIMESTAMPTZ
);

CREATE INDEX idx_patients_name_trgm
    ON patients USING gin(immutable_unaccent(first_name || ' ' || last_name) gin_trgm_ops);
CREATE INDEX idx_patients_rut     ON patients(rut) WHERE rut IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_patients_status  ON patients(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_patients_deleted ON patients(deleted_at) WHERE deleted_at IS NULL;

-- Clinical record (1:1 with patients)
CREATE TABLE clinical_records (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id              UUID UNIQUE NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    main_diagnosis          TEXT,
    allergies               TEXT,
    current_medications     TEXT,
    relevant_history        TEXT,
    family_history          TEXT,
    physical_restrictions   TEXT,
    alerts                  TEXT,
    occupation              VARCHAR(150),
    consent_signed          BOOLEAN NOT NULL DEFAULT false,
    consent_date            DATE,
    consent_version         VARCHAR(50),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ,
    created_by              UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by              UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Active ailments / conditions
CREATE TABLE patient_ailments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id      UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    description     VARCHAR(255) NOT NULL,
    body_area       VARCHAR(100),
    icd10_code      VARCHAR(10),
    severity        VARCHAR(20) CHECK (severity IN ('leve','moderado','severo')),
    onset_date      DATE,
    resolved_date   DATE,
    is_chronic      BOOLEAN NOT NULL DEFAULT false,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_ailments_patient        ON patient_ailments(patient_id);
CREATE INDEX idx_ailments_patient_active ON patient_ailments(patient_id) WHERE is_active = true;

-- Progress notes (separate from SOAP in sessions)
CREATE TABLE progress_notes (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id      UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    care_session_id UUID,               -- FK added after care_sessions table
    worker_id       UUID,               -- FK added after amaur_workers table
    note_type       VARCHAR(50) CHECK (note_type IN ('evolución','observación','alerta','seguimiento','informe')),
    note            TEXT NOT NULL,
    is_private      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_progress_notes_patient ON progress_notes(patient_id);
