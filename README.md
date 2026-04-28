# Rekall

> **The workspace where every call is remembered.**

Rekall is a self-hostable **call intelligence platform** — a place for teams to meet, talk, and later look back at what was said, decided, and agreed. Think of it as the space between a video-conferencing tool and a shared organizational memory: meetings happen *inside* Rekall, and the knowledge from them stays there.

---

## Why Rekall?

Most meetings evaporate the moment they end. Notes get scattered across chats, decisions get lost, and action items quietly go unowned. Teams repeat the same conversations because nobody can find what was said three weeks ago.

Rekall is built on a simple premise:

> **If the meeting happened in Rekall, the team shouldn't have to remember it — Rekall already does.**

That means real-time conversation *and* durable recall, in one place, owned by your organization.

---

## What you get

- 🏢 **A workspace.** Organizations, departments, invitations — a real team directory, not just a login screen.
- 🎥 **Video meetings that work.** Shareable join codes, peer-to-peer media, waiting-room knock flow, host controls. No client to install.
- 🙋 **A polished room.** Pre-meeting device check, virtual backgrounds, in-room chat, raise hand, emoji reactions, screen share.
- 📞 **Persistent call history.** Every meeting becomes a record in the workspace, ready for what comes next.
- 🔒 **Self-hosted by design.** Your video, your transcripts, your data — on infrastructure you control.
- 🧠 **A path to organizational memory.** Transcription, AI Notes, AI Ask, topic and sentiment classification, and observability dashboards are on the roadmap. See [`ROADMAP.md`](.docs/ROADMAP.md).

For the full picture, read the [Overview](.docs/OVERVIEW.md) and [Features](.docs/FEATURES.md).

---

## Who it's for

- **Small to mid-sized teams** who want a self-hosted alternative to *"Zoom + Notion + Otter"* stitched together.
- **Engineering-led companies** that prefer to own their data and run their own infrastructure.
- **Distributed teams** that need meetings to leave a trail behind them — decisions, action items, agreements.
- **Organizations in regulated spaces** (healthcare, legal, finance) where meeting recordings and transcripts must stay inside the org's perimeter.

If you're looking for a video-call tool that's *just* a video-call tool, Rekall is probably overkill. If you're looking for the memory layer of your organization, keep reading.

---

## Quickstart

Rekall runs anywhere Docker runs. Two paths: **basic** (postgres + backend +
frontend + mailpit, 2 min to first login) or **with live captions** (adds the
C++ ASR microservice, 10–15 min on first build).

### Prerequisites

| | Basic | + Live captions |
|---|---|---|
| Docker Desktop ≥ 24 (or Docker Engine + Compose v2) | ✅ | ✅ |
| Free RAM | 2 GB | 4 GB |
| Free disk | 3 GB | 6 GB |
| Free ports | `3000`, `8080`, `5432`, `8025` | + `8081`, `9091` |
| Git | ✅ | ✅ (submodule init) |
| Network access | ✅ | ✅ (downloads vcpkg deps + whisper.cpp + model) |

> **Windows users:** all commands work in PowerShell. Where the snippet shows
> `bash …`, run it from Git Bash or WSL — or use the Windows-native equivalent
> shown alongside.

---

## Path A — Basic install (no captions)

```bash
# 1. Clone
git clone <repo-url> rekall
cd rekall

# 2. Bootstrap the .env file (defaults work for local dev).
cp .env.example .env

# 3. Bring up postgres + backend + frontend + mailpit.
make up
# equivalent: docker compose up -d --build
```

That's it. The first run takes ~3 min while images build (mostly the Go
backend image).

### Open the app

| | URL |
|---|---|
| **Rekall** | http://localhost:3000 |
| **Mail inbox** (dev) | http://localhost:8025 |
| **API** | http://localhost:8080 |
| **API docs** (Swagger) | http://localhost:8080/docs |

Register the first account, create an organization, and you're in.

### Verify it's healthy

