-- One row per ASR session lifecycle. The PK is the session_id issued by the
-- C++ ASR service (StartSessionOutput.SessionID), so a single id joins backend
-- logs, ASR logs, OpenAI logs, and DB rows.
--
-- Exactly one of (call_id, meeting_id) is set: solo Calls populate call_id,
-- multi-party Meetings populate meeting_id. The (engine_mode, engine_target,
-- model_id) triple is a snapshot taken at session-open time so a future model
-- rollover does not retroactively rewrite history.
CREATE TABLE IF NOT EXISTS transcript_sessions (
    id                       UUID        NOT NULL,
    speaker_user_id          UUID        NOT NULL,
    call_id                  UUID,
    meeting_id               UUID,
    scope_type               TEXT,
    scope_id                 UUID,
    engine_mode              TEXT        NOT NULL,
    engine_target            TEXT        NOT NULL,
    model_id                 TEXT        NOT NULL,
    language_requested       TEXT,
    sample_rate              INTEGER     NOT NULL,
    frame_format             TEXT        NOT NULL,
    correlation_id           TEXT,
    status                   TEXT        NOT NULL DEFAULT 'active',
    started_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at                 TIMESTAMPTZ,
    expires_at               TIMESTAMPTZ NOT NULL,
    finalized_segment_count  INTEGER     NOT NULL DEFAULT 0,
    audio_seconds_total      NUMERIC(10,3) NOT NULL DEFAULT 0,
    error_code               TEXT,
    error_message            TEXT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT transcript_sessions_pkey         PRIMARY KEY (id),
    CONSTRAINT transcript_sessions_speaker_fk   FOREIGN KEY (speaker_user_id) REFERENCES users(id)    ON DELETE RESTRICT,
    CONSTRAINT transcript_sessions_call_fk      FOREIGN KEY (call_id)         REFERENCES calls(id)    ON DELETE CASCADE,
    CONSTRAINT transcript_sessions_meeting_fk   FOREIGN KEY (meeting_id)      REFERENCES meetings(id) ON DELETE CASCADE,
    CONSTRAINT transcript_sessions_call_xor_meeting CHECK (
        (call_id IS NOT NULL)::int + (meeting_id IS NOT NULL)::int = 1
    ),
    CONSTRAINT transcript_sessions_engine_mode_chk CHECK (
        engine_mode IN ('local', 'openai', 'legacy')
    ),
    CONSTRAINT transcript_sessions_status_chk CHECK (
        status IN ('active', 'ended', 'errored', 'expired')
    )
);

CREATE INDEX IF NOT EXISTS idx_transcript_sessions_speaker_started
    ON transcript_sessions (speaker_user_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_transcript_sessions_call
    ON transcript_sessions (call_id)
    WHERE call_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transcript_sessions_meeting
    ON transcript_sessions (meeting_id)
    WHERE meeting_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transcript_sessions_scope
    ON transcript_sessions (scope_type, scope_id, started_at DESC)
    WHERE scope_type IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transcript_sessions_status_active
    ON transcript_sessions (expires_at)
    WHERE status = 'active';
