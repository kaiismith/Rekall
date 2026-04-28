#include "rekall/asr/session/session_manager.hpp"

#include <chrono>
#include <mutex>
#include <shared_mutex>
#include <thread>
#include <utility>

#include "rekall/asr/observ/log_catalog.hpp"
#include "rekall/asr/observ/metrics.hpp"
#include "rekall/asr/util/correlation_id.hpp"

namespace rekall::asr::session {

SessionManager::SessionManager(rekall::asr::engine::ModelRegistry* models,
                               rekall::asr::config::Config cfg,
                               rekall::asr::observ::Metrics* metrics)
    : models_(models), cfg_(std::move(cfg)), metrics_(metrics) {
    sweeper_ = std::jthread([this](std::stop_token st) { sweeper_loop(st); });
}

SessionManager::~SessionManager() { sweeper_.request_stop(); }

CreateResult SessionManager::create(const CreateInput& in) {
    auto model = (models_ != nullptr) ? models_->get_or_default(in.model_id_request)
                                      : nullptr;
    std::string canonical = (model ? model->id() : in.model_id_request);

    // OpenAI mode has no local model registry, so `models_` is null and the
    // canonical id falls through to whatever the caller sent — which is
    // typically empty (callers don't override). Fall back to the engine's
    // configured model name so downstream consumers (Go-side persistence,
    // logs, dashboards) always see the canonical model id, never an empty
    // string.
    if (canonical.empty()) {
        if (cfg_.engine.mode == rekall::asr::config::EngineMode::OpenAi) {
            canonical = cfg_.engine.openai.model;
        } else {
            canonical = cfg_.models.default_id;
        }
    }

    std::uint32_t ttl = in.requested_ttl_seconds;
    if (ttl == 0) ttl = cfg_.auth.token_default_ttl_seconds;
    if (ttl < cfg_.auth.token_min_ttl_seconds) ttl = cfg_.auth.token_min_ttl_seconds;
    if (ttl > cfg_.auth.token_max_ttl_seconds) ttl = cfg_.auth.token_max_ttl_seconds;

    auto session = std::make_shared<Session>(
        cfg_.admission.inbound_audio_buffer_frames,
        cfg_.admission.outbound_event_buffer);
    session->sid            = rekall::asr::util::new_uuid_v4();
    session->cid            = in.call_id;
    session->uid            = in.user_id;
    session->model_id       = canonical;
    session->correlation_id = rekall::asr::util::ensure(in.correlation_id);
    session->model          = model;
    session->started_at     = std::chrono::system_clock::now();
    session->expires_at     = session->started_at + std::chrono::seconds(ttl);
    session->touch();

    {
        std::unique_lock g(mu_);
        by_sid_.emplace(session->sid, session);
    }
    update_active_gauge();

    rekall::asr::observ::info(rekall::asr::observ::SESSION_STARTED, {
        {"session_id",     session->sid},
        {"call_id",        session->cid},
        {"user_id",        session->uid},
        {"model_id",       session->model_id},
        {"correlation_id", session->correlation_id},
    });
    return CreateResult{
        .session            = session,
        .canonical_model_id = canonical,
        .expires_at         = session->expires_at,
    };
}

std::shared_ptr<Session> SessionManager::get(std::string_view sid) const {
    std::shared_lock g(mu_);
    auto it = by_sid_.find(std::string(sid));
    if (it == by_sid_.end()) return nullptr;
    return it->second;
}

std::shared_ptr<Session> SessionManager::bind(std::string_view sid) {
    std::shared_ptr<Session> s;
    {
        std::shared_lock g(mu_);
        auto it = by_sid_.find(std::string(sid));
        if (it == by_sid_.end()) return nullptr;
        s = it->second;
    }
    State expected = State::Pending;
    if (!s->state.compare_exchange_strong(expected, State::Active,
                                          std::memory_order_acq_rel,
                                          std::memory_order_acquire)) {
        return nullptr;
    }
    s->touch();
    return s;
}

SessionManager::EndOutcome SessionManager::end(std::string_view sid) {
    std::shared_ptr<Session> s;
    {
        std::unique_lock g(mu_);
        auto it = by_sid_.find(std::string(sid));
        if (it == by_sid_.end()) return EndOutcome{};
        s = it->second;
        by_sid_.erase(it);
    }
    s->state.store(State::Closed, std::memory_order_release);
    s->stop_source.request_stop();
    update_active_gauge();

    auto out = EndOutcome{
        .final_transcript = s->final_text.snapshot(),
        .final_count      = s->stats.final_count.load(),
        .present          = true,
    };

    using namespace std::chrono;
    auto dur = duration<double>(system_clock::now() - s->started_at).count();
    if (metrics_ != nullptr) metrics_->session_duration_seconds().Observe(dur);
    rekall::asr::observ::info(rekall::asr::observ::SESSION_ENDED, {
        {"session_id",     s->sid},
        {"correlation_id", s->correlation_id},
        {"duration_s",     dur},
        {"final_count",    out.final_count},
        {"partial_count",  s->stats.partial_count.load()},
    });
    return out;
}

std::size_t SessionManager::force_close_remaining() {
    std::vector<std::shared_ptr<Session>> snapshot;
    {
        std::unique_lock g(mu_);
        snapshot.reserve(by_sid_.size());
        for (auto& kv : by_sid_) snapshot.push_back(kv.second);
        by_sid_.clear();
    }
    for (auto& s : snapshot) {
        s->state.store(State::Closed, std::memory_order_release);
        s->stop_source.request_stop();
    }
    update_active_gauge();
    return snapshot.size();
}

std::size_t SessionManager::active_count() const {
    std::shared_lock g(mu_);
    return by_sid_.size();
}

void SessionManager::update_active_gauge() {
    if (metrics_ == nullptr) return;
    metrics_->active_sessions().Set(static_cast<double>(active_count()));
}

void SessionManager::sweeper_loop(std::stop_token st) {
    using namespace std::chrono;
    using namespace std::chrono_literals;
    while (!st.stop_requested()) {
        for (int i = 0; i < 5 && !st.stop_requested(); ++i) {
            std::this_thread::sleep_for(1s);
        }
        if (st.stop_requested()) break;

        auto now    = system_clock::now();
        auto now_ms = duration_cast<milliseconds>(now.time_since_epoch()).count();
        auto idle_ms = static_cast<std::int64_t>(cfg_.session.idle_timeout_seconds) * 1000;
        auto hard    = seconds(cfg_.session.hard_timeout_seconds);

        std::vector<std::string> to_close;
        {
            std::shared_lock g(mu_);
            for (const auto& [sid, s] : by_sid_) {
                if (s->state.load() == State::Closed) continue;

                auto last = s->last_activity_at_ms.load();
                bool idle = (last > 0) && (now_ms - last > idle_ms) &&
                            (s->state.load() == State::Active);
                bool too_long = (now - s->started_at) > hard;

                if (idle || too_long) to_close.push_back(sid);
            }
        }
        for (const auto& sid : to_close) {
            (void)end(sid);
        }
    }
}

}  // namespace rekall::asr::session
