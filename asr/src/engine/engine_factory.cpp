#include "rekall/asr/engine/engine_factory.hpp"

#include <stdexcept>
#include <utility>

#ifdef REKALL_ASR_HAS_LOCAL
#include "rekall/asr/engine/local_engine.hpp"
#endif

#ifdef REKALL_ASR_HAS_OPENAI
#include "rekall/asr/engine/openai/openai_engine.hpp"
#include "rekall/asr/engine/openai/openai_http_client.hpp"
#endif

namespace rekall::asr::engine {

std::unique_ptr<TranscriptionEngine> make_engine(const EngineDeps& d) {
    switch (d.engine_cfg.mode) {
        case rekall::asr::config::EngineMode::Local:
#ifdef REKALL_ASR_HAS_LOCAL
            return std::make_unique<LocalEngine>(
                d.session, d.model, d.session_cfg, d.metrics);
#else
            throw rekall::asr::config::ConfigError{
                "engine.mode=local but this build was compiled without the "
                "local engine — rebuild with -DREKALL_ASR_ENGINE=local or =both"};
#endif

        case rekall::asr::config::EngineMode::OpenAi: {
#ifdef REKALL_ASR_HAS_OPENAI
            rekall::asr::engine::openai::OpenAiHttpClient::Config cc{
                .api_key      = d.engine_cfg.openai.api_key,
                .organization = d.engine_cfg.openai.organization,
                .base_url     = d.engine_cfg.openai.base_url,
                .user_agent   = "rekall-asr/0.1.0 openai-cpp",
            };
            auto client = std::make_unique<rekall::asr::engine::openai::OpenAiHttpClient>(std::move(cc));
            return std::make_unique<rekall::asr::engine::openai::OpenAiEngine>(
                d.session, std::move(client), d.session_cfg,
                d.engine_cfg.openai, d.metrics);
#else
            throw rekall::asr::config::ConfigError{
                "engine.mode=openai but this build was compiled without the "
                "openai engine — rebuild with -DREKALL_ASR_ENGINE=openai or =both"};
#endif
        }
    }
    throw std::logic_error("unknown engine mode");
}

std::unique_ptr<TranscriptionEngine> make_engine(
    const EngineDeps&                                  d,
    std::unique_ptr<rekall::asr::engine::openai::OpenAiClient> client) {
    if (d.engine_cfg.mode == rekall::asr::config::EngineMode::Local) {
        // The injected client is irrelevant for local; defer to the
        // production overload.
        return make_engine(d);
    }

#ifdef REKALL_ASR_HAS_OPENAI
    if (!client) {
        throw std::logic_error("make_engine(deps, client): client must be non-null for openai mode");
    }
    return std::make_unique<rekall::asr::engine::openai::OpenAiEngine>(
        d.session, std::move(client), d.session_cfg,
        d.engine_cfg.openai, d.metrics);
#else
    (void)client;
    throw rekall::asr::config::ConfigError{
        "engine.mode=openai but this build was compiled without the openai engine"};
#endif
}

}  // namespace rekall::asr::engine
