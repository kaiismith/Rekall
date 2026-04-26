#include "rekall/asr/session/transcript_event.hpp"

#include <nlohmann/json.hpp>

namespace rekall::asr::session {

namespace {
const char* type_name(EventType t) {
    switch (t) {
        case EventType::Ready:   return "ready";
        case EventType::Partial: return "partial";
        case EventType::Final:   return "final";
        case EventType::Info:    return "info";
        case EventType::Error:   return "error";
        case EventType::Pong:    return "pong";
    }
    return "unknown";
}
}  // namespace

std::string serialise(const TranscriptEvent& e) {
    nlohmann::json j;
    j["type"] = type_name(e.type);

    switch (e.type) {
        case EventType::Ready:
            // The hook caller is expected to populate sid/model_id/sample_rate
            // via the `code`/`message` field convention; ws_session builds the
            // ready event explicitly, so this branch is rarely hit here.
            if (!e.code.empty())    j["session_id"] = e.code;
            if (!e.message.empty()) j["model_id"]   = e.message;
            j["sample_rate"] = 16000;
            break;
        case EventType::Partial:
            j["segment_id"] = e.segment_id;
            j["text"]       = e.text;
            j["start_ms"]   = e.start_ms;
            j["end_ms"]     = e.end_ms;
            j["confidence"] = e.confidence;
            break;
        case EventType::Final: {
            j["segment_id"] = e.segment_id;
            j["text"]       = e.text;
            j["language"]   = e.language;
            j["start_ms"]   = e.start_ms;
            j["end_ms"]     = e.end_ms;
            nlohmann::json words = nlohmann::json::array();
            for (const auto& w : e.words) {
                words.push_back({
                    {"w",        w.w},
                    {"start_ms", w.start_ms},
                    {"end_ms",   w.end_ms},
                    {"p",        w.p},
                });
            }
            j["words"] = std::move(words);
            break;
        }
        case EventType::Info:
            j["code"] = e.code;
            if (!e.message.empty()) j["message"] = e.message;
            break;
        case EventType::Error:
            j["code"]    = e.code;
            j["message"] = e.message;
            break;
        case EventType::Pong:
            j["ts"] = e.ts_unix_ms;
            break;
    }
    return j.dump();
}

}  // namespace rekall::asr::session
