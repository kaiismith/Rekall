-- Run once by an elevated DB user (NOT by Alembic).
-- Sets up the `intellikat` runtime user with the minimal grants required.

-- 1. Create the role (idempotent — wrap in DO block if your provider rejects IF NOT EXISTS).
CREATE USER intellikat WITH PASSWORD 'CHANGEME';

-- 2. Read-only on shared tables owned by transcript-persistence + the backend.
GRANT SELECT ON transcript_sessions, transcript_segments, users, calls, meetings TO intellikat;

-- 3. Read/write on the three intellikat-owned tables (created by Alembic).
GRANT SELECT, INSERT, UPDATE, DELETE
    ON transcript_segment_sentiments,
       transcript_session_summaries,
       intellikat_jobs
    TO intellikat;

-- 4. Sequence usage for the UUID default (gen_random_uuid uses the pgcrypto extension; no sequences).
--    No GRANTs needed if all PKs use gen_random_uuid().

-- Explicitly NOT granted: CREATE, ALTER, TRUNCATE, DROP. Schema changes go through Alembic
-- with a separate elevated migration user (typically the DB owner).
