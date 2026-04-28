// Boost.Beast-based WebSocket data plane.
//
// One WsServer per process. On each accepted connection it spawns a WsSession
// instance which authenticates (JWTValidator), binds to a SessionManager
// session, then runs the read-loop / write-loop pair that bridges the WS
// frames ↔ the session's two ring buffers.

#pragma once

#include <atomic>
#include <memory>
#include <string>
#include <thread>
#include <vector>

#include <boost/asio.hpp>

#include "rekall/asr/auth/jwt_validator.hpp"
#include "rekall/asr/config/config.hpp"
#include "rekall/asr/engine/transcription_engine.hpp"
#include "rekall/asr/engine/worker_pool.hpp"
#include "rekall/asr/session/session_manager.hpp"

namespace rekall::asr::observ { class Metrics; }

namespace rekall::asr::transport {

struct WsServerDeps {
    rekall::asr::auth::JWTValidator*        validator;
    rekall::asr::session::SessionManager*   sessions;
    rekall::asr::engine::WorkerPool*        workers;
    rekall::asr::engine::ModelRegistry*     models;
    rekall::asr::observ::Metrics*           metrics;
    rekall::asr::config::Config             cfg;
};

class WsServer {
   public:
    explicit WsServer(WsServerDeps deps);
    ~WsServer();

    WsServer(const WsServer&)            = delete;
    WsServer& operator=(const WsServer&) = delete;

    // Starts accepting connections on cfg.server.ws_listen. Non-blocking;
    // the accept loop runs on an internal thread pool.
    void start();

    // Stops accepting new connections and unblocks the io_context. In-flight
    // sessions continue under SessionManager's drain.
    void stop();

   private:
    void run_io();
    void do_accept();

    WsServerDeps deps_;
    boost::asio::io_context io_;
    std::unique_ptr<boost::asio::ip::tcp::acceptor> acceptor_;
    std::vector<std::thread> threads_;
    std::atomic<bool> stopping_{false};
};

}  // namespace rekall::asr::transport
