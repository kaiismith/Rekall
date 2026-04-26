// Transcriber wraps a per-decode whisper_state against one Session: it owns
// the sliding-window decode loop, emits partial/final TranscriptEvents into
// the session's outbound RingBuffer, and consults the VadSegmenter to bound
// segment lengths.
//
// One Transcriber instance per active session. Run on a dedicated thread the
// caller provides; loop terminates when the supplied stop_token is requested.

#pragma once

#include <atomic>
#include <cstdint>
#include <memory>
#include <stop_token>

#include "rekall/asr/audio/vad_segmenter.hpp"
#include "rekall/asr/config/config.hpp"
#include "rekall/asr/engine/model_registry.hpp"

struct whisper_state;

namespace rekall::asr::observ { class Metrics; }

namespace rekall::asr::session {
struct Session;
}

namespace rekall::asr::engine {

class Transcriber {
   public:
    Transcriber(std::shared_ptr<rekall::asr::session::Session> session,
                std::shared_ptr<LoadedModel> model,
                rekall::asr::config::SessionConfig session_cfg,
                rekall::asr::observ::Metrics* metrics);
    ~Transcriber();

    Transcriber(const Transcriber&)            = delete;
    Transcriber& operator=(const Transcriber&) = delete;

    // Pumps inbound audio into the sliding window, runs whisper at the
    // configured cadence, and emits TranscriptEvents to the session outbound
    // queue. Returns when stop_token is requested or the session enters a
    // terminal state.
    void run(std::stop_token st);

   private:
    void emit_partial();
    void emit_final(std::uint64_t end_sample);
    bool ensure_state();

    std::shared_ptr<rekall::asr::session::Session> session_;
    std::shared_ptr<LoadedModel>                   model_;
    rekall::asr::config::SessionConfig             cfg_;
    rekall::asr::observ::Metrics*                  metrics_;

    whisper_state* state_ = nullptr;

    // Sliding window of f32 samples (whisper's preferred input format).
    std::vector<float>     window_;
    std::size_t            window_max_samples_ = 0;
    rekall::asr::audio::VadSegmenter vad_;

    // Throttling and segment accounting.
    std::int32_t   segment_id_ = 0;
    std::string    last_partial_text_;
    std::chrono::steady_clock::time_point last_partial_emit_{};
};

}  // namespace rekall::asr::engine
