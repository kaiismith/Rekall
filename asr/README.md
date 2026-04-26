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