```bash
docker compose ps
# Every container should show "Up" (postgres + backend marked "(healthy)").

docker compose logs backend | grep SYS_SERVER_LISTENING
# → look for: "rekall API listening on :8080"
```

### Other commands

```bash
make down              # stop everything (containers + network; data persists)
make logs              # tail container logs
make restart           # down + up --build
make test              # backend + frontend tests (no docker needed)
make migrate-up        # apply pending DB migrations
make migrate-down      # roll back the most recent migration

# Pre-commit hook (auto-installed by `npm install` in frontend/):
#   - Frontend: prettier --write + eslint --fix on staged files; tsc --noEmit
#               when any *.ts/*.tsx is staged
#   - Go:       gofmt + goimports auto-fix on staged backend/*.go; whole-tree
#               go vet + golangci-lint + go build (requires goimports +
#               golangci-lint on $PATH — see backend/README.md for install)
# Bypass: `git commit --no-verify`, `HUSKY=0`,
#         `REKALL_PRE_COMMIT_SKIP_GO=1`, `REKALL_PRE_COMMIT_SKIP_FRONTEND=1`.
# CI re-runs the same checks.
# See [`frontend/README.md`](frontend/README.md#pre-commit-hook) and
# [`backend/README.md`](backend/README.md#pre-commit-hook) for details.

# Build commands
make build             # serial build of every basic-stack image (no asr)
make build-all         # PARALLEL build of every image including asr
make build-bake        # parallel build via docker buildx bake (best caching)
make build-asr         # build only the asr image (skip Go/JS rebuilds)
make up-asr            # build + start including the asr profile
```

> By default `docker compose build` builds services serially. Pass `--parallel`
> (or use `make build-all` / `make build-bake`) to compile every Dockerfile at
> once — much faster on multi-core machines, especially for the slow C++ asr
> image.

---

## Path B — Add live captions (ASR microservice)

