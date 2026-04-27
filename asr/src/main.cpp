// rekall-asr — service entrypoint.
//
// Lifecycle:
//   1. parse args, load config (env-var > yaml > default)
//   2. init logger + metrics + tracer
//   3. build jti cache, JWT validator, model registry, worker pool, sessions
//   4. start metrics HTTP listener, gRPC control plane, WS data plane
//   5. wait for SIGTERM/SIGINT
//   6. graceful drain: flip health, stop accepting, wait drain budget
//   7. force-close remainder, return 0
//
// Exit codes: 0 clean, 1 startup failure, 2 unrecoverable runtime failure.

#include <atomic>
#include <chrono>
#include <csignal>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <iostream>
#include <memory>
#include <string>
#include <thread>

#include "rekall/asr/auth/jwt_validator.hpp"
#include "rekall/asr/config/config.hpp"
#include "rekall/asr/engine/model_registry.hpp"
#include "rekall/asr/engine/worker_pool.hpp"
#include "rekall/asr/observ/log_catalog.hpp"
#include "rekall/asr/observ/metrics.hpp"
#include "rekall/asr/observ/tracing.hpp"
#include "rekall/asr/session/session_manager.hpp"
#include "rekall/asr/transport/grpc_server.hpp"
#include "rekall/asr/transport/ws_server.hpp"

#ifndef _WIN32
#include <unistd.h>
#include <sys/types.h>
#endif

namespace {

constexpr const char* kVersion = "0.1.0";

std::atomic<int> g_signal{0};

void on_signal(int sig) { g_signal.store(sig); }

std::filesystem::path config_path_from_args(int argc, char** argv) {
    std::filesystem::path path = "/etc/rekall-asr/config.yaml";
    if (const char* env = std::getenv("ASR_CONFIG_PATH"); env != nullptr && *env != '\0') {
        path = env;
    }
    for (int i = 1; i + 1 < argc; ++i) {
        std::string a = argv[i];
        if (a == "--config" || a == "-c") {
            path = argv[i + 1];
            break;
        }
    }
    return path;
}

void maybe_drop_privs(const rekall::asr::config::DropPrivs& dp) {
#ifdef _WIN32
    (void)dp;
#else
    if (!dp.enabled) return;
    if (setgid(dp.gid) != 0) {
        rekall::asr::observ::error(rekall::asr::observ::FATAL, {
            {"reason", "setgid failed"}, {"gid", dp.gid},
        });
        std::exit(1);
    }
    if (setuid(dp.uid) != 0) {
        rekall::asr::observ::error(rekall::asr::observ::FATAL, {
            {"reason", "setuid failed"}, {"uid", dp.uid},
        });
        std::exit(1);
    }
#endif
}

}  // namespace

