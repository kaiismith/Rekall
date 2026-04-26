// Owns the live set of Sessions plus a sweeper that closes idle / over-limit
// sessions. Sessions are created by gRPC StartSession (Pending) and bound by
// the WS upgrade handler (→ Active).

#pragma once

#include <atomic>
#include <chrono>
#include <memory>
#include <mutex>
#include <optional>
#include <shared_mutex>
#include <stop_token>
#include <string>
#include <string_view>
#include <thread>
#include <unordered_map>
#include <vector>

#include "rekall/asr/config/config.hpp"
#include "rekall/asr/engine/model_registry.hpp"
#include "rekall/asr/session/session.hpp"

namespace rekall::asr::observ { class Metrics; }

namespace rekall::asr::session {

struct CreateInput {
    std::string user_id;
    std::string call_id;
    std::string model_id_request;     // empty → default
    std::string language;
    std::uint32_t requested_ttl_seconds = 0;  // 0 → server default
    std::string correlation_id;
};

struct CreateResult {
    std::shared_ptr<Session> session;
    std::string canonical_model_id;
    std::chrono::system_clock::time_point expires_at;
};

class SessionManager {
   public:
    SessionManager(rekall::asr::engine::ModelRegistry* models,
                   rekall::asr::config::Config cfg,
                   rekall::asr::observ::Metrics* metrics);
    ~SessionManager();

    SessionManager(const SessionManager&)            = delete;
    SessionManager& operator=(const SessionManager&) = delete;

    // Creates a Pending session and stores it. The returned shared_ptr is
    // also retained internally; the bind() call resolves it on WS upgrade.
    CreateResult create(const CreateInput& in);

    // Returns the session for `sid`, or nullptr.
    std::shared_ptr<Session> get(std::string_view sid) const;

    // Transitions Pending → Active and stamps last_activity_at. Returns the
    // session if found and was Pending; nullptr otherwise.
    std::shared_ptr<Session> bind(std::string_view sid);

    // Removes the session from the table; returns its accumulated final
    // transcript and final_count. Idempotent: returns empty payload for
    // unknown / already-removed sids.
    struct EndOutcome {
        std::string final_transcript;
        std::uint32_t final_count = 0;
        bool present = false;
    };
    EndOutcome end(std::string_view sid);

    // Force-closes any sessions still in the table. Used during graceful drain.
    std::size_t force_close_remaining();

    std::size_t active_count() const;

   private:
    void sweeper_loop(std::stop_token st);
    void update_active_gauge();

    rekall::asr::engine::ModelRegistry* models_;
    rekall::asr::config::Config         cfg_;
    rekall::asr::observ::Metrics*       metrics_;

    mutable std::shared_mutex mu_;
    std::unordered_map<std::string, std::shared_ptr<Session>> by_sid_;

    std::jthread sweeper_;
};

}  // namespace rekall::asr::session
