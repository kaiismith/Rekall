-- One row per ASR `final` TranscriptEvent. text + per-word timings (JSONB) +
-- engine snapshot are stored verbatim so downstream insight extraction can
-- iterate without re-running ASR.
--
-- (session_id, segment_index) is UNIQUE so retransmissions from the browser
-- become an UPSERT — see infrastructure/repositories/transcript_repository.go.
-- engine_mode and model_id are denormalised from the session for fast
-- per-segment provenance queries without a join.
CREATE TABLE IF NOT EXISTS transcript_segments (
    id                  UUID        NOT NULL DEFAULT gen_random_uuid(),
    session_id          UUID        NOT NULL,
    segment_index       INTEGER     NOT NULL,
    speaker_user_id     UUID        NOT NULL,
    call_id             UUID,
    meeting_id          UUID,
    text                TEXT        NOT NULL,
    language            TEXT,
    confidence          REAL,
    start_ms            INTEGER     NOT NULL,
    end_ms              INTEGER     NOT NULL,
    words               JSONB,
    engine_mode         TEXT        NOT NULL,
    model_id            TEXT        NOT NULL,
    segment_started_at  TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT transcript_segments_pkey       PRIMARY KEY (id),
    CONSTRAINT transcript_segments_session_fk FOREIGN KEY (session_id)      REFERENCES transcript_sessions(id) ON DELETE CASCADE,
    CONSTRAINT transcript_segments_speaker_fk FOREIGN KEY (speaker_user_id) REFERENCES users(id)               ON DELETE RESTRICT,
    CONSTRAINT transcript_segments_call_fk    FOREIGN KEY (call_id)         REFERENCES calls(id)               ON DELETE CASCADE,
    CONSTRAINT transcript_segments_meeting_fk FOREIGN KEY (meeting_id)      REFERENCES meetings(id)            ON DELETE CASCADE,
    CONSTRAINT transcript_segments_call_xor_meeting CHECK (
        (call_id IS NOT NULL)::int + (meeting_id IS NOT NULL)::int = 1
    ),
    CONSTRAINT transcript_segments_time_order CHECK (end_ms > start_ms),
    CONSTRAINT transcript_segments_unique_per_session UNIQUE (session_id, segment_index)
);

CREATE INDEX IF NOT EXISTS idx_transcript_segments_session_index
    ON transcript_segments (session_id, segment_index);

CREATE INDEX IF NOT EXISTS idx_transcript_segments_call_started
    ON transcript_segments (call_id, segment_started_at)
    WHERE call_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transcript_segments_meeting_started
    ON transcript_segments (meeting_id, segment_started_at)
    WHERE meeting_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transcript_segments_speaker_started
    ON transcript_segments (speaker_user_id, segment_started_at DESC);

-- Foundation for the future "search transcripts" feature; building the GIN
-- index now (table is empty) avoids a multi-hour CONCURRENTLY build later.
CREATE INDEX IF NOT EXISTS idx_transcript_segments_text_search
    ON transcript_segments USING GIN (to_tsvector('english', text));
