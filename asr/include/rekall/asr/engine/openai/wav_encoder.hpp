// In-memory WAV-RIFF encoder for the OpenAI engine.
//
// Produces a standard PCM WAVE buffer (mono, 16-bit signed LE) suitable as
// the `file` field of OpenAI's /v1/audio/transcriptions multipart request.
// 50 lines of code, hand-rolled because we never need any other format and
// pulling in a dependency for one fixed shape isn't worth it.

#pragma once

#include <cstddef>
#include <cstdint>
#include <span>
#include <vector>

namespace rekall::asr::engine::openai {

// Encodes the int16 PCM samples into a complete RIFF WAVE byte buffer.
// Returns an empty vector for empty input. Always little-endian.
std::vector<std::byte> encode_wav_pcm16(std::span<const std::int16_t> samples,
                                        std::uint32_t sample_rate);

}  // namespace rekall::asr::engine::openai
