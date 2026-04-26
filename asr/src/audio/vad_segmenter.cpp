#include "rekall/asr/audio/vad_segmenter.hpp"

#include <cmath>
#include <cstdint>

namespace rekall::asr::audio {

namespace {

constexpr std::size_t kFrameSize = 320;  // 20 ms @ 16 kHz

std::uint32_t rms_int16(const std::int16_t* data, std::size_t n) {
    if (n == 0) return 0;
    long double acc = 0;
    for (std::size_t i = 0; i < n; ++i) {
        long double s = data[i];
        acc += s * s;
    }
    return static_cast<std::uint32_t>(std::sqrt(acc / static_cast<long double>(n)));
}

}  // namespace

VadSegmenter::VadSegmenter(std::uint32_t sample_rate_hz,
                           std::uint32_t silence_threshold_rms,
                           std::uint32_t silence_debounce_ms)
    : sample_rate_hz_(sample_rate_hz),
      silence_threshold_rms_(silence_threshold_rms),
      silence_debounce_samples_(
          static_cast<std::uint64_t>(sample_rate_hz_) *
          static_cast<std::uint64_t>(silence_debounce_ms) / 1000ULL) {}

std::optional<SegmentEnd> VadSegmenter::push(std::span<const std::int16_t> samples) {
    std::optional<SegmentEnd> result;

    // Walk the input in 20 ms frames so the RMS measurement is local in time.
    std::size_t i = 0;
    while (i < samples.size()) {
        std::size_t take = std::min<std::size_t>(kFrameSize, samples.size() - i);
        const auto* frame = samples.data() + i;
        std::uint32_t r = rms_int16(frame, take);
        bool frame_silent = (r < silence_threshold_rms_);

        if (frame_silent) {
            if (!in_silence_) {
                in_silence_           = true;
                silence_run_samples_  = take;
                silence_run_started_  = total_samples_ + i;
                emitted_for_silence_  = false;
            } else {
                silence_run_samples_ += take;
            }
            if (!emitted_for_silence_ &&
                silence_run_samples_ >= silence_debounce_samples_) {
                result.emplace(SegmentEnd{
                    .silence_start_sample     = silence_run_started_,
                    .silence_duration_samples = silence_run_samples_,
                });
                emitted_for_silence_ = true;
            }
        } else {
            in_silence_           = false;
            silence_run_samples_  = 0;
            emitted_for_silence_  = false;
        }
        i += take;
    }

    total_samples_ += samples.size();
    return result;
}

void VadSegmenter::reset() {
    in_silence_          = true;
    emitted_for_silence_ = true;
    silence_run_samples_ = 0;
    silence_run_started_ = 0;
}

}  // namespace rekall::asr::audio
