// Bounded worker pool. Each Worker owns nothing persistent across sessions —
// the per-decode whisper_state is created/destroyed inside Transcriber, which
// the WS handler instantiates after acquiring a worker handle.
//
// The pool itself is purely a counting semaphore expressed as a free-list of
// Worker objects, so admission control is `try_acquire().has_value()`.

#pragma once

#include <atomic>
#include <cstddef>
#include <memory>
#include <mutex>
#include <optional>
#include <stop_token>
#include <vector>

namespace rekall::asr::observ { class Metrics; }

namespace rekall::asr::engine {

class WorkerPool;

class Worker {
   public:
    Worker(std::size_t id, WorkerPool* pool) : id_(id), pool_(pool) {}
    std::size_t id() const noexcept { return id_; }

   private:
    std::size_t id_;
    WorkerPool* pool_;
};

// RAII handle returned by try_acquire. Releases the worker back to the pool
// on destruction. Move-only.
class WorkerHandle {
   public:
    WorkerHandle() = default;
    WorkerHandle(Worker* w, WorkerPool* pool);
    ~WorkerHandle();

    WorkerHandle(const WorkerHandle&)            = delete;
    WorkerHandle& operator=(const WorkerHandle&) = delete;
    WorkerHandle(WorkerHandle&& other) noexcept;
    WorkerHandle& operator=(WorkerHandle&& other) noexcept;

    Worker* operator->() const noexcept { return w_; }
    Worker& operator*()  const noexcept { return *w_; }
    explicit operator bool() const noexcept { return w_ != nullptr; }

    void release() noexcept;

   private:
    Worker* w_              = nullptr;
    WorkerPool* pool_       = nullptr;
};

class WorkerPool {
   public:
    explicit WorkerPool(std::size_t size, rekall::asr::observ::Metrics* metrics = nullptr);
    ~WorkerPool();

    WorkerPool(const WorkerPool&)            = delete;
    WorkerPool& operator=(const WorkerPool&) = delete;

    // Returns std::nullopt if all workers are in use.
    std::optional<WorkerHandle> try_acquire();

    std::size_t size()    const noexcept { return workers_.size(); }
    std::size_t in_use()  const noexcept { return in_use_.load(std::memory_order_relaxed); }
    std::size_t free()    const noexcept { return size() - in_use(); }

   private:
    friend class WorkerHandle;
    void release_unsafe(Worker* w);

    std::mutex mu_;
    std::vector<std::unique_ptr<Worker>> workers_;
    std::vector<Worker*> free_;
    std::atomic<std::size_t> in_use_{0};
    rekall::asr::observ::Metrics* metrics_;
};

}  // namespace rekall::asr::engine
