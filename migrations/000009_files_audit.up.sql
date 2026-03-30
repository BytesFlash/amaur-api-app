-- Files (polymorphic), audit logs, notifications
CREATE TABLE files (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type             VARCHAR(50) NOT NULL,  -- 'patient','company','visit','care_session','contract'
    entity_id               UUID NOT NULL,
    file_name               VARCHAR(255) NOT NULL,
    storage_path            VARCHAR(500) NOT NULL,
    file_size_bytes         INT,
    mime_type               VARCHAR(100),
    description             TEXT,
    is_visible_to_patient   BOOLEAN NOT NULL DEFAULT false,
    uploaded_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by             UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted_at              TIMESTAMPTZ
);

CREATE INDEX idx_files_entity ON files(entity_type, entity_id) WHERE deleted_at IS NULL;

-- Audit log — never deleted from application
CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    entity_type     VARCHAR(100) NOT NULL,
    entity_id       UUID NOT NULL,
    action          VARCHAR(50) NOT NULL,  -- 'create','update','delete','view_sensitive','login','logout'
    old_values      JSONB,
    new_values      JSONB,
    changed_fields  TEXT[],
    ip_address      VARCHAR(45),
    user_agent      TEXT,
    session_id      VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_entity  ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_user    ON audit_logs(user_id);
CREATE INDEX idx_audit_created ON audit_logs(created_at DESC);

-- Internal notifications
CREATE TABLE notifications (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type   VARCHAR(100),
    title               VARCHAR(255) NOT NULL,
    body                TEXT,
    is_read             BOOLEAN NOT NULL DEFAULT false,
    read_at             TIMESTAMPTZ,
    related_entity_type VARCHAR(100),
    related_entity_id   UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ
);

CREATE INDEX idx_notifications_user_unread ON notifications(user_id) WHERE is_read = false;
