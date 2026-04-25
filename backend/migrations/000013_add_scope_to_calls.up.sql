-- Migration: 000013_add_scope_to_calls (up)
-- Adds organization/department scope to the calls table, mirroring meetings.
-- Existing rows are treated as Open Items (scope_type IS NULL).

ALTER TABLE calls
    ADD COLUMN scope_type TEXT,
    ADD COLUMN scope_id   UUID;

ALTER TABLE calls
    ADD CONSTRAINT calls_scope_type_check
        CHECK (scope_type IS NULL OR scope_type IN ('organization', 'department'));

ALTER TABLE calls
    ADD CONSTRAINT calls_scope_coherent
        CHECK ((scope_type IS NULL AND scope_id IS NULL)
            OR (scope_type IS NOT NULL AND scope_id IS NOT NULL));

CREATE INDEX IF NOT EXISTS idx_calls_scope
    ON calls (scope_type, scope_id)
    WHERE scope_type IS NOT NULL;

COMMENT ON COLUMN calls.scope_type IS 'organization | department | NULL (open)';
COMMENT ON COLUMN calls.scope_id   IS 'UUID of the org or dept; NULL for open items';
