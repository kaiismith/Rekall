// Lightweight RMS-based voice activity detector.
//
// Pushed audio is 16 kHz int16 mono. The segmenter tracks rolling RMS; when
// the signal drops below `silence_threshold_rms` for at least
// `silence_debounce_ms`, it reports a SegmentEnd with the position (in samples
// since session start) where the silence began.
//
// Whisper.cpp ships a more sophisticated VAD on master; this is the simple
// energy detector used to bound segment lengths in the streaming loop. It is
// intentionally kept separate so it can be replaced by webrtc-vad or
// whisper.cpp's own VAD without touching the Transcriber wiring.

#pragma once

#include <cstddef>
#include <cstdint>
#include <optional>
#include <span>

namespace rekall::asr::audio {

struct SegmentEnd {
    std::uint64_t silence_start_sample;
    std::uint64_t silence_duration_samples;
};

class VadSegmenter {
   public:
    // sample_rate_hz: typically 16000.
    // silence_threshold_rms: int16 RMS amplitude below which we treat the
    //     audio as silence. 200 is a reasonable starting point (~ -45 dBFS).
    // silence_debounce_ms: how long the silence must persist before we report
    //     a segment-end (default 400 ms).
    VadSegmenter(std::uint32_t sample_rate_hz,
                 std::uint32_t silence_threshold_rms = 200,
                 std::uint32_t silence_debounce_ms   = 400);

    // Push a chunk of int16 PCM. Returns SegmentEnd if a silent stretch of
    // sufficient duration just terminated; std::nullopt otherwise.
    // Multiple calls may yield a single SegmentEnd (when silence first crosses
    // the debounce threshold); repeated silence beyond that does not re-emit
    // until a new speech-then-silence transition.
    std::optional<SegmentEnd> push(std::span<const std::int16_t> samples);

    // Resets internal counters; called when a segment is consumed by the
    // Transcriber so the next segment starts clean.
    void reset();

    std::uint64_t total_samples() const noexcept { return total_samples_; }

   private:
    std::uint32_t sample_rate_hz_;
    std::uint32_t silence_threshold_rms_;
    std::uint64_t silence_debounce_samples_;

    bool          in_silence_           = true;
    bool          emitted_for_silence_  = true;   // true while we haven't seen speech yet
    std::uint64_t silence_run_samples_  = 0;
    std::uint64_t silence_run_started_  = 0;
    std::uint64_t total_samples_        = 0;
};

}  // namespace rekall::asr::audio
