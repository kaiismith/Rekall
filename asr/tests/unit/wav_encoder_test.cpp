// Verifies the in-memory WAV encoder produces a well-formed RIFF/WAVE buffer
// that OpenAI's /v1/audio/transcriptions endpoint will accept.

#include "rekall/asr/engine/openai/wav_encoder.hpp"

#include <cmath>
#include <cstdint>
#include <cstring>
#include <gtest/gtest.h>
#include <numbers>
#include <vector>

namespace {

template <class T>
T read_le(const std::byte* p) {
    T v = 0;
    for (std::size_t i = 0; i < sizeof(T); ++i) {
        v |= static_cast<T>(std::to_integer<std::uint8_t>(p[i])) << (8 * i);
    }
    return v;
}

bool fourcc_eq(const std::byte* p, const char (&tag)[5]) {
    return std::memcmp(p, tag, 4) == 0;
}

std::vector<std::int16_t> make_tone_1khz_1s() {
    constexpr std::uint32_t rate = 16'000;
    std::vector<std::int16_t> out(rate);
    for (std::uint32_t i = 0; i < rate; ++i) {
        const double t = static_cast<double>(i) / rate;
        const double s = std::sin(2.0 * std::numbers::pi * 1000.0 * t);
        out[i]         = static_cast<std::int16_t>(s * 30000.0);
    }
    return out;
}

}  // namespace

TEST(WavEncoderTest, EmptyInputReturnsEmptyVector) {
    auto out = rekall::asr::engine::openai::encode_wav_pcm16({}, 16000);
    EXPECT_TRUE(out.empty());
}

TEST(WavEncoderTest, OneSecondToneByteLengthAndHeader) {
    const auto samples = make_tone_1khz_1s();
    const auto wav     = rekall::asr::engine::openai::encode_wav_pcm16(samples, 16000);

    ASSERT_EQ(wav.size(), 44u + samples.size() * sizeof(std::int16_t));

    // RIFF header
    EXPECT_TRUE(fourcc_eq(wav.data() + 0, "RIFF"));
    EXPECT_EQ(read_le<std::uint32_t>(wav.data() + 4),
              static_cast<std::uint32_t>(36u + samples.size() * sizeof(std::int16_t)));
    EXPECT_TRUE(fourcc_eq(wav.data() + 8, "WAVE"));

    // fmt chunk
    EXPECT_TRUE(fourcc_eq(wav.data() + 12, "fmt "));
    EXPECT_EQ(read_le<std::uint32_t>(wav.data() + 16), 16u);   // chunk size
    EXPECT_EQ(read_le<std::uint16_t>(wav.data() + 20), 1u);    // PCM
    EXPECT_EQ(read_le<std::uint16_t>(wav.data() + 22), 1u);    // mono
    EXPECT_EQ(read_le<std::uint32_t>(wav.data() + 24), 16'000u);
    EXPECT_EQ(read_le<std::uint32_t>(wav.data() + 28), 32'000u);  // byte_rate
    EXPECT_EQ(read_le<std::uint16_t>(wav.data() + 32), 2u);    // block_align
    EXPECT_EQ(read_le<std::uint16_t>(wav.data() + 34), 16u);   // bits_per_sample

    // data chunk
    EXPECT_TRUE(fourcc_eq(wav.data() + 36, "data"));
    EXPECT_EQ(read_le<std::uint32_t>(wav.data() + 40),
              static_cast<std::uint32_t>(samples.size() * sizeof(std::int16_t)));
}

TEST(WavEncoderTest, SamplesRoundTripWithinOneLsb) {
    const auto samples = make_tone_1khz_1s();
    const auto wav     = rekall::asr::engine::openai::encode_wav_pcm16(samples, 16000);

    // Decode the data chunk back to int16.
    const std::byte* p = wav.data() + 44;
    std::vector<std::int16_t> decoded(samples.size());
    std::memcpy(decoded.data(), p, samples.size() * sizeof(std::int16_t));

    ASSERT_EQ(decoded.size(), samples.size());
    EXPECT_EQ(decoded.front(), samples.front());
    EXPECT_EQ(decoded[samples.size() / 2], samples[samples.size() / 2]);
    EXPECT_EQ(decoded.back(), samples.back());
}
