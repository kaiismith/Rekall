# Smoke Checklist: Transcript Persistence

Manual smoke for the [transcript-persistence](../../.kiro/specs/transcript-persistence/) feature. NOT executed in CI — run this once after a fresh deploy of phases A–C, then again any time the backend ASR wiring or the frontend captions flow is touched.

## Prerequisites

- Local stack running: `make up` from the repo root.
- ASR feature flag on: `ASR_FEATURE_ENABLED=true` in `.env` (defaults true in `dev`).
- A running ASR service. Either:
  - Local engine: `make asr-run` (slow on a CPU-only laptop — see [asr-engine-mode-switching](../../.kiro/specs/asr-engine-mode-switching/) for the openai-mode shortcut).
  - OpenAI engine: `ASR_ENGINE_MODE=openai`, `OPENAI_API_KEY=sk-…`, `make asr-run-openai`.
- Browser pointed at `http://localhost:5173` and signed in.
- `psql` access to the dev DB (or `docker compose exec postgres psql -U rekall rekall_db`).

## Solo Call lifecycle

1. **Create a call.** Calls page → "New call" → give it a title → save.
2. **Open the call.** Navigate into the new call — the captions panel renders.
3. **Issue a session.** Click *Start captions*. Within ~1 s a `transcript_sessions` row should appear:
   ```sql
   SELECT id, speaker_user_id, call_id, engine_mode, model_id, status, started_at, expires_at
   FROM transcript_sessions
   ORDER BY created_at DESC LIMIT 1;
   ```
   Expected: `engine_mode` = `local` or `openai` matching your service config; `status` = `active`; `expires_at` ~5 min in the future.
4. **Speak ~30 seconds.** Read a paragraph aloud. Captions stream in the panel as the engine produces finals.
5. **Watch segments accumulate.** Re-run periodically:
   ```sql
   SELECT segment_index, start_ms, end_ms, language, model_id, left(text, 60) AS preview
   FROM transcript_segments
   WHERE session_id = '<the-id-from-step-3>'
   ORDER BY segment_index;
   ```
   Expected: one row per VAD-segmented utterance, ordered, no gaps in `segment_index`. `model_id` denormalised onto each row.
6. **Confirm the counters.**
   ```sql
   SELECT finalized_segment_count, audio_seconds_total
   FROM transcript_sessions WHERE id = '<id>';
   ```
   `finalized_segment_count` = count from step 5; `audio_seconds_total` ≈ sum of `(end_ms - start_ms) / 1000` from step 5.
7. **End the session.** Click *Stop*. Verify:
   - `transcript_sessions.status` flips to `ended`, `ended_at` is stamped.
   - `calls.transcript` is now populated with the stitched text (space-joined segments).
   - The Calls list page still shows the call's transcript preview unchanged.
8. **Read the new endpoint.**
   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/v1/calls/<call-id>/transcript | jq
   ```
   Expected: `{ "session": {...}, "segments": [...] }` matching the rows from step 5.

## Meeting lifecycle

1. **Create a meeting** with `transcription_enabled=true`.
2. **Open it in two browsers** (different users; use an incognito window for the second).
3. **Both participants enable captions** in the room.
4. Verify two `transcript_sessions` rows exist, one per `speaker_user_id`:
   ```sql
   SELECT id, speaker_user_id, meeting_id, status FROM transcript_sessions
   WHERE meeting_id = '<meeting-id>';
   ```
5. **Both participants speak** in turn. Each user's captions appear in their own panel AND in the other's (relayed via the WS hub).
6. **Verify cross-user attribution.** Each segment's `speaker_user_id` matches the speaker; never the other participant.
   ```sql
   SELECT speaker_user_id, segment_index, left(text, 50) AS preview
   FROM transcript_segments
   WHERE meeting_id = '<meeting-id>'
   ORDER BY segment_started_at;
   ```
7. **Hit the meeting transcript endpoint.**
   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/v1/meetings/<code>/transcript | jq
   ```
   Expected: `sessions[]` lists both users; `segments[]` interleaved chronologically.
8. **End the meeting** (host clicks *End* or one participant leaves and the cleanup job fires). Both `transcript_sessions` rows should transition to `ended`.

## Failure modes

- **Spoofing attempt.** In a meeting, open the dev tools and manually send a `caption_chunk` over the WS with a `session_id` belonging to the other participant. Expected: backend log emits `TRANSCRIPT_SESSION_NOT_OWNED` (warn); the segment is dropped (no row inserted); the broadcast still fires (legacy behaviour). Other participants see the spoofed caption (relay-only) but nothing is persisted under the wrong user.
- **Browser refresh mid-session.** Refresh the page while captions are running. The browser's WS closes; ASR reaps after ~5 min; the cleanup job (next `cfg.Meeting.CleanupInterval` tick) flips status to `expired` and stitches the partial transcript into `calls.transcript`.
- **Duplicate POST.** Re-send the same `final` event twice (e.g. by clicking *Stop* and *Start* rapidly with overlapping timing). Expected: `finalized_segment_count` increments by 1, not 2 (UPSERT idempotency on `(session_id, segment_index)`).

## Backfill (one-shot, on first deploy only)

After running `make migrate-up` for the first time post-spec:

```sql
-- Count of legacy calls.transcript rows that should have been backfilled:
SELECT count(*) FROM calls WHERE transcript IS NOT NULL AND transcript <> '';
-- Synthetic legacy sessions inserted:
SELECT count(*) FROM transcript_sessions WHERE engine_mode = 'legacy';
-- These two should be equal.

-- Re-running the migration must NOT change either count (idempotent).
```