int main(int argc, char** argv) {
    rekall::asr::config::Config cfg;
    try {
        cfg = rekall::asr::config::Config::load(config_path_from_args(argc, argv));
    } catch (const std::exception& e) {
        std::fprintf(stderr, "FATAL [ASR_CONFIG_INVALID] %s\n", e.what());
        return 1;
    }

    rekall::asr::observ::init_logger(cfg.logging.level, cfg.logging.format);
    rekall::asr::observ::info(rekall::asr::observ::SERVICE_STARTING, {
        {"version", kVersion},
        {"ws_listen",      cfg.server.ws_listen},
        {"grpc_listen",    cfg.server.grpc_listen},
        {"metrics_listen", cfg.server.metrics_listen},
    });

    auto tracer = rekall::asr::observ::init_tracer(cfg.telemetry.otel_endpoint);

    std::unique_ptr<rekall::asr::observ::Metrics> metrics;
    try {
        metrics = std::make_unique<rekall::asr::observ::Metrics>(cfg.server.metrics_listen);
    } catch (const std::exception& e) {
        rekall::asr::observ::error(rekall::asr::observ::FATAL, {
            {"phase", "metrics_init"}, {"error", e.what()},
        });
        return 1;
    }

    // Audit log: which engine is doing inference?
    rekall::asr::observ::info(rekall::asr::observ::ENGINE_SELECTED, {
        {"mode",   std::string(rekall::asr::config::to_string(cfg.engine.mode))},
        {"target", cfg.engine.mode == rekall::asr::config::EngineMode::OpenAi
                       ? (cfg.engine.openai.base_url.empty()
                              ? std::string{"https://api.openai.com/v1"}
                              : cfg.engine.openai.base_url)
                       : std::string{"<local-models>"}},
        {"model",  cfg.engine.mode == rekall::asr::config::EngineMode::OpenAi
                       ? cfg.engine.openai.model
                       : cfg.models.default_id},
    });
    if (cfg.engine.mode == rekall::asr::config::EngineMode::OpenAi) {
        rekall::asr::observ::warn(rekall::asr::observ::DATA_LEAVES_HOST, {
            {"destination", cfg.engine.openai.base_url.empty()
                            ? std::string{"https://api.openai.com/v1"}
                            : cfg.engine.openai.base_url},
            {"model",       cfg.engine.openai.model},
        });
    }

    auto jti = std::make_shared<rekall::asr::auth::JtiCache>(
        std::chrono::seconds(cfg.auth.jti_cache_ttl_seconds));
    rekall::asr::auth::JWTValidator validator(cfg.auth.token_secret, cfg.auth.token_audience,
                                              cfg.auth.token_issuer, jti);

    rekall::asr::engine::ModelRegistry models(metrics.get());
    if (cfg.engine.mode == rekall::asr::config::EngineMode::Local) {
        bool any_loaded = false;
        for (const auto& e : cfg.models.entries) {
            any_loaded = models.load(e) || any_loaded;
        }
        if (!any_loaded) {
            rekall::asr::observ::error(rekall::asr::observ::FATAL, {
                {"reason", "no models could be loaded"},
            });
            return 1;
        }
        models.set_default(cfg.models.default_id);
    } else {
        // OpenAI mode: skip local model loading entirely. The session manager
        // tolerates an empty registry; the WS bind doesn't dereference
        // session->model when the openai engine runs.
        rekall::asr::observ::info(rekall::asr::observ::ENGINE_PROBE_OK, {
            {"mode", "openai"},
            {"note", "skipping local model registry — engine is openai"},
        });
    }

    rekall::asr::engine::WorkerPool workers(cfg.worker_pool.size, metrics.get());
    rekall::asr::session::SessionManager sessions(&models, cfg, metrics.get());

    rekall::asr::transport::GrpcServerDeps grpc_deps{
        .sessions    = &sessions,
        .models      = &models,
        .workers     = &workers,
        .metrics     = metrics.get(),
        .cfg         = cfg,
        .started_at  = std::chrono::system_clock::now(),
        .version     = kVersion,
    };
    rekall::asr::transport::GrpcServer grpc(grpc_deps);

    rekall::asr::transport::WsServerDeps ws_deps{
        .validator = &validator,
        .sessions  = &sessions,
        .workers   = &workers,
        .models    = &models,
        .metrics   = metrics.get(),
        .cfg       = cfg,
    };
    rekall::asr::transport::WsServer ws(ws_deps);

    try {
        grpc.start();
        ws.start();
    } catch (const std::exception& e) {
        rekall::asr::observ::error(rekall::asr::observ::FATAL, {
            {"phase", "transport_start"}, {"error", e.what()},
        });
        return 1;
    }

    maybe_drop_privs(cfg.drop_privs);

    std::signal(SIGINT,  on_signal);
    std::signal(SIGTERM, on_signal);

    rekall::asr::observ::info(rekall::asr::observ::SERVICE_READY, {
        {"version",       kVersion},
        {"models_loaded", static_cast<int>(models.loaded_ids().size())},
        {"workers",       static_cast<int>(workers.size())},
    });

    while (g_signal.load() == 0) {
        std::this_thread::sleep_for(std::chrono::milliseconds(200));
    }

    rekall::asr::observ::info(rekall::asr::observ::GRACEFUL_DRAIN_BEGIN, {
        {"signal", g_signal.load()},
        {"drain_seconds", cfg.session.graceful_drain_seconds},
    });

    grpc.set_serving(false);
    ws.stop();

    auto deadline = std::chrono::steady_clock::now() +
                    std::chrono::seconds(cfg.session.graceful_drain_seconds);
    while (sessions.active_count() > 0 && std::chrono::steady_clock::now() < deadline) {
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }
    auto remaining = sessions.force_close_remaining();
    grpc.stop();

    rekall::asr::observ::info(rekall::asr::observ::GRACEFUL_DRAIN_END, {
        {"force_closed", remaining},
    });

    return 0;
}
