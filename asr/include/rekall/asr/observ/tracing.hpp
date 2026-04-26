// OpenTelemetry-compatible tracing facade.
//
// V1 ships a no-op implementation: when OTEL_EXPORTER_OTLP_ENDPOINT is unset
// (or this header's tracing backend is not compiled in) every Span call is a
// no-op. The interface is in place so callers (Transcriber, JWTValidator,
// gRPC handlers) can be instrumented today without conditional compilation.

#pragma once

#include <chrono>
#include <memory>
#include <string>
#include <string_view>
#include <unordered_map>

namespace rekall::asr::observ {

class Span {
   public:
    virtual ~Span() = default;
    virtual void set_attribute(std::string_view key, std::string_view value) = 0;
    virtual void set_attribute(std::string_view key, std::int64_t value)     = 0;
    virtual void set_status_ok() = 0;
    virtual void set_status_error(std::string_view message) = 0;
    virtual void end() = 0;
};

class Tracer {
   public:
    virtual ~Tracer() = default;
    virtual std::unique_ptr<Span> start_span(std::string_view name) = 0;
};

// Returns a no-op tracer. The full OTLP exporter wiring lands in a follow-up
// commit so this file only carries the interface today.
std::shared_ptr<Tracer> make_noop_tracer();

// Initialise the global tracer. If `otel_endpoint` is empty, returns a no-op.
std::shared_ptr<Tracer> init_tracer(std::string_view otel_endpoint);

}  // namespace rekall::asr::observ
