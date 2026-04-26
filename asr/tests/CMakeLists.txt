# gtest is provided by vcpkg.
find_package(GTest CONFIG REQUIRED)

add_executable(rekall_asr_unit_tests
    unit/jwt_validator_test.cpp
    unit/jti_cache_test.cpp
    unit/ring_buffer_test.cpp
    unit/vad_segmenter_test.cpp
    unit/model_registry_test.cpp
    unit/worker_pool_test.cpp
    unit/config_loader_test.cpp
    unit/correlation_id_test.cpp
    unit/transcript_event_test.cpp
)
target_link_libraries(rekall_asr_unit_tests PRIVATE
    rekall_asr_core
    GTest::gtest
    GTest::gtest_main
)
add_test(NAME rekall_asr_unit_tests COMMAND rekall_asr_unit_tests)

add_subdirectory(integration)
if(REKALL_ASR_BUILD_LOAD)
    add_subdirectory(load)
endif()
