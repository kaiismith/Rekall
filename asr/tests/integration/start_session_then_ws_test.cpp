#include "test_helpers.hpp"

#include <gtest/gtest.h>

#include <chrono>
#include <thread>

#include <boost/asio.hpp>
#include <boost/beast.hpp>
#include <boost/beast/websocket.hpp>
#include <nlohmann/json.hpp>

namespace beast     = boost::beast;
namespace asio      = boost::asio;
namespace websocket = beast::websocket;
using tcp           = asio::ip::tcp;

using namespace rekall::asr;

TEST(StartSessionThenWS, ConnectsAndReceivesReadyEvent) {
    if (!test::integration_enabled()) {
        GTEST_SKIP() << "set REKALL_ASR_INTEGRATION=1 to run integration suite";
    }

    auto cfg = test::make_test_config(/*ws=*/18101, /*grpc=*/19101);

    observ::init_logger("warn", "json");
    observ::Metrics metrics(/*listen=*/"");

    auto jti = std::make_shared<auth::JtiCache>(std::chrono::seconds(60));
    auth::JWTValidator validator(cfg.auth.token_secret, cfg.auth.token_audience,
                                 cfg.auth.token_issuer, jti);

    engine::ModelRegistry models(&metrics);
    ASSERT_TRUE(models.load(cfg.models.entries.front()));
    models.set_default(cfg.models.default_id);

    engine::WorkerPool   workers(cfg.worker_pool.size, &metrics);
    session::SessionManager sessions(&models, cfg, &metrics);

    transport::GrpcServer grpc({.sessions = &sessions, .models = &models, .workers = &workers,
                                .metrics  = &metrics, .cfg = cfg,
                                .started_at = std::chrono::system_clock::now(),
                                .version = "test"});
    transport::WsServer ws({.validator = &validator, .sessions = &sessions, .workers = &workers,
                            .models = &models, .metrics = &metrics, .cfg = cfg});
    grpc.start();
    ws.start();

    // Pre-register a session directly with the SessionManager (bypassing gRPC).
    session::CreateInput ci;
    ci.user_id = "user-1";
    ci.call_id = "call-1";
    auto out = sessions.create(ci);
    auto sid = out.session->sid;
    auto token = test::sign_token(cfg, sid);

    // Open the WS client.
    asio::io_context ioc;
    tcp::resolver resolver(ioc);
    auto endpoints = resolver.resolve("127.0.0.1", "18101");
    websocket::stream<tcp::socket> wsc(ioc);
    asio::connect(wsc.next_layer(), endpoints);
    wsc.set_option(websocket::stream_base::decorator(
        [](websocket::request_type& req) { req.set(beast::http::field::origin, "http://test"); }));
    wsc.handshake("127.0.0.1:18101", "/v1/asr/stream?token=" + token);

    // First text frame should be a ready/info event.
    beast::flat_buffer buf;
    wsc.read(buf);
    auto payload = beast::buffers_to_string(buf.data());
    auto j = nlohmann::json::parse(payload);
    EXPECT_TRUE(j["type"] == "info" || j["type"] == "ready");

    beast::error_code ec;
    wsc.close(websocket::close_code::normal, ec);
    ws.stop();
    grpc.stop();
}
