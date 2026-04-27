// Abstract transcription-engine interface.
//
// Two implementations exist: LocalEngine wraps whisper.cpp for in-process
// streaming inference; OpenAiEngine buffers per-segment PCM and uploads to
// OpenAI's /v1/audio/transcriptions over HTTPS. Both are constructed by
// engine_factory::make_engine based on the running engine.mode.
//
// The interface is intentionally narrow: one blocking run loop driven by the
// caller's stop_token, plus an identifier surfaced in logs and the WS `ready`
// event.

#pragma once

#include <stop_token>
#include <string_view>

namespace rekall::asr::engine {

class TranscriptionEngine {
   public:
    virtual ~TranscriptionEngine() = default;

    // Pumps the bound Session's inbound queue until `st` is requested or the
    // session reaches a terminal state. MUST emit TranscriptEvents to the
    // session's outbound queue. MUST NOT mutate session state outside the
    // documented stat counters.
    virtual void run(std::stop_token st) = 0;

    // Identifier for logs and the WS `ready` event. "local" | "openai".
    virtual std::string_view name() const noexcept = 0;
};

}  // namespace rekall::asr::engine
