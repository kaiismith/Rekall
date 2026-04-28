// HTTPS implementation of OpenAiClient that delegates to the vendored
// `olrea/openai-cpp` header-only library.
//
// Only one of these is created per process; the openai-cpp singleton is
// initialised on construction.

#pragma once

#include "rekall/asr/engine/openai/openai_client.hpp"

#include <chrono>
#include <string>

namespace rekall::asr::engine::openai {

class OpenAiHttpClient final : public OpenAiClient {
   public:
    struct Config {
        std::string  api_key;
        std::string  organization;
        std::string  base_url;          // "" → openai-cpp default
        std::string  user_agent;        // "rekall-asr/<ver> openai-cpp"
    };

    explicit OpenAiHttpClient(Config cfg);
    ~OpenAiHttpClient() override = default;

    OpenAiHttpClient(const OpenAiHttpClient&)            = delete;
    OpenAiHttpClient& operator=(const OpenAiHttpClient&) = delete;

    TranscribeResult transcribe(
        std::span<const std::byte> wav_bytes,
        const OpenAiParams&        params,
        std::stop_token            st) override;

   private:
    Config cfg_;
};

}  // namespace rekall::asr::engine::openai
