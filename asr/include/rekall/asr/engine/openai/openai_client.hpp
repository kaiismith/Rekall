// Abstract OpenAI transcription client.
//
// The OpenAiEngine speaks to this seam — never directly to the upstream
// header-only `openai-cpp` library — so unit tests can plug in a fake without
// involving libcurl or a fake HTTPS server.
//
// One shot per segment: caller hands over WAV bytes + parameters, receives a
// transcript or a typed error. No streaming on this surface; OpenAI's
// /v1/audio/transcriptions endpoint is request/response.

#pragma once

#include <chrono>
#include <cstddef>
#include <cstdint>
#include <span>
#include <stop_token>
#include <string>
#include <variant>
#include <vector>

#include "rekall/asr/session/transcript_event.hpp"

namespace rekall::asr::engine::openai {

struct OpenAiParams {
    std::string         model;
    std::string         language;
    float               temperature     = 0.0F;
    std::string         response_format;
    std::string         prompt;
    std::chrono::seconds timeout {30};
    // Optional: forwarded as `X-Rekall-Correlation-Id` so the upload can be
    // joined with backend / ASR logs.
    std::string         correlation_id;
};

struct OpenAiTranscript {
    std::string                                   text;
    std::string                                   language;
    float                                         avg_logprob = 0.0F;
    std::vector<rekall::asr::session::WordTiming> words;
};

enum class OpenAiError {
    Network,
    Timeout,
    Unauthorized,
    RateLimited,
    BadRequest,
    ServerError,
    Cancelled,
    ParseError,
};

// Stable string label used by metrics/logs. Stable for any version of the
// service so dashboards keep working.
std::string_view to_label(OpenAiError e) noexcept;

// C++20-friendly stand-in for std::expected<T, E>. The real one lives in
// <expected> (C++23) and isn't available on the gcc-11 build container.
struct TranscribeResult {
    bool             ok = false;
    OpenAiTranscript value;
    OpenAiError      error = OpenAiError::Network;

    static TranscribeResult success(OpenAiTranscript v) {
        TranscribeResult r;
        r.ok    = true;
        r.value = std::move(v);
        return r;
    }
    static TranscribeResult failure(OpenAiError e) {
        TranscribeResult r;
        r.ok    = false;
        r.error = e;
        return r;
    }

    explicit operator bool() const noexcept { return ok; }
    bool has_value() const noexcept { return ok; }
};

class OpenAiClient {
   public:
    virtual ~OpenAiClient() = default;

    // Synchronous — caller blocks for the network round-trip. The stop_token
    // lets the engine abort an in-flight request when the session shuts
    // down; implementations SHOULD honour it within ~100 ms.
    virtual TranscribeResult transcribe(
        std::span<const std::byte> wav_bytes,
        const OpenAiParams&        params,
        std::stop_token            st) = 0;
};

}  // namespace rekall::asr::engine::openai
