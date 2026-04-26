// In-memory per-session state shared between the WS coroutine pair and the
// worker thread. Mutation discipline:
//   - inbound:   produced by WS read coroutine, consumed by Transcriber
//   - outbound:  produced by Transcriber, consumed by WS write coroutine
//   - stats/state: written by Transcriber, read by SessionManager + gRPC
//   - stop_source: any party may request_stop; observed by Transcriber and
//     the WS coroutines via stop_token

#pragma once

#include <atomic>
#include <chrono>
#include <cstdint>
#include <memory>
#include <mutex>
#include <stop_token>
#include <string>
#include <string_view>
#include <vector>

#include "rekall/asr/audio/ring_buffer.hpp"
#include "rekall/asr/session/transcript_event.hpp"

namespace rekall::asr::engine { class LoadedModel; }

namespace rekall::asr::session {

enum class State : std::uint8_t {
    Pending,   // created via gRPC StartSession; awaiting WS upgrade
    Active,    // WS bound, audio flowing
    Closing,   // graceful tear-down in progress
    Closed,    // terminal
};

const char* state_name(State s) noexcept;

// Each binary frame copied off the wire becomes one InboundFrame in the
// inbound RingBuffer. We carry int16 PCM and a flag denoting whether the
// frame is the synthetic "flush" sentinel.
struct InboundFrame {
    std::vector<std::int16_t> samples;
    bool flush = false;

    static InboundFrame audio(std::vector<std::int16_t> s) {
        InboundFrame f;
        f.samples = std::move(s);
        return f;
    }
    static InboundFrame flush_sentinel() {
        InboundFrame f;
        f.flush = true;
        return f;
    }
};

struct Stats {
    std::atomic<std::uint64_t> audio_samples_processed{0};
    std::atomic<std::uint32_t> partial_count{0};
    std::atomic<std::uint32_t> final_count{0};
    std::atomic<std::uint32_t> dropped_partials{0};
};

// Aggregated final transcript text — appended to as each final lands so that
// gRPC EndSession can return the full transcript without requiring callers to
// stitch events themselves.
class FinalAccumulator {
   public:
    void append(std::string_view text);
    std::string snapshot() const;

   private:
    mutable std::mutex mu_;
    std::string buf_;
};

struct Session {
    std::string sid;
    std::string cid;
    std::string uid;
    std::string model_id;
    std::string correlation_id;

    std::shared_ptr<rekall::asr::engine::LoadedModel> model;

    rekall::asr::audio::RingBuffer<InboundFrame>    inbound;
    rekall::asr::audio::RingBuffer<TranscriptEvent> outbound;

    std::atomic<State> state{State::Pending};
    std::chrono::system_clock::time_point started_at      = std::chrono::system_clock::now();
    std::atomic<std::int64_t>             last_activity_at_ms{0};
    std::chrono::system_clock::time_point expires_at;

    Stats             stats;
    FinalAccumulator  final_text;

    std::stop_source  stop_source;

    Session(std::size_t inbound_capacity, std::size_t outbound_capacity)
        : inbound(inbound_capacity), outbound(outbound_capacity) {}

    void touch();
    bool is_terminal() const noexcept {
        State s = state.load(std::memory_order_acquire);
        return s == State::Closing || s == State::Closed;
    }
};

}  // namespace rekall::asr::session
