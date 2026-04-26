cmake_minimum_required(VERSION 3.22)
project(rekall_asr LANGUAGES CXX C VERSION 0.1.0)

# ── Standards ────────────────────────────────────────────────────────────────
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
endif()

# ── Options ──────────────────────────────────────────────────────────────────
option(REKALL_ASR_BUILD_TESTS "Build the gtest suite"           ON)
option(REKALL_ASR_BUILD_LOAD  "Build the opt-in load benchmark" OFF)
option(REKALL_ASR_USE_TSAN    "Enable ThreadSanitizer"          OFF)
option(REKALL_ASR_USE_ASAN    "Enable AddressSanitizer"         OFF)

if(REKALL_ASR_USE_TSAN)
    add_compile_options(-fsanitize=thread -g -O1)
    add_link_options(-fsanitize=thread)
endif()
if(REKALL_ASR_USE_ASAN)
    add_compile_options(-fsanitize=address -fno-omit-frame-pointer -g -O1)
    add_link_options(-fsanitize=address)
endif()

# Strict warnings on OUR code only — applied later to the rekall_asr_core
# target via target_compile_options. Setting them globally here would propagate
# into add_subdirectory(third_party/whisper.cpp) and flood the build with
# upstream -Wshadow noise (gguf.cpp, ggml-cpu, ops.cpp). Defined as a list so
# the same flags can be applied to test executables too.
if(MSVC)
    set(REKALL_ASR_STRICT_WARNINGS /W4 /permissive-)
else()
    set(REKALL_ASR_STRICT_WARNINGS
        -Wall -Wextra -Wpedantic -Wshadow -Wnon-virtual-dtor)
endif()

# ── Vendored: whisper.cpp ────────────────────────────────────────────────────
# Force STATIC linkage for whisper / ggml so everything ends up inside the
# single `rekall-asr` binary. Otherwise the build emits libwhisper.so.1,
# libggml.so.0, libggml-cpu.so.0, libggml-base.so.0 which the distroless
# runtime image can't resolve at startup. BUILD_SHARED_LIBS scopes only to
# this subdirectory because we set it before add_subdirectory.
set(BUILD_SHARED_LIBS      OFF CACHE BOOL "" FORCE)
set(WHISPER_BUILD_TESTS    OFF CACHE BOOL "" FORCE)
set(WHISPER_BUILD_EXAMPLES OFF CACHE BOOL "" FORCE)
set(WHISPER_BUILD_SERVER   OFF CACHE BOOL "" FORCE)
if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/third_party/whisper.cpp/CMakeLists.txt")
    add_subdirectory(third_party/whisper.cpp EXCLUDE_FROM_ALL)
else()
    message(WARNING
        "third_party/whisper.cpp is empty. Run: git submodule update --init --recursive")
endif()

# ── Third-party (vcpkg) ──────────────────────────────────────────────────────
find_package(Threads             REQUIRED)
find_package(spdlog              CONFIG REQUIRED)
find_package(yaml-cpp            CONFIG REQUIRED)
find_package(nlohmann_json       CONFIG REQUIRED)
find_package(prometheus-cpp      CONFIG REQUIRED)
find_package(jwt-cpp             CONFIG REQUIRED)
find_package(Boost 1.83          REQUIRED COMPONENTS system)
find_package(Protobuf            CONFIG REQUIRED)
find_package(gRPC                CONFIG REQUIRED)
find_package(absl                CONFIG REQUIRED)

# ── Generated proto ──────────────────────────────────────────────────────────
add_subdirectory(proto)

# ── Core library ─────────────────────────────────────────────────────────────
add_library(rekall_asr_core STATIC
    src/auth/jwt_validator.cpp
    src/auth/jti_cache.cpp
    src/audio/ring_buffer.cpp
    src/audio/vad_segmenter.cpp
    src/config/config.cpp
    src/engine/model_registry.cpp
    src/engine/transcriber.cpp
    src/engine/worker_pool.cpp
    src/observ/log_catalog.cpp
    src/observ/metrics.cpp
    src/observ/tracing.cpp
    src/session/session.cpp
    src/session/session_manager.cpp
    src/session/transcript_event.cpp
    src/transport/grpc_server.cpp
    src/transport/grpc_health.cpp
    src/transport/ws_server.cpp
    src/transport/ws_session.cpp
    src/util/correlation_id.cpp
)

# Apply strict warnings ONLY to our own code (PRIVATE so they don't propagate
# to consumers like the executable / tests, which can opt in separately).
target_compile_options(rekall_asr_core PRIVATE ${REKALL_ASR_STRICT_WARNINGS})

target_include_directories(rekall_asr_core PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(rekall_asr_core PUBLIC
    rekall_asr_proto
    spdlog::spdlog
    yaml-cpp::yaml-cpp
    nlohmann_json::nlohmann_json
    prometheus-cpp::core
    prometheus-cpp::pull
    jwt-cpp::jwt-cpp
    Boost::system
    gRPC::grpc++
    gRPC::grpc++_reflection
    protobuf::libprotobuf
    absl::base
    absl::strings
    absl::flat_hash_map
    Threads::Threads
)

if(TARGET whisper)
    target_link_libraries(rekall_asr_core PUBLIC whisper)
endif()

# ── Binary ───────────────────────────────────────────────────────────────────
add_executable(rekall-asr src/main.cpp)
target_link_libraries(rekall-asr PRIVATE rekall_asr_core)

# ── Tests ────────────────────────────────────────────────────────────────────
if(REKALL_ASR_BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()

# ── Install ──────────────────────────────────────────────────────────────────
install(TARGETS rekall-asr RUNTIME DESTINATION bin)
install(FILES   config/config.example.yaml DESTINATION etc/rekall-asr RENAME config.yaml)
