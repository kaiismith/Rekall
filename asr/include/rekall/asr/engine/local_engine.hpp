// LocalEngine — whisper.cpp sliding-window streaming engine.
//
// Wraps a per-decode whisper_state against one Session: owns the sliding-
// window decode loop, emits partial/final TranscriptEvents into the session's
// outbound RingBuffer, and consults the VadSegmenter to bound segment lengths.
//
// One instance per active session. Run on a dedicated thread the caller
// provides; loop terminates when the supplied stop_token is requested.

#pragma once

#include <atomic>
#include <chrono>
#include <cstdint>
#include <memory>
#include <stop_token>
#include <string>
#include <vector>

#include "rekall/asr/audio/vad_segmenter.hpp"
#include "rekall/asr/config/config.hpp"
#include "rekall/asr/engine/model_registry.hpp"
#include "rekall/asr/engine/transcription_engine.hpp"

struct whisper_state;

namespace rekall::asr::observ { class Metrics; }

namespace rekall::asr::session {
struct Session;
}

namespace rekall::asr::engine {

class LocalEngine final : public TranscriptionEngine {
   public:
    LocalEngine(std::shared_ptr<rekall::asr::session::Session> session,
                std::shared_ptr<LoadedModel>                   model,
                rekall::asr::config::SessionConfig             session_cfg,
                rekall::asr::observ::Metrics*                  metrics);
    ~LocalEngine() override;

    LocalEngine(const LocalEngine&)            = delete;
    LocalEngine& operator=(const LocalEngine&) = delete;

    void             run(std::stop_token st) override;
    std::string_view name() const noexcept override { return "local"; }

   private:
    void emit_partial();
    void emit_final(std::uint64_t end_sample);
    bool ensure_state();

    std::shared_ptr<rekall::asr::session::Session> session_;
    std::shared_ptr<LoadedModel>                   model_;
    rekall::asr::config::SessionConfig             cfg_;
    rekall::asr::observ::Metrics*                  metrics_;

    whisper_state* state_ = nullptr;

    std::vector<float>                    window_;
    std::size_t                           window_max_samples_ = 0;
    rekall::asr::audio::VadSegmenter      vad_;

    std::int32_t                          segment_id_ = 0;
    std::string                           last_partial_text_;
    std::chrono::steady_clock::time_point last_partial_emit_{};
};

}  // namespace rekall::asr::engine
