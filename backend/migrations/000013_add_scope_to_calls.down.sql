-- Migration: 000013_add_scope_to_calls (down)
-- Drops only the columns; never deletes rows. Any scope attribution is lost
-- on rollback but the call records themselves are preserved.

DROP INDEX IF EXISTS idx_calls_scope;

ALTER TABLE calls
    DROP CONSTRAINT IF EXISTS calls_scope_coherent,
    DROP CONSTRAINT IF EXISTS calls_scope_type_check;

ALTER TABLE calls
    DROP COLUMN IF EXISTS scope_id,
    DROP COLUMN IF EXISTS scope_type;
