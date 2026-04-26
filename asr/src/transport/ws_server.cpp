#include "rekall/asr/transport/ws_server.hpp"

#include <algorithm>
#include <thread>
#include <utility>

#include <boost/asio.hpp>

#include "rekall/asr/observ/log_catalog.hpp"
#include "rekall/asr/transport/ws_session.hpp"

namespace rekall::asr::transport {

namespace {

asio::ip::tcp::endpoint parse_listen(const std::string& s) {
    auto colon = s.rfind(':');
    if (colon == std::string::npos) {
        throw std::runtime_error("invalid listen address: " + s);
    }
    std::string host = s.substr(0, colon);
    auto port_n = static_cast<unsigned short>(std::stoi(s.substr(colon + 1)));
    if (host == "0.0.0.0" || host.empty()) {
        return {asio::ip::tcp::v4(), port_n};
    }
    return {asio::ip::make_address(host), port_n};
}

}  // namespace

WsServer::WsServer(WsServerDeps deps) : deps_(std::move(deps)) {}

WsServer::~WsServer() { stop(); }

void WsServer::start() {
    auto endpoint = parse_listen(deps_.cfg.server.ws_listen);
    acceptor_ = std::make_unique<asio::ip::tcp::acceptor>(io_, endpoint);
    do_accept();

    // Two IO threads is plenty for the WS workload (the heavy lifting happens
    // on worker threads, not on the IO loop).
    const std::size_t io_threads = 2;
    for (std::size_t i = 0; i < io_threads; ++i) {
        threads_.emplace_back([this] { run_io(); });
    }
    rekall::asr::observ::info(rekall::asr::observ::SERVICE_READY, {
        {"component", "ws_server"},
        {"listen",    deps_.cfg.server.ws_listen},
    });
}

void WsServer::stop() {
    if (stopping_.exchange(true)) return;
    boost::system::error_code ec;
    if (acceptor_) acceptor_->close(ec);
    io_.stop();
    for (auto& t : threads_) {
        if (t.joinable()) t.join();
    }
    threads_.clear();
}

void WsServer::run_io() {
    try {
        io_.run();
    } catch (const std::exception& e) {
        rekall::asr::observ::error(rekall::asr::observ::FATAL, {
            {"component", "ws_server.io"},
            {"error",     e.what()},
        });
    }
}

void WsServer::do_accept() {
    if (!acceptor_) return;
    acceptor_->async_accept(
        [this](boost::system::error_code ec, asio::ip::tcp::socket socket) {
            if (!stopping_.load() && !ec) {
                std::make_shared<WsSession>(std::move(socket), deps_)->run();
            }
            if (!stopping_.load()) do_accept();
        });
}

}  // namespace rekall::asr::transport
