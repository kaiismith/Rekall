#include "rekall/asr/config/config.hpp"

#include <yaml-cpp/yaml.h>

#include <algorithm>
#include <cctype>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <optional>
#include <sstream>
#include <string>
#include <thread>
#include <vector>

namespace rekall::asr::config {

namespace {

// Splits a comma-separated string, trimming each element.
std::vector<std::string> split_csv(std::string_view s) {
    std::vector<std::string> out;
    std::string cur;
    for (char c : s) {
        if (c == ',') {
            // Trim and emit.
            auto begin = cur.find_first_not_of(" \t");
            auto end   = cur.find_last_not_of(" \t");
            if (begin != std::string::npos) out.emplace_back(cur.substr(begin, end - begin + 1));
            cur.clear();
        } else {
            cur += c;
        }
    }
    auto begin = cur.find_first_not_of(" \t");
    auto end   = cur.find_last_not_of(" \t");
    if (begin != std::string::npos) out.emplace_back(cur.substr(begin, end - begin + 1));
    return out;
}

bool parse_bool(std::string_view s) {
    std::string lc(s);
    std::transform(lc.begin(), lc.end(), lc.begin(), [](unsigned char ch) { return std::tolower(ch); });
    return lc == "1" || lc == "true" || lc == "yes" || lc == "on";
}

std::optional<std::string> getenv_opt(const std::string& key) {
    const char* v = std::getenv(key.c_str());
    if (v == nullptr) return std::nullopt;
    if (*v == '\0')   return std::nullopt;
    return std::string(v);
}

YAML::Node walk(const YAML::Node& root, std::string_view path) {
    YAML::Node cur = root;
    std::size_t i = 0;
    while (i <= path.size()) {
        std::size_t j = path.find('.', i);
        if (j == std::string_view::npos) j = path.size();
        std::string key(path.substr(i, j - i));
        if (!cur.IsMap() || !cur[key]) return {};
        cur = cur[key];
        if (j == path.size()) break;
        i = j + 1;
    }
    return cur;
}

template <typename T>
T pick(const YAML::Node& root, std::string_view yaml_path, std::string_view env_key, T fallback) {
    if (auto e = getenv_opt(std::string(env_key))) {
        try {
            if constexpr (std::is_same_v<T, bool>) {
                return parse_bool(*e);
            } else if constexpr (std::is_same_v<T, std::string>) {
                return *e;
            } else {
                std::stringstream ss(*e);
                T v{};
                ss >> v;
                if (ss.fail()) {
                    throw ConfigError("invalid value for " + std::string(env_key) + ": " + *e);
                }
                return v;
            }
        } catch (const ConfigError&) {
            throw;
        } catch (const std::exception&) {
            throw ConfigError("invalid value for " + std::string(env_key));
        }
    }
    auto node = walk(root, yaml_path);
    if (node && !node.IsNull()) {
        try {
            return node.as<T>();
        } catch (const std::exception& e) {
            throw ConfigError("yaml key " + std::string(yaml_path) + " has wrong type: " + e.what());
        }
    }
    return fallback;
}

std::vector<std::string> pick_csv(const YAML::Node& root, std::string_view yaml_path,
                                  std::string_view env_key,
                                  std::vector<std::string> fallback) {
    if (auto e = getenv_opt(std::string(env_key))) {
        return split_csv(*e);
    }
    auto node = walk(root, yaml_path);
    if (node && node.IsSequence()) {
        std::vector<std::string> out;
        out.reserve(node.size());
        for (const auto& n : node) out.emplace_back(n.as<std::string>());
        return out;
    }
    if (node && node.IsScalar()) {
        return split_csv(node.as<std::string>());
    }
    return fallback;
}

}  // namespace

std::string Config::env_var_for(std::string_view yaml_path) {
    std::string out = "ASR_";
    out.reserve(out.size() + yaml_path.size());
    for (char c : yaml_path) {
        if (c == '.') {
            out += '_';
        } else {
            out += static_cast<char>(std::toupper(static_cast<unsigned char>(c)));
        }
    }
    return out;
}

Config Config::load(const std::filesystem::path& path) {
    YAML::Node root;
    if (!path.empty() && std::filesystem::exists(path)) {
        try {
            root = YAML::LoadFile(path.string());
        } catch (const std::exception& e) {
            throw ConfigError("failed to parse YAML at " + path.string() + ": " + e.what());
        }
    }
    // If the file is missing, every key falls back to env var or default.
    // This is intentional for container deployments that pass everything via env.

    Config c;

    // ── server ───────────────────────────────────────────────────────────────
    c.server.ws_listen          = pick<std::string>(root, "server.ws_listen",
                                                    env_var_for("server.ws_listen"),
                                                    c.server.ws_listen);
    c.server.grpc_listen        = pick<std::string>(root, "server.grpc_listen",
                                                    env_var_for("server.grpc_listen"),
                                                    c.server.grpc_listen);
    c.server.metrics_listen     = pick<std::string>(root, "server.metrics_listen",
                                                    env_var_for("server.metrics_listen"),
                                                    c.server.metrics_listen);
    c.server.allow_insecure_ws  = pick<bool>(root, "server.allow_insecure_ws",
                                             "ASR_ALLOW_INSECURE_WS",
                                             c.server.allow_insecure_ws);
    c.server.grpc_bind_all      = pick<bool>(root, "server.grpc_bind_all",
                                             "ASR_GRPC_BIND_ALL",
                                             c.server.grpc_bind_all);
    c.server.ws_allowed_origins = pick_csv(root, "server.ws_allowed_origins",
                                           "ASR_WS_ALLOWED_ORIGINS", {});

    // ── auth ─────────────────────────────────────────────────────────────────
    c.auth.token_secret_env     = pick<std::string>(root, "auth.token_secret_env",
                                                    "ASR_TOKEN_SECRET_ENV", c.auth.token_secret_env);
    c.auth.token_audience       = pick<std::string>(root, "auth.token_audience",
                                                    "ASR_TOKEN_AUDIENCE", c.auth.token_audience);
    c.auth.token_issuer         = pick<std::string>(root, "auth.token_issuer",
                                                    "ASR_TOKEN_ISSUER", c.auth.token_issuer);
    c.auth.token_default_ttl_seconds = pick<std::uint32_t>(root, "auth.token_default_ttl_seconds",
        "ASR_AUTH_TOKEN_DEFAULT_TTL_SECONDS", c.auth.token_default_ttl_seconds);
    c.auth.token_min_ttl_seconds     = pick<std::uint32_t>(root, "auth.token_min_ttl_seconds",
        "ASR_AUTH_TOKEN_MIN_TTL_SECONDS", c.auth.token_min_ttl_seconds);
    c.auth.token_max_ttl_seconds     = pick<std::uint32_t>(root, "auth.token_max_ttl_seconds",
        "ASR_AUTH_TOKEN_MAX_TTL_SECONDS", c.auth.token_max_ttl_seconds);
    c.auth.jti_cache_ttl_seconds     = pick<std::uint32_t>(root, "auth.jti_cache_ttl_seconds",
        "ASR_AUTH_JTI_CACHE_TTL_SECONDS", c.auth.jti_cache_ttl_seconds);

    // The token secret itself comes from the env var named in `token_secret_env`.
    if (auto v = getenv_opt(c.auth.token_secret_env)) {
        c.auth.token_secret = *v;
    }

    // ── tls ──────────────────────────────────────────────────────────────────
    c.tls.ws_cert        = pick<std::string>(root, "tls.ws_cert",        "ASR_TLS_WS_CERT",        "");
    c.tls.ws_key         = pick<std::string>(root, "tls.ws_key",         "ASR_TLS_WS_KEY",         "");
    c.tls.grpc_cert      = pick<std::string>(root, "tls.grpc_cert",      "ASR_GRPC_SERVER_CERT",   "");
    c.tls.grpc_key       = pick<std::string>(root, "tls.grpc_key",       "ASR_GRPC_SERVER_KEY",    "");
    c.tls.grpc_client_ca = pick<std::string>(root, "tls.grpc_client_ca", "ASR_GRPC_CLIENT_CA",     "");

    // ── worker_pool ──────────────────────────────────────────────────────────
    c.worker_pool.size   = pick<std::uint32_t>(root, "worker_pool.size",
                                               "ASR_WORKER_POOL_SIZE", c.worker_pool.size);
    if (c.worker_pool.size == 0) {
        unsigned hw = std::thread::hardware_concurrency();
        c.worker_pool.size = std::min<std::uint32_t>(8U, hw == 0 ? 4U : hw);
    }

    // ── admission ────────────────────────────────────────────────────────────
    c.admission.inbound_audio_buffer_frames = pick<std::uint32_t>(root,
        "admission.inbound_audio_buffer_frames",
        "ASR_ADMISSION_INBOUND_AUDIO_BUFFER_FRAMES",
        c.admission.inbound_audio_buffer_frames);
    c.admission.outbound_event_buffer = pick<std::uint32_t>(root,
        "admission.outbound_event_buffer",
        "ASR_ADMISSION_OUTBOUND_EVENT_BUFFER",
        c.admission.outbound_event_buffer);
    c.admission.backpressure_kill_ms = pick<std::uint32_t>(root,
        "admission.backpressure_kill_ms",
        "ASR_ADMISSION_BACKPRESSURE_KILL_MS",
        c.admission.backpressure_kill_ms);
    c.admission.outbound_drop_ms = pick<std::uint32_t>(root,
        "admission.outbound_drop_ms",
        "ASR_ADMISSION_OUTBOUND_DROP_MS",
        c.admission.outbound_drop_ms);

    // ── session ──────────────────────────────────────────────────────────────
    c.session.idle_timeout_seconds = pick<std::uint32_t>(root,
        "session.idle_timeout_seconds", "ASR_SESSION_IDLE_TIMEOUT_SECONDS",
        c.session.idle_timeout_seconds);
    c.session.hard_timeout_seconds = pick<std::uint32_t>(root,
        "session.hard_timeout_seconds", "ASR_SESSION_HARD_TIMEOUT_SECONDS",
        c.session.hard_timeout_seconds);
    c.session.graceful_drain_seconds = pick<std::uint32_t>(root,
        "session.graceful_drain_seconds", "ASR_SESSION_GRACEFUL_DRAIN_SECONDS",
        c.session.graceful_drain_seconds);
    c.session.partial_emit_interval_ms = pick<std::uint32_t>(root,
        "session.partial_emit_interval_ms", "ASR_SESSION_PARTIAL_EMIT_INTERVAL_MS",
        c.session.partial_emit_interval_ms);
    c.session.final_emit_max_latency_ms = pick<std::uint32_t>(root,
        "session.final_emit_max_latency_ms", "ASR_SESSION_FINAL_EMIT_MAX_LATENCY_MS",
        c.session.final_emit_max_latency_ms);
    c.session.audio_window_seconds = pick<std::uint32_t>(root,
        "session.audio_window_seconds", "ASR_SESSION_AUDIO_WINDOW_SECONDS",
        c.session.audio_window_seconds);

    // ── models ───────────────────────────────────────────────────────────────
    c.models.default_id = pick<std::string>(root, "models.default", "ASR_MODELS_DEFAULT",
                                            c.models.default_id);

    // First, dump the keys yaml-cpp actually parsed at the root so we can see
    // what's there if the resolution below misses.
    const bool root_is_map = root && root.IsMap();
    if (root_is_map) {
        std::fprintf(stderr, "config root has %zu keys:", root.size());
        for (auto kv : root) {
            try {
                auto k = kv.first.as<std::string>();
                std::fprintf(stderr, " '%s'", k.c_str());
            } catch (...) {
                std::fprintf(stderr, " <non-string>");
            }
        }
        std::fprintf(stderr, "\n");
    }

    // Resolve the models block by iterating root, NOT via operator[]. yaml-cpp
    // 0.8's non-const operator[] auto-vivifies, which can mask the real key
    // and return an undefined Node even when the key parsed correctly.
    YAML::Node models_node;
    if (root_is_map) {
        for (auto kv : root) {
            try {
                if (kv.first.as<std::string>() == "models") {
                    models_node = kv.second;
                    break;
                }
            } catch (...) { /* skip non-string keys */ }
        }
    }

    YAML::Node entries_node;
    if (models_node && models_node.IsMap()) {
        for (auto kv : models_node) {
            try {
                if (kv.first.as<std::string>() == "entries") {
                    entries_node = kv.second;
                    break;
                }
            } catch (...) {}
        }
    }

    const bool has_models     = static_cast<bool>(models_node);
    const bool models_is_map  = has_models && models_node.IsMap();
    const bool has_entries    = models_is_map && static_cast<bool>(entries_node);
    const bool entries_is_seq = has_entries && entries_node.IsSequence();
    const std::size_t yaml_entries_size = entries_is_seq ? entries_node.size() : 0;

    if (entries_is_seq) {
        for (const auto& e : entries_node) {
            ModelEntry m;
            m.id          = e["id"]          ? e["id"].as<std::string>()  : "";
            m.path        = e["path"]        ? e["path"].as<std::string>(): "";
            m.language    = e["language"]    ? e["language"].as<std::string>() : m.language;
            m.n_threads   = e["n_threads"]   ? e["n_threads"].as<std::int32_t>() : m.n_threads;
            m.beam_size   = e["beam_size"]   ? e["beam_size"].as<std::int32_t>() : m.beam_size;
            m.best_of     = e["best_of"]     ? e["best_of"].as<std::int32_t>()   : m.best_of;
            m.temperature = e["temperature"] ? e["temperature"].as<float>()      : m.temperature;
            m.translate   = e["translate"]   ? e["translate"].as<bool>()         : m.translate;
            m.suppress_blank = e["suppress_blank"] ? e["suppress_blank"].as<bool>()
                                                  : m.suppress_blank;
            m.suppress_non_speech_tokens = e["suppress_non_speech_tokens"]
                ? e["suppress_non_speech_tokens"].as<bool>() : m.suppress_non_speech_tokens;
            c.models.entries.push_back(std::move(m));
        }
    }
    std::fprintf(stderr,
        "config: file=%s exists=%d size=%lld root_is_map=%d has_models=%d "
        "models_is_map=%d has_entries=%d entries_is_seq=%d yaml_entries=%zu "
        "loaded_entries=%zu\n",
        path.string().c_str(),
        std::filesystem::exists(path) ? 1 : 0,
        std::filesystem::exists(path)
            ? static_cast<long long>(std::filesystem::file_size(path)) : -1LL,
        root_is_map, has_models, models_is_map, has_entries, entries_is_seq,
        yaml_entries_size, c.models.entries.size());

    // ── logging ──────────────────────────────────────────────────────────────
    c.logging.level  = pick<std::string>(root, "logging.level",  "ASR_LOG_LEVEL",  c.logging.level);
    c.logging.format = pick<std::string>(root, "logging.format", "ASR_LOG_FORMAT", c.logging.format);

    // ── telemetry ────────────────────────────────────────────────────────────
    c.telemetry.otel_endpoint = pick<std::string>(root, "telemetry.otel_endpoint",
                                                  "OTEL_EXPORTER_OTLP_ENDPOINT",
                                                  c.telemetry.otel_endpoint);

    // ── drop_privs (Linux only) ──────────────────────────────────────────────
    if (auto v = getenv_opt("ASR_DROP_PRIVS_TO")) {
        auto colon = v->find(':');
        if (colon == std::string::npos) {
            throw ConfigError("ASR_DROP_PRIVS_TO must be in 'uid:gid' form");
        }
        try {
            c.drop_privs.uid     = static_cast<std::uint32_t>(std::stoul(v->substr(0, colon)));
            c.drop_privs.gid     = static_cast<std::uint32_t>(std::stoul(v->substr(colon + 1)));
            c.drop_privs.enabled = true;
        } catch (const std::exception&) {
            throw ConfigError("ASR_DROP_PRIVS_TO must be 'uid:gid' with integer values");
        }
    }

    c.validate();
    return c;
}

namespace {
void require_readable(const std::string& path, std::string_view what) {
    if (path.empty()) return;
    std::ifstream f(path);
    if (!f.good()) throw ConfigError(std::string(what) + " not readable: " + path);
}
}  // namespace

void Config::validate() const {
    if (worker_pool.size < 1 || worker_pool.size > 64) {
        throw ConfigError("worker_pool.size must be in [1, 64]");
    }
    if (models.entries.empty()) {
        throw ConfigError("models.entries must not be empty");
    }
    bool default_found = false;
    for (const auto& m : models.entries) {
        if (m.id.empty())   throw ConfigError("models.entries[].id must not be empty");
        if (m.path.empty()) throw ConfigError("models.entries[" + m.id + "].path must not be empty");
        if (m.id == models.default_id) default_found = true;
    }
    if (!default_found) {
        throw ConfigError("models.default '" + models.default_id +
                          "' does not match any models.entries[].id");
    }
    if (auth.token_max_ttl_seconds > 300) {
        throw ConfigError("auth.token_max_ttl_seconds must be <= 300");
    }
    if (auth.token_max_ttl_seconds < auth.token_min_ttl_seconds) {
        throw ConfigError("auth.token_max_ttl_seconds must be >= token_min_ttl_seconds");
    }
    if (auth.token_default_ttl_seconds < auth.token_min_ttl_seconds ||
        auth.token_default_ttl_seconds > auth.token_max_ttl_seconds) {
        throw ConfigError("auth.token_default_ttl_seconds must lie in [min, max]");
    }
    if (auth.token_secret.size() < 32) {
        throw ConfigError("env var '" + auth.token_secret_env +
                          "' must hold a secret of at least 32 bytes");
    }
    require_readable(tls.ws_cert,        "tls.ws_cert");
    require_readable(tls.ws_key,         "tls.ws_key");
    require_readable(tls.grpc_cert,      "tls.grpc_cert");
    require_readable(tls.grpc_key,       "tls.grpc_key");
    require_readable(tls.grpc_client_ca, "tls.grpc_client_ca");
}

}  // namespace rekall::asr::config
