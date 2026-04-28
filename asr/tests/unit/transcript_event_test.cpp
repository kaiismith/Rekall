#include "rekall/asr/session/transcript_event.hpp"

#include <gtest/gtest.h>
#include <nlohmann/json.hpp>

using namespace rekall::asr::session;

TEST(TranscriptEventTest, PartialSerialisesCoreFields) {
    TranscriptEvent e;
    e.type       = EventType::Partial;
    e.segment_id = 3;
    e.text       = "hello world";
    e.start_ms   = 1000;
    e.end_ms     = 1500;
    e.confidence = 0.42F;
    auto j = nlohmann::json::parse(serialise(e));
    EXPECT_EQ(j["type"],       "partial");
    EXPECT_EQ(j["segment_id"], 3);
    EXPECT_EQ(j["text"],       "hello world");
    EXPECT_EQ(j["start_ms"],   1000);
    EXPECT_EQ(j["end_ms"],     1500);
    EXPECT_NEAR(j["confidence"].get<float>(), 0.42F, 1e-5);
}

TEST(TranscriptEventTest, FinalCarriesWordTimings) {
    TranscriptEvent e;
    e.type       = EventType::Final;
    e.segment_id = 1;
    e.text       = "hi";
    e.language   = "en";
    e.start_ms   = 0;
    e.end_ms     = 200;
    e.words      = {{"hi", 0, 200, 0.9F}};
    auto j = nlohmann::json::parse(serialise(e));
    EXPECT_EQ(j["type"],     "final");
    EXPECT_EQ(j["language"], "en");
    ASSERT_TRUE(j["words"].is_array());
    ASSERT_EQ(j["words"].size(), 1U);
    EXPECT_EQ(j["words"][0]["w"], "hi");
}

TEST(TranscriptEventTest, ErrorIncludesCodeAndMessage) {
    TranscriptEvent e;
    e.type    = EventType::Error;
    e.code    = "ASR_INFERENCE_FAILED";
    e.message = "out of memory";
    auto j = nlohmann::json::parse(serialise(e));
    EXPECT_EQ(j["type"],    "error");
    EXPECT_EQ(j["code"],    "ASR_INFERENCE_FAILED");
    EXPECT_EQ(j["message"], "out of memory");
}

TEST(TranscriptEventTest, PongCarriesTimestamp) {
    TranscriptEvent e;
    e.type       = EventType::Pong;
    e.ts_unix_ms = 1735689600123ULL;
    auto j = nlohmann::json::parse(serialise(e));
    EXPECT_EQ(j["type"], "pong");
    EXPECT_EQ(j["ts"],   1735689600123ULL);
}
