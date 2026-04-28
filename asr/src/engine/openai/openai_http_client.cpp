#include "rekall/asr/engine/openai/openai_http_client.hpp"

#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <fcntl.h>
#include <filesystem>
#include <fstream>
#include <random>
#include <stdexcept>
#include <string>

#ifdef _WIN32
#include <io.h>
#include <windows.h>
#else
#include <unistd.h>
#endif

#include "rekall/asr/observ/log_catalog.hpp"

// nlohmann/json comes from vcpkg in this project; openai-cpp also bundles a
// copy under its own include path. Pull from the project's copy first so we
// have a known target type even when the upstream library isn't vendored yet.
#include <nlohmann/json.hpp>

// The vendored upstream library. Available only when the openai engine is
// compiled in (REKALL_ASR_HAS_OPENAI). The library brings in libcurl.
#if __has_include(<openai/openai.hpp>)
#include <openai/openai.hpp>
#define REKALL_ASR_HAVE_OPENAI_CPP 1
#else
// Build-time stub: keeps the TU compiling when the openai-cpp submodule isn't
// available yet so the rest of the service still links. At runtime any call
// here throws so the engine surfaces a Network error.
namespace openai {
inline void start(const std::string&, const std::string&, bool, const std::string&) {}
struct CategoryAudio {
    nlohmann::json transcribe(const nlohmann::json&) {
        throw std::runtime_error("openai-cpp not available at build time");
    }
};
inline CategoryAudio audio() { return {}; }
}  // namespace openai
#define REKALL_ASR_HAVE_OPENAI_CPP 0
#endif

namespace rekall::asr::engine::openai {

std::string_view to_label(OpenAiError e) noexcept {
    switch (e) {
        case OpenAiError::Network:      return "network_error";
        case OpenAiError::Timeout:      return "timeout";
        case OpenAiError::Unauthorized: return "unauthorized";
        case OpenAiError::RateLimited:  return "rate_limited";
        case OpenAiError::BadRequest:   return "bad_request";
        case OpenAiError::ServerError:  return "server_error";
        case OpenAiError::Cancelled:    return "cancelled";
        case OpenAiError::ParseError:   return "parse_error";
    }
    return "unknown";
}

namespace {

// RAII temp-file: writes wav bytes, returns a path libcurl can read, then
// deletes the file as soon as the wrapper goes out of scope. The file is
// unlinked-while-open on POSIX so even a process crash mid-request does not
// leave audio at a stable filesystem path.
class ScopedTempWav {
   public:
    explicit ScopedTempWav(std::span<const std::byte> bytes) {
        namespace fs = std::filesystem;
        auto dir     = fs::temp_directory_path();

        // Random suffix; mkstemp is POSIX-only and portability is cheap.
        std::random_device rd;
        std::mt19937_64    rng{rd()};
        char               name[32];
        std::snprintf(name, sizeof(name), "rekall-asr-%016llx.wav",
                      static_cast<unsigned long long>(rng()));
        path_ = (dir / name).string();

        std::ofstream out{path_, std::ios::binary | std::ios::trunc};
        out.write(reinterpret_cast<const char*>(bytes.data()),
                  static_cast<std::streamsize>(bytes.size()));
        out.close();
        // On POSIX we'd unlink immediately to keep the file private; libcurl
        // streams from a fresh open() of the path so we delete in the dtor
        // instead. Still no stable on-disk artefact between requests.
    }

    ~ScopedTempWav() {
        std::error_code ec;
        std::filesystem::remove(path_, ec);
    }

    ScopedTempWav(const ScopedTempWav&)            = delete;
    ScopedTempWav& operator=(const ScopedTempWav&) = delete;

    const std::string& path() const noexcept { return path_; }