The captions feature uses an optional **C++ microservice** powered by
[whisper.cpp](https://github.com/ggml-org/whisper.cpp). It's built into its
own container under a docker-compose profile, so the rest of Rekall is
completely unaffected if you skip it.

### One-time setup

```bash
# 1. Pull whisper.cpp source (~80 MB, single-shot).
git submodule update --init --recursive
# If the submodule is empty (clean clone), add it explicitly:
#   git submodule add https://github.com/ggml-org/whisper.cpp.git \
#       asr/third_party/whisper.cpp
#   git -C asr/third_party/whisper.cpp checkout v1.7.6  # any v1.7.* tag

# 2. Download the tiny.en model (~39 MB) — enough to smoke-test.
bash asr/scripts/download_models.sh tiny.en
# PowerShell equivalent (no bash needed):
#   New-Item -ItemType Directory -Force asr/models | Out-Null
#   Invoke-WebRequest -UseBasicParsing `
#     -Uri "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.en.bin" `
#     -OutFile asr/models/ggml-tiny.en.bin

# 3. Add ASR config to .env. The shared HS256 secret MUST be ≥ 32 bytes
#    and identical on both Go backend and asr container.
cat >> .env <<'EOF'
ASR_FEATURE_ENABLED=true
ASR_GRPC_ADDR=asr:9090
ASR_TOKEN_SECRET=<paste 32-byte hex string here>
ASR_WS_URL_BASE=ws://localhost:8081
ASR_TOKEN_DEFAULT_TTL=3m
ASR_TOKEN_MAX_TTL=5m
ASR_WS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
ASR_CIRCUIT_BREAKER_FAILURES=3
ASR_CIRCUIT_BREAKER_COOLDOWN=30s
EOF
```

Generate the shared secret:

```bash
openssl rand -hex 32
# PowerShell equivalent:
#   $b = New-Object byte[] 32
#   [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b)
#   ($b | ForEach-Object { $_.ToString("x2") }) -join ""
```

### Bring it up

```bash
make up-asr
# equivalent: docker compose --profile asr up -d --build
```

The first build is **slow** — 10–15 min — because it compiles whisper.cpp + a
full vcpkg dependency tree (gRPC, Boost, spdlog, jwt-cpp, prometheus-cpp). All
subsequent builds reuse the cached layers and complete in seconds.

> **Transcript persistence.** Every `final` ASR event is now stored in
> `transcript_sessions` + `transcript_segments` (text + per-word timings + the
> engine + model snapshot) so downstream features — summaries, action items,
> sentiment, search — can iterate without re-running ASR. The legacy
> `calls.transcript` column is kept as a denormalised cache, rebuilt at session
> end. The smoke checklist at
> [`backend/docs/smoke-transcript-persistence.md`](backend/docs/smoke-transcript-persistence.md)
> walks through the verification end-to-end.

If you want to build all images (basic + asr) at once in **parallel** rather
than letting compose serialise them service-by-service:

```bash
make build-all          # docker compose --profile asr build --parallel
# or, with BuildKit's bake driver (better cache reuse):
make build-bake         # docker buildx bake --file docker-compose.yml
docker compose --profile asr up -d   # then start what was built
```

### Verify ASR is healthy

```bash
# Quickest "is it up" check:
docker compose --profile asr ps --format "table {{.Name}}\t{{.Status}}"
# Healthy stack:
#   rekall_asr        Up 12 seconds (healthy)
#   rekall_backend    Up 30 seconds (healthy)
#   rekall_frontend   Up 30 seconds
#   rekall_mailpit    Up 30 seconds
#   rekall_postgres   Up 30 seconds (healthy)

# Confirm the C++ service finished startup:
docker compose --profile asr logs asr | grep ASR_SERVICE_READY

# Confirm the Go backend wired the gRPC client:
docker compose --profile asr logs backend | grep ASR

# Probe Prometheus from the host:
curl http://localhost:9091/metrics | grep asr_active_sessions
# → asr_active_sessions 0
```

### Use it

1. Log in at http://localhost:3000
2. Create a new meeting — **tick the "Live captions / transcription"
   toggle** on the create form
3. Open the meeting room → click **Start captions** in the right sidebar →
   allow the mic prompt → start talking
4. Partials appear in italic, finals in regular text

### Stop / rebuild

```bash
docker compose --profile asr down            # stop ASR + everything else
docker compose --profile asr build asr       # rebuild only the asr image
docker compose --profile asr up -d           # start without rebuilding
```

When `ASR_FEATURE_ENABLED=false` (the default in [.env.example](.env.example))
the backend returns `503 ASR_NOT_CONFIGURED` for any `*/asr-session` call and
the frontend captions UI hides itself. The `asr` profile being absent from a
plain `docker compose up` means the asr container is never even built unless
you explicitly opt in. See [`asr/README.md`](asr/README.md) for the full
architecture, security model, and debugging tips, and
[`.kiro/specs/asr-service/`](.kiro/specs/asr-service/) for the spec.

### Path B-cloud — captions without compiling whisper.cpp

If your machine is too slow to run `whisper.cpp` on CPU (or you just want to
validate the captions flow without paying the 5–15 minute cold-build cost),
the asr service ships a second engine that uploads each VAD-bounded segment
to OpenAI's `/v1/audio/transcriptions` endpoint via the vendored
[`olrea/openai-cpp`](https://github.com/olrea/openai-cpp) library. The image
is ~80 MB, builds in ~1 minute, and needs no model files on disk.

This is **operator-driven** — there is no UI to toggle the engine. The
choice lives in `.env` and (optionally) [`asr/config/config.yaml`](asr/config/config.yaml).
The frontend captions experience is identical aside from cloud mode emitting
finals only (no rolling partial inside a segment).

```bash
# 1. Pull only the openai-cpp submodule.
git submodule update --init -- asr/third_party/openai-cpp
# Fresh clone? Add it explicitly:
#   git submodule add https://github.com/olrea/openai-cpp.git \
#       asr/third_party/openai-cpp

# 2. Get a key at https://platform.openai.com/api-keys, then in .env set:
#       OPENAI_API_KEY=sk-...
#       ASR_ENGINE_MODE=openai
#    Optional, for the slim Docker image:
#       REKALL_ASR_ENGINE=openai
#       REKALL_ASR_IMAGE_TAG=openai

# 3a. Host build (fastest — ~1 min cold).
make asr-build-openai
make asr-run-openai

# 3b. OR: slim Docker image alongside the existing rekall-asr:latest.
REKALL_ASR_ENGINE=openai REKALL_ASR_IMAGE_TAG=openai \
  docker compose --profile asr up -d --build asr
```

Production safety: the asr binary refuses to start with `ASR_ENGINE_MODE=openai`
when `SERVER_ENV=production` unless `ASR_ALLOW_OPENAI_IN_PRODUCTION=true` is
also set. A boot-time `ASR_DATA_LEAVES_HOST` warning is logged whenever cloud
mode is active so audit pipelines can flag pods that send audio off-host.

Switching back to local: in `.env` set `ASR_ENGINE_MODE=local`,
`REKALL_ASR_ENGINE=both`, `REKALL_ASR_IMAGE_TAG=latest`, then
`docker compose --profile asr up -d`. See
[`asr/README.md` § Engine modes](asr/README.md#engine-modes) for the full
reference and [`asr/docs/smoke-openai-mode.md`](asr/docs/smoke-openai-mode.md)
for the manual smoke checklist.

---

## Path C — Local backend / ASR container (developer loop)

When iterating on Go code, skip the backend image rebuild by running
`go run` on the host while keeping postgres + asr in Docker:

```bash
# 1. Supporting services in Docker
docker compose up -d postgres mailpit
docker compose --profile asr up -d --build asr     # only if you want captions

# 2. Backend on the host
cd backend
export ASR_FEATURE_ENABLED=true                        # only if asr is up
export ASR_GRPC_ADDR=127.0.0.1:9090
export ASR_TOKEN_SECRET=$(grep '^ASR_TOKEN_SECRET=' ../.env | cut -d= -f2)
export ASR_WS_URL_BASE=ws://localhost:8081
go run ./cmd/server

# 3. (Optional) Frontend on the host with HMR
cd frontend
npm install
npm run dev      # Vite at http://localhost:5173
```

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `make up` fails with port conflicts | Another process is on `3000`, `8080`, `5432`, `8025`, `8081`, or `9091`. Stop it or change the host-side port in `docker-compose.yml`. |
| Backend stuck on DB connection | Postgres is still warming up. `docker compose ps` should show postgres `(healthy)` first. |
| `FATAL [SYS_CONFIG_INVALID]` on backend startup | A required env var is missing. Check the message; common one is `ASR_TOKEN_SECRET` < 32 bytes when `ASR_FEATURE_ENABLED=true`. |
| `error getting credentials - err: exec: "docker-credential-desktop"` | Docker Desktop's bin dir isn't on your shell PATH. Run from the Docker Desktop terminal, or `$env:PATH = "C:\Program Files\Docker\Docker\resources\bin;$env:PATH"` in PowerShell. |
| `git submodule add -b v1.7.6 …` fails with `'origin/v1.7.6' is not a commit` | `v1.7.6` is a tag, not a branch. Drop `-b` and check out the tag manually inside the submodule. See [`.gitmodules.example`](asr/.gitmodules.example). |
| ASR container restarts in a loop with `ASR_MODEL_LOAD_FAILED file not found` | Models aren't mounted. The compose file mounts `./asr/models:/var/lib/rekall-asr/models:ro`. Confirm the file exists on the host: `ls asr/models/`. |
| Captions panel never appears in a meeting | The host didn't tick the toggle when creating that meeting. Per-meeting opt-in is required even when ASR is enabled globally. |
| Captions panel shows but Start fails with `503 ASR_NOT_CONFIGURED` | The backend isn't seeing `ASR_FEATURE_ENABLED=true`. Restart the backend container after editing `.env`: `docker compose restart backend`. |
| WS close 4401 in browser console | `ASR_TOKEN_SECRET` mismatch between Go and ASR (or one is shorter than 32 bytes). Both must read the same `.env`. |
| WS close 4503 | Worker pool saturated. Bump `ASR_WORKER_POOL_SIZE` (default `min(8, cpu)`). |

---

## Organizations and scopes

Rekall organises work along two axes: **who** (account → organization → department) and **where the item lives** (an item's *scope*).

### Hierarchy

```
Account                        — your individual user
  └─ Organization              — a workspace of teammates (e.g. "Acme")
        └─ Department          — a sub-team within the org (e.g. "Engineering")
```

A user may belong to multiple organizations, and to multiple departments within the same org.

### Scope

Every meeting and call has a **scope**, which is one of:

- **Open** — not attached to any team. Visible to the host and the participants only. Meetings created from the Recall page default to Open.
- **Organization** — attached to an org. Visible to every member of that org.
- **Department** — attached to a specific department within an org. Visible to every member of that dept.

### URL grammar

The hierarchy maps directly onto routes — every level is bookmarkable:

| | |
|---|---|
| `/dashboard` | Personal landing |
| `/meetings`, `/calls` | Flat lists across all the items you can see, with a Scope filter chip |
| `/organizations` | List of orgs you belong to |
| `/organizations/:id` | Org detail with tabs: Overview / Departments / Meetings / Calls |
| `/organizations/:id/meetings` | Org-scoped meetings list |
| `/organizations/:id/calls` | Org-scoped calls list |
| `/organizations/:orgId/departments/:deptId` | Department detail with tabs: Overview / Meetings / Calls |
| `/organizations/:orgId/departments/:deptId/meetings` | Dept-scoped meetings list |
| `/organizations/:orgId/departments/:deptId/calls` | Dept-scoped calls list |

The flat lists understand a `?scope=` query parameter (`open`, `org:<uuid>`, or `dept:<orgUuid>:<deptUuid>`) so any filter is shareable.

The TopBar's **Org Switcher** lets you jump between Personal and any org you're a member of.

---

## Platform administration

Some operations — most notably *creating an organization* — are restricted to **platform admins**. Platform admins are declared via environment variables, not via in-app promotion, so the admin list is a deployment concern.

### Role hierarchy

| Role | Granted by | Capabilities |
|---|---|---|
| **Platform admin** | `PLATFORM_ADMIN_EMAILS` env var | Create orgs (optionally on behalf of any user via `owner_email`); intervene on any org or department as if they held the highest role |
| **Org owner / admin** | `OrgMembership.role = owner \| admin` | Invite members, create/rename/delete departments, assign department heads |
| **Department head** | `DepartmentMembership.role = head` | Add/remove members of *their* department; cannot rename or delete the department, cannot create new departments |
| **Member** | Default | Read meetings and calls in scopes they belong to |

### Environment variables

| Var | Description |
|---|---|
| `PLATFORM_ADMIN_EMAILS` | Comma-separated, lowercased emails. On every server boot the listed users are promoted to `role=admin`; any current admin not on the list is demoted to `member`. |
| `PLATFORM_ADMIN_BOOTSTRAP_PASSWORD` | Optional. When set, missing admin users are auto-created on first boot with this password. Subsequent boots do **not** re-apply it — rotation goes through the normal password-reset flow. |

The reconciliation runs once on startup before the HTTP server begins accepting requests; the `created`/`promoted`/`demoted` counts are logged for ops visibility.

---

## Project status

Rekall is in **active early development**. The meeting and workspace layers are usable today; the recall and AI layers — transcription, summaries, AI Ask, dashboards — are what come next.

See the [Roadmap](.docs/ROADMAP.md) for the path forward.

---

## Documentation

| | |
|---|---|
| 🎯 [Overview](.docs/OVERVIEW.md) | What Rekall is, why it exists, and who it's for |
| ✨ [Features](.docs/FEATURES.md) | What you can do with Rekall today |
| 🗺️ [Roadmap](.docs/ROADMAP.md) | Where Rekall is headed next |

---

## License

See [LICENSE](LICENSE).
