-- 004_add_2fa_to_users.up.sql
ALTER TABLE users
    ADD COLUMN two_factor_secret VARCHAR(32),
    ADD COLUMN two_factor_enabled BOOLEAN NOT NULL DEFAULT false;

-- Add an index on the two_factor_enabled column for faster queries
CREATE INDEX idx_users_two_factor_enabled ON users(two_factor_enabled);