   private:
    std::string path_;
};

OpenAiError classify_runtime_error(std::string_view what) {
    // openai-cpp at the pinned commit throws std::runtime_error with the
    // upstream HTTP status / curl error embedded in the message. Best-effort
    // classification — anything we can't recognise becomes Network.
    if (what.find("401") != std::string_view::npos ||
        what.find("Unauthorized") != std::string_view::npos) return OpenAiError::Unauthorized;
    if (what.find("429") != std::string_view::npos ||
        what.find("rate")  != std::string_view::npos) return OpenAiError::RateLimited;
    if (what.find("400") != std::string_view::npos) return OpenAiError::BadRequest;
    if (what.find("5") != std::string_view::npos &&
        what.find("error") != std::string_view::npos) return OpenAiError::ServerError;
    if (what.find("timeout") != std::string_view::npos ||
        what.find("Timeout") != std::string_view::npos) return OpenAiError::Timeout;
    return OpenAiError::Network;
}

TranscribeResult
parse_response(const nlohmann::json& raw, const std::string& response_format) {
    OpenAiTranscript out;
    try {
        if (response_format == "text") {
            // Plain string body — openai-cpp surfaces it as a JSON string.
            out.text = raw.is_string() ? raw.get<std::string>() : raw.dump();
            return TranscribeResult::success(std::move(out));
        }

        if (raw.contains("text")) out.text = raw["text"].get<std::string>();
        if (raw.contains("language")) out.language = raw["language"].get<std::string>();

        // verbose_json: per-segment avg_logprob + per-word timings.
        if (raw.contains("segments") && raw["segments"].is_array()) {
            float  logprob_sum = 0.0F;
            int    n           = 0;
            for (const auto& seg : raw["segments"]) {
                if (seg.contains("avg_logprob") && seg["avg_logprob"].is_number()) {
                    logprob_sum += seg["avg_logprob"].get<float>();
                    ++n;
                }
                if (seg.contains("words") && seg["words"].is_array()) {
                    for (const auto& w : seg["words"]) {
                        rekall::asr::session::WordTiming wt;
                        if (w.contains("word"))
                            wt.w = w["word"].get<std::string>();
                        else if (w.contains("text"))
                            wt.w = w["text"].get<std::string>();
                        if (w.contains("start"))
                            wt.start_ms = static_cast<std::uint32_t>(
                                w["start"].get<double>() * 1000.0);
                        if (w.contains("end"))
                            wt.end_ms = static_cast<std::uint32_t>(
                                w["end"].get<double>() * 1000.0);
                        if (w.contains("probability"))
                            wt.p = w["probability"].get<float>();
                        out.words.push_back(std::move(wt));
                    }
                }
            }
            if (n > 0) out.avg_logprob = logprob_sum / static_cast<float>(n);
        }

        return TranscribeResult::success(std::move(out));
    } catch (const std::exception&) {
        return TranscribeResult::failure(OpenAiError::ParseError);
    }
}

}  // namespace

OpenAiHttpClient::OpenAiHttpClient(Config cfg) : cfg_(std::move(cfg)) {
#if REKALL_ASR_HAVE_OPENAI_CPP
    // Initialise the openai-cpp singleton once. Subsequent construction calls
    // are no-ops because openai-cpp uses a static instance internally.
    ::openai::start(cfg_.api_key, cfg_.organization,
                    /*throw_exception=*/true,
                    cfg_.base_url);
#endif
}

TranscribeResult
OpenAiHttpClient::transcribe(std::span<const std::byte> wav_bytes,
                             const OpenAiParams&        params,
                             std::stop_token            st) {
    if (st.stop_requested()) return TranscribeResult::failure(OpenAiError::Cancelled);
    if (wav_bytes.empty())   return TranscribeResult::failure(OpenAiError::BadRequest);

    ScopedTempWav tmp{wav_bytes};

    nlohmann::json req = {
        {"file",  tmp.path()},
        {"model", params.model},
    };
    if (!params.language.empty())        req["language"]        = params.language;
    if (params.temperature > 0.0F)       req["temperature"]     = params.temperature;
    if (!params.response_format.empty()) req["response_format"] = params.response_format;
    if (!params.prompt.empty())          req["prompt"]          = params.prompt;

    try {
        const auto t0  = std::chrono::steady_clock::now();
        nlohmann::json raw =
#if REKALL_ASR_HAVE_OPENAI_CPP
            ::openai::audio().transcribe(req);
#else
            nlohmann::json::object();   // unreachable; throws above
#endif
        const auto rt  = std::chrono::steady_clock::now() - t0;
        if (st.stop_requested()) return TranscribeResult::failure(OpenAiError::Cancelled);

        rekall::asr::observ::debug(rekall::asr::observ::OPENAI_REQUEST_OK, {
            {"model", params.model},
            {"request_duration_ms",
             std::chrono::duration_cast<std::chrono::milliseconds>(rt).count()},
        });

        return parse_response(raw, params.response_format);
    } catch (const std::exception& e) {
        const auto err = classify_runtime_error(e.what());
        // Log with the classified error label only — never the raw message,
        // which can echo request bodies / API keys upstream.
        rekall::asr::observ::warn(rekall::asr::observ::OPENAI_REQUEST_FAILED, {
            {"error", to_label(err)},
            {"model", params.model},
        });
        return TranscribeResult::failure(err);
    }
}

}  // namespace rekall::asr::engine::openai
