// Prometheus metrics registry for the ASR service.
//
// All metrics are exposed on a separate HTTP listener (default :9091) at
// `GET /metrics` in the standard Prometheus text format. The ASR_Service
// owns one Metrics instance and passes it by reference to subsystems.

#pragma once

#include <prometheus/counter.h>
#include <prometheus/exposer.h>
#include <prometheus/family.h>
#include <prometheus/gauge.h>
#include <prometheus/histogram.h>
#include <prometheus/registry.h>

#include <memory>
#include <string>

namespace rekall::asr::observ {

class Metrics {
   public:
    // `listen` is host:port. If empty, no HTTP exposer is started (useful in tests).
    explicit Metrics(const std::string& listen);
    ~Metrics();

    Metrics(const Metrics&)            = delete;
    Metrics& operator=(const Metrics&) = delete;

    // Gauges
    prometheus::Gauge& active_sessions();
    prometheus::Gauge& worker_pool_size();
    prometheus::Gauge& worker_pool_in_use();

    // Counters (labelled)
    prometheus::Counter& audio_seconds_processed_total(const std::string& model_id);
    prometheus::Counter& partial_events_total         (const std::string& model_id);
    prometheus::Counter& final_events_total           (const std::string& model_id);
    prometheus::Counter& admission_rejected_total     (const std::string& reason);
    prometheus::Counter& dropped_partials_total();
    prometheus::Counter& auth_failed_total            (const std::string& reason);
    prometheus::Counter& ws_close_total               (const std::string& code);

    // Histograms
    prometheus::Histogram& inference_duration_seconds (const std::string& model_id,
                                                       const std::string& event);
    prometheus::Histogram& session_duration_seconds();
    prometheus::Histogram& model_load_duration_seconds(const std::string& model_id);

    prometheus::Registry& registry() noexcept { return *registry_; }

   private:
    std::shared_ptr<prometheus::Registry> registry_;
    std::unique_ptr<prometheus::Exposer>  exposer_;

    // Families — single instance, lookup via Add({{label,value}}).
    prometheus::Family<prometheus::Gauge>&     fam_active_sessions_;
    prometheus::Family<prometheus::Gauge>&     fam_worker_pool_size_;
    prometheus::Family<prometheus::Gauge>&     fam_worker_pool_in_use_;
    prometheus::Family<prometheus::Counter>&   fam_audio_seconds_processed_;
    prometheus::Family<prometheus::Counter>&   fam_partial_events_;
    prometheus::Family<prometheus::Counter>&   fam_final_events_;
    prometheus::Family<prometheus::Counter>&   fam_admission_rejected_;
    prometheus::Family<prometheus::Counter>&   fam_dropped_partials_;
    prometheus::Family<prometheus::Counter>&   fam_auth_failed_;
    prometheus::Family<prometheus::Counter>&   fam_ws_close_;
    prometheus::Family<prometheus::Histogram>& fam_inference_duration_;
    prometheus::Family<prometheus::Histogram>& fam_session_duration_;
    prometheus::Family<prometheus::Histogram>& fam_model_load_duration_;
};

}  // namespace rekall::asr::observ
