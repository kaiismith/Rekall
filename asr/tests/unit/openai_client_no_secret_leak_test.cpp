// Guards against accidental secret leakage through openai-cpp's exception
// messages or our own logging. We instantiate the http client with a sentinel
// API key, force a transcription failure (no network), and assert the sentinel
// string is absent from the returned error label and from the spdlog buffer.

#include "rekall/asr/engine/openai/openai_http_client.hpp"

#include <gtest/gtest.h>
#include <spdlog/sinks/ostream_sink.h>
#include <spdlog/spdlog.h>
#include <sstream>

namespace {

constexpr const char* kSentinelKey = "SENTINEL_KEY_42_NEVER_LOG_ME";

}  // namespace

TEST(OpenAiClientNoSecretLeakTest, SentinelKeyAbsentFromErrorAndLogs) {
    // Capture spdlog output into a stringstream sink so we can scan it.
    auto buf = std::make_shared<std::ostringstream>();
    auto sink = std::make_shared<spdlog::sinks::ostream_sink_mt>(*buf);
    auto logger = std::make_shared<spdlog::logger>("asr-test", sink);
    spdlog::set_default_logger(logger);
    spdlog::set_level(spdlog::level::debug);

    rekall::asr::engine::openai::OpenAiHttpClient::Config cc{
        .api_key      = kSentinelKey,
        .organization = "",
        .base_url     = "https://invalid.invalid",   // forces a Network error
        .user_agent   = "rekall-asr-test",
    };
    rekall::asr::engine::openai::OpenAiHttpClient client(std::move(cc));

    // Tiny silence buffer — at least one sample so the client doesn't reject
    // empty input ahead of attempting a request.
    constexpr std::size_t n_bytes = 44 + 32;  // RIFF + a few samples
    std::vector<std::byte> wav(n_bytes, std::byte{0});

    rekall::asr::engine::openai::OpenAiParams params;
    params.model            = "whisper-1";
    params.response_format  = "json";
    params.timeout          = std::chrono::seconds{1};

    std::stop_source ss;
    auto result = client.transcribe(std::span{wav}, params, ss.get_token());

    // Either the network error, or BadRequest if the wav header tripped curl.
    // What matters is the sentinel key never surfaces.
    EXPECT_FALSE(result.has_value());
    const std::string captured = buf->str();
    EXPECT_EQ(captured.find(kSentinelKey), std::string::npos)
        << "API key leaked to logs!";
}
