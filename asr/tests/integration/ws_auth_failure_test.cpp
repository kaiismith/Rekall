#include "test_helpers.hpp"

#include <gtest/gtest.h>

#include <chrono>

#include <boost/asio.hpp>
#include <boost/beast.hpp>
#include <boost/beast/websocket.hpp>

namespace beast     = boost::beast;
namespace asio      = boost::asio;
namespace websocket = beast::websocket;
using tcp           = asio::ip::tcp;
using namespace rekall::asr;

namespace {

bool tries_handshake(int port, const std::string& token) {
    asio::io_context ioc;
    tcp::resolver resolver(ioc);
    auto endpoints = resolver.resolve("127.0.0.1", std::to_string(port));
    websocket::stream<tcp::socket> wsc(ioc);
    try {
        asio::connect(wsc.next_layer(), endpoints);
        wsc.set_option(websocket::stream_base::decorator(
            [](websocket::request_type& req) {
                req.set(beast::http::field::origin, "http://test");
            }));
        wsc.handshake("127.0.0.1", "/v1/asr/stream?token=" + token);
        return true;
    } catch (const std::exception&) {
        return false;
    }
}

}  // namespace

TEST(WsAuthFailure, GarbageTokenIsRejected) {
    if (!test::integration_enabled()) GTEST_SKIP();

    auto cfg = test::make_test_config(/*ws=*/18102, /*grpc=*/19102, /*pool=*/2);
    observ::init_logger("warn", "json");
    observ::Metrics metrics("");
    auto jti = std::make_shared<auth::JtiCache>(std::chrono::seconds(60));
    auth::JWTValidator validator(cfg.auth.token_secret, cfg.auth.token_audience,
                                 cfg.auth.token_issuer, jti);
    engine::ModelRegistry models(&metrics);
    ASSERT_TRUE(models.load(cfg.models.entries.front()));
    models.set_default(cfg.models.default_id);
    engine::WorkerPool workers(cfg.worker_pool.size, &metrics);
    session::SessionManager sessions(&models, cfg, &metrics);
    transport::GrpcServer grpc({&sessions, &models, &workers, &metrics, cfg,
                                std::chrono::system_clock::now(), "test"});
    transport::WsServer ws({&validator, &sessions, &workers, &models, &metrics, cfg});
    grpc.start();
    ws.start();

    EXPECT_FALSE(tries_handshake(18102, "not.a.jwt"));

    ws.stop();
    grpc.stop();
}

TEST(WsAuthFailure, UnknownSessionIsRejected) {
    if (!test::integration_enabled()) GTEST_SKIP();

    auto cfg = test::make_test_config(/*ws=*/18103, /*grpc=*/19103, /*pool=*/2);
    observ::init_logger("warn", "json");
    observ::Metrics metrics("");
    auto jti = std::make_shared<auth::JtiCache>(std::chrono::seconds(60));
    auth::JWTValidator validator(cfg.auth.token_secret, cfg.auth.token_audience,
                                 cfg.auth.token_issuer, jti);
    engine::ModelRegistry models(&metrics);
    ASSERT_TRUE(models.load(cfg.models.entries.front()));
    models.set_default(cfg.models.default_id);
    engine::WorkerPool workers(cfg.worker_pool.size, &metrics);
    session::SessionManager sessions(&models, cfg, &metrics);
    transport::GrpcServer grpc({&sessions, &models, &workers, &metrics, cfg,
                                std::chrono::system_clock::now(), "test"});
    transport::WsServer ws({&validator, &sessions, &workers, &models, &metrics, cfg});
    grpc.start();
    ws.start();

    // Token with valid signature but a sid that was never registered.
    auto token = test::sign_token(cfg, /*sid=*/"phantom-sid", /*jti=*/"jti-phantom");
    EXPECT_FALSE(tries_handshake(18103, token));

    ws.stop();
    grpc.stop();
}
