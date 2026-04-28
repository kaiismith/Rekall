#include "rekall/asr/engine/model_registry.hpp"

#include <chrono>
#include <filesystem>
#include <mutex>
#include <shared_mutex>
#include <utility>

#include "rekall/asr/observ/log_catalog.hpp"
#include "rekall/asr/observ/metrics.hpp"

#if __has_include(<whisper.h>)
#include <whisper.h>
#define REKALL_HAVE_WHISPER 1
#else
// Allow the registry to compile in environments where the whisper.cpp
// submodule has not yet been initialised (e.g. early CI bootstrap).
struct whisper_context;
extern "C" {
inline whisper_context* whisper_init_from_file(const char*) { return nullptr; }
inline void whisper_free(whisper_context*) {}
}
#define REKALL_HAVE_WHISPER 0
#endif

namespace rekall::asr::engine {

LoadedModel::LoadedModel(std::string id, std::string path, whisper_context* ctx,
                         rekall::asr::config::ModelEntry entry)
    : id_(std::move(id)), path_(std::move(path)), ctx_(ctx), entry_(std::move(entry)) {}

LoadedModel::~LoadedModel() {
    if (ctx_ != nullptr) {
        whisper_free(ctx_);
        ctx_ = nullptr;
    }
}

ModelRegistry::ModelRegistry(rekall::asr::observ::Metrics* metrics) : metrics_(metrics) {}

std::shared_ptr<LoadedModel> ModelRegistry::load_one(
    const rekall::asr::config::ModelEntry& entry) {

    if (!std::filesystem::exists(entry.path)) {
        rekall::asr::observ::error(rekall::asr::observ::MODEL_LOAD_FAILED, {
            {"model_id", entry.id},
            {"path",     entry.path},
            {"reason",   "file not found"},
        });
        return nullptr;
    }

    auto t0 = std::chrono::steady_clock::now();
#if REKALL_HAVE_WHISPER
    whisper_context_params cparams = whisper_context_default_params();
    whisper_context* ctx = whisper_init_from_file_with_params(entry.path.c_str(), cparams);
#else
    whisper_context* ctx = whisper_init_from_file(entry.path.c_str());
#endif
    auto t1 = std::chrono::steady_clock::now();

    if (ctx == nullptr) {
        rekall::asr::observ::error(rekall::asr::observ::MODEL_LOAD_FAILED, {
            {"model_id", entry.id},
            {"path",     entry.path},
            {"reason",   "whisper_init_from_file returned null"},
        });
        return nullptr;
    }

    auto secs = std::chrono::duration<double>(t1 - t0).count();
    if (metrics_ != nullptr) {
        metrics_->model_load_duration_seconds(entry.id).Observe(secs);
    }
    rekall::asr::observ::info(rekall::asr::observ::MODEL_LOADED, {
        {"model_id",     entry.id},
        {"path",         entry.path},
        {"load_seconds", secs},
    });
    return std::make_shared<LoadedModel>(entry.id, entry.path, ctx, entry);
}

bool ModelRegistry::load(const rekall::asr::config::ModelEntry& entry) {
    auto m = load_one(entry);
    if (!m) return false;
    std::unique_lock g(mu_);
    by_id_[entry.id] = std::move(m);
    if (default_id_.empty()) default_id_ = entry.id;
    return true;
}

ReloadResult ModelRegistry::reload(
    const std::vector<rekall::asr::config::ModelEntry>& entries) {

    ReloadResult result;
    for (const auto& e : entries) {
        auto m = load_one(e);
        if (!m) {
            result.failed.push_back(e.id);
            continue;
        }
        {
            std::unique_lock g(mu_);
            by_id_[e.id] = std::move(m);
        }
        result.loaded.push_back(e.id);
    }
    return result;
}

std::shared_ptr<LoadedModel> ModelRegistry::get(std::string_view id) const {
    std::shared_lock g(mu_);
    auto it = by_id_.find(std::string(id));
    if (it == by_id_.end()) return nullptr;
    return it->second;
}

std::shared_ptr<LoadedModel> ModelRegistry::get_or_default(std::string_view id) const {
    std::shared_lock g(mu_);
    if (!id.empty()) {
        if (auto it = by_id_.find(std::string(id)); it != by_id_.end()) return it->second;
    }
    if (auto it = by_id_.find(default_id_); it != by_id_.end()) return it->second;
    // No default registered yet — return any model we have, or nullptr.
    if (!by_id_.empty()) return by_id_.begin()->second;
    return nullptr;
}

void ModelRegistry::set_default(std::string_view id) {
    std::unique_lock g(mu_);
    if (by_id_.find(std::string(id)) == by_id_.end()) return;
    default_id_ = std::string(id);
}

std::string ModelRegistry::default_id() const {
    std::shared_lock g(mu_);
    return default_id_;
}

std::vector<std::string> ModelRegistry::loaded_ids() const {
    std::shared_lock g(mu_);
    std::vector<std::string> out;
    out.reserve(by_id_.size());
    for (const auto& [k, _] : by_id_) out.push_back(k);
    return out;
}

bool ModelRegistry::empty() const {
    std::shared_lock g(mu_);
    return by_id_.empty();
}

}  // namespace rekall::asr::engine
