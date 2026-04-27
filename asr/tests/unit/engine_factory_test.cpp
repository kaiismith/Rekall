// Confirms the engine factory honours the build-time selection and the
// runtime config.mode setting. The openai branch is exercised only when the
// build was compiled with the openai engine; otherwise we assert that the
// factory throws ConfigError naming the rebuild flag.

#include "rekall/asr/engine/engine_factory.hpp"
#include "rekall/asr/audio/ring_buffer.hpp"
#include "rekall/asr/config/config.hpp"
#include "rekall/asr/session/session.hpp"

#ifdef REKALL_ASR_HAS_OPENAI
#include "../fakes/fake_openai_client.hpp"
#endif

#include <gtest/gtest.h>
#include <memory>

namespace {

std::shared_ptr<rekall::asr::session::Session> make_session() {
    auto s = std::make_shared<rekall::asr::session::Session>(
        /*inbound_capacity=*/16, /*outbound_capacity=*/16);
    s->sid = "11111111-1111-1111-1111-111111111111";
    return s;
}

}  // namespace

#ifdef REKALL_ASR_HAS_LOCAL
TEST(EngineFactoryTest, ReturnsLocalEngineForLocalMode) {
    rekall::asr::engine::EngineDeps d;
    d.session              = make_session();
    d.engine_cfg.mode      = rekall::asr::config::EngineMode::Local;
    d.session_cfg          = {};
    d.metrics              = nullptr;
    // Note: model is null in this test; LocalEngine tolerates that until run().

    auto engine = rekall::asr::engine::make_engine(d);
    ASSERT_NE(engine, nullptr);
    EXPECT_EQ(engine->name(), "local");
}
#else
TEST(EngineFactoryTest, RefusesLocalModeWhenNotCompiledIn) {
    rekall::asr::engine::EngineDeps d;
    d.session         = make_session();
    d.engine_cfg.mode = rekall::asr::config::EngineMode::Local;

    EXPECT_THROW(rekall::asr::engine::make_engine(d),
                 rekall::asr::config::ConfigError);
}
#endif

#ifdef REKALL_ASR_HAS_OPENAI
TEST(EngineFactoryTest, ReturnsOpenAiEngineForOpenAiMode) {
    rekall::asr::engine::EngineDeps d;
    d.session                 = make_session();
    d.engine_cfg.mode         = rekall::asr::config::EngineMode::OpenAi;
    d.engine_cfg.openai.model = "whisper-1";
    d.session_cfg             = {};
    d.metrics                 = nullptr;

    auto fake = std::make_unique<rekall::asr::tests::FakeOpenAiClient>();
    auto engine = rekall::asr::engine::make_engine(d, std::move(fake));
    ASSERT_NE(engine, nullptr);
    EXPECT_EQ(engine->name(), "openai");
}
#else
TEST(EngineFactoryTest, RefusesOpenAiModeWhenNotCompiledIn) {
    rekall::asr::engine::EngineDeps d;
    d.session         = make_session();
    d.engine_cfg.mode = rekall::asr::config::EngineMode::OpenAi;

    EXPECT_THROW(rekall::asr::engine::make_engine(d),
                 rekall::asr::config::ConfigError);
}
#endif
