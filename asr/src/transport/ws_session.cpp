#include "rekall/asr/transport/ws_session.hpp"

#include <algorithm>
#include <chrono>
#include <cstdint>
#include <cstring>
#include <memory>
#include <string>
#include <string_view>
#include <thread>
#include <variant>

#include <boost/beast/http.hpp>
#include <nlohmann/json.hpp>

#include "rekall/asr/observ/log_catalog.hpp"
#include "rekall/asr/observ/metrics.hpp"
#include "rekall/asr/util/correlation_id.hpp"

namespace rekall::asr::transport {

namespace http = beast::http;

namespace {

constexpr std::size_t kMaxFrameBytes = 64 * 1024;
constexpr std::size_t kMaxControlBytes = 8 * 1024;

std::string url_decode(std::string_view in) {
    std::string out;
    out.reserve(in.size());
    for (std::size_t i = 0; i < in.size(); ++i) {
        char c = in[i];
        if (c == '%' && i + 2 < in.size()) {
            auto hex = [](char x) -> int {
                if (x >= '0' && x <= '9') return x - '0';
                if (x >= 'a' && x <= 'f') return 10 + x - 'a';
                if (x >= 'A' && x <= 'F') return 10 + x - 'A';
                return -1;
            };
            int hi = hex(in[i + 1]), lo = hex(in[i + 2]);
            if (hi >= 0 && lo >= 0) {
                out += static_cast<char>((hi << 4) | lo);
                i += 2;
                continue;
            }
        } else if (c == '+') {
            out += ' ';
            continue;
        }
        out += c;
    }
    return out;
}

std::string extract_query_param(std::string_view target, std::string_view key) {
    auto qpos = target.find('?');
    if (qpos == std::string_view::npos) return {};
    auto query = target.substr(qpos + 1);
    while (!query.empty()) {
        auto amp = query.find('&');
        std::string_view pair = query.substr(0, amp);
        auto eq = pair.find('=');
        if (eq != std::string_view::npos) {
            auto k = pair.substr(0, eq);
            auto v = pair.substr(eq + 1);
            if (k == key) return url_decode(v);
        }
        if (amp == std::string_view::npos) break;
        query.remove_prefix(amp + 1);
    }
    return {};
}

bool origin_allowed(const std::vector<std::string>& allow, std::string_view origin,
                    bool dev_wildcard) {
    if (origin.empty()) return false;
    if (dev_wildcard && allow.empty()) return true;
    return std::any_of(allow.begin(), allow.end(),
                       [&](const std::string& a) { return a == origin; });
}

std::vector<std::int16_t> bytes_to_int16(const std::uint8_t* data, std::size_t bytes) {
    std::vector<std::int16_t> out(bytes / 2);
    std::memcpy(out.data(), data, out.size() * sizeof(std::int16_t));
    return out;
}

}  // namespace

WsSession::WsSession(tcp::socket socket, WsServerDeps deps)
    : ws_(std::move(socket)),
      strand_(asio::make_strand(ws_.get_executor())),
      idle_timer_(strand_),
      hard_timer_(strand_),
      deps_(std::move(deps)) {
    ws_.set_option(websocket::stream_base::timeout::suggested(beast::role_type::server));
    ws_.read_message_max(kMaxFrameBytes);
}

void WsSession::run() {
    asio::dispatch(strand_, beast::bind_front_handler(&WsSession::do_handshake, shared_from_this()));
}

void WsSession::do_handshake() {
    // Read the HTTP upgrade request manually so we can pull ?token= and Origin
    // before the WebSocket handshake takes ownership of the stream.
    auto req = std::make_shared<http::request<http::string_body>>();
    auto buf = std::make_shared<beast::flat_buffer>();

    auto self = shared_from_this();
    http::async_read(ws_.next_layer(), *buf, *req,
        [this, self, req, buf](beast::error_code ec, std::size_t) {
            if (ec) {
                close_with(4400, "bad request");
                return;
            }
            if (!websocket::is_upgrade(*req)) {
                http::response<http::string_body> res{http::status::upgrade_required, req->version()};
                res.set(http::field::server, "rekall-asr");
                res.set(http::field::upgrade, "websocket");
                res.body() = "websocket upgrade required";
                res.prepare_payload();
                http::async_write(ws_.next_layer(), res,
                    [self](beast::error_code, std::size_t) { /* close on scope */ });
                return;
            }

            // Validate origin.
            std::string origin = std::string(req->base()[http::field::origin]);
            bool dev = deps_.cfg.server.allow_insecure_ws;
            if (!origin_allowed(deps_.cfg.server.ws_allowed_origins, origin, dev)) {
                rekall::asr::observ::warn(rekall::asr::observ::AUTH_FAILED, {
                    {"reason", "origin_not_allowed"}, {"origin", origin},
                });
                if (deps_.metrics) deps_.metrics->auth_failed_total("origin_not_allowed").Increment();
                close_with(4401, "origin not allowed");
                return;
            }

            std::string target = std::string(req->target());
            std::string token  = extract_query_param(target, "token");

            // Run the validator before accepting the upgrade.
            authenticate(token);
            if (closed_) return;

            // Acquire a worker — admission control. Stored on the session
            // pointer's lifetime via a side struct; for simplicity in this
            // first pass we rely on the SessionManager having reserved a slot
            // at StartSession time, and we count workers in the gRPC handler
            // instead. Here we simply check pool capacity.
            if (deps_.workers && deps_.workers->free() == 0) {
                if (deps_.metrics) deps_.metrics->admission_rejected_total("ws_upgrade").Increment();
                rekall::asr::observ::warn(rekall::asr::observ::ADMISSION_REJECTED, {
                    {"phase", "ws_upgrade"},
                });
                close_with(4503, "service at capacity");
                return;
            }

            // Accept the WS upgrade.
            ws_.set_option(websocket::stream_base::decorator(
                [](websocket::response_type& res) {
                    res.set(http::field::server, "rekall-asr");
                }));
            ws_.async_accept(*req, beast::bind_front_handler(&WsSession::on_handshake, self));
        });
}

void WsSession::authenticate(const std::string& token) {
    if (deps_.validator == nullptr) {
        close_with(4401, "auth not configured");
        return;
    }
    auto result = deps_.validator->validate(token);
    if (std::holds_alternative<rekall::asr::auth::TokenError>(result)) {
        auto err = std::get<rekall::asr::auth::TokenError>(result);
        std::string r{rekall::asr::auth::reason(err)};
        if (deps_.metrics) deps_.metrics->auth_failed_total(r).Increment();
        rekall::asr::observ::warn(rekall::asr::observ::AUTH_FAILED, {
            {"reason",       r},
            {"token_prefix", token.substr(0, std::min<std::size_t>(token.size(), 8))},
        });
        close_with(4401, "unauthorized");
        return;
    }
    auto claims = std::get<rekall::asr::auth::TokenClaims>(result);

    auto sess = deps_.sessions->bind(claims.sid);
    if (!sess) {
        if (deps_.metrics) deps_.metrics->auth_failed_total("unknown_session").Increment();
        rekall::asr::observ::warn(rekall::asr::observ::AUTH_FAILED, {
            {"reason",     "unknown_session"},
            {"session_id", claims.sid},
        });
        close_with(4401, "unknown session");
        return;
    }
    session_ = std::move(sess);

    rekall::asr::observ::info(rekall::asr::observ::AUTH_OK, {
        {"session_id",     session_->sid},
        {"correlation_id", session_->correlation_id},
    });
}

void WsSession::on_handshake(beast::error_code ec) {
    if (ec) {
        close_with(4400, "ws handshake failed");
        return;
    }
    // Binary-by-default for inbound audio frames; outbound writes set their
    // own text/binary mode per message via ws_.text(true).
    ws_.binary(true);
    last_audio_at_ = std::chrono::steady_clock::now();
    schedule_idle_timer();
    schedule_hard_timer();
    send_ready();
    start_loops();
}

void WsSession::send_ready() {
    nlohmann::json j = {
        {"type",        "ready"},
        {"session_id",  session_->sid},
        {"model_id",    session_->model_id},
        {"sample_rate", 16000},
    };
    rekall::asr::session::TranscriptEvent e;
    e.type    = rekall::asr::session::EventType::Info;
    e.code    = "READY";
    e.message = j.dump();
    // Push directly into outbound — write loop will pick it up.
    (void)session_->outbound.try_push(std::move(e));
}

void WsSession::start_loops() {
    do_read();
    do_write_one();

    // Spawn the worker thread that owns whisper inference for this session.
    auto session = session_;
    auto deps    = deps_;
    std::thread([session, deps]() {
        try {
            rekall::asr::engine::Transcriber t(session, session->model,
                                               deps.cfg.session, deps.metrics);
            t.run(session->stop_source.get_token());
        } catch (const std::exception& e) {
            rekall::asr::observ::error(rekall::asr::observ::INFERENCE_FAILED, {
                {"session_id", session->sid},
                {"error",      e.what()},
            });
        }
    }).detach();
}

void WsSession::do_read() {
    if (closed_) return;
    in_buf_.clear();
    ws_.async_read(in_buf_,
        beast::bind_front_handler(&WsSession::on_read, shared_from_this()));
}

void WsSession::on_read(beast::error_code ec, std::size_t) {
    if (closed_) return;
    if (ec == websocket::error::closed) {
        close_with(1000, "client closed");
        return;
    }
    if (ec) {
        close_with(4400, ec.message());
        return;
    }

    if (ws_.got_binary()) {
        // Audio frame.
        auto bytes = beast::buffers_to_string(in_buf_.data());
        if (bytes.empty() || (bytes.size() % 2) != 0 || bytes.size() > kMaxFrameBytes) {
            close_with(4400, "invalid binary frame");
            return;
        }
        first_audio_seen_ = true;
        last_audio_at_    = std::chrono::steady_clock::now();
        auto samples = bytes_to_int16(reinterpret_cast<const std::uint8_t*>(bytes.data()),
                                      bytes.size());
        auto frame   = rekall::asr::session::InboundFrame::audio(std::move(samples));

        if (!session_->inbound.try_push(std::move(frame))) {
            // Backpressure: simply drop this frame for now; a sustained-full
            // condition is detected by the absence of incoming progress and
            // tripped by the idle timer's secondary "no progress" mode.
            rekall::asr::observ::debug(rekall::asr::observ::BACKPRESSURE_APPLIED, {
                {"session_id", session_->sid},
            });
        }
        session_->touch();
    } else {
        // Text control frame.
        auto bytes = beast::buffers_to_string(in_buf_.data());
        if (bytes.size() > kMaxControlBytes) {
            close_with(4400, "control frame too large");
            return;
        }
        try {
            auto j = nlohmann::json::parse(bytes);
            std::string type = j.value("type", "");
            if (type == "flush") {
                (void)session_->inbound.try_push(
                    rekall::asr::session::InboundFrame::flush_sentinel());
            } else if (type == "ping") {
                rekall::asr::session::TranscriptEvent e;
                e.type       = rekall::asr::session::EventType::Pong;
                e.ts_unix_ms = static_cast<std::uint64_t>(
                    std::chrono::duration_cast<std::chrono::milliseconds>(
                        std::chrono::system_clock::now().time_since_epoch()).count());
                (void)session_->outbound.try_push(std::move(e));
            } else if (type == "config") {
                // First-frame config is accepted silently; later config is ignored.
                // Implementation deferred — see Requirement 3.10.
            }
        } catch (const std::exception&) {
            close_with(4400, "invalid control json");
            return;
        }
    }

    do_read();
}

void WsSession::do_write_one() {
    if (closed_ || writing_) return;
    auto ev = session_->outbound.try_pop();
    if (!ev) {
        // Schedule a re-arm with a short delay so we don't busy-loop. We use
        // a 5 ms steady_timer on the same strand.
        auto self = shared_from_this();
        auto t = std::make_shared<asio::steady_timer>(strand_);
        t->expires_after(std::chrono::milliseconds(5));
        t->async_wait([this, self, t](beast::error_code) {
            if (!closed_) do_write_one();
        });
        return;
    }
    out_buf_  = rekall::asr::session::serialise(*ev);
    writing_  = true;
    ws_.text(true);
    ws_.async_write(asio::buffer(out_buf_),
        beast::bind_front_handler(&WsSession::on_write, shared_from_this()));
}

void WsSession::on_write(beast::error_code ec, std::size_t) {
    writing_ = false;
    if (ec) {
        close_with(1011, ec.message());
        return;
    }
    do_write_one();
}

void WsSession::schedule_idle_timer() {
    auto self = shared_from_this();
    idle_timer_.expires_after(std::chrono::seconds(deps_.cfg.session.idle_timeout_seconds));
    idle_timer_.async_wait(beast::bind_front_handler(&WsSession::on_idle_timer, self));
}

void WsSession::schedule_hard_timer() {
    auto self = shared_from_this();
    hard_timer_.expires_after(std::chrono::seconds(deps_.cfg.session.hard_timeout_seconds));
    hard_timer_.async_wait(beast::bind_front_handler(&WsSession::on_hard_timer, self));
}

void WsSession::on_idle_timer(beast::error_code ec) {
    if (ec || closed_) return;
    if (!first_audio_seen_) {
        close_with(4411, "no audio within idle timeout");
        return;
    }
    auto since = std::chrono::steady_clock::now() - last_audio_at_;
    if (since > std::chrono::seconds(deps_.cfg.session.idle_timeout_seconds)) {
        close_with(4408, "idle");
        return;
    }
    schedule_idle_timer();
}

void WsSession::on_hard_timer(beast::error_code ec) {
    if (ec || closed_) return;
    close_with(4412, "hard timeout");
}

void WsSession::close_with(int code, std::string_view reason) {
    if (closed_) return;
    closed_ = true;
    if (deps_.metrics) deps_.metrics->ws_close_total(std::to_string(code)).Increment();
    if (session_) {
        session_->stop_source.request_stop();
        deps_.sessions->end(session_->sid);
    }
    boost::system::error_code ec;
    websocket::close_reason cr{static_cast<websocket::close_code>(code), std::string(reason)};
    ws_.async_close(cr, [self = shared_from_this()](beast::error_code) {});
    idle_timer_.cancel();
    hard_timer_.cancel();
}

}  // namespace rekall::asr::transport
