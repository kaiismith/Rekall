-- Migration: 000002_create_calls (down)

DROP INDEX IF EXISTS idx_calls_active;
DROP INDEX IF EXISTS idx_calls_created_at;
DROP INDEX IF EXISTS idx_calls_status;
DROP INDEX IF EXISTS idx_calls_user_id;
DROP TABLE IF EXISTS calls;
