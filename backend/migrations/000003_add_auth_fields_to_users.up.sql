-- Migration: 000003_add_auth_fields_to_users (up)
-- Adds password hash and email verification flag required by the auth system.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_hash   TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS email_verified  BOOLEAN NOT NULL DEFAULT FALSE;

-- Remove the placeholder default so future inserts must supply the hash explicitly.
ALTER TABLE users ALTER COLUMN password_hash DROP DEFAULT;
