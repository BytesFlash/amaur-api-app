-- Migration 000019: make users.email unique only among non-deleted rows
--
-- The original UNIQUE constraint on users.email is table-wide, which means a
-- soft-deleted user row continues to "occupy" its email address forever.
-- This blocked the EnableLogin flow when a patient's login was disabled and
-- then re-enabled with the same email (INSERT would fail with a duplicate-key
-- violation even though the application-level checks passed).
--
-- The fix:
--   1. Drop the table-level UNIQUE constraint.
--   2. Add a partial unique index restricted to rows where deleted_at IS NULL.
--
-- The existing idx_users_email partial index (WHERE deleted_at IS NULL) already
-- serves query optimisation; we are promoting it to carry the uniqueness guarantee.

-- Step 1: remove the table-level unique constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;

-- Step 2: replace/ensure the partial unique index exists
--         (DROP first in case the index already existed without the UNIQUE flag)
DROP INDEX IF EXISTS idx_users_email;
CREATE UNIQUE INDEX idx_users_email ON users(LOWER(email)) WHERE deleted_at IS NULL;
