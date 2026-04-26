#include "rekall/asr/config/config.hpp"

#include <gtest/gtest.h>

#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <string>

using namespace rekall::asr::config;

namespace {

std::filesystem::path write_yaml(const std::string& body) {
    auto p = std::filesystem::temp_directory_path() /
             ("asr_cfg_" + std::to_string(std::rand()) + ".yaml");
    std::ofstream f(p);
    f << body;
    return p;
}

void set_env(const char* k, const char* v) {
#ifdef _WIN32
    _putenv_s(k, v == nullptr ? "" : v);
#else
    if (v == nullptr) ::unsetenv(k);
    else              ::setenv(k, v, 1);
#endif
}

const char* kSecret = "0123456789abcdef0123456789abcdef0123456789ab";

const char* kMinimalYaml = R"(
models:
  default: tiny.en
  entries:
    - id: tiny.en
      path: /tmp/ggml-tiny.en.bin
)";

}  // namespace

TEST(ConfigLoaderTest, EnvVarNameDerivation) {
    EXPECT_EQ(Config::env_var_for("server.ws_listen"),    "ASR_SERVER_WS_LISTEN");
    EXPECT_EQ(Config::env_var_for("worker_pool.size"),    "ASR_WORKER_POOL_SIZE");
    EXPECT_EQ(Config::env_var_for("models.default"),      "ASR_MODELS_DEFAULT");
}

TEST(ConfigLoaderTest, SecretEnvRequired) {
    set_env("ASR_TOKEN_SECRET", nullptr);
    auto p = write_yaml(kMinimalYaml);
    EXPECT_THROW({ (void)Config::load(p); }, ConfigError);
}

TEST(ConfigLoaderTest, MinimalYamlWithSecretValidates) {
    set_env("ASR_TOKEN_SECRET", kSecret);
    auto p = write_yaml(kMinimalYaml);
    Config c;
    ASSERT_NO_THROW(c = Config::load(p));
    EXPECT_EQ(c.models.default_id, "tiny.en");
    EXPECT_GE(c.worker_pool.size, 1U);
}

TEST(ConfigLoaderTest, RejectsDefaultWithNoMatchingEntry) {
    set_env("ASR_TOKEN_SECRET", kSecret);
    auto p = write_yaml(R"(
models:
  default: not-loaded
  entries:
    - id: tiny.en
      path: /tmp/x.bin
)");
    EXPECT_THROW({ (void)Config::load(p); }, ConfigError);
}

TEST(ConfigLoaderTest, RejectsTtlOver300) {
    set_env("ASR_TOKEN_SECRET", kSecret);
    auto p = write_yaml(R"(
auth:
  token_max_ttl_seconds: 600
models:
  default: tiny.en
  entries:
    - id: tiny.en
      path: /tmp/x.bin
)");
    EXPECT_THROW({ (void)Config::load(p); }, ConfigError);
}

TEST(ConfigLoaderTest, EnvOverridesYaml) {
    set_env("ASR_TOKEN_SECRET", kSecret);
    set_env("ASR_WORKER_POOL_SIZE", "16");
    auto p = write_yaml(R"(
worker_pool:
  size: 4
models:
  default: tiny.en
  entries:
    - id: tiny.en
      path: /tmp/x.bin
)");
    auto c = Config::load(p);
    EXPECT_EQ(c.worker_pool.size, 16U);
    set_env("ASR_WORKER_POOL_SIZE", nullptr);
}
