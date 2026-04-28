-- Backfill is forward-only: rolling back would require deleting rows we
-- can't reliably distinguish from real legacy-tagged data inserted later by
-- a future code path. Operators who need to undo this should restore from a
-- pre-migration backup.
--
-- Intentionally a no-op so `migrate down 1` succeeds.
SELECT 1;
