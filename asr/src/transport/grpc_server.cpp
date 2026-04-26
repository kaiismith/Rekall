#include "rekall/asr/transport/grpc_server.hpp"

#include <chrono>
#include <fstream>
#include <memory>
#include <sstream>
#include <utility>

#include <google/protobuf/empty.pb.h>
#include <google/protobuf/timestamp.pb.h>
#include <grpcpp/grpcpp.h>
#include <grpcpp/security/server_credentials.h>
#include <grpcpp/server_builder.h>

#include "asr.grpc.pb.h"
#include "rekall/asr/observ/log_catalog.hpp"
#include "rekall/asr/observ/metrics.hpp"
#include "rekall/asr/util/correlation_id.hpp"

namespace rekall::asr::transport {

namespace {

std::string slurp(const std::string& path) {
    std::ifstream f(path, std::ios::binary);
    if (!f.good()) throw std::runtime_error("cannot read " + path);
    std::stringstream ss;
    ss << f.rdbuf();
    return ss.str();
}

std::string correlation_from(const grpc::ServerContext& ctx) {
    auto md = ctx.client_metadata();
    auto it = md.find("x-correlation-id");
    if (it != md.end()) return std::string(it->second.data(), it->second.size());
    return rekall::asr::util::new_uuid_v4();
}

void timestamp_from_tp(google::protobuf::Timestamp* out,
                      std::chrono::system_clock::time_point tp) {
    using namespace std::chrono;
    auto secs = duration_cast<seconds>(tp.time_since_epoch()).count();
    auto nano = duration_cast<nanoseconds>(tp.time_since_epoch()).count() % 1000000000LL;
    out->set_seconds(secs);
    out->set_nanos(static_cast<int>(nano));
}

}  // namespace

class GrpcServer::Impl final : public rekall::asr::v1::ASR::Service {
   public:
    explicit Impl(GrpcServerDeps* deps, std::atomic<bool>* serving)
        : deps_(deps), serving_(serving) {}

    grpc::Status StartSession(grpc::ServerContext* ctx,
                              const rekall::asr::v1::StartSessionRequest* req,
                              rekall::asr::v1::StartSessionResponse* resp) override {
        std::string corr = correlation_from(*ctx);

        // Admission gate.
        if (deps_->workers != nullptr && deps_->workers->free() == 0) {
            if (deps_->metrics) deps_->metrics->admission_rejected_total("grpc_start").Increment();
            rekall::asr::observ::warn(rekall::asr::observ::ADMISSION_REJECTED, {
                {"phase", "grpc_start_session"}, {"correlation_id", corr},
            });
            return grpc::Status(grpc::StatusCode::RESOURCE_EXHAUSTED,
                                "all asr workers busy");
        }

        rekall::asr::session::CreateInput in;
        in.user_id              = req->user_id();
        in.call_id              = req->call_id();
        in.model_id_request     = req->model_id();
        in.language             = req->language();
        in.requested_ttl_seconds = req->requested_token_ttl_seconds();
        in.correlation_id       = corr;

        auto out = deps_->sessions->create(in);

        resp->set_session_id(out.session->sid);
        resp->set_model_id(out.canonical_model_id);
        resp->set_sample_rate(16000);
        resp->set_frame_format("pcm_s16le_mono");
        timestamp_from_tp(resp->mutable_expires_at(), out.expires_at);
        return grpc::Status::OK;
    }

    grpc::Status EndSession(grpc::ServerContext* /*ctx*/,
                            const rekall::asr::v1::EndSessionRequest* req,
                            rekall::asr::v1::EndSessionResponse* resp) override {
        auto out = deps_->sessions->end(req->session_id());
        resp->set_final_transcript(out.final_transcript);
        resp->set_final_count(out.final_count);
        return grpc::Status::OK;
    }

    grpc::Status GetSession(grpc::ServerContext* /*ctx*/,
                            const rekall::asr::v1::GetSessionRequest* req,
                            rekall::asr::v1::SessionInfo* resp) override {
        auto s = deps_->sessions->get(req->session_id());
        if (!s) return grpc::Status(grpc::StatusCode::NOT_FOUND, "session not found");
        resp->set_state(rekall::asr::session::state_name(s->state.load()));
        timestamp_from_tp(resp->mutable_started_at(), s->started_at);
        std::chrono::system_clock::time_point last =
            std::chrono::system_clock::time_point(
                std::chrono::milliseconds(s->last_activity_at_ms.load()));
        timestamp_from_tp(resp->mutable_last_activity_at(), last);
        resp->set_audio_seconds_processed(static_cast<std::uint32_t>(
            s->stats.audio_samples_processed.load() / 16000));
        resp->set_partial_count(s->stats.partial_count.load());
        resp->set_final_count(s->stats.final_count.load());
        return grpc::Status::OK;
    }

