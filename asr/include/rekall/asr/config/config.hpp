// Configuration loader for the ASR service.
//
// Source-of-truth precedence: env var > YAML file > built-in default.
// Env var keys are derived from YAML paths: lowercased path with '.' → '_',
// prefixed `ASR_`. e.g. worker_pool.size ⇄ ASR_WORKER_POOL_SIZE.
//
// `Config::load(path)` throws ConfigError on any validation failure or on a
// missing required env var. Callers in main() catch and exit(1).

#pragma once

#include <cstdint>
#include <filesystem>
#include <stdexcept>
#include <string>
#include <vector>

namespace rekall::asr::config {

class ConfigError : public std::runtime_error {
   public:
    using std::runtime_error::runtime_error;
};

struct ServerConfig {
    std::string ws_listen      = "0.0.0.0:8081";
    std::string grpc_listen    = "127.0.0.1:9090";
    std::string metrics_listen = "0.0.0.0:9091";
    bool allow_insecure_ws     = false;
    bool allow_insecure_grpc   = false;  // dev only: gRPC off loopback w/o mTLS
    bool grpc_bind_all         = false;
    std::vector<std::string> ws_allowed_origins;
};

struct AuthConfig {
    std::string token_secret;             // resolved from env var named below
    std::string token_secret_env = "ASR_TOKEN_SECRET";
    std::string token_audience   = "rekall-asr";
    std::string token_issuer     = "rekall-backend";
    std::uint32_t token_default_ttl_seconds = 180;
    std::uint32_t token_min_ttl_seconds     = 60;
    std::uint32_t token_max_ttl_seconds     = 300;
    std::uint32_t jti_cache_ttl_seconds     = 600;
};

struct TLSConfig {
    std::string ws_cert;
    std::string ws_key;
    std::string grpc_cert;
    std::string grpc_key;
    std::string grpc_client_ca;
};

struct WorkerPoolConfig {
    std::uint32_t size = 0;               // 0 → derive from hardware_concurrency
};

struct AdmissionConfig {
    std::uint32_t inbound_audio_buffer_frames = 256;
    std::uint32_t outbound_event_buffer       = 64;
    std::uint32_t backpressure_kill_ms        = 5000;
    std::uint32_t outbound_drop_ms            = 200;
};

struct SessionConfig {
    std::uint32_t idle_timeout_seconds        = 30;
    std::uint32_t hard_timeout_seconds        = 1800;
    std::uint32_t graceful_drain_seconds      = 30;
    std::uint32_t partial_emit_interval_ms    = 250;
    std::uint32_t final_emit_max_latency_ms   = 1500;
    std::uint32_t audio_window_seconds        = 8;
};

struct ModelEntry {
    std::string id;
    std::string path;
    std::string language = "en";
    std::int32_t n_threads  = 4;
    std::int32_t beam_size  = 5;
    std::int32_t best_of    = 5;
    float        temperature = 0.0F;
    bool         translate   = false;
    bool         suppress_blank = true;
    bool         suppress_non_speech_tokens = true;
};

struct ModelsConfig {
    std::string default_id = "small.en";
    std::vector<ModelEntry> entries;
};

struct LoggingConfig {
    std::string level  = "info";
    std::string format = "json";
};

struct TelemetryConfig {
    std::string otel_endpoint;             // empty disables exporter
};

struct DropPrivs {
    bool enabled = false;
    std::uint32_t uid = 0;
    std::uint32_t gid = 0;
};

struct Config {
    ServerConfig     server;
    AuthConfig       auth;
    TLSConfig        tls;
    WorkerPoolConfig worker_pool;
    AdmissionConfig  admission;
    SessionConfig    session;
    ModelsConfig     models;
    LoggingConfig    logging;
    TelemetryConfig  telemetry;
    DropPrivs        drop_privs;

    // Loads YAML from `path`, applies env-var overrides, validates, and returns.
    // Throws ConfigError on any failure.
    static Config load(const std::filesystem::path& path);

    // Returns the env-var name corresponding to a YAML dotted path
    // (e.g. "worker_pool.size" → "ASR_WORKER_POOL_SIZE").
    static std::string env_var_for(std::string_view yaml_path);

    // Throws ConfigError on any rule violation. Called by load(); exposed for
    // tests that want to validate a hand-built Config.
    void validate() const;
};

}  // namespace rekall::asr::config
