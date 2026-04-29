#include "rekall/asr/engine/local_engine.hpp"

#include <algorithm>
#include <chrono>
#include <cmath>
#include <cstdint>
#include <utility>
#include <vector>

#include "rekall/asr/observ/log_catalog.hpp"
#include "rekall/asr/observ/metrics.hpp"
#include "rekall/asr/session/session.hpp"
#include "rekall/asr/session/transcript_event.hpp"

#if __has_include(<whisper.h>)
#include <whisper.h>
#define REKALL_HAVE_WHISPER 1
#else
// Build-time stub: keeps the TU compiling when the whisper.cpp submodule is
// not yet available. The runtime behaviour is degraded (no transcripts are
// produced) but the rest of the service still links.
struct whisper_state {};
struct whisper_full_params {
    int  strategy;
    int  n_threads;
    bool translate;
    bool no_context;
    bool single_segment;
    bool token_timestamps;
    bool suppress_blank;
    // whisper.cpp v1.7.x renamed `suppress_non_speech_tokens` → `suppress_nst`.
    bool suppress_nst;
    const char* language;
    // Per-strategy knobs nested as in real whisper.cpp v1.7.x.
    struct { int best_of; }   greedy;
    struct { int beam_size; } beam_search;
};
extern "C" {
inline whisper_state* whisper_init_state(struct whisper_context*)         { return nullptr; }
inline void           whisper_free_state(whisper_state*)                  {}
inline whisper_full_params whisper_full_default_params(int)               { return {}; }
inline int  whisper_full_with_state(struct whisper_context*, whisper_state*,
                                    whisper_full_params, const float*, int) { return -1; }
inline int  whisper_full_n_segments_from_state(whisper_state*)            { return 0; }
inline const char* whisper_full_get_segment_text_from_state(whisper_state*, int) { return ""; }
inline std::int64_t whisper_full_get_segment_t0_from_state(whisper_state*, int) { return 0; }
inline std::int64_t whisper_full_get_segment_t1_from_state(whisper_state*, int) { return 0; }
inline int  whisper_full_n_tokens_from_state(whisper_state*, int)         { return 0; }
inline const char* whisper_full_get_token_text_from_state(struct whisper_context*,
                                                           whisper_state*, int, int) { return ""; }
struct whisper_token_data { float p; std::int64_t t0; std::int64_t t1; };
inline whisper_token_data whisper_full_get_token_data_from_state(whisper_state*, int, int) {
    return {};
}
}
#define WHISPER_SAMPLING_BEAM_SEARCH 1
#define WHISPER_SAMPLING_GREEDY      0
#define REKALL_HAVE_WHISPER 0
#endif

