-- Migration: 000002_create_calls (up)

CREATE TABLE IF NOT EXISTS calls (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         TEXT        NOT NULL,
    duration_sec  INTEGER     NOT NULL DEFAULT 0 CHECK (duration_sec >= 0),
    status        TEXT        NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    recording_url TEXT,
    transcript    TEXT,
    metadata      JSONB       NOT NULL DEFAULT '{}',
    started_at    TIMESTAMPTZ,
    ended_at      TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    CONSTRAINT calls_ended_after_started
        CHECK (ended_at IS NULL OR started_at IS NULL OR ended_at >= started_at)
);

CREATE INDEX idx_calls_user_id      ON calls (user_id);
CREATE INDEX idx_calls_status       ON calls (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_calls_created_at   ON calls (created_at DESC);
CREATE INDEX idx_calls_active       ON calls (user_id, created_at DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE calls IS 'Recorded conversations ingested into the Rekall platform.';
COMMENT ON COLUMN calls.status IS 'pending | processing | done | failed';
