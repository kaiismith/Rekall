#include "rekall/asr/util/correlation_id.hpp"

#include <gtest/gtest.h>

#include <unordered_set>

using namespace rekall::asr::util;

TEST(CorrelationIdTest, GeneratesUuidV4Shape) {
    for (int i = 0; i < 32; ++i) {
        auto id = new_uuid_v4();
        EXPECT_TRUE(looks_like_uuid_v4(id)) << id;
    }
}

TEST(CorrelationIdTest, GeneratesDistinctValues) {
    std::unordered_set<std::string> seen;
    for (int i = 0; i < 256; ++i) {
        seen.insert(new_uuid_v4());
    }
    EXPECT_EQ(seen.size(), 256U);
}

TEST(CorrelationIdTest, EnsurePassesThroughNonEmpty) {
    EXPECT_EQ(ensure("incoming-id"), "incoming-id");
}

TEST(CorrelationIdTest, EnsureGeneratesWhenEmpty) {
    auto id = ensure("");
    EXPECT_TRUE(looks_like_uuid_v4(id));
}

TEST(CorrelationIdTest, RejectsObviousNonUuid) {
    EXPECT_FALSE(looks_like_uuid_v4(""));
    EXPECT_FALSE(looks_like_uuid_v4("not-a-uuid"));
    EXPECT_FALSE(looks_like_uuid_v4("00000000-0000-0000-0000-000000000000"));
}
