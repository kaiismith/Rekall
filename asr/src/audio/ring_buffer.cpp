// Header-only template — this TU exists so the build system has a non-empty
// object for the library target and so linkers under MSVC don't complain
// about an empty archive entry.
#include "rekall/asr/audio/ring_buffer.hpp"

namespace rekall::asr::audio {
// Intentionally empty.
}  // namespace rekall::asr::audio
