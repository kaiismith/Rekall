#include "test_helpers.hpp"

#include <gtest/gtest.h>

#include <chrono>
#include <thread>

using namespace rekall::asr;

TEST(GracefulShutdown, FlipsHealthAndForceClosesRemaining) {
    if (!test::integration_enabled()) GTEST_SKIP();

    auto cfg = test::make_test_config(/*ws=*/18105, /*grpc=*/19105, /*pool=*/2);
    cfg.session.graceful_drain_seconds = 1;

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

    auto out = sessions.create({});
    EXPECT_EQ(sessions.active_count(), 1U);

    transport::GrpcServer grpc({&sessions, &models, &workers, &metrics, cfg,
                                std::chrono::system_clock::now(), "test"});
    transport::WsServer ws({&validator, &sessions, &workers, &models, &metrics, cfg});
    grpc.start();
    ws.start();

    grpc.set_serving(false);
    ws.stop();

    auto deadline = std::chrono::steady_clock::now() +
                    std::chrono::seconds(cfg.session.graceful_drain_seconds);
    while (sessions.active_count() > 0 && std::chrono::steady_clock::now() < deadline) {
        std::this_thread::sleep_for(std::chrono::milliseconds(50));
    }
    auto remaining = sessions.force_close_remaining();
    grpc.stop();

    EXPECT_GE(remaining, 1U);
    EXPECT_EQ(sessions.active_count(), 0U);
}
