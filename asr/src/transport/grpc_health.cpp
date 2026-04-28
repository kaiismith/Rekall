// gRPC health-check service is provided by the official `grpc::Server` itself
// when EnableDefaultHealthCheckService is set. The wiring lives in main.cpp;
// this TU is a placeholder so the CMakeLists.txt source list compiles
// uniformly. Future work: per-service health probes (e.g. "ASR/inference"
// SERVING vs NOT_SERVING based on worker pool saturation).
namespace rekall::asr::transport {
}  // namespace rekall::asr::transport
