// Correlation-id helpers. The Go backend emits `x-correlation-id` on every
// gRPC call; the WS path generates one on upgrade. We thread it through every
// log line so a single ID joins backend ↔ ASR logs.

#pragma once

#include <string>
#include <string_view>

namespace rekall::asr::util {

// Returns true if `s` looks like a UUIDv4 (8-4-4-4-12 hex with hyphens).
// Used for input validation; we accept any non-empty short string as a
// correlation_id but only emit our own as UUIDv4.
bool looks_like_uuid_v4(std::string_view s);

// Generates a random UUIDv4 string (lowercase, with hyphens).
std::string new_uuid_v4();

// Returns `incoming` if non-empty, otherwise generates a fresh UUIDv4.
std::string ensure(std::string_view incoming);

}  // namespace rekall::asr::util
