// Engine factory: constructs the configured TranscriptionEngine for a session.
//
// The single source of truth for engine selection. Worker / WS code only ever
// holds an abstract `unique_ptr<TranscriptionEngine>` — they don't know which
// concrete impl is running. This is also the seam that integration tests use
// to swap in a FakeOpenAiClient.

#pragma once

#include <memory>

#include "rekall/asr/config/config.hpp"
#include "rekall/asr/engine/transcription_engine.hpp"

namespace rekall::asr::engine { class LoadedModel; }
namespace rekall::asr::engine::openai { class OpenAiClient; }
namespace rekall::asr::observ { class Metrics; }
namespace rekall::asr::session { struct Session; }

namespace rekall::asr::engine {

struct EngineDeps {
    std::shared_ptr<rekall::asr::session::Session> session;
    std::shared_ptr<LoadedModel>                   model;          // nullable for openai
    rekall::asr::config::SessionConfig             session_cfg;
    rekall::asr::config::EngineConfig              engine_cfg;
    rekall::asr::observ::Metrics*                  metrics = nullptr;
};

// Production overload — constructs an OpenAiHttpClient internally for
// engine.mode=openai. Throws rekall::asr::config::ConfigError when the
// requested mode is not compiled into this build.
std::unique_ptr<TranscriptionEngine> make_engine(const EngineDeps& deps);

// Test overload — caller injects a pre-built OpenAiClient (typically a fake).
// `client` must be non-null when engine.mode=openai; ignored for local.
std::unique_ptr<TranscriptionEngine> make_engine(
    const EngineDeps&                                  deps,
    std::unique_ptr<rekall::asr::engine::openai::OpenAiClient> client);

}  // namespace rekall::asr::engine
