// Per-connection WebSocket coroutine pair: read loop forwards binary PCM into
// the session's inbound queue, write loop drains the outbound queue and emits
// JSON text frames.

#pragma once

#include <chrono>
#include <memory>
#include <string>

#include <boost/asio.hpp>
#include <boost/beast/core.hpp>
#include <boost/beast/websocket.hpp>

#include "rekall/asr/transport/ws_server.hpp"

namespace rekall::asr::transport {

namespace beast     = boost::beast;
namespace asio      = boost::asio;
namespace websocket = beast::websocket;
using tcp           = asio::ip::tcp;

class WsSession : public std::enable_shared_from_this<WsSession> {
   public:
    WsSession(tcp::socket socket, WsServerDeps deps);

    // Performs the HTTP upgrade handshake and dispatches into the WS lifecycle.
    void run();

   private:
    void do_handshake();
    void on_handshake(beast::error_code ec);

    void authenticate(const std::string& target);
    void send_ready();
    void start_loops();

    void do_read();
    void on_read(beast::error_code ec, std::size_t bytes);

    void do_write_one();
    void on_write(beast::error_code ec, std::size_t bytes);

    void schedule_idle_timer();
    void schedule_hard_timer();
    void on_idle_timer(beast::error_code ec);
    void on_hard_timer(beast::error_code ec);

    void close_with(int code, std::string_view reason);

    websocket::stream<beast::tcp_stream> ws_;
    asio::strand<asio::any_io_executor>  strand_;
    asio::steady_timer                   idle_timer_;
    asio::steady_timer                   hard_timer_;

    beast::flat_buffer in_buf_;
    std::string        out_buf_;

    WsServerDeps                                deps_;
    std::shared_ptr<rekall::asr::session::Session> session_;

    std::chrono::steady_clock::time_point last_audio_at_;
    bool first_audio_seen_ = false;
    bool closed_           = false;
    bool writing_          = false;
};

}  // namespace rekall::asr::transport
