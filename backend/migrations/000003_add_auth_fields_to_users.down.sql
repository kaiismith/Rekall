-- Migration: 000003_add_auth_fields_to_users (down)

ALTER TABLE users
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS email_verified;
