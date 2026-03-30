-- Contracts, plans and service packages
CREATE TABLE contracts (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id          UUID NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
    name                VARCHAR(255) NOT NULL,
    contract_type       VARCHAR(50) CHECK (contract_type IN ('mensual','anual','paquete','puntual')),
    status              VARCHAR(20) NOT NULL DEFAULT 'active'
                            CHECK (status IN ('draft','active','paused','expired','terminated')),
    start_date          DATE NOT NULL,
    end_date            DATE,
    renewal_date        DATE,
    value_clp           DECIMAL(12,2),
    billing_cycle       VARCHAR(50),
    notes               TEXT,
    signed_document_url VARCHAR(500),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by          UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_contracts_company  ON contracts(company_id);
CREATE INDEX idx_contracts_status   ON contracts(status);
CREATE INDEX idx_contracts_end_date ON contracts(end_date) WHERE status = 'active';

-- A contract can have multiple service lines, each with sessions OR hours quota
CREATE TABLE contract_services (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contract_id           UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    service_type_id       UUID NOT NULL REFERENCES service_types(id) ON DELETE RESTRICT,
    quota_type            VARCHAR(20) NOT NULL DEFAULT 'sessions'
                              CHECK (quota_type IN ('sessions','hours','unlimited')),
    quantity_per_period   INT,
    period_unit           VARCHAR(20) CHECK (period_unit IN ('month','week','total')),
    sessions_included     INT,
    sessions_used         INT NOT NULL DEFAULT 0,
    hours_included        DECIMAL(8,2),
    hours_used            DECIMAL(8,2) NOT NULL DEFAULT 0,
    price_per_unit        DECIMAL(10,2),
    notes                 TEXT
);

CREATE INDEX idx_contract_services_contract ON contract_services(contract_id);