    grpc::Status Health(grpc::ServerContext* /*ctx*/,
                        const google::protobuf::Empty* /*req*/,
                        rekall::asr::v1::HealthResponse* resp) override {
        resp->set_status(serving_->load() ? "SERVING" : "NOT_SERVING");
        resp->set_version(deps_->version);
        auto uptime = std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now() - deps_->started_at).count();
        resp->set_uptime_seconds(static_cast<std::uint64_t>(uptime));
        if (deps_->models != nullptr) {
            for (const auto& id : deps_->models->loaded_ids()) {
                resp->add_loaded_models(id);
            }
        }
        if (deps_->sessions != nullptr) {
            resp->set_active_sessions(static_cast<std::uint32_t>(deps_->sessions->active_count()));
        }
        if (deps_->workers != nullptr) {
            resp->set_worker_pool_size(static_cast<std::uint32_t>(deps_->workers->size()));
            resp->set_worker_pool_in_use(static_cast<std::uint32_t>(deps_->workers->in_use()));
        }
        return grpc::Status::OK;
    }

    grpc::Status ReloadModels(grpc::ServerContext* /*ctx*/,
                              const rekall::asr::v1::ReloadModelsRequest* req,
                              rekall::asr::v1::ReloadModelsResponse* resp) override {
        std::vector<rekall::asr::config::ModelEntry> entries;
        entries.reserve(req->entries_size());
        for (const auto& e : req->entries()) {
            rekall::asr::config::ModelEntry m;
            m.id        = e.id();
            m.path      = e.path();
            m.language  = e.language();
            m.n_threads = e.n_threads();
            m.beam_size = e.beam_size();
            entries.push_back(std::move(m));
        }
        if (deps_->models == nullptr) {
            return grpc::Status(grpc::StatusCode::FAILED_PRECONDITION,
                                "model registry unavailable");
        }
        auto result = deps_->models->reload(entries);
        for (auto& id : result.loaded) resp->add_loaded(std::move(id));
        for (auto& id : result.failed) resp->add_failed(std::move(id));
        return grpc::Status::OK;
    }

    // StreamSession is wired in but intentionally returns UNIMPLEMENTED for v1
    // because the production data path is the WebSocket. Tests that need a
    // server-side stream interface will land alongside the integration suite.
    grpc::Status StreamSession(grpc::ServerContext* /*ctx*/,
                               grpc::ServerReaderWriter<
                                   rekall::asr::v1::TranscriptEvent,
                                   rekall::asr::v1::StreamChunk>* /*stream*/) override {
        return grpc::Status(grpc::StatusCode::UNIMPLEMENTED,
                            "use the websocket data plane");
    }

   private:
    GrpcServerDeps* deps_;
    std::atomic<bool>* serving_;
};

GrpcServer::GrpcServer(GrpcServerDeps deps)
    : deps_(std::move(deps)), impl_(std::make_unique<Impl>(&deps_, &serving_)) {}

GrpcServer::~GrpcServer() { stop(); }

void GrpcServer::start() {
    grpc::ServerBuilder builder;

    std::shared_ptr<grpc::ServerCredentials> creds;
    bool have_mtls = !deps_.cfg.tls.grpc_cert.empty() &&
                     !deps_.cfg.tls.grpc_key.empty()  &&
                     !deps_.cfg.tls.grpc_client_ca.empty();
    if (have_mtls) {
        grpc::SslServerCredentialsOptions opts(
            GRPC_SSL_REQUEST_AND_REQUIRE_CLIENT_CERTIFICATE_AND_VERIFY);
        opts.pem_root_certs = slurp(deps_.cfg.tls.grpc_client_ca);
        opts.pem_key_cert_pairs.push_back({slurp(deps_.cfg.tls.grpc_key),
                                           slurp(deps_.cfg.tls.grpc_cert)});
        creds = grpc::SslServerCredentials(opts);
    } else {
        // Insecure is acceptable only when bound to loopback, OR the operator
        // explicitly opted in via allow_insecure_grpc (dev / docker-compose
        // bridge networking, where the backend reaches asr:9090 across a
        // user-defined network and loopback isn't reachable).
        if (deps_.cfg.server.grpc_bind_all && !deps_.cfg.server.allow_insecure_grpc) {
            throw std::runtime_error(
                "grpc bound off loopback without mTLS — refuse to start "
                "(set ASR_ALLOW_INSECURE_GRPC=true for local dev)");
        }
        creds = grpc::InsecureServerCredentials();
    }

    builder.AddListeningPort(deps_.cfg.server.grpc_listen, creds);
    builder.RegisterService(impl_.get());
    server_ = builder.BuildAndStart();
    if (!server_) throw std::runtime_error("failed to start gRPC server on " +
                                           deps_.cfg.server.grpc_listen);

    rekall::asr::observ::info(rekall::asr::observ::SERVICE_READY, {
        {"component", "grpc_server"},
        {"listen",    deps_.cfg.server.grpc_listen},
        {"mtls",      have_mtls},
    });
}

void GrpcServer::stop() {
    if (!server_) return;
    auto deadline = std::chrono::system_clock::now() +
                    std::chrono::seconds(deps_.cfg.session.graceful_drain_seconds);
    serving_.store(false);
    server_->Shutdown(deadline);
    server_.reset();
}

void GrpcServer::set_serving(bool serving) { serving_.store(serving); }

}  // namespace rekall::asr::transport
