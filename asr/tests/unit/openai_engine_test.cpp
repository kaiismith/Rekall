// Drives the OpenAiEngine through canned FakeOpenAiClient responses to verify
// segmentation, retry, and fail-stop behaviours. Compiled only when the
// build includes the openai engine.

#include "../fakes/fake_openai_client.hpp"
#include "rekall/asr/audio/ring_buffer.hpp"
#include "rekall/asr/config/config.hpp"
#include "rekall/asr/engine/openai/openai_engine.hpp"
#include "rekall/asr/session/session.hpp"

#include <chrono>
#include <cstdint>
#include <gtest/gtest.h>
#include <memory>
#include <thread>
#include <vector>

namespace {

using rekall::asr::engine::openai::OpenAiTranscript;
using rekall::asr::engine::openai::OpenAiError;

std::shared_ptr<rekall::asr::session::Session> make_session() {
    auto s = std::make_shared<rekall::asr::session::Session>(
        /*inbound_capacity=*/64, /*outbound_capacity=*/64);
    s->sid = "22222222-2222-2222-2222-222222222222";
    return s;
}

// Generates `seconds` of synthetic 16 kHz int16 audio with non-trivial energy
// so the VAD treats it as speech.
std::vector<std::int16_t> tone(double seconds, std::int16_t amp = 12000) {
    const std::size_t n = static_cast<std::size_t>(seconds * 16000.0);
    std::vector<std::int16_t> v(n);
    for (std::size_t i = 0; i < n; ++i) v[i] = (i % 2 == 0) ? amp : -amp;
    return v;
}

rekall::asr::config::OpenAiEngineConfig make_oai_cfg() {
    rekall::asr::config::OpenAiEngineConfig c;
    c.api_key                 = "test-key";
    c.model                   = "whisper-1";
    c.response_format         = "verbose_json";
    c.request_timeout_seconds = 5;
    c.max_segment_seconds     = 5;
    c.min_segment_seconds     = 1;
    c.retries                 = 0;     // tests drive retry behaviour explicitly
    c.retry_backoff_ms        = 1;
    return c;
}

}  // namespace

TEST(OpenAiEngineTest, HappyPathEmitsFinalAfterFlush) {
    auto sess = make_session();
    auto fake = std::make_unique<rekall::asr::tests::FakeOpenAiClient>();
    OpenAiTranscript t; t.text = "hello world";
    fake->enqueue(rekall::asr::engine::openai::TranscribeResult::success(t));
    auto* fake_ptr = fake.get();

    rekall::asr::engine::openai::OpenAiEngine engine(
        sess, std::move(fake), {}, make_oai_cfg(), nullptr);

    sess->inbound.try_push(rekall::asr::session::InboundFrame::audio(tone(2.0)));
    sess->inbound.try_push(rekall::asr::session::InboundFrame::flush_sentinel());

    std::stop_source ss;
    std::thread th([&] { engine.run(ss.get_token()); });

    // Drain inbound; engine should pop the flush sentinel, fire client once,
    // emit a final, and the test then stops it.
    auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(2);
    while (std::chrono::steady_clock::now() < deadline) {
        if (sess->stats.final_count.load() >= 1) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
    }
    ss.request_stop();
    sess->inbound.try_push(rekall::asr::session::InboundFrame::flush_sentinel());
    th.join();

    EXPECT_EQ(fake_ptr->call_count(), 1);
    EXPECT_EQ(sess->stats.final_count.load(), 1u);
}

TEST(OpenAiEngineTest, SubMinClipIsDiscardedWithoutCallingClient) {
    auto sess = make_session();
    auto fake = std::make_unique<rekall::asr::tests::FakeOpenAiClient>();
    auto* fake_ptr = fake.get();

    rekall::asr::engine::openai::OpenAiEngine engine(
        sess, std::move(fake), {}, make_oai_cfg(), nullptr);

    // 0.3 s — below min_segment_seconds=1; flushing should drop it silently.
    sess->inbound.try_push(rekall::asr::session::InboundFrame::audio(tone(0.3)));
    sess->inbound.try_push(rekall::asr::session::InboundFrame::flush_sentinel());

    std::stop_source ss;
    std::thread th([&] { engine.run(ss.get_token()); });
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    ss.request_stop();
    sess->inbound.try_push(rekall::asr::session::InboundFrame::flush_sentinel());
    th.join();

    EXPECT_EQ(fake_ptr->call_count(), 0);
    EXPECT_EQ(sess->stats.final_count.load(), 0u);
}

TEST(OpenAiEngineTest, ThreeConsecutiveFailuresStopSession) {
    auto sess = make_session();
    auto fake = std::make_unique<rekall::asr::tests::FakeOpenAiClient>();
    fake->enqueue(rekall::asr::engine::openai::TranscribeResult::failure(OpenAiError::Network));
    fake->enqueue(rekall::asr::engine::openai::TranscribeResult::failure(OpenAiError::Network));
    fake->enqueue(rekall::asr::engine::openai::TranscribeResult::failure(OpenAiError::Network));

    rekall::asr::engine::openai::OpenAiEngine engine(
        sess, std::move(fake), {}, make_oai_cfg(), nullptr);

    // Three full segments worth of audio.
    for (int i = 0; i < 3; ++i) {
        sess->inbound.try_push(rekall::asr::session::InboundFrame::audio(tone(2.0)));
        sess->inbound.try_push(rekall::asr::session::InboundFrame::flush_sentinel());
    }

    std::stop_source ss;
    std::thread th([&] { engine.run(ss.get_token()); });

    auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(3);
    while (std::chrono::steady_clock::now() < deadline) {
        if (sess->stop_source.stop_requested()) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
    }
    ss.request_stop();
    sess->inbound.try_push(rekall::asr::session::InboundFrame::flush_sentinel());
    th.join();

    EXPECT_TRUE(sess->stop_source.stop_requested());
}
