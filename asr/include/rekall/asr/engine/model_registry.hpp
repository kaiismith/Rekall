// In-memory registry of loaded whisper.cpp models.
//
// Models are pointer-stable: hot-reload mutates the map under a unique_lock,
// but existing sessions hold a shared_ptr<LoadedModel> so a swap is safe and
// the old context is destroyed only after the last session releases it.

#pragma once

#include <chrono>
#include <memory>
#include <mutex>
#include <optional>
#include <shared_mutex>
#include <string>
#include <string_view>
#include <unordered_map>
#include <vector>

#include "rekall/asr/config/config.hpp"

struct whisper_context;

namespace rekall::asr::observ { class Metrics; }

namespace rekall::asr::engine {

// LoadedModel owns the heap-allocated whisper_context. Destruction releases it.
class LoadedModel {
   public:
    LoadedModel(std::string id, std::string path, whisper_context* ctx,
                rekall::asr::config::ModelEntry entry);
    ~LoadedModel();

    LoadedModel(const LoadedModel&)            = delete;
    LoadedModel& operator=(const LoadedModel&) = delete;

    const std::string& id()   const noexcept { return id_; }
    const std::string& path() const noexcept { return path_; }
    whisper_context*   ctx()  const noexcept { return ctx_; }
    const rekall::asr::config::ModelEntry& entry() const noexcept { return entry_; }

   private:
    std::string id_;
    std::string path_;
    whisper_context* ctx_;
    rekall::asr::config::ModelEntry entry_;
};

struct ReloadResult {
    std::vector<std::string> loaded;
    std::vector<std::string> failed;
};

class ModelRegistry {
   public:
    explicit ModelRegistry(rekall::asr::observ::Metrics* metrics = nullptr);

    // Loads each entry. Existing models with the same id are replaced; sessions
    // already holding the old shared_ptr keep using it until they release.
    // Returns the loaded/failed ids for ReloadModelsResponse plumbing.
    ReloadResult reload(const std::vector<rekall::asr::config::ModelEntry>& entries);

    // Convenience for first-boot registration.
    bool load(const rekall::asr::config::ModelEntry& entry);

    // Returns the model for `id`, or the default if `id` is empty / unknown.
    // The caller may inspect the returned shared_ptr's ->id() to discover the
    // canonical model that was actually selected (after fallback).
    std::shared_ptr<LoadedModel> get_or_default(std::string_view id) const;

    // Returns nullptr if the id is unknown.
    std::shared_ptr<LoadedModel> get(std::string_view id) const;

    // Sets the default model id. The registry retains the previous default if
    // `id` is unknown.
    void set_default(std::string_view id);

    std::string default_id() const;

    std::vector<std::string> loaded_ids() const;

    bool empty() const;

   private:
    std::shared_ptr<LoadedModel> load_one(const rekall::asr::config::ModelEntry& entry);

    mutable std::shared_mutex mu_;
    std::unordered_map<std::string, std::shared_ptr<LoadedModel>> by_id_;
    std::string default_id_;
    rekall::asr::observ::Metrics* metrics_;
};

}  // namespace rekall::asr::engine
