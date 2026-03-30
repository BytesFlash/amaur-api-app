-- Worker weekly availability rules (recurring by weekday)
CREATE TABLE IF NOT EXISTS worker_availability_rules (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_id   UUID        NOT NULL REFERENCES amaur_workers(id) ON DELETE CASCADE,
    weekday     SMALLINT    NOT NULL CHECK (weekday BETWEEN 0 AND 6), -- 0=Sunday … 6=Saturday
    start_time  TIME        NOT NULL,
    end_time    TIME        NOT NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT  chk_avail_time CHECK (end_time > start_time)
);

CREATE INDEX IF NOT EXISTS idx_worker_avail_worker
    ON worker_availability_rules(worker_id)
    WHERE is_active = TRUE;
