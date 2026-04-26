#include "rekall/asr/observ/metrics.hpp"

#include <prometheus/histogram.h>

namespace rekall::asr::observ {

namespace {
const prometheus::Histogram::BucketBoundaries kInferenceBuckets = {
    0.025, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0, 10.0,
};
const prometheus::Histogram::BucketBoundaries kSessionBuckets = {
    1.0, 5.0, 30.0, 60.0, 300.0, 900.0, 1800.0,
};
const prometheus::Histogram::BucketBoundaries kModelLoadBuckets = {
    0.1, 0.5, 1.0, 5.0, 15.0, 60.0,
};
}  // namespace

Metrics::Metrics(const std::string& listen)
    : registry_(std::make_shared<prometheus::Registry>()),
      fam_active_sessions_(prometheus::BuildGauge()
          .Name("asr_active_sessions").Help("currently active asr sessions")
          .Register(*registry_)),
      fam_worker_pool_size_(prometheus::BuildGauge()
          .Name("asr_worker_pool_size").Help("configured worker pool size")
          .Register(*registry_)),
      fam_worker_pool_in_use_(prometheus::BuildGauge()
          .Name("asr_worker_pool_in_use").Help("workers currently servicing sessions")
          .Register(*registry_)),
      fam_audio_seconds_processed_(prometheus::BuildCounter()
          .Name("asr_audio_seconds_processed_total")
          .Help("seconds of audio processed by whisper").Register(*registry_)),
      fam_partial_events_(prometheus::BuildCounter()
          .Name("asr_partial_events_total").Help("partial transcripts emitted")
          .Register(*registry_)),
      fam_final_events_(prometheus::BuildCounter()
          .Name("asr_final_events_total").Help("final transcripts emitted")
          .Register(*registry_)),
      fam_admission_rejected_(prometheus::BuildCounter()
          .Name("asr_admission_rejected_total")
          .Help("session admission rejected").Register(*registry_)),
      fam_dropped_partials_(prometheus::BuildCounter()
          .Name("asr_dropped_partials_total")
          .Help("partial events dropped under outbound backpressure")
          .Register(*registry_)),
      fam_auth_failed_(prometheus::BuildCounter()
          .Name("asr_auth_failed_total")
          .Help("session-token validation failures").Register(*registry_)),
      fam_ws_close_(prometheus::BuildCounter()
          .Name("asr_ws_close_total")
          .Help("websocket connections closed by code").Register(*registry_)),
      fam_inference_duration_(prometheus::BuildHistogram()
          .Name("asr_inference_duration_seconds")
          .Help("whisper inference duration by event type").Register(*registry_)),
      fam_session_duration_(prometheus::BuildHistogram()
          .Name("asr_session_duration_seconds")
          .Help("session wall-clock duration").Register(*registry_)),
      fam_model_load_duration_(prometheus::BuildHistogram()
          .Name("asr_model_load_duration_seconds")
          .Help("whisper model load duration").Register(*registry_)) {
    if (!listen.empty()) {
        exposer_ = std::make_unique<prometheus::Exposer>(listen);
        exposer_->RegisterCollectable(registry_);
    }
}

Metrics::~Metrics() = default;

prometheus::Gauge& Metrics::active_sessions()       { return fam_active_sessions_.Add({}); }
prometheus::Gauge& Metrics::worker_pool_size()      { return fam_worker_pool_size_.Add({}); }
prometheus::Gauge& Metrics::worker_pool_in_use()    { return fam_worker_pool_in_use_.Add({}); }

prometheus::Counter& Metrics::audio_seconds_processed_total(const std::string& m) {
    return fam_audio_seconds_processed_.Add({{"model_id", m}});
}
prometheus::Counter& Metrics::partial_events_total(const std::string& m) {
    return fam_partial_events_.Add({{"model_id", m}});
}
prometheus::Counter& Metrics::final_events_total(const std::string& m) {
    return fam_final_events_.Add({{"model_id", m}});
}
prometheus::Counter& Metrics::admission_rejected_total(const std::string& reason) {
    return fam_admission_rejected_.Add({{"reason", reason}});
}
prometheus::Counter& Metrics::dropped_partials_total() {
    return fam_dropped_partials_.Add({});
}
prometheus::Counter& Metrics::auth_failed_total(const std::string& reason) {
    return fam_auth_failed_.Add({{"reason", reason}});
}
prometheus::Counter& Metrics::ws_close_total(const std::string& code) {
    return fam_ws_close_.Add({{"code", code}});
}

prometheus::Histogram& Metrics::inference_duration_seconds(const std::string& m,
                                                          const std::string& event) {
    return fam_inference_duration_.Add({{"model_id", m}, {"event", event}}, kInferenceBuckets);
}
prometheus::Histogram& Metrics::session_duration_seconds() {
    return fam_session_duration_.Add({}, kSessionBuckets);
}
prometheus::Histogram& Metrics::model_load_duration_seconds(const std::string& m) {
    return fam_model_load_duration_.Add({{"model_id", m}}, kModelLoadBuckets);
}

}  // namespace rekall::asr::observ
