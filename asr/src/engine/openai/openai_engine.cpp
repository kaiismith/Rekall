#include "rekall/asr/engine/openai/openai_engine.hpp"

#include <algorithm>
#include <chrono>
#include <cmath>
#include <cstdint>
#include <thread>
#include <utility>

#include "rekall/asr/engine/openai/wav_encoder.hpp"
#include "rekall/asr/observ/log_catalog.hpp"
#include "rekall/asr/observ/metrics.hpp"
#include "rekall/asr/session/session.hpp"

namespace rekall::asr::engine::openai {

namespace {

constexpr std::uint32_t kSampleRate = 16'000;

bool is_retryable(OpenAiError e) noexcept {
    switch (e) {
        case OpenAiError::Network:
        case OpenAiError::Timeout:
        case OpenAiError::RateLimited:
        case OpenAiError::ServerError:
            return true;
        default:
            return false;
    }
}

std::string_view retry_reason(OpenAiError e) noexcept {
    switch (e) {
        case OpenAiError::Network:     return "network";
        case OpenAiError::Timeout:     return "timeout";
        case OpenAiError::RateLimited: return "rate_limited";
        case OpenAiError::ServerError: return "server_error";
        default:                       return "other";
    }
}

}  // namespace

OpenAiEngine::OpenAiEngine(std::shared_ptr<rekall::asr::session::Session> session,
                           std::unique_ptr<OpenAiClient>                  client,
                           rekall::asr::config::SessionConfig             session_cfg,
                           rekall::asr::config::OpenAiEngineConfig        oai_cfg,
                           rekall::asr::observ::Metrics*                  metrics)
    : session_(std::move(session)),
      client_(std::move(client)),
      cfg_(session_cfg),
      oai_(std::move(oai_cfg)),
      metrics_(metrics),
      vad_(kSampleRate) {
    // Reasonable upper bound for buffered PCM: max_segment_seconds @ 16 kHz int16.
    segment_pcm_.reserve(static_cast<std::size_t>(kSampleRate) *
                         oai_.max_segment_seconds);
}

void OpenAiEngine::run(std::stop_token st) {
    using namespace std::chrono;

    while (!st.stop_requested() && !session_->is_terminal()) {
        auto frame = session_->inbound.pop_blocking(st);
        if (!frame) break;  // stop_token tripped

        const bool is_flush = frame->flush;
        bool       vad_end  = false;

        if (segment_pcm_.empty() && !frame->samples.empty()) {
            segment_started_ = steady_clock::now();
        }

        if (!frame->samples.empty()) {
            session_->stats.audio_samples_processed.fetch_add(frame->samples.size());
            session_->touch();

            if (auto eos = vad_.push(frame->samples); eos.has_value()) {
                vad_end = true;
                rekall::asr::observ::debug(rekall::asr::observ::VAD_SEGMENT_END, {
                    {"session_id", session_->sid},
                });
            }

            // Append into the segment buffer.
            segment_pcm_.insert(segment_pcm_.end(),
                                frame->samples.begin(), frame->samples.end());
        }

        const double secs_buffered =
            static_cast<double>(segment_pcm_.size()) / kSampleRate;
        const bool over_cap  = secs_buffered >= oai_.max_segment_seconds;
        const bool under_min = secs_buffered <  oai_.min_segment_seconds;

        if (vad_end || is_flush || over_cap) {
            if (vad_end) emit_partial_not_supported_once();

            if (under_min) {
                // Discard sub-min clips — they waste an API call and OpenAI
                // returns gibberish on near-empty audio.
                segment_pcm_.clear();
                vad_.reset();
                continue;
            }
            on_segment_close(st);
        }
    }

    // Best-effort tail flush on shutdown if we still have a meaningful clip.
    if (!st.stop_requested() && !session_->is_terminal() &&
        !segment_pcm_.empty() &&
        static_cast<double>(segment_pcm_.size()) / kSampleRate >= oai_.min_segment_seconds) {
        on_segment_close(st);
    }
}

void OpenAiEngine::on_segment_close(std::stop_token st) {
    using namespace std::chrono;

    const auto wav = encode_wav_pcm16(segment_pcm_, kSampleRate);

    OpenAiParams params;
    params.model           = oai_.model;
    params.language        = oai_.language;
    params.temperature     = oai_.temperature;
    params.response_format = oai_.response_format;
    params.prompt          = oai_.prompt;
    params.timeout         = std::chrono::seconds(oai_.request_timeout_seconds);
    params.correlation_id  = session_->correlation_id;

    const double audio_seconds =
        static_cast<double>(segment_pcm_.size()) / kSampleRate;
    if (metrics_ != nullptr) {
        metrics_->openai_audio_seconds_uploaded_total().Increment(audio_seconds);
    }

    TranscribeResult result = TranscribeResult::failure(OpenAiError::Network);

    const std::uint32_t max_attempts = oai_.retries + 1;
    for (std::uint32_t attempt = 1; attempt <= max_attempts; ++attempt) {
        if (st.stop_requested()) {
            result = TranscribeResult::failure(OpenAiError::Cancelled);
            break;
        }

        const auto t0 = steady_clock::now();
        result = client_->transcribe(std::span{wav}, params, st);
        const auto rt = steady_clock::now() - t0;

        if (metrics_ != nullptr) {
            metrics_->openai_request_duration_seconds().Observe(
                duration<double>(rt).count());
        }

        if (result.has_value()) {
            if (metrics_ != nullptr) {
                metrics_->openai_requests_total("ok").Increment();
            }
            break;
        }

        const auto err = result.error;
        if (metrics_ != nullptr) {
            metrics_->openai_requests_total(std::string(to_label(err))).Increment();
        }

        if (err == OpenAiError::RateLimited) {
            rekall::asr::observ::warn(rekall::asr::observ::OPENAI_RATE_LIMITED, {
                {"session_id", session_->sid},
                {"segment_id", segment_id_},
                {"attempt",    static_cast<int>(attempt)},
            });
        }

        if (!is_retryable(err) || attempt == max_attempts) break;

        if (metrics_ != nullptr) {
            metrics_->openai_retries_total(std::string(retry_reason(err))).Increment();
        }
        // Exponential backoff: 500 ms, 1 s, 2 s, ...
        const auto backoff_ms = oai_.retry_backoff_ms *
                                (1u << std::min<std::uint32_t>(attempt - 1, 5));
        std::this_thread::sleep_for(milliseconds(backoff_ms));
    }

    if (result.has_value()) {
        consecutive_failures_ = 0;
        emit_final_event(result.value);
    } else {
        ++consecutive_failures_;
        rekall::asr::observ::warn(rekall::asr::observ::OPENAI_REQUEST_FAILED, {
            {"session_id", session_->sid},
            {"segment_id", segment_id_},
            {"error",      to_label(result.error)},
        });
        emit_error_event("ASR_OPENAI_REQUEST_FAILED",
                         to_label(result.error));
        if (consecutive_failures_ >= 3) {
            rekall::asr::observ::error(rekall::asr::observ::OPENAI_REQUEST_FAILED, {
                {"session_id", session_->sid},
                {"reason",     "3 consecutive segment failures — stopping session"},
            });
            session_->stop_source.request_stop();
        }
    }

    segment_pcm_.clear();
    vad_.reset();
    ++segment_id_;
}

void OpenAiEngine::emit_partial_not_supported_once() {
    if (warned_no_partials_) return;
    warned_no_partials_ = true;
    rekall::asr::observ::info(rekall::asr::observ::PARTIAL_NOT_SUPPORTED, {
        {"session_id", session_->sid},
        {"engine",     "openai"},
    });
}

void OpenAiEngine::emit_error_event(std::string_view code, std::string_view message) {
    rekall::asr::session::TranscriptEvent ev;
    ev.type    = rekall::asr::session::EventType::Error;
    ev.code    = code;
    ev.message = message;
    (void)session_->outbound.try_push(std::move(ev));
}

void OpenAiEngine::emit_final_event(const OpenAiTranscript& t) {
    using namespace std::chrono;

    rekall::asr::session::TranscriptEvent ev;
    ev.type       = rekall::asr::session::EventType::Final;
    ev.segment_id = segment_id_;
    ev.text       = t.text;
    ev.language   = !t.language.empty() ? t.language : oai_.language;
    ev.words      = t.words;
    // Wall-clock-derived timing: OpenAI returns relative timestamps inside the
    // clip, but the engine needs session-relative timing. Use the buffered
    // segment's elapsed bounds.
    const auto now = steady_clock::now();
    const auto dur = duration_cast<milliseconds>(now - segment_started_).count();
    ev.start_ms    = 0;
    ev.end_ms      = static_cast<std::uint32_t>(std::max<std::int64_t>(0, dur));
    if (t.avg_logprob != 0.0F) {
        // exp(avg_logprob) maps to [0, 1] confidence-ish.
        ev.confidence = std::min(1.0F,
            std::max(0.0F, static_cast<float>(std::exp(static_cast<double>(t.avg_logprob)))));
    }

    (void)session_->outbound.try_push(std::move(ev));

    session_->stats.final_count.fetch_add(1);
    if (metrics_ != nullptr) {
        metrics_->final_events_total(oai_.model).Increment();
    }
    session_->final_text.append(t.text);
    rekall::asr::observ::debug(rekall::asr::observ::FINAL_EMITTED, {
        {"session_id", session_->sid}, {"segment_id", segment_id_},
        {"engine",     "openai"},
    });
}

}  // namespace rekall::asr::engine::openai
