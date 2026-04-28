#include "rekall/asr/engine/worker_pool.hpp"

#include <mutex>
#include <utility>

#include "rekall/asr/observ/metrics.hpp"

namespace rekall::asr::engine {

WorkerHandle::WorkerHandle(Worker* w, WorkerPool* pool) : w_(w), pool_(pool) {}

WorkerHandle::WorkerHandle(WorkerHandle&& other) noexcept
    : w_(other.w_), pool_(other.pool_) {
    other.w_    = nullptr;
    other.pool_ = nullptr;
}

WorkerHandle& WorkerHandle::operator=(WorkerHandle&& other) noexcept {
    if (this != &other) {
        release();
        w_          = other.w_;
        pool_       = other.pool_;
        other.w_    = nullptr;
        other.pool_ = nullptr;
    }
    return *this;
}

WorkerHandle::~WorkerHandle() { release(); }

void WorkerHandle::release() noexcept {
    if (w_ != nullptr && pool_ != nullptr) {
        try {
            std::lock_guard g(pool_->mu_);
            pool_->release_unsafe(w_);
        } catch (...) {
            // Destructor must not throw; releasing under contention is a
            // straight push_back so a memory exception is the only realistic
            // failure mode and we'd rather leak the slot than crash.
        }
    }
    w_    = nullptr;
    pool_ = nullptr;
}

WorkerPool::WorkerPool(std::size_t size, rekall::asr::observ::Metrics* metrics)
    : metrics_(metrics) {
    workers_.reserve(size);
    free_.reserve(size);
    for (std::size_t i = 0; i < size; ++i) {
        workers_.emplace_back(std::make_unique<Worker>(i, this));
        free_.push_back(workers_.back().get());
    }
    if (metrics_ != nullptr) {
        metrics_->worker_pool_size().Set(static_cast<double>(size));
        metrics_->worker_pool_in_use().Set(0.0);
    }
}

WorkerPool::~WorkerPool() = default;

std::optional<WorkerHandle> WorkerPool::try_acquire() {
    std::lock_guard g(mu_);
    if (free_.empty()) return std::nullopt;
    Worker* w = free_.back();
    free_.pop_back();
    in_use_.fetch_add(1, std::memory_order_relaxed);
    if (metrics_ != nullptr) {
        metrics_->worker_pool_in_use().Set(static_cast<double>(in_use_.load()));
    }
    return WorkerHandle(w, this);
}

void WorkerPool::release_unsafe(Worker* w) {
    free_.push_back(w);
    in_use_.fetch_sub(1, std::memory_order_relaxed);
    if (metrics_ != nullptr) {
        metrics_->worker_pool_in_use().Set(static_cast<double>(in_use_.load()));
    }
}

}  // namespace rekall::asr::engine
