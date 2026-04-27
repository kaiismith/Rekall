# Smoke checklist — `ASR_ENGINE_MODE=openai`

Manual smoke test for the cloud engine path. Run this before merging any change
that touches `OpenAiHttpClient`, the openai-cpp pinned commit, or the engine
factory.

Not executed in CI — every test that exercises the cloud engine in CI uses
`FakeOpenAiClient` instead.

## Prerequisites

- `OPENAI_API_KEY` exported in your shell.
- The frontend running locally (`make frontend-dev` or via `docker compose up frontend`).
- The Go backend running locally with `ASR_FEATURE_ENABLED=true`.

## 1. Build the cloud-only binary

```bash
make asr-build-openai
```

Expected: completes in ~1 minute on a fresh checkout (no whisper.cpp compile,
no model download). The build directory is `asr/build-openai/`.

## 2. Run it

```bash
make asr-run-openai
```

Expected log lines (JSON):

- `ASR_ENGINE_SELECTED` with `"mode": "openai"` and `"target": "https://api.openai.com/v1"`.
- `ASR_DATA_LEAVES_HOST` (warn) — the audit signal you want for grep dashboards.
- `ASR_SERVICE_READY`.

## 3. Open the calls page and start captions

1. Open `http://localhost:5173/calls/<some-call>` in a browser.
2. Click **Start captions**.
3. Speak a short sentence (e.g. "hello world").

Expected:

- The captions panel renders a small **Cloud** chip.
- After ~1–2 seconds, the spoken text appears as a finalised segment (no
  rolling partial — the cloud engine is one-shot).
- A "Listening…" placeholder shows between segments.

## 4. Verify the audit signals

```bash
grep ASR_OPENAI_REQUEST_OK <log-stream>
grep ASR_DATA_LEAVES_HOST  <log-stream>
```

Both should appear at least once. The `ASR_OPENAI_REQUEST_OK` lines carry
`session_id`, `model`, and `request_duration_ms` — confirm the API key is
NOT present in any of them.

## 5. Tear down

`Ctrl+C` the binary. Confirm `ASR_GRACEFUL_DRAIN_BEGIN` and
`ASR_GRACEFUL_DRAIN_END` log lines.

## Common failures

| Symptom | Likely cause |
|---|---|
| `ASR_CONFIG_INVALID engine.mode=openai but env var 'OPENAI_API_KEY' is empty` | export it before `make asr-run-openai` |
| `ASR_ENGINE_PROBE_FAILED error=unauthorized` | the key is set but invalid |
| Captions panel never shows the Cloud chip | older asr binary still running; check the `engine_mode` field in the WS `ready` event |
| `ASR_OPENAI_REQUEST_FAILED error=rate_limited` repeatedly | bursting too fast — bump `ASR_ENGINE_OPENAI_RETRY_BACKOFF_MS` or wait |
| Build fails with `third_party/openai-cpp/include/openai/openai.hpp: file not found` | run `git submodule update --init -- asr/third_party/openai-cpp` |
