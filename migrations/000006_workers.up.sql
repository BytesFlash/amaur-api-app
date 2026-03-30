-- AMAUR workers (operational team) — separate from system users
CREATE TABLE amaur_workers (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID UNIQUE REFERENCES users(id) ON DELETE SET NULL,  -- optional system access
    rut                 VARCHAR(12) UNIQUE,
    first_name          VARCHAR(100) NOT NULL,
    last_name           VARCHAR(100) NOT NULL,
    email               VARCHAR(255),
    phone               VARCHAR(20),
    role_title          VARCHAR(100),   -- 'Kinesiólogo', 'Terapeuta', etc.
    specialty           VARCHAR(150),
    hire_date           DATE,
    termination_date    DATE,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    availability_notes  TEXT,
    internal_notes      TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_workers_active   ON amaur_workers(is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_workers_user     ON amaur_workers(user_id) WHERE user_id IS NOT NULL;

-- Service catalogue
CREATE TABLE service_types (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                        VARCHAR(255) NOT NULL,
    category                    VARCHAR(100),   -- 'bienestar','terapia','evaluacion','capacitacion'
    description                 TEXT,
    default_duration_minutes    INT,
    is_group_service            BOOLEAN NOT NULL DEFAULT false,  -- pausa activa grupal vs individual
    requires_clinical_record    BOOLEAN NOT NULL DEFAULT false,
    is_active                   BOOLEAN NOT NULL DEFAULT true,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ
);

-- Which workers can perform which services
CREATE TABLE worker_services (
    worker_id       UUID NOT NULL REFERENCES amaur_workers(id) ON DELETE CASCADE,
    service_type_id UUID NOT NULL REFERENCES service_types(id) ON DELETE CASCADE,
    PRIMARY KEY (worker_id, service_type_id)
);

-- Add FK for progress_notes.worker_id now that amaur_workers exists
ALTER TABLE progress_notes
    ADD CONSTRAINT fk_progress_notes_worker
    FOREIGN KEY (worker_id) REFERENCES amaur_workers(id) ON DELETE SET NULL;
