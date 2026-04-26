// Centralised log event catalog for the ASR service.
//
// Every log line carries a stable `event_code` (SCREAMING_SNAKE_CASE) and a
// human-readable message defined here, plus runtime context fields supplied
// by the call site. This mirrors the Go-side pkg/logger/catalog convention
// so a single Loki/Datadog query can join both services by event_code.

#pragma once

#include <nlohmann/json.hpp>
#include <spdlog/spdlog.h>

#include <string_view>

namespace rekall::asr::observ {

struct Event {
    std::string_view code;
    std::string_view message;
};

// ── Session lifecycle ────────────────────────────────────────────────────────
inline constexpr Event SESSION_STARTED        {"ASR_SESSION_STARTED",        "asr session started"};
inline constexpr Event SESSION_ENDED          {"ASR_SESSION_ENDED",          "asr session ended"};

// ── Auth ─────────────────────────────────────────────────────────────────────
inline constexpr Event AUTH_OK                {"ASR_AUTH_OK",                "session token validated"};
inline constexpr Event AUTH_FAILED            {"ASR_AUTH_FAILED",            "session token validation failed"};

// ── Audio / transcript (Debug-level: high frequency) ─────────────────────────
inline constexpr Event FRAME_RECEIVED         {"ASR_FRAME_RECEIVED",         "audio frame received"};
inline constexpr Event PARTIAL_EMITTED        {"ASR_PARTIAL_EMITTED",        "partial transcript emitted"};
inline constexpr Event FINAL_EMITTED          {"ASR_FINAL_EMITTED",          "final transcript emitted"};
inline constexpr Event VAD_SEGMENT_END        {"ASR_VAD_SEGMENT_END",        "vad detected end of segment"};

// ── Engine failures ──────────────────────────────────────────────────────────
inline constexpr Event INFERENCE_FAILED       {"ASR_INFERENCE_FAILED",       "whisper inference returned an error"};

// ── Admission / backpressure ─────────────────────────────────────────────────
inline constexpr Event ADMISSION_REJECTED     {"ASR_ADMISSION_REJECTED",     "admission rejected — pool saturated"};
inline constexpr Event BACKPRESSURE_APPLIED   {"ASR_BACKPRESSURE_APPLIED",   "inbound buffer full — pausing reads"};
inline constexpr Event BACKPRESSURE_TIMEOUT   {"ASR_BACKPRESSURE_TIMEOUT",   "inbound buffer remained full beyond limit"};
inline constexpr Event DROPPED_PARTIAL        {"ASR_DROPPED_PARTIAL",        "outbound partial dropped under writer slow drain"};

// ── Models ───────────────────────────────────────────────────────────────────
inline constexpr Event MODEL_LOADED           {"ASR_MODEL_LOADED",           "whisper model loaded"};
inline constexpr Event MODEL_LOAD_FAILED      {"ASR_MODEL_LOAD_FAILED",      "whisper model failed to load"};

// ── Lifecycle ────────────────────────────────────────────────────────────────
inline constexpr Event SERVICE_STARTING       {"ASR_SERVICE_STARTING",       "service starting"};
inline constexpr Event SERVICE_READY          {"ASR_SERVICE_READY",          "service ready"};
inline constexpr Event GRACEFUL_DRAIN_BEGIN   {"ASR_GRACEFUL_DRAIN_BEGIN",   "shutdown drain begin"};
inline constexpr Event GRACEFUL_DRAIN_END     {"ASR_GRACEFUL_DRAIN_END",     "shutdown drain end"};
inline constexpr Event FATAL                  {"ASR_FATAL",                  "fatal runtime failure"};
inline constexpr Event CONFIG_INVALID         {"ASR_CONFIG_INVALID",         "configuration invalid"};

// ── Logger init ──────────────────────────────────────────────────────────────
// Initialise the global spdlog logger named "asr". `level` is one of
// debug/info/warn/error; `format` is "json" (default) or "text".
void init_logger(std::string_view level, std::string_view format);

// ── Emit helpers ─────────────────────────────────────────────────────────────
// Each helper merges the event_code, event_ts, and supplied fields into a
// single JSON object and emits as the message. `fields` MAY include
// `session_id` and `correlation_id`; they are first-class but not enforced.
void debug(const Event& e, const nlohmann::json& fields = nlohmann::json::object());
void info (const Event& e, const nlohmann::json& fields = nlohmann::json::object());
void warn (const Event& e, const nlohmann::json& fields = nlohmann::json::object());
void error(const Event& e, const nlohmann::json& fields = nlohmann::json::object());

}  // namespace rekall::asr::observ
