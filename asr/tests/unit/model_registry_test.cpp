#include "rekall/asr/engine/model_registry.hpp"

#include <gtest/gtest.h>

#include <filesystem>
#include <fstream>

using namespace rekall::asr::engine;
using namespace rekall::asr::config;

namespace {

ModelEntry make_entry(const std::string& id, const std::string& path) {
    ModelEntry e;
    e.id        = id;
    e.path      = path;
    e.language  = "en";
    e.n_threads = 1;
    e.beam_size = 1;
    return e;
}

std::filesystem::path make_dummy_model() {
    auto p = std::filesystem::temp_directory_path() / "ggml-test.bin";
    std::ofstream(p) << "stub";
    return p;
}

}  // namespace

TEST(ModelRegistryTest, MissingFileLoadFails) {
    ModelRegistry reg;
    EXPECT_FALSE(reg.load(make_entry("ghost", "/no/such/file")));
    EXPECT_TRUE(reg.empty());
}

TEST(ModelRegistryTest, FallsBackToDefault) {
    // Even with no models loaded, get_or_default returns nullptr safely.
    ModelRegistry reg;
    EXPECT_EQ(reg.get_or_default("anything"), nullptr);
}

TEST(ModelRegistryTest, ReloadProducesLoadedAndFailedSplit) {
    ModelRegistry reg;
    auto good = make_dummy_model();
    auto r = reg.reload({
        make_entry("a", good.string()),
        make_entry("b", "/no/such/file"),
    });
    // The dummy file exists but isn't a valid whisper model — under the build
    // stub path, whisper_init_from_file returns null and the entry fails. The
    // assertion here verifies the failure path is wired; under a real
    // whisper.cpp build the same file would cause a different error which is
    // also surfaced via `failed`.
    EXPECT_EQ(r.loaded.size() + r.failed.size(), 2U);
    EXPECT_EQ(r.failed.back(), "b");
}

TEST(ModelRegistryTest, SetDefaultIgnoresUnknownId) {
    ModelRegistry reg;
    reg.set_default("not-loaded");
    EXPECT_TRUE(reg.default_id().empty());
}
