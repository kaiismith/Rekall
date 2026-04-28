# Integration tests run in-process: they spin up the gRPC server + WS server
# against a tiny.en model when REKALL_ASR_INTEGRATION=1 is set, otherwise the
# tests skip themselves so CI without the model file still passes.

find_package(GTest CONFIG REQUIRED)

add_executable(rekall_asr_integration_tests
    start_session_then_ws_test.cpp
    ws_auth_failure_test.cpp
    ws_admission_at_capacity_test.cpp
    graceful_shutdown_test.cpp
)
target_link_libraries(rekall_asr_integration_tests PRIVATE
    rekall_asr_core
    GTest::gtest
    GTest::gtest_main
)
add_test(NAME rekall_asr_integration_tests COMMAND rekall_asr_integration_tests)
