// OpenAiEngine — buffers per-segment PCM and uploads to OpenAI's
// /v1/audio/transcriptions endpoint when a segment closes (VAD-end, flush, or
// max_segment_seconds cap). Emits Final TranscriptEvents per segment; never
// emits Partial — the cloud endpoint is one-shot.

#pragma once

#include <chrono>
#include <cstdint>
#include <memory>
#include <stop_token>
#include <vector>

#include "rekall/asr/audio/vad_segmenter.hpp"
#include "rekall/asr/config/config.hpp"
#include "rekall/asr/engine/openai/openai_client.hpp"
#include "rekall/asr/engine/transcription_engine.hpp"

namespace rekall::asr::observ { class Metrics; }
namespace rekall::asr::session { struct Session; }

namespace rekall::asr::engine::openai {

class OpenAiEngine final : public TranscriptionEngine {
   public:
    OpenAiEngine(std::shared_ptr<rekall::asr::session::Session> session,
                 std::unique_ptr<OpenAiClient>                  client,
                 rekall::asr::config::SessionConfig             session_cfg,
                 rekall::asr::config::OpenAiEngineConfig        oai_cfg,
                 rekall::asr::observ::Metrics*                  metrics);
    ~OpenAiEngine() override = default;

    OpenAiEngine(const OpenAiEngine&)            = delete;
    OpenAiEngine& operator=(const OpenAiEngine&) = delete;

    void             run(std::stop_token st) override;
    std::string_view name() const noexcept override { return "openai"; }

   private:
    void on_segment_close(std::stop_token st);
    void emit_error_event(std::string_view code, std::string_view message);
    void emit_final_event(const OpenAiTranscript& t);
    void emit_partial_not_supported_once();

    std::shared_ptr<rekall::asr::session::Session> session_;
    std::unique_ptr<OpenAiClient>                  client_;
    rekall::asr::config::SessionConfig             cfg_;
    rekall::asr::config::OpenAiEngineConfig        oai_;
    rekall::asr::observ::Metrics*                  metrics_;

    rekall::asr::audio::VadSegmenter        vad_;
    std::vector<std::int16_t>               segment_pcm_;
    std::chrono::steady_clock::time_point   segment_started_{};
    std::int32_t                            segment_id_           = 0;
    std::int32_t                            consecutive_failures_ = 0;
    bool                                    warned_no_partials_   = false;
};

}  // namespace rekall::asr::engine::openai
