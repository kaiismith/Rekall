-- Idempotent one-shot backfill: lift every non-NULL calls.transcript into the
-- new tables tagged engine_mode='legacy' so downstream features can treat
-- transcript_segments as the canonical source of truth without special-casing
-- pre-spec data.
--
-- Re-runnable: the WHERE NOT EXISTS clauses skip any call that already has a
-- legacy session. Running this migration twice is a no-op.

-- 1. One synthetic transcript_sessions row per legacy call.
WITH legacy_calls AS (
    SELECT
        c.id          AS call_id,
        c.user_id     AS user_id,
        c.transcript  AS transcript,
        c.created_at  AS created_at,
        c.updated_at  AS updated_at
    FROM calls c
    WHERE c.transcript IS NOT NULL
      AND c.transcript <> ''
      AND NOT EXISTS (
          SELECT 1
          FROM transcript_sessions ts
          WHERE ts.call_id = c.id
            AND ts.engine_mode = 'legacy'
      )
)
INSERT INTO transcript_sessions (
    id,
    speaker_user_id,
    call_id,
    engine_mode,
    engine_target,
    model_id,
    sample_rate,
    frame_format,
    status,
    started_at,
    ended_at,
    expires_at,
    finalized_segment_count,
    audio_seconds_total,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    user_id,
    call_id,
    'legacy',
    'legacy',
    'legacy',
    16000,
    'pcm_s16le_mono',
    'ended',
    created_at,
    updated_at,
    created_at,                -- expires_at: in the past, so the cleanup job ignores it
    1,
    0,
    NOW(),
    NOW()
FROM legacy_calls;

-- 2. One synthetic transcript_segments row per just-inserted legacy session.
-- The text gets a sentinel start_ms/end_ms so the time_order CHECK passes;
-- analytics consumers should treat engine_mode='legacy' as "no fine timing".
INSERT INTO transcript_segments (
    id,
    session_id,
    segment_index,
    speaker_user_id,
    call_id,
    text,
    language,
    confidence,
    start_ms,
    end_ms,
    words,
    engine_mode,
    model_id,
    segment_started_at,
    created_at
)
SELECT
    gen_random_uuid(),
    ts.id,
    0,
    ts.speaker_user_id,
    ts.call_id,
    c.transcript,
    NULL,
    NULL,
    0,
    1,
    NULL,
    'legacy',
    'legacy',
    ts.started_at,
    NOW()
FROM transcript_sessions ts
JOIN calls c ON c.id = ts.call_id
WHERE ts.engine_mode = 'legacy'
  AND NOT EXISTS (
      SELECT 1
      FROM transcript_segments tsg
      WHERE tsg.session_id = ts.id
  );
