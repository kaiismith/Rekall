#include "rekall/asr/auth/jwt_validator.hpp"

#include <gtest/gtest.h>

#include <chrono>
#include <thread>
#include <vector>

using namespace rekall::asr::auth;
using namespace std::chrono_literals;

TEST(JtiCacheTest, FirstConsumeSucceedsSecondReplays) {
    JtiCache cache(60s);
    auto exp = std::chrono::system_clock::now() + 60s;
    EXPECT_TRUE (cache.try_consume("jti-a", exp));
    EXPECT_FALSE(cache.try_consume("jti-a", exp));
}

TEST(JtiCacheTest, IndependentJtisCoexist) {
    JtiCache cache(60s);
    auto exp = std::chrono::system_clock::now() + 60s;
    EXPECT_TRUE(cache.try_consume("a", exp));
    EXPECT_TRUE(cache.try_consume("b", exp));
    EXPECT_EQ(cache.size(), 2U);
}

TEST(JtiCacheTest, SweepRemovesExpired) {
    JtiCache cache(0s);
    auto past = std::chrono::system_clock::now() - 5s;
    cache.try_consume("expired", past);
    EXPECT_EQ(cache.size(), 1U);
    cache.sweep_expired();
    EXPECT_EQ(cache.size(), 0U);
}

TEST(JtiCacheTest, ConcurrentConsumeOfSameJti_ExactlyOneSucceeds) {
    JtiCache cache(60s);
    auto exp = std::chrono::system_clock::now() + 60s;
    constexpr int N = 64;
    std::vector<std::thread> threads;
    std::atomic<int> wins{0};
    for (int i = 0; i < N; ++i) {
        threads.emplace_back([&]() {
            if (cache.try_consume("race", exp)) wins.fetch_add(1);
        });
    }
    for (auto& t : threads) t.join();
    EXPECT_EQ(wins.load(), 1);
}
