#include "rekall/asr/observ/tracing.hpp"

namespace rekall::asr::observ {

namespace {

class NoopSpan final : public Span {
   public:
    void set_attribute(std::string_view, std::string_view) override {}
    void set_attribute(std::string_view, std::int64_t)     override {}
    void set_status_ok()                                   override {}
    void set_status_error(std::string_view)                override {}
    void end()                                             override {}
};

class NoopTracer final : public Tracer {
   public:
    std::unique_ptr<Span> start_span(std::string_view) override {
        return std::make_unique<NoopSpan>();
    }
};

}  // namespace

std::shared_ptr<Tracer> make_noop_tracer() {
    return std::make_shared<NoopTracer>();
}

std::shared_ptr<Tracer> init_tracer(std::string_view otel_endpoint) {
    // OTLP wiring is deferred — see Requirement 8.6 / Out of Scope. Until the
    // OpenTelemetry C++ SDK is added to vcpkg.json, every endpoint resolves
    // to the no-op tracer so call sites never branch on null.
    (void)otel_endpoint;
    return make_noop_tracer();
}

}  // namespace rekall::asr::observ
