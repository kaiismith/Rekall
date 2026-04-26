// Session_Token (HS256 JWT) validation for the WebSocket data plane.
//
// The Go backend mints a token after registering the session over gRPC and
// hands it to the browser. The browser presents it as `?token=<jwt>` on the
// WS upgrade. JWTValidator::validate verifies signature, claims, and consumes
// the jti (single-use) atomically.
//
// All failure modes collapse to WS close 4401 at the call site; the
// discriminator only goes to the log line and to the
// `asr_auth_failed_total{reason}` Prometheus counter.

#pragma once

#include <absl/container/flat_hash_map.h>

#include <chrono>
#include <memory>
#include <optional>
#include <shared_mutex>
#include <stop_token>
#include <string>
#include <string_view>
#include <thread>
#include <variant>

namespace rekall::asr::auth {

enum class TokenError {
    InvalidSignature,
    Expired,
    NotYetValid,
    WrongAudience,
    WrongIssuer,
    UnknownSession,    // sid not currently registered with the SessionManager
    JtiReplay,
    Malformed,
};

// Maps TokenError to a stable lowercase reason string for metrics/logs.
std::string_view reason(TokenError e) noexcept;

struct TokenClaims {
    std::string sub;            // user_id (uuid)
    std::string sid;            // asr session_id (uuid)
    std::string cid;            // call_id (uuid)
    std::string model;          // requested model_id
    std::string jti;            // unique token id
    std::chrono::system_clock::time_point exp;
};

// Bounded TTL cache enforcing single-use semantics on the jti claim.
// Entries auto-expire via a sweeper goroutine; cache TTL is the configured
// `jti_cache_ttl_seconds`.
class JtiCache {
   public:
    explicit JtiCache(std::chrono::seconds ttl);
    ~JtiCache();

    JtiCache(const JtiCache&)            = delete;
    JtiCache& operator=(const JtiCache&) = delete;

    // Returns true on first insertion, false if the jti was already present
    // (replay). Successful insertion records `exp` so the sweeper can evict
    // it once the absolute expiry has passed.
    bool try_consume(std::string_view jti, std::chrono::system_clock::time_point exp);

    // Removes entries whose recorded exp is in the past. Called by the sweeper
    // every 30 s; safe to call from tests directly.
    void sweep_expired();

    std::size_t size() const noexcept;

   private:
    void sweeper_loop(std::stop_token st);

    mutable std::shared_mutex mu_;
    absl::flat_hash_map<std::string, std::chrono::system_clock::time_point> seen_;
    std::chrono::seconds ttl_;
    std::jthread sweeper_;
};

class JWTValidator {
   public:
    JWTValidator(std::string secret, std::string audience, std::string issuer,
                 std::shared_ptr<JtiCache> jti);

    // Validates signature, claims, and consumes the jti atomically. Returns
    // the parsed claims on success, or a TokenError on any failure.
    std::variant<TokenClaims, TokenError> validate(std::string_view token) const;

   private:
    std::string secret_;
    std::string audience_;
    std::string issuer_;
    std::shared_ptr<JtiCache> jti_;
};

}  // namespace rekall::asr::auth
