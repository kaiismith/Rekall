// In-memory representation of a transcript event before JSON serialisation.
// Lives between the Transcriber (producer, on the worker thread) and the
// WebSocket writer (consumer, on the network strand) via the session's
// outbound RingBuffer.

#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace rekall::asr::session {

enum class EventType {
    Ready,
    Partial,
    Final,
    Info,
    Error,
    Pong,
};

struct WordTiming {
    std::string w;
    std::uint32_t start_ms = 0;
    std::uint32_t end_ms   = 0;
    float p                = 0.0F;
};

struct TranscriptEvent {
    EventType type;
    std::int32_t segment_id = 0;
    std::string text;
    std::uint32_t start_ms  = 0;
    std::uint32_t end_ms    = 0;
    float confidence        = 0.0F;
    std::vector<WordTiming> words;
    std::string language;
    // For Info/Error events.
    std::string code;
    std::string message;
    // For Pong.
    std::uint64_t ts_unix_ms = 0;
};

// Serialises a TranscriptEvent to a JSON string ready for WebSocket text frame.
// Defined in transcript_event.cpp.
std::string serialise(const TranscriptEvent& e);

}  // namespace rekall::asr::session
