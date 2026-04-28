#include "rekall/asr/session/session.hpp"

#include <chrono>
#include <mutex>

#include "rekall/asr/session/transcript_event.hpp"

namespace rekall::asr::session {

const char* state_name(State s) noexcept {
    switch (s) {
        case State::Pending: return "pending";
        case State::Active:  return "active";
        case State::Closing: return "closing";
        case State::Closed:  return "closed";
    }
    return "unknown";
}

void Session::touch() {
    using namespace std::chrono;
    auto ms = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
    last_activity_at_ms.store(static_cast<std::int64_t>(ms), std::memory_order_relaxed);
}

void FinalAccumulator::append(std::string_view text) {
    std::lock_guard g(mu_);
    if (!buf_.empty() && !text.empty() && buf_.back() != ' ') buf_ += ' ';
    buf_.append(text.data(), text.size());
}

std::string FinalAccumulator::snapshot() const {
    std::lock_guard g(mu_);
    return buf_;
}

}  // namespace rekall::asr::session
