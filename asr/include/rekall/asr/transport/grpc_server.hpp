// gRPC control plane. Implements rekall.asr.v1.ASR over a separate listener
// — by default loopback only, mTLS-required for non-loopback binds.

#pragma once

#include <atomic>
#include <chrono>
#include <memory>
#include <string>

#include "rekall/asr/config/config.hpp"
#include "rekall/asr/engine/model_registry.hpp"
#include "rekall/asr/engine/worker_pool.hpp"
#include "rekall/asr/session/session_manager.hpp"

namespace grpc { class Server; }

namespace rekall::asr::observ { class Metrics; }

namespace rekall::asr::transport {

struct GrpcServerDeps {
    rekall::asr::session::SessionManager* sessions;
    rekall::asr::engine::ModelRegistry*   models;
    rekall::asr::engine::WorkerPool*      workers;
    rekall::asr::observ::Metrics*         metrics;
    rekall::asr::config::Config           cfg;
    std::chrono::system_clock::time_point started_at;
    std::string                           version;
};

class GrpcServer {
   public:
    explicit GrpcServer(GrpcServerDeps deps);
    ~GrpcServer();

    GrpcServer(const GrpcServer&)            = delete;
    GrpcServer& operator=(const GrpcServer&) = delete;

    // Builds, configures (mTLS if requested), and starts the gRPC server.
    void start();

    // Initiates graceful shutdown with the configured drain budget.
    void stop();

    // Atomically toggles the health status returned by Health RPC. Used by
    // the lifecycle code to flip to NOT_SERVING on SIGTERM.
    void set_serving(bool serving);

   private:
    GrpcServerDeps deps_;
    std::unique_ptr<grpc::Server> server_;
    class Impl;
    std::unique_ptr<Impl> impl_;
    std::atomic<bool> serving_{true};
};

}  // namespace rekall::asr::transport
