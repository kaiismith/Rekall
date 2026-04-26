# Multi-stage build for the rekall-asr C++ service.
#
# Stage 1: Ubuntu builder with vcpkg + full toolchain. Also fetches the
#          grpc-health-probe release binary so we don't depend on a
#          dedicated image for it.
# Stage 2: distroless runtime — final image is small (~120 MB w/o models).
#
# Build with BuildKit (Docker Desktop has it on by default; for raw `docker
# build` set DOCKER_BUILDKIT=1). The vcpkg clone, vcpkg downloads, and
# vcpkg build trees all live in BuildKit cache mounts so subsequent builds
# don't re-fetch or re-compile gRPC/Boost/spdlog/etc even when the source
# layer is invalidated.

# syntax=docker/dockerfile:1.6

# ── Stage 1: build ───────────────────────────────────────────────────────────
FROM ubuntu:22.04 AS build

# VCPKG_REF is the GIT TAG used to clone vcpkg itself (passed to `git clone --branch`).
# Two related but distinct values exist:
#   - VCPKG_REF             — git tag, here in the Dockerfile (e.g. "2024.10.21").
#   - "builtin-baseline" in asr/vcpkg.json — 40-char commit SHA. MUST be a SHA,
#     never a tag. To bump after changing VCPKG_REF here, run inside any clone:
#         git -C /tmp/vcpkg fetch --tags origin
#         git -C /tmp/vcpkg rev-parse <new-tag>
#     and paste the SHA into vcpkg.json.
ARG VCPKG_REF=2024.10.21

# Pinned grpc_health_probe release. Statically linked, ~5 MB.
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.24

ENV DEBIAN_FRONTEND=noninteractive
# patchelf is included to bypass vcpkg's runtime download of
# patchelf-0.14.5 from github.com/NixOS/patchelf/releases — that mirror
# returns transient 502s and breaks otherwise-clean boost-cmake builds.
# vcpkg_find_acquire_program(PATCHELF) prefers the system binary when present.
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential cmake git curl ca-certificates pkg-config \
        zip unzip tar ninja-build patchelf \
        autoconf automake libtool python3 \
    && rm -rf /var/lib/apt/lists/*

# grpc-health-probe — statically linked Go binary published only as a release
# asset (no public Docker image exists). Verifying the URL succeeds before
# the long vcpkg/whisper.cpp build is wasted.
RUN curl -fsSL \
        "https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64" \
        -o /usr/local/bin/grpc_health_probe \
    && chmod +x /usr/local/bin/grpc_health_probe

ENV VCPKG_ROOT=/opt/vcpkg
# vcpkg lives in a BuildKit cache mount so the (~250 MB) clone survives across
# rebuilds even when this layer is invalidated. The cache is keyed by
# VCPKG_REF, so bumping the ref version triggers a fresh clone automatically.
#
# Fail-fast (`set -e`); validate the cache by the presence of bootstrap-vcpkg.sh
# (more meaningful than .git, which can survive a partial clone). If the cache
# is invalid, wipe it and re-clone. Then copy CONTENTS into VCPKG_ROOT using
# the explicit `SRC/. DST/` form so it works whether or not VCPKG_ROOT
# pre-exists.
#
# A shallow clone breaks vcpkg's builtin-baseline resolution (the manifest's
# baseline SHA must be reachable from HEAD), so we do a full clone.
RUN --mount=type=cache,target=/var/cache/vcpkg-src,id=vcpkg-src-${VCPKG_REF} \
    set -eu && \
    if [ ! -f /var/cache/vcpkg-src/bootstrap-vcpkg.sh ]; then \
        rm -rf /var/cache/vcpkg-src/* /var/cache/vcpkg-src/.[!.]* 2>/dev/null || true; \
        git clone --branch ${VCPKG_REF} https://github.com/microsoft/vcpkg.git /var/cache/vcpkg-src; \
    fi && \
    mkdir -p ${VCPKG_ROOT} && \
    cp -a /var/cache/vcpkg-src/. ${VCPKG_ROOT}/ && \
    ${VCPKG_ROOT}/bootstrap-vcpkg.sh -disableMetrics

WORKDIR /src
COPY . .

# Initialise the whisper.cpp submodule if it was committed; otherwise skip
# (the CMakeLists prints a warning when the directory is empty).
RUN if [ -d third_party/whisper.cpp/.git ]; then \
        git -C third_party/whisper.cpp fetch --depth 1 origin || true; \
    fi

# Cache mounts on vcpkg's downloads (tarballs) and buildtrees (compiler output)
# turn the slow "compile gRPC + Boost + …" step from minutes into seconds on
# every rebuild after the first. The `installed` tree must NOT be mounted —
# CMake reads from it during configure.
#
# CONCURRENCY_FOR_VCPKG controls how many g++ processes run in parallel during
# the vcpkg dep build AND the subsequent whisper.cpp compile.
#
# whisper.cpp's ggml-cpu/ops.cpp consumes ~5 GB RAM in cc1plus on its own;
# boost/gRPC compile units consume ~1.5 GB each. Default to 1 so the build
# is reliable on WSL2's stock ~8 GB memory split — once you've bumped
# `~/.wslconfig` or are building on bare metal with 16 GB+, override at
# build time:
#   docker compose --profile asr build --build-arg CONCURRENCY_FOR_VCPKG=4 asr
ARG CONCURRENCY_FOR_VCPKG=1
ENV VCPKG_MAX_CONCURRENCY=${CONCURRENCY_FOR_VCPKG}
RUN --mount=type=cache,target=${VCPKG_ROOT}/downloads,id=vcpkg-downloads \
    --mount=type=cache,target=${VCPKG_ROOT}/buildtrees,id=vcpkg-buildtrees \
    --mount=type=cache,target=/root/.cache/vcpkg,id=vcpkg-binarycache \
    cmake -G Ninja -B build -S . \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_TOOLCHAIN_FILE=${VCPKG_ROOT}/scripts/buildsystems/vcpkg.cmake \
        -DREKALL_ASR_BUILD_TESTS=OFF \
        -DVCPKG_MAX_CONCURRENCY=${CONCURRENCY_FOR_VCPKG} \
    && cmake --build build --target rekall-asr -j${CONCURRENCY_FOR_VCPKG}

# ── Stage 2: distroless runtime ──────────────────────────────────────────────
FROM gcr.io/distroless/cc-debian12 AS runtime

COPY --from=build /src/build/rekall-asr /usr/local/bin/rekall-asr
# Active config baked into the image at /etc/rekall-asr/config.yaml. Operators
# can still mount a different one over this path at runtime.
COPY --from=build /src/config/config.yaml /etc/rekall-asr/config.yaml
COPY --from=build /usr/local/bin/grpc_health_probe /usr/local/bin/grpc_health_probe

EXPOSE 8081 9090 9091

# Distroless ships nonroot user uid:gid 65532.
USER 65532:65532

HEALTHCHECK --interval=15s --timeout=5s --retries=3 \
  CMD ["/usr/local/bin/grpc_health_probe", "-addr=127.0.0.1:9090"]

ENTRYPOINT ["/usr/local/bin/rekall-asr"]
CMD ["--config", "/etc/rekall-asr/config.yaml"]
