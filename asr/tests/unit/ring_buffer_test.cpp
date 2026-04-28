#include "rekall/asr/audio/ring_buffer.hpp"

#include <gtest/gtest.h>

#include <atomic>
#include <thread>
#include <vector>

using namespace rekall::asr::audio;

TEST(RingBufferTest, FifoOrder) {
    RingBuffer<int> rb(4);
    EXPECT_TRUE(rb.try_push(1));
    EXPECT_TRUE(rb.try_push(2));
    EXPECT_TRUE(rb.try_push(3));
    EXPECT_EQ(rb.try_pop().value(), 1);
    EXPECT_EQ(rb.try_pop().value(), 2);
    EXPECT_EQ(rb.try_pop().value(), 3);
    EXPECT_FALSE(rb.try_pop().has_value());
}

TEST(RingBufferTest, FullPushReturnsFalse) {
    RingBuffer<int> rb(2);
    EXPECT_TRUE(rb.try_push(10));
    EXPECT_TRUE(rb.try_push(11));
    EXPECT_FALSE(rb.try_push(12));
    EXPECT_TRUE(rb.full());
}

TEST(RingBufferTest, DropOldestWhenFull) {
    RingBuffer<int> rb(2);
    EXPECT_FALSE(rb.push_or_drop_oldest(1));
    EXPECT_FALSE(rb.push_or_drop_oldest(2));
    EXPECT_TRUE (rb.push_or_drop_oldest(3));   // dropped 1
    EXPECT_EQ(rb.try_pop().value(), 2);
    EXPECT_EQ(rb.try_pop().value(), 3);
}

TEST(RingBufferTest, ConcurrentSpscIsSafe) {
    RingBuffer<int> rb(64);
    constexpr int N = 10000;
    std::atomic<bool> done{false};
    std::vector<int> out;
    out.reserve(N);

    std::thread consumer([&]() {
        std::stop_source stop;
        while (out.size() < static_cast<std::size_t>(N)) {
            auto v = rb.try_pop();
            if (v) out.push_back(*v);
            else std::this_thread::yield();
        }
        done.store(true);
    });
    std::thread producer([&]() {
        for (int i = 0; i < N; ++i) {
            while (!rb.try_push(i)) std::this_thread::yield();
        }
    });

    producer.join();
    consumer.join();
    ASSERT_EQ(out.size(), static_cast<std::size_t>(N));
    for (int i = 0; i < N; ++i) EXPECT_EQ(out[i], i);
}