namespace rekall::asr::engine {

namespace {

constexpr std::uint32_t kSampleRate = 16000;

// Convert int16 PCM → float32 in [-1, 1] for whisper.
void append_int16_to_float(std::vector<float>& dst, const std::vector<std::int16_t>& src) {
    const std::size_t old = dst.size();
    dst.resize(old + src.size());
    for (std::size_t i = 0; i < src.size(); ++i) {
        dst[old + i] = static_cast<float>(src[i]) / 32768.0F;
    }
}

}  // namespace

LocalEngine::LocalEngine(std::shared_ptr<rekall::asr::session::Session> session,
                         std::shared_ptr<LoadedModel> model,
                         rekall::asr::config::SessionConfig session_cfg,
                         rekall::asr::observ::Metrics* metrics)
    : session_(std::move(session)),
      model_(std::move(model)),
      cfg_(session_cfg),
      metrics_(metrics),
      vad_(kSampleRate) {
    window_max_samples_ = static_cast<std::size_t>(kSampleRate) * cfg_.audio_window_seconds;
    window_.reserve(window_max_samples_);
}

LocalEngine::~LocalEngine() {
    if (state_ != nullptr) {
        whisper_free_state(state_);
        state_ = nullptr;
    }
}

bool LocalEngine::ensure_state() {
    if (state_ != nullptr) return true;
    if (!model_ || model_->ctx() == nullptr) return false;
    state_ = whisper_init_state(model_->ctx());
    return state_ != nullptr;
}

void LocalEngine::run(std::stop_token st) {
    using namespace std::chrono;
    last_partial_emit_ = steady_clock::now();

    while (!st.stop_requested() && !session_->is_terminal()) {
        auto frame = session_->inbound.pop_blocking(st);
        if (!frame) break;  // stop_token tripped

        bool flush         = frame->flush;
        bool vad_speech_end = false;

        if (!flush && !frame->samples.empty()) {
            session_->stats.audio_samples_processed.fetch_add(frame->samples.size());
            session_->touch();

            // VAD speech-end is the segmentation signal. When the energy
            // detector trips a "trailing silence" boundary we finalize the
            // current window and start fresh — without this, whisper keeps
            // re-decoding overlapping audio every 150 ms and hallucinates
            // the previous utterance's text into new ones.
            if (auto seg_end = vad_.push(frame->samples)) {
                vad_speech_end = true;
                rekall::asr::observ::debug(rekall::asr::observ::VAD_SEGMENT_END, {
                    {"session_id", session_->sid},
                });
            }

            // Append into the sliding window, trimming the oldest samples
            // when over capacity.
            append_int16_to_float(window_, frame->samples);
            if (window_.size() > window_max_samples_) {
                std::size_t excess = window_.size() - window_max_samples_;
                window_.erase(window_.begin(),
                              window_.begin() + static_cast<std::ptrdiff_t>(excess));
            }
        }

        // Decide whether to decode now: throttle partials by
        // partial_emit_interval_ms; always decode on flush or VAD speech-end.
        auto now   = steady_clock::now();
        auto since = duration_cast<milliseconds>(now - last_partial_emit_).count();
        bool due   = since >= cfg_.partial_emit_interval_ms;
        if (!flush && !vad_speech_end && !due) continue;

        if (!ensure_state()) {
            rekall::asr::observ::error(rekall::asr::observ::INFERENCE_FAILED, {
                {"session_id", session_->sid},
                {"reason",     "whisper_init_state returned null"},
            });
            // Surface to client.
            rekall::asr::session::TranscriptEvent err;
            err.type    = rekall::asr::session::EventType::Error;
            err.code    = "ASR_INFERENCE_FAILED";
            err.message = "engine state init failed";
            (void)session_->outbound.try_push(std::move(err));
            session_->stop_source.request_stop();
            break;
        }

        if (window_.empty()) {
            last_partial_emit_ = now;
            continue;
        }

        if (flush || vad_speech_end) {
            // Commit the current utterance, then wipe state so the next
            // utterance starts with a clean buffer + a new segment_id.
            emit_final(static_cast<std::uint64_t>(window_.size()));
            window_.clear();
            vad_.reset();
            last_partial_text_.clear();
            last_partial_emit_ = steady_clock::now();
        } else {
            emit_partial();
            last_partial_emit_ = now;
        }
    }
}

void LocalEngine::emit_partial() {
    using namespace std::chrono;
    auto t0 = steady_clock::now();

    whisper_full_params params = whisper_full_default_params(WHISPER_SAMPLING_GREEDY);
    if (model_ && !model_->entry().language.empty()) params.language = model_->entry().language.c_str();
    params.n_threads               = std::max(1, model_ ? model_->entry().n_threads : 1);
    params.no_context              = true;
    params.single_segment          = true;
    params.token_timestamps        = false;
    params.suppress_blank          = model_ ? model_->entry().suppress_blank : true;
    params.suppress_nst            = model_ ? model_->entry().suppress_non_speech_tokens : true;
    params.translate               = model_ ? model_->entry().translate : false;

    int rc = whisper_full_with_state(model_->ctx(), state_, params,
                                     window_.data(), static_cast<int>(window_.size()));
    auto t1 = steady_clock::now();
    auto secs = duration<double>(t1 - t0).count();
    if (metrics_ != nullptr) {
        metrics_->inference_duration_seconds(model_->id(), "partial").Observe(secs);
    }
    if (rc != 0) {
        rekall::asr::observ::warn(rekall::asr::observ::INFERENCE_FAILED, {
            {"session_id", session_->sid}, {"phase", "partial"}, {"rc", rc},
        });
        return;
    }

    int n = whisper_full_n_segments_from_state(state_);
    if (n <= 0) return;
    std::string text = whisper_full_get_segment_text_from_state(state_, n - 1);
    if (text.empty() || text == last_partial_text_) return;
    last_partial_text_ = text;

    rekall::asr::session::TranscriptEvent ev;
    ev.type       = rekall::asr::session::EventType::Partial;
    ev.segment_id = segment_id_;
    ev.text       = std::move(text);
    ev.start_ms   = static_cast<std::uint32_t>(
        whisper_full_get_segment_t0_from_state(state_, n - 1) * 10);
    ev.end_ms     = static_cast<std::uint32_t>(
        whisper_full_get_segment_t1_from_state(state_, n - 1) * 10);
    ev.confidence = 0.0F;

    // Outbound is bounded; if full, drop the oldest event (which is by design
    // a partial — we never enqueue a final after a partial in the same tick
    // without giving the writer a chance to drain).
    bool dropped = session_->outbound.push_or_drop_oldest(std::move(ev));
    if (dropped) {
        session_->stats.dropped_partials.fetch_add(1);
        if (metrics_ != nullptr) metrics_->dropped_partials_total().Increment();
        rekall::asr::observ::debug(rekall::asr::observ::DROPPED_PARTIAL, {
            {"session_id", session_->sid},
        });
    }
    session_->stats.partial_count.fetch_add(1);
    if (metrics_ != nullptr) metrics_->partial_events_total(model_->id()).Increment();
}

void LocalEngine::emit_final(std::uint64_t /*end_sample*/) {
    using namespace std::chrono;
    auto t0 = steady_clock::now();

    // Note: whisper.cpp v1.7.x nests the per-strategy knobs under
    // `params.beam_search.beam_size` and `params.greedy.best_of`. The
    // `suppress_non_speech_tokens` field was renamed to `suppress_nst`.
    whisper_full_params params = whisper_full_default_params(WHISPER_SAMPLING_BEAM_SEARCH);
    if (model_ && !model_->entry().language.empty()) params.language = model_->entry().language.c_str();
    params.n_threads               = std::max(1, model_ ? model_->entry().n_threads : 1);
    params.beam_search.beam_size   = std::max(1, model_ ? model_->entry().beam_size : 5);
    params.greedy.best_of          = std::max(1, model_ ? model_->entry().best_of   : 5);
    params.no_context              = true;
    params.single_segment          = false;
    params.token_timestamps        = true;
    params.suppress_blank          = model_ ? model_->entry().suppress_blank : true;
    params.suppress_nst            = model_ ? model_->entry().suppress_non_speech_tokens : true;
    params.translate               = model_ ? model_->entry().translate : false;

    int rc = whisper_full_with_state(model_->ctx(), state_, params,
                                     window_.data(), static_cast<int>(window_.size()));
    auto t1 = steady_clock::now();
    auto secs = duration<double>(t1 - t0).count();
    if (metrics_ != nullptr) {
        metrics_->inference_duration_seconds(model_->id(), "final").Observe(secs);
    }
    if (rc != 0) {
        rekall::asr::observ::warn(rekall::asr::observ::INFERENCE_FAILED, {
            {"session_id", session_->sid}, {"phase", "final"}, {"rc", rc},
        });
        return;
    }

    int n = whisper_full_n_segments_from_state(state_);
    std::string full_text;
    std::vector<rekall::asr::session::WordTiming> words;
    std::int64_t start_t = 0, end_t = 0;
    for (int s = 0; s < n; ++s) {
        const char* st = whisper_full_get_segment_text_from_state(state_, s);
        if (st != nullptr && *st != '\0') {
            if (!full_text.empty()) full_text += ' ';
            full_text += st;
        }
        if (s == 0)     start_t = whisper_full_get_segment_t0_from_state(state_, s);
        if (s == n - 1) end_t   = whisper_full_get_segment_t1_from_state(state_, s);

        int nt = whisper_full_n_tokens_from_state(state_, s);
        for (int i = 0; i < nt; ++i) {
            const char* w = whisper_full_get_token_text_from_state(model_->ctx(), state_, s, i);
            if (w == nullptr || *w == '\0') continue;
            std::string ws(w);
            if (!ws.empty() && ws.front() == '[') continue;  // special tokens
            auto td = whisper_full_get_token_data_from_state(state_, s, i);
            words.push_back({
                std::move(ws),
                static_cast<std::uint32_t>(td.t0 * 10),
                static_cast<std::uint32_t>(td.t1 * 10),
                td.p,
            });
        }
    }

    rekall::asr::session::TranscriptEvent ev;
    ev.type       = rekall::asr::session::EventType::Final;
    ev.segment_id = segment_id_;
    ev.text       = full_text;
    ev.start_ms   = static_cast<std::uint32_t>(start_t * 10);
    ev.end_ms     = static_cast<std::uint32_t>(end_t   * 10);
    ev.language   = model_ ? model_->entry().language : "";
    ev.words      = std::move(words);
    (void)session_->outbound.try_push(std::move(ev));

    session_->stats.final_count.fetch_add(1);
    if (metrics_ != nullptr) metrics_->final_events_total(model_->id()).Increment();
    if (metrics_ != nullptr) {
        metrics_->audio_seconds_processed_total(model_->id())
            .Increment(static_cast<double>(window_.size()) / kSampleRate);
    }
    session_->final_text.append(full_text);
    rekall::asr::observ::debug(rekall::asr::observ::FINAL_EMITTED, {
        {"session_id", session_->sid}, {"segment_id", segment_id_},
    });

    last_partial_text_.clear();
    ++segment_id_;
}

}  // namespace rekall::asr::engine
