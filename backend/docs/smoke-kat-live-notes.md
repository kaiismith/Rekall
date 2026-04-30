# Smoke Checklist — Kat Live Notes

This is the manual verification path for the Kat live-notes feature. NOT
executed in CI — operators run this once per deploy mode (api-key / managed
identity) when the spec ships and after material changes.

See [`.kiro/specs/kat-live-notes/`](../../.kiro/specs/kat-live-notes/) for
the requirements / design / tasks. Notes are intentionally **ephemeral** —
nothing Kat produces is written to the database; the smoke includes a
`psql -c "\dt kat_*"` check to confirm.

## Prerequisites

- A running Rekall stack (backend, frontend, postgres) with the ASR service
  enabled and at least one user able to sign in.
- An Azure AI Foundry (Azure OpenAI) deployment of a chat-completions model
  reachable from the backend host. v1 was developed against `gpt-4o-mini`.
- For the API-key path: a working API key with permission to call that
  deployment.
- For the managed-identity path: either `az login` on the developer machine
  OR a workload identity / managed identity assigned to the backend with
  the `Cognitive Services OpenAI User` role on the Foundry resource.

## Path A — API-key auth

1. Set in `backend/.env`:
   ```
   KAT_ENABLED=true
   KAT_FOUNDRY_ENDPOINT=https://<your-foundry>.openai.azure.com
   KAT_FOUNDRY_DEPLOYMENT=gpt-4o-mini
   KAT_FOUNDRY_API_VERSION=2024-08-01-preview
   KAT_FOUNDRY_API_KEY=<paste-the-key>
   ```
2. Restart the backend. **Watch the boot log** — exactly one
   `KAT_FOUNDRY_INITIALIZED` line should appear with:
   - `auth_mode=api_key`
   - `endpoint_host=<your-foundry>.openai.azure.com` (host only, no path)
   - `deployment=gpt-4o-mini`
   - The API key value MUST NOT appear anywhere in the log.
3. Probe the health endpoint:
   ```
   curl -s http://localhost:8080/healthz/kat | jq
   ```
   Expect:
   ```
   { "configured": true, "auth_mode": "api_key",
     "deployment": "gpt-4o-mini",
     "endpoint_host": "<your-foundry>.openai.azure.com" }
   ```
4. Sign in to the frontend in two browsers (User A and User B). Have User A
   create a meeting and User B join via the meeting code.
5. Both users open captions; speak for 90 seconds with normal cadence.
   - **Within ~25 s of the first finalized segment** the Kat panel on both
     clients should flip from "Kat is listening…" to a rendered note.
   - The summary should reference what was actually said (not generic
     filler). The "Notes are not saved — they live only during this meeting"
     hint is visible.
   - Subsequent ticks every ~20 s should refresh the note in place.
6. Sit silent for 40 seconds. Confirm:
   - No new note appears (no-op tick — expected when fewer than 2 new
     finalized segments accumulate).
   - The previous note stays visible.
7. Open a third tab, join the meeting, observe:
   - The Kat panel briefly shows "Kat is listening…" then immediately
     receives the recent in-memory ring-buffer notes via the late-join
     replay (same `kat.note` WS channel as live updates).
8. **No-persistence sanity check**:
   ```
   psql -c "\dt kat_*" "$(grep DB_URL .env)"
   ```
   Expect `Did not find any relation named "kat_*"`. If any table appears,
   STOP — a regression has snuck Kat persistence in.
9. Kill the backend mid-meeting. Reconnect both clients. Observe:
   - Kat panel returns to "Kat is listening…" until the next successful
     tick produces a fresh note. There is no error toast, no red state.

## Path B — DefaultAzureCredential (managed identity / az login)

1. Edit `backend/.env`:
   ```
   KAT_FOUNDRY_API_KEY=
   ```
   (everything else from Path A unchanged.)
2. On a developer machine: `az login`. On a deployed environment: ensure
   the workload / managed identity has the `Cognitive Services OpenAI User`
   role on the Foundry resource.
3. Restart the backend. The boot log should now show
   `auth_mode=managed_identity` (no other change).
4. Re-run steps 3–9 above. Behaviour should be identical from the user's
   perspective — only the authentication mechanism differs at the seam.

## Failure-mode spot checks

- **Foundry timeout**: temporarily set `KAT_FOUNDRY_REQUEST_TIMEOUT_MS=100`,
  restart, talk in a meeting. Expect `KAT_FOUNDRY_TIMEOUT` warn logs and the
  panel staying on the previous note (no error toast, no red state).
- **Foundry unconfigured**: clear `KAT_FOUNDRY_ENDPOINT`, restart. Boot log
  should emit `KAT_FOUNDRY_UNCONFIGURED` (warn). `/healthz/kat` returns
  `{configured: false, auth_mode: "none"}`. The frontend Kat panel renders
  the offline card with the operator hint.
- **KAT_ENABLED=false**: same as unconfigured but no Foundry construction
  is attempted at all; the scheduler is never started. `/healthz/kat`
  returns `configured=false`. Meetings work normally.

## What this checklist does NOT cover (intentional)

- **Long-term cost monitoring** — the cohort + concurrency cap design keeps
  Foundry call volume bounded but actual per-day spend depends on meeting
  load; track it via your Azure cost dashboard, not via this checklist.
- **Cross-process state** — Kat is single-instance by design; HA / multi-pod
  deployments would need a separate cohort coordinator (out of scope, see
  Requirement 10.10).
- **Persisting Kat history** — explicitly out of scope. The transcript
  itself is the durable record; intellikat-service produces the post-call
  durable summary as a separate workstream.
