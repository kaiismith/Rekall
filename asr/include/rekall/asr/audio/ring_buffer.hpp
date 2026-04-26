// Bounded ring buffer used as the inbound (PCM frames) and outbound
// (transcript events) queue between WebSocket coroutines and the worker
// thread. Single-producer-single-consumer in normal use; the mutex makes
// SPMC and MPSC safe under TSan as well.

#pragma once

#include <condition_variable>
#include <cstddef>
#include <mutex>
#include <optional>
#include <stop_token>
#include <vector>

namespace rekall::asr::audio {

template <class T>
class RingBuffer {
   public:
    explicit RingBuffer(std::size_t capacity)
        : buf_(capacity), capacity_(capacity), head_(0), tail_(0), size_(0) {}

    RingBuffer(const RingBuffer&)            = delete;
    RingBuffer& operator=(const RingBuffer&) = delete;

    // Returns false if full.
    bool try_push(T item) {
        std::lock_guard g(mu_);
        if (size_ == capacity_) return false;
        buf_[tail_] = std::move(item);
        tail_       = (tail_ + 1) % capacity_;
        ++size_;
        not_empty_.notify_one();
        return true;
    }

    // Drops the oldest entry if the buffer is full and inserts the new one.
    // Returns true if a slot was overwritten (caller may want to track this).
    bool push_or_drop_oldest(T item) {
        std::lock_guard g(mu_);
        bool dropped = false;
        if (size_ == capacity_) {
            head_   = (head_ + 1) % capacity_;
            --size_;
            dropped = true;
        }
        buf_[tail_] = std::move(item);
        tail_       = (tail_ + 1) % capacity_;
        ++size_;
        not_empty_.notify_one();
        return dropped;
    }

    std::optional<T> try_pop() {
        std::lock_guard g(mu_);
        if (size_ == 0) return std::nullopt;
        T out      = std::move(buf_[head_]);
        head_      = (head_ + 1) % capacity_;
        --size_;
        return out;
    }

    // Blocks until an item is available or the stop token is requested.
    std::optional<T> pop_blocking(std::stop_token st) {
        std::unique_lock g(mu_);
        std::stop_callback cb(st, [this]() {
            std::lock_guard inner(mu_);
            not_empty_.notify_all();
        });
        not_empty_.wait(g, [&] { return size_ > 0 || st.stop_requested(); });
        if (size_ == 0) return std::nullopt;
        T out      = std::move(buf_[head_]);
        head_      = (head_ + 1) % capacity_;
        --size_;
        return out;
    }

    std::size_t size() const {
        std::lock_guard g(mu_);
        return size_;
    }

    std::size_t capacity() const noexcept { return capacity_; }

    bool full() const {
        std::lock_guard g(mu_);
        return size_ == capacity_;
    }

   private:
    mutable std::mutex mu_;
    std::condition_variable not_empty_;
    std::vector<T> buf_;
    std::size_t capacity_;
    std::size_t head_;
    std::size_t tail_;
    std::size_t size_;
};

}  // namespace rekall::asr::audio
