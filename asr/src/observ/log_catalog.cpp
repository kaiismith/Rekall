#include "rekall/asr/observ/log_catalog.hpp"

#include <spdlog/sinks/stdout_color_sinks.h>

#include <chrono>
#include <iomanip>
#include <memory>
#include <mutex>
#include <sstream>
#include <string>
#include <string_view>

namespace rekall::asr::observ {

namespace {

std::shared_ptr<spdlog::logger> g_logger;
std::once_flag g_once;
bool g_json = true;

spdlog::level::level_enum parse_level(std::string_view s) {
    if (s == "debug") return spdlog::level::debug;
    if (s == "warn" || s == "warning") return spdlog::level::warn;
    if (s == "error") return spdlog::level::err;
    return spdlog::level::info;
}

std::string iso8601_utc_now() {
    using namespace std::chrono;
    auto now = system_clock::now();
    auto ms  = duration_cast<milliseconds>(now.time_since_epoch()) % 1000;
    auto t   = system_clock::to_time_t(now);
    std::tm tm{};
#ifdef _WIN32
    gmtime_s(&tm, &t);
#else
    gmtime_r(&t, &tm);
#endif
    std::ostringstream os;
    os << std::put_time(&tm, "%Y-%m-%dT%H:%M:%S")
       << '.' << std::setw(3) << std::setfill('0') << ms.count() << 'Z';
    return os.str();
}

std::string severity_string(spdlog::level::level_enum level) {
    switch (level) {
        case spdlog::level::debug: return "debug";
        case spdlog::level::info:  return "info";
        case spdlog::level::warn:  return "warn";
        case spdlog::level::err:   return "error";
        default:                   return "info";
    }
}

void emit(spdlog::level::level_enum level, const Event& e, const nlohmann::json& fields) {
    if (!g_logger) return;
    if (!g_logger->should_log(level)) return;

    if (g_json) {
        nlohmann::json line = {
            {"event_code",     std::string(e.code)},
            {"event_ts",       iso8601_utc_now()},
            {"level",          severity_string(level)},
            {"msg",            std::string(e.message)},
        };
        if (fields.is_object()) {
            for (auto it = fields.begin(); it != fields.end(); ++it) {
                line[it.key()] = it.value();
            }
        }
        g_logger->log(level, line.dump());
    } else {
        std::ostringstream os;
        os << '[' << e.code << "] " << e.message;
        if (fields.is_object() && !fields.empty()) {
            os << ' ' << fields.dump();
        }
        g_logger->log(level, os.str());
    }
}

}  // namespace

void init_logger(std::string_view level, std::string_view format) {
    std::call_once(g_once, []() {
        g_logger = spdlog::stdout_color_mt("asr");
        // We render the entire JSON ourselves; spdlog only adds a trailing
        // newline. Pattern is just the message.
        g_logger->set_pattern("%v");
    });
    g_json = (format != "text");
    g_logger->set_level(parse_level(level));
}

void debug(const Event& e, const nlohmann::json& fields) { emit(spdlog::level::debug, e, fields); }
void info (const Event& e, const nlohmann::json& fields) { emit(spdlog::level::info,  e, fields); }
void warn (const Event& e, const nlohmann::json& fields) { emit(spdlog::level::warn,  e, fields); }
void error(const Event& e, const nlohmann::json& fields) { emit(spdlog::level::err,   e, fields); }

}  // namespace rekall::asr::observ
