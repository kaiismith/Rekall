#include "rekall/asr/auth/jwt_validator.hpp"

#include <chrono>
#include <mutex>
#include <shared_mutex>
#include <thread>

#include <absl/container/flat_hash_map.h>

namespace rekall::asr::auth {

JtiCache::JtiCache(std::chrono::seconds ttl) : ttl_(ttl) {
    sweeper_ = std::jthread([this](std::stop_token st) { sweeper_loop(st); });
}

JtiCache::~JtiCache() {
    // jthread destructor signals stop and joins; explicit for clarity.
    sweeper_.request_stop();
}

bool JtiCache::try_consume(std::string_view jti, std::chrono::system_clock::time_point exp) {
    // Pin lifetime of the absolute expiry; we extend by ttl_ so legitimate
    // clock skew between issuer and verifier doesn't free the slot too early.
    auto until = exp + ttl_;
    std::unique_lock g(mu_);
    auto [it, inserted] = seen_.try_emplace(std::string(jti), until);
    return inserted;
}

void JtiCache::sweep_expired() {
    auto now = std::chrono::system_clock::now();
    std::unique_lock g(mu_);
    // absl::flat_hash_map::erase(it) returns void (unlike std::unordered_map's
    // iterator return). Post-increment captures the current iterator before
    // erase invalidates it, then advances to the next bucket.
    for (auto it = seen_.begin(); it != seen_.end();) {
        if (it->second <= now) {
            seen_.erase(it++);
        } else {
            ++it;
        }
    }
}

std::size_t JtiCache::size() const noexcept {
    std::shared_lock g(mu_);
    return seen_.size();
}

void JtiCache::sweeper_loop(std::stop_token st) {
    using namespace std::chrono_literals;
    while (!st.stop_requested()) {
        // Wait in 1 s slices so stop_requested is observed promptly.
        for (int i = 0; i < 30 && !st.stop_requested(); ++i) {
            std::this_thread::sleep_for(1s);
        }
        if (st.stop_requested()) break;
        sweep_expired();
    }
}

}  // namespace rekall::asr::auth
