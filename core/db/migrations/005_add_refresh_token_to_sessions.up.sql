-- 005_add_refresh_token_to_sessions.up.sql
ALTER TABLE sessions
    ADD COLUMN refresh_token TEXT;