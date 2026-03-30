-- Revert migration 000019
DROP INDEX IF EXISTS idx_users_email;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
