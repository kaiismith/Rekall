# Rekall ASR Service

A standalone C++20 microservice that performs real-time speech-to-text using
[whisper.cpp](https://github.com/ggml-org/whisper.cpp).

- **WebSocket data plane** — browsers stream 16 kHz mono PCM in, receive JSON
  transcript events out (`partial`, `final`, `info`, `error`).
- **gRPC control plane** — the Rekall Go backend manages session lifecycle,
  health, and model registry over mTLS.
- **Auth** — short-lived single-use JWT minted by the Go backend after it has
  registered the session, presented as `?token=<jwt>` on the WS upgrade.

## Build

```bash
git submodule update --init --recursive
make asr-build
```

Requires CMake ≥ 3.22, a C++20 compiler (gcc-12 / clang-15 / MSVC-2022), and
`vcpkg` for the third-party dependencies declared in `vcpkg.json`.

### Engine modes

The service ships two transcription back-ends, selected at **build time** by
the `REKALL_ASR_ENGINE` CMake / Docker build-arg:

| Build flag           | Image tag           | Compile-in deps                | Cold build  |
|----------------------|---------------------|--------------------------------|-------------|
| `both` (default)     | `rekall-asr:latest` | whisper.cpp + libcurl + openai-cpp | 5–15 min |
| `local`              | (no published tag)  | whisper.cpp                    | 5–15 min    |
| `openai`             | `rekall-asr:openai` | libcurl + olrea/openai-cpp     | **~1 min**  |

The same binary then chooses which engine to *run* at process start via the
`ASR_ENGINE_MODE=local|openai` env var. `mode=openai` requires `OPENAI_API_KEY`
to be exported and refuses to start in `SERVER_ENV=production` unless
`ASR_ALLOW_OPENAI_IN_PRODUCTION=true` is also set (audit signal:
`ASR_DATA_LEAVES_HOST` log event).

The cloud engine is **one-shot** — no `partial` events, only `final`s emitted
when a VAD silence boundary or `flush` closes a segment.

#### One-time setup — choosing your mode

The frontend has no engine selector; mode is operator-driven via config. The
two pieces of state you need to align are:

1. **Build-time** — `REKALL_ASR_ENGINE` (CMake flag / Dockerfile build-arg /
   `docker-compose.yml` build-arg). Decides which engine code is compiled in.
2. **Runtime** — `ASR_ENGINE_MODE` (env var / `engine.mode` in YAML). Decides
   which engine the running binary actually uses. Must be a subset of what
   was compiled in at build time, otherwise startup fails fast with
   `ASR_CONFIG_INVALID` naming the rebuild flag.

##### `local` (production default — what you already have)

```bash
git submodule update --init -- asr/third_party/whisper.cpp
bash asr/scripts/download_models.sh tiny.en
# .env entries (already present after a fresh clone):
#   REKALL_ASR_ENGINE=both
#   ASR_ENGINE_MODE=local
make asr-build         # or:  make up-asr  (docker)
```

##### `openai` (developer accelerant for slow CPUs)

```bash
# 1. Pull only the openai-cpp submodule (~30 s, no whisper.cpp clone).
git submodule update --init -- asr/third_party/openai-cpp
# If the submodule entry is missing on a fresh clone, add it explicitly:
#   git submodule add https://github.com/olrea/openai-cpp.git \
#       asr/third_party/openai-cpp

# 2. Get an API key from https://platform.openai.com/api-keys
#    and paste it into .env (the variable already exists, just empty):
#       OPENAI_API_KEY=sk-...
#
#    Then flip the runtime mode in the same .env:
#       ASR_ENGINE_MODE=openai
#
#    On Windows PowerShell, use Set-Content / notepad. On bash:
sed -i 's/^OPENAI_API_KEY=$/OPENAI_API_KEY=sk-paste-yours-here/' .env
sed -i 's/^ASR_ENGINE_MODE=local$/ASR_ENGINE_MODE=openai/'        .env

# 3a. Local host build (fastest path — ~1 min, no Docker).
make asr-build-openai
make asr-run-openai

# 3b. OR: Docker — produces a slim rekall-asr:openai image (~80 MB).
#     The two compose env vars are read by docker-compose.yml.
REKALL_ASR_ENGINE=openai REKALL_ASR_IMAGE_TAG=openai \
  docker compose --profile asr up -d --build asr
```

##### Switching back to `local`

```bash
sed -i 's/^ASR_ENGINE_MODE=openai$/ASR_ENGINE_MODE=local/' .env
# If you ran the cloud-only Docker image, also revert the build-arg pair:
#   REKALL_ASR_ENGINE=both REKALL_ASR_IMAGE_TAG=latest
docker compose --profile asr up -d
```

#### Fast onboarding path (cloud-only build, summary)

For new contributors / CPU-only laptops where the whisper.cpp compile is
prohibitive:

```bash
export OPENAI_API_KEY=sk-...
make asr-build-openai     # ~1 min cold; no whisper, no models
make asr-run-openai
# open the calls page, click Start captions, speak — finals appear in ~2 s.
```

The smoke checklist lives at [`docs/smoke-openai-mode.md`](docs/smoke-openai-mode.md).

#### `olrea/openai-cpp` upgrade procedure

1. Bump the submodule to the new commit SHA.
2. Verify the upstream LICENCE is still MIT.
3. Run the secret-leak unit test (`unit/openai_client_no_secret_leak_test`).
4. Update the pinned SHA in `.gitmodules`.

## Run

```bash
./scripts/download_models.sh tiny.en small.en
export ASR_TOKEN_SECRET=$(openssl rand -hex 32)
./build/rekall-asr --config config/config.example.yaml
```

## Test

```bash
make asr-test          # unit + integration
make asr-load CONCURRENCY=8   # opt-in load benchmark
```

## Layout

```
asr/
├── proto/              # rekall.asr.v1 — gRPC contract
├── include/            # public headers
├── src/                # implementation
├── third_party/
│   └── whisper.cpp/    # vendored submodule, pinned tag
├── tests/              # gtest unit + integration + load
├── docker/             # multi-stage Dockerfile → distroless
├── scripts/            # download_models.sh, gen_dev_certs.sh, format.sh
└── config/             # config.example.yaml
```

## Architecture (1-paragraph summary)

The service has three concurrent subsystems coordinated by a `SessionManager`.
The **WS server** (Boost.Beast) accepts browser audio frames into a per-session
inbound `RingBuffer` and drains a per-session outbound queue back as JSON text
frames. The **gRPC control plane** services `StartSession`/`EndSession`/`Health`/
`ReloadModels` calls from the Go backend over loopback (mTLS required when
binding off loopback). A **WorkerPool** of `std::jthread`s pulls inbound frames,
runs `whisper.cpp` over a sliding window, emits throttled `partial` events at
the configured cadence, and emits a single `final` event per VAD-detected
segment or on client `flush`. Authentication is performed once at WS upgrade by
`JWTValidator` (HS256 + single-use jti cache); admission is enforced both at
gRPC `StartSession` and at WS upgrade time. Graceful shutdown flips Health to
NOT_SERVING, stops accepting new sessions, gives existing sessions a configurable
drain budget, then force-closes the rest.

## Configuration

YAML at `config/config.example.yaml`; every key may be overridden by an env var
named `ASR_<DOTTED_PATH_UPPERCASED_AND_UNDERSCORE_JOINED>` (e.g.
`worker_pool.size` ↔ `ASR_WORKER_POOL_SIZE`). The full list of env vars and
required-vs-optional matrix is in [`.kiro/specs/asr-service/requirements.md`](../.kiro/specs/asr-service/requirements.md)
§12.

## Security model

- Browser ↔ ASR is **WebSocket only**; the gRPC port defaults to loopback and
  refuses to bind off loopback without mTLS.
- The Session_Token is single-use, ≤ 5 min, bound to `{user_id, call_id, session_id}`,
  and never appears in logs (only the 8-char prefix).
- Origin is validated against the explicit `ws_allowed_origins` allow-list; the
  dev wildcard only applies when `SERVER_ENV=development`.
- Audio frames are never logged at any level.
- Drop-privileges support on Linux via `ASR_DROP_PRIVS_TO=uid:gid`.

## Debugging tips

- Set `ASR_LOG_LEVEL=debug` to see `ASR_FRAME_RECEIVED` and `ASR_PARTIAL_EMITTED`
  events. They're noisy by design — keep this OFF in production.
- `grpc_health_probe -addr=127.0.0.1:9090` returns SERVING/NOT_SERVING; useful
  for reproducing load-balancer drain races.
- `curl localhost:9091/metrics` exposes the Prometheus surface; key gauges:
  `asr_active_sessions`, `asr_worker_pool_in_use`, `asr_dropped_partials_total`.
