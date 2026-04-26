#include "rekall/asr/audio/vad_segmenter.hpp"

#include <gtest/gtest.h>

#include <cmath>
#include <cstdint>
#include <vector>

using namespace rekall::asr::audio;

namespace {

std::vector<std::int16_t> tone(std::size_t n_samples, double amp, double freq_hz, double sr_hz) {
    std::vector<std::int16_t> out(n_samples);
    for (std::size_t i = 0; i < n_samples; ++i) {
        double s = amp * std::sin(2.0 * M_PI * freq_hz * i / sr_hz);
        out[i] = static_cast<std::int16_t>(s);
    }
    return out;
}

std::vector<std::int16_t> silence(std::size_t n_samples) {
    return std::vector<std::int16_t>(n_samples, 0);
}

}  // namespace

TEST(VadSegmenterTest, SilenceAfterSpeechFiresSegmentEnd) {
    VadSegmenter vad(16000, /*threshold=*/200, /*debounce_ms=*/200);

    auto speech = tone(/*samples=*/16000, /*amp=*/8000.0, /*freq=*/440.0, 16000.0);
    auto sil    = silence(16000 / 2);  // 500 ms — well over 200 ms debounce

    EXPECT_FALSE(vad.push(speech).has_value());
    auto end = vad.push(sil);
    ASSERT_TRUE(end.has_value());
    EXPECT_GT(end->silence_duration_samples, 0U);
}

TEST(VadSegmenterTest, SustainedSpeechDoesNotFire) {
    VadSegmenter vad(16000, 200, 200);
    for (int i = 0; i < 10; ++i) {
        auto speech = tone(8000, 8000.0, 440.0, 16000.0);
        EXPECT_FALSE(vad.push(speech).has_value());
    }
}

TEST(VadSegmenterTest, ShortSilenceBelowDebounceDoesNotFire) {
    VadSegmenter vad(16000, 200, 400);
    EXPECT_FALSE(vad.push(tone(16000, 8000.0, 440.0, 16000.0)).has_value());
    // 100 ms of silence — under the 400 ms debounce
    EXPECT_FALSE(vad.push(silence(1600)).has_value());
}

TEST(VadSegmenterTest, ResetClearsState) {
    VadSegmenter vad(16000, 200, 100);
    (void)vad.push(tone(8000, 8000.0, 440.0, 16000.0));
    (void)vad.push(silence(8000));
    vad.reset();
    // After reset we should not refire on continued silence.
    EXPECT_FALSE(vad.push(silence(8000)).has_value());
}
