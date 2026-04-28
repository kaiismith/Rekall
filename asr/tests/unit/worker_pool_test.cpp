#include "rekall/asr/engine/worker_pool.hpp"

#include <gtest/gtest.h>

#include <atomic>
#include <thread>
#include <vector>

using namespace rekall::asr::engine;

TEST(WorkerPoolTest, AcquireReturnsHandleWhenFree) {
    WorkerPool pool(2);
    auto a = pool.try_acquire();
    auto b = pool.try_acquire();
    EXPECT_TRUE(a.has_value());
    EXPECT_TRUE(b.has_value());
    EXPECT_EQ(pool.in_use(), 2U);
}

TEST(WorkerPoolTest, ReturnsNulloptWhenSaturated) {
    WorkerPool pool(1);
    auto a = pool.try_acquire();
    auto b = pool.try_acquire();
    EXPECT_TRUE (a.has_value());
    EXPECT_FALSE(b.has_value());
}

TEST(WorkerPoolTest, ReleaseReclaims) {
    WorkerPool pool(1);
    {
        auto a = pool.try_acquire();
        EXPECT_TRUE(a.has_value());
        EXPECT_EQ(pool.in_use(), 1U);
    }
    EXPECT_EQ(pool.in_use(), 0U);
    auto b = pool.try_acquire();
    EXPECT_TRUE(b.has_value());
}

TEST(WorkerPoolTest, ConcurrentStormLeavesAccountingConsistent) {
    constexpr std::size_t kSize = 8;
    WorkerPool pool(kSize);
    constexpr int N = 200;
    std::vector<std::thread> threads;
    std::atomic<int> grants{0};
    for (int i = 0; i < N; ++i) {
        threads.emplace_back([&]() {
            auto h = pool.try_acquire();
            if (h) {
                grants.fetch_add(1);
                std::this_thread::sleep_for(std::chrono::milliseconds(1));
            }
        });
    }
    for (auto& t : threads) t.join();
    EXPECT_EQ(pool.in_use(), 0U);
    EXPECT_GE(grants.load(), static_cast<int>(kSize));  // at least one full saturation
}
