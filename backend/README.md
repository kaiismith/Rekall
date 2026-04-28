# Rekall — Backend

Go API server (Gin) backing the Rekall call-intelligence platform. Owns
authentication, organizations / departments, meetings, calls, the WebRTC
signalling plane, and the gRPC bridge to the C++ ASR microservice.

For the platform overview, deployment paths, and full feature list, see the
root [`README.md`](../README.md).

---

## Local development

### Prerequisites

| | Version |
|---|---|
| Go | ≥ 1.25 (matches `go.mod` toolchain directive) |
| `golang-migrate` | any recent (`brew install golang-migrate` / `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`) |
| Postgres | 16 (Docker recommended — see root README's "Path A") |
| `goimports` | `go install golang.org/x/tools/cmd/goimports@latest` |
| `golangci-lint` | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0` |

The last two are required by the **pre-commit hook** (see below). Without them
on `$PATH`, the hook will refuse to run and tell you which install command to
copy-paste.

### Run

```bash
cd backend

# 1. Bring up Postgres (any way you like — Docker is easiest):
docker compose up -d postgres

# 2. Apply migrations:
make migrate-up

# 3. Run the server:
go run ./cmd/server
# or: make run
```

The server listens on `:8080` by default. See [`.env.example`](../.env.example)
for the full env-var contract.

---

## Make targets

| Command | What it does |
|---|---|
| `make build` | Regenerate Swagger docs, then build `bin/rekall-server` |
| `make run` | `make build` + run the binary |
| `make test` | `go test -v -race -count=1 ./tests/unit/...` |
| `make test-integration` | Same flags against `./tests/integration/...` |
| `make test-all` | Run every Go test under `./...` |
| `make test-coverage` | Coverage profile + HTML report at `coverage.html` |
| `make test-hook` | Integration test for the Go pre-commit hook (Linux/macOS only) |
| `make lint` | `golangci-lint run ./...` (matches the pre-commit hook) |
| `make fmt` | `gofmt -w .` + `goimports -w .` over the whole tree |
| `make vet` | `go vet ./...` |
| `make tidy` | `go mod tidy` |
| `make docs` | Regenerate Swagger from handler annotations |
| `make docs-check` | Fail if committed `docs/` is stale vs. annotations |
| `make migrate-up` | Apply pending migrations |
| `make migrate-down` | Roll back the most recent migration |
| `make clean` | Remove `bin/` and coverage artefacts |

---

## Pre-commit hook

A local git hook runs on every `git commit` and, when any `backend/**/*.go`
file is staged, performs:

1. **Toolchain preflight** — verifies `gofmt`, `goimports`, `golangci-lint`
   are on `$PATH`. Missing tools fail loudly with the install command.
2. **Auto-format** — `gofmt -w` and `goimports -w -local github.com/rekall/backend`
   on each staged file. Modifications are silently re-staged.
3. **Vet** — `go vet ./...` against the whole `backend/` tree.
4. **Lint** — `golangci-lint run ./...` using [`.golangci.yml`](./.golangci.yml).
5. **Build** — `go build ./...` to confirm the project still compiles.

Auto-generated files are excluded: `backend/docs/` (Swagger output) and
`*.pb.go` (protobuf).

The hook is wired through [husky](https://typicode.github.io/husky/) — the
same `.husky/pre-commit` script that the [`frontend/`](../frontend/) hook uses.
Wall-clock cost on a typical Go-only commit is ~5–6 seconds (warm cache).

### Installation

The hook is installed automatically the first time you run `npm install` from
[`frontend/`](../frontend/). The Go-side install is just the two `go install`
commands from the prerequisites table above. There is no Go-specific install
script — the hook's `.husky/` directory is shared with the frontend.

Verify the hook is wired:

```bash
git config --get core.hooksPath        # should print: .husky/_
ls ../.husky/_                         # should list pre-commit + helpers
```

If `core.hooksPath` is empty, run `npm install` from `frontend/` once to
trigger the install.

### `golangci-lint` version pin

The pre-commit hook and CI MUST use the same `golangci-lint` version, otherwise
local-pass / CI-fail (or vice-versa) is possible. The recommended pin is:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
```

Bumping the pin requires:
1. Update this README's install command
2. Update CI's install step (if a workflow file exists)
3. Verify `make lint` still passes against the whole `backend/` tree

### Bypassing the hook

The hook is a developer convenience, not an enforcement gate. CI re-runs every
check, so a bypass on `main` will still be caught — bypass is for handoffs and
emergency commits, not the steady state.

| How | When |
|---|---|
| `git commit --no-verify` (or `-n`) | Skip the entire hook for this commit |
| `HUSKY=0 git commit ...` | Same as above, via env var |
| `REKALL_PRE_COMMIT_SKIP_GO=1 git commit ...` | Skip ONLY the Go block; frontend block still runs |
| `REKALL_PRE_COMMIT_SKIP_FRONTEND=1 git commit ...` | Skip ONLY the frontend block; Go block still runs |

The Dockerfile sets `ENV HUSKY=0` (defensive — the Go image build does not run
`npm install` so husky is never invoked, but the env var documents intent).

### Manual smoke test

```bash
# From repo root, with Go tools installed.
cat > backend/cmd/server/bad.go <<'EOF'
package main

import "fmt"

func Bad() { fmt.Printf("%s", 42) }
EOF

git add backend/cmd/server/bad.go
git commit -m "smoke"
# Expected: hook fires, vet reports the format/arg mismatch, commit blocked.

# Cleanup
git restore --staged backend/cmd/server/bad.go
rm backend/cmd/server/bad.go
```

### Troubleshooting

| Symptom | Fix |
|---|---|
| `pre-commit: golangci-lint not found on $PATH` | Run the install command from the prerequisites table; verify with `command -v golangci-lint` |
| `pre-commit: goimports not found on $PATH` | `go install golang.org/x/tools/cmd/goimports@latest` |
| Hook does not fire after fresh clone | Run `npm install` from `frontend/` once to trigger the husky `prepare` script. Verify `git config --get core.hooksPath` prints `.husky/_` |
| Local lint passes, CI fails (or vice-versa) | Version skew. Confirm your local `golangci-lint version` matches the CI install pin |
| First commit after `git pull` is slow | `golangci-lint`'s cache invalidates on a `go.sum` change. Subsequent commits are fast again |
| Hook reports vet/lint error in a file you did not touch | A staged change broke an unstaged importer — that is the point of whole-tree scope. Fix the importer or revert the breaking change |
| Hook is too slow | Use `--no-verify` for the immediate commit; report friction so the budget can be tuned |

---

## Project layout

```
backend/
├── cmd/server/           # main package — composition root + Gin router
├── internal/
│   ├── application/      # use-cases (handlers' service layer)
│   ├── domain/           # entities + ports (interfaces)
│   ├── infrastructure/   # adapters: postgres, asr-grpc, websocket, etc.
│   └── interfaces/http/  # Gin handlers, middleware, DTOs
├── pkg/                  # reusable packages (no internal/ dependency)
├── migrations/           # golang-migrate SQL files
├── tests/
│   ├── unit/             # fast tests, no DB / no network
│   └── integration/      # need a running Postgres
├── docs/                 # auto-generated Swagger output (do NOT edit by hand)
├── .golangci.yml         # lint config (single source of truth)
├── go.mod / go.sum
├── Dockerfile
└── Makefile
```

---

## Database

Migrations live under [`migrations/`](./migrations/) and are managed with
[`golang-migrate`](https://github.com/golang-migrate/migrate). The
[`Makefile`](./Makefile) wraps the common operations:

```bash
make migrate-up              # apply all pending migrations
make migrate-down            # roll back the most recent migration
make migrate-drop            # drop everything (DESTRUCTIVE — local only)
```

The `DB_URL` make variable defaults to the local-dev URL; override for staging
test runs:

```bash
DB_URL="postgres://user:pass@host:5432/db?sslmode=require" make migrate-up
```

---

## Swagger / OpenAPI

Handler annotations are processed by [`swag`](https://github.com/swaggo/swag)
into [`docs/swagger.json`](./docs/swagger.yaml) on every `make build`. The
generated file is committed so CI does not need a working `swag` toolchain.

If you change a handler annotation, run `make docs` and commit the result.
`make docs-check` is the CI guard that fails when the committed output drifts
from the source annotations.
