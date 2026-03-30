-- Companies and branches
CREATE TABLE companies (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rut                 VARCHAR(15) UNIQUE,
    name                VARCHAR(255) NOT NULL,
    fantasy_name        VARCHAR(255),
    industry            VARCHAR(100),
    size_category       VARCHAR(20) CHECK (size_category IN ('micro','pequeña','mediana','grande')),
    contact_name        VARCHAR(200),
    contact_email       VARCHAR(255),
    contact_phone       VARCHAR(20),
    billing_email       VARCHAR(255),
    address             TEXT,
    city                VARCHAR(100),
    region              VARCHAR(100),
    website             VARCHAR(255),
    status              VARCHAR(20) NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active','inactive','prospect','churned')),
    commercial_notes    TEXT,
    lead_source         VARCHAR(100),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_companies_name_trgm ON companies USING gin(immutable_unaccent(name) gin_trgm_ops);
CREATE INDEX idx_companies_status    ON companies(status) WHERE deleted_at IS NULL;

CREATE TABLE company_branches (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id      UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    address         TEXT,
    city            VARCHAR(100),
    region          VARCHAR(100),
    contact_name    VARCHAR(200),
    contact_phone   VARCHAR(20),
    is_main         BOOLEAN NOT NULL DEFAULT false,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_branches_company ON company_branches(company_id);

-- Many-to-many patients <-> companies (with context)
CREATE TABLE patient_companies (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id  UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    company_id  UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    position    VARCHAR(150),
    department  VARCHAR(150),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    start_date  DATE,
    end_date    DATE,
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE (patient_id, company_id)
);

CREATE INDEX idx_patient_companies_patient ON patient_companies(patient_id);
CREATE INDEX idx_patient_companies_company ON patient_companies(company_id);
