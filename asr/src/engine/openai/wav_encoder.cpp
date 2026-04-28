#include "rekall/asr/engine/openai/wav_encoder.hpp"

#include <cstring>

namespace rekall::asr::engine::openai {

namespace {

template <class T>
void put_le(std::byte*& p, T v) {
    for (std::size_t i = 0; i < sizeof(T); ++i) {
        *p++ = static_cast<std::byte>((v >> (8 * i)) & 0xFFu);
    }
}

void put_fourcc(std::byte*& p, const char (&tag)[5]) {
    std::memcpy(p, tag, 4);
    p += 4;
}

}  // namespace

std::vector<std::byte> encode_wav_pcm16(std::span<const std::int16_t> samples,
                                        std::uint32_t                 sample_rate) {
    if (samples.empty()) return {};

    constexpr std::uint16_t channels        = 1;
    constexpr std::uint16_t bits_per_sample = 16;
    const std::uint32_t     byte_rate      = sample_rate * channels * (bits_per_sample / 8);
    const std::uint16_t     block_align    = channels * (bits_per_sample / 8);
    const std::uint32_t     data_size      =
        static_cast<std::uint32_t>(samples.size()) * sizeof(std::int16_t);
    const std::uint32_t     riff_size      = 36u + data_size;

    std::vector<std::byte> buf(44u + data_size);
    auto* p = buf.data();

    put_fourcc(p, "RIFF");
    put_le(p, riff_size);
    put_fourcc(p, "WAVE");

    put_fourcc(p, "fmt ");
    put_le(p, std::uint32_t{16});       // PCM fmt chunk size
    put_le(p, std::uint16_t{1});        // audio format = PCM
    put_le(p, channels);
    put_le(p, sample_rate);
    put_le(p, byte_rate);
    put_le(p, block_align);
    put_le(p, bits_per_sample);

    put_fourcc(p, "data");
    put_le(p, data_size);

    // PCM samples — host int16 is LE on every platform we ship to.
    std::memcpy(p, samples.data(), data_size);

    return buf;
}

}  // namespace rekall::asr::engine::openai
