// Test fixtures for the C++ integration suite. Spins up a gRPC + WS server
// against an in-memory configuration and exposes helpers to mint tokens and
// connect a Beast WS client.

#pragma once

#include <chrono>
#include <cstdlib>
#include <filesystem>
#include <memory>
#include <string>
#include <thread>

#include <jwt-cpp/jwt.h>

#include "rekall/asr/auth/jwt_validator.hpp"
#include "rekall/asr/config/config.hpp"
#include "rekall/asr/engine/model_registry.hpp"
#include "rekall/asr/engine/worker_pool.hpp"
#include "rekall/asr/observ/log_catalog.hpp"
#include "rekall/asr/observ/metrics.hpp"
#include "rekall/asr/session/session_manager.hpp"
#include "rekall/asr/transport/grpc_server.hpp"
#include "rekall/asr/transport/ws_server.hpp"

namespace rekall::asr::test {

inline bool integration_enabled() {
    const char* v = std::getenv("REKALL_ASR_INTEGRATION");
    return v != nullptr && (*v == '1' || *v == 't' || *v == 'T');
}

inline std::filesystem::path tiny_model_path() {
    if (const char* p = std::getenv("REKALL_ASR_TINY_MODEL"); p && *p) return p;
    return "/var/lib/rekall-asr/models/ggml-tiny.en.bin";
}

inline rekall::asr::config::Config make_test_config(int ws_port, int grpc_port,
                                                    std::size_t worker_pool_size = 4) {
    rekall::asr::config::Config c;
    c.server.ws_listen          = "127.0.0.1:" + std::to_string(ws_port);
    c.server.grpc_listen        = "127.0.0.1:" + std::to_string(grpc_port);
    c.server.metrics_listen     = "";
    c.server.allow_insecure_ws  = true;
    c.server.ws_allowed_origins = {};   // dev wildcard

    c.auth.token_secret  = "0123456789abcdef0123456789abcdef0123456789ab";
    c.auth.token_secret_env = "ASR_TOKEN_SECRET";
    c.auth.token_audience = "rekall-asr";
    c.auth.token_issuer   = "rekall-backend";
    c.auth.token_default_ttl_seconds = 60;
    c.auth.token_min_ttl_seconds     = 60;
    c.auth.token_max_ttl_seconds     = 300;
    c.auth.jti_cache_ttl_seconds     = 60;

    c.worker_pool.size = static_cast<std::uint32_t>(worker_pool_size);
    c.session.idle_timeout_seconds   = 5;
    c.session.hard_timeout_seconds   = 60;
    c.session.graceful_drain_seconds = 2;

    rekall::asr::config::ModelEntry m;
    m.id        = "tiny.en";
    m.path      = tiny_model_path().string();
    m.language  = "en";
    m.n_threads = 1;
    m.beam_size = 1;
    c.models.entries.push_back(m);
    c.models.default_id = "tiny.en";

    return c;
}

inline std::string sign_token(const rekall::asr::config::Config& cfg,
                              const std::string& sid,
                              const std::string& jti = "jti-" + std::to_string(std::rand()),
                              std::chrono::seconds ttl = std::chrono::seconds{60}) {
    auto now = std::chrono::system_clock::now();
    return jwt::create()
        .set_issuer(cfg.auth.token_issuer)
        .set_audience(cfg.auth.token_audience)
        .set_subject("user-1")
        .set_id(jti)
        .set_payload_claim("sid",   jwt::claim(sid))
        .set_payload_claim("cid",   jwt::claim(std::string("call-1")))
        .set_payload_claim("model", jwt::claim(std::string("tiny.en")))
        .set_issued_at(now)
        .set_not_before(now - std::chrono::seconds{1})
        .set_expires_at(now + ttl)
        .sign(jwt::algorithm::hs256{cfg.auth.token_secret});
}

}  // namespace rekall::asr::test
