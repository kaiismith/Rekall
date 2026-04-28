// Header-only test double for OpenAiClient.
//
// Tests enqueue() canned responses (Ok or any OpenAiError); transcribe() pops
// them in order. call_count() exposes how many transcribe() calls fired.

#pragma once

#include "rekall/asr/engine/openai/openai_client.hpp"

#include <cstddef>
#include <span>
#include <stop_token>
#include <utility>
#include <vector>

namespace rekall::asr::tests {

class FakeOpenAiClient final : public rekall::asr::engine::openai::OpenAiClient {
   public:
    using Result = rekall::asr::engine::openai::TranscribeResult;

    void enqueue(Result r) { responses_.push_back(std::move(r)); }
    int  call_count() const noexcept { return calls_; }

    Result transcribe(std::span<const std::byte>,
                      const rekall::asr::engine::openai::OpenAiParams&,
                      std::stop_token) override {
        ++calls_;
        if (responses_.empty()) {
            return Result::failure(rekall::asr::engine::openai::OpenAiError::Network);
        }
        auto r = std::move(responses_.front());
        responses_.erase(responses_.begin());
        return r;
    }

   private:
    std::vector<Result> responses_;
    int                 calls_ = 0;
};

}  // namespace rekall::asr::tests
