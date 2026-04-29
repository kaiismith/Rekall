# Rekall — Intellikat

Standalone Python 3.12 microservice that derives **per-segment sentiment** and
**session-level summarization** from transcripts produced by the ASR pipeline.

Triggered by lightweight reference messages on Azure Service Bus (the message
carries a `transcript_session_id` only — never transcript text), reads the
canonical transcripts from the shared Postgres DB, runs a Hugging Face BERT
model for sentiment (`MarieAngeA13/Sentiment-Analysis-BERT` — hosted via
the HF Inference API or loaded locally), runs Foundry AI for summarization,
and writes results to two intellikat-owned tables.

## Architecture in one diagram

```
   browser ──▶ ASR ──▶ backend ──▶ Postgres (transcript_sessions, transcript_segments)
                          │
                          └─▶ Service Bus topic "rekall.transcript.insights"
                                    │  (reference message: session_id only)
                                    ▼
                              intellikat (this service)
                                    │  reads segments from Postgres
                                    │  sentiment per segment ─▶ HF (hosted or local)
                                    │  summary  per session ──▶ Foundry AI
                                    ▼
                              Postgres (transcript_segment_sentiments, transcript_session_summaries, intellikat_jobs)
```

Hexagonal / clean architecture — see [config/importlinter.toml](config/importlinter.toml)
for the contracts CI enforces:

```
domain/        # entities (one file per aggregate), value objects, port Protocols
application/   # use cases — depend on domain only
infrastructure/# adapters: Postgres, HF (hosted + local), Foundry, Service Bus, logging
interfaces/    # FastAPI health endpoints + the worker entrypoint
```

For the full spec see [`.kiro/specs/intellikat-service/`](../.kiro/specs/intellikat-service/).

## Quick start (dev)

```bash
cd intellikat

# 1. install deps
uv sync                         # hosted-mode
# OR
uv sync --extra local           # local-mode (~3 GB: torch + tokenizers + accelerate)

# 2. configure
cp .env.example .env
# Fill in at minimum:
#   INTELLIKAT_DATABASE_URL
#   INTELLIKAT_HF_TOKEN  (when HF_MODE=hosted)
#   INTELLIKAT_FOUNDRY_ENDPOINT + INTELLIKAT_FOUNDRY_API_KEY
#   INTELLIKAT_SERVICEBUS_CONNECTION_STRING (or _FULLY_QUALIFIED_NAMESPACE)

# 3. apply migrations (once per fresh DB)
uv run alembic -c migrations/alembic.ini upgrade head

# 4. run the worker
uv run intellikat --mode worker
```

## Modes

- `--mode worker` (default) — run the Service Bus consumer loop
- `--mode migrate` — apply Alembic migrations and exit (Kubernetes init-container)
- `--mode health-check` — run readiness checks once and exit 0/1

## Hugging Face execution modes

| Mode | Image size | When |
|---|---|---|
| `hosted` (default) | ~150 MB | Dev, low volume, no torch |
| `local` | ~3 GB | High volume, data-residency, GPU pods |

Switch with `INTELLIKAT_HF_MODE`. Hosted requires `INTELLIKAT_HF_TOKEN`;
local requires `uv sync --extra local` (or build the `intellikat-local`
Docker target).

## DB permissions

Intellikat only INSERTs into its own three tables — see [scripts/grant_permissions.sql](scripts/grant_permissions.sql).
The shared `transcript_sessions` / `transcript_segments` tables are
read-only via this user.

## Debugging

- Audit row for every job: `SELECT * FROM intellikat_jobs ORDER BY started_at DESC LIMIT 20;`
- Re-process one session manually: `python scripts/enqueue_reprocess.py <session_id>`
- Hand-publish a synthetic message: `python scripts/enqueue_test_message.py <session_id>`
- Health probes: `curl http://127.0.0.1:8090/healthz` and `/readyz`

## Tests

```bash
uv run pytest tests/unit                                  # always fast
INTELLIKAT_BUILD=1 uv run pytest tests/integration        # requires Docker (testcontainers)
```

Lints: `uv run ruff check`, `uv run ruff format --check`, `uv run mypy src`,
`uv run lint-imports --config config/importlinter.toml`.

## Out of scope (v1)

See [Requirement 19](../.kiro/specs/intellikat-service/requirements.md#requirement-19-out-of-scope-v1) — frontend rendering,
APM/metrics, streaming sentiment, action-item extraction, and per-tenant
prompt customisation are deferred.
