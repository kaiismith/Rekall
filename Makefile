.PHONY: up down logs restart build build-all build-bake up-asr build-asr migrate \
        backend-test backend-lint backend-build \
        frontend-test frontend-lint frontend-build \
        asr-build asr-test asr-lint asr-load asr-image asr-proto-go

# ─── Docker Compose ───────────────────────────────────────────────────────────
# `up` builds + starts the basic stack (postgres + backend + frontend + mailpit).
# Use `up-asr` to also build + start the C++ ASR microservice.
#
# `build-all` and `build-bake` build every image in PARALLEL (default compose
# behaviour serialises image builds across services); use these when iterating
# on multiple Dockerfiles at once.
up:
	docker compose up -d --build

up-asr:
	docker compose --profile asr up -d --build

down:
	docker compose --profile asr down

logs:
	docker compose --profile asr logs -f

restart:
	docker compose --profile asr down && docker compose --profile asr up -d --build

# Build only — does NOT start the containers. Sequential.
build:
	docker compose build

# Build EVERY service image (basic + asr) in parallel via compose.
# Faster on multi-core machines than the default serial build.
build-all:
	docker compose --profile asr build --parallel

# Same as build-all but uses BuildKit's bake driver — parallel by default,
# better cache reuse, and can build cross-arch with --set "*.platform=...".
build-bake:
	docker buildx bake --file docker-compose.yml

# Convenience: build only the asr image (handy after touching asr/Dockerfile
# or asr/CMakeLists.txt — avoids rebuilding the Go/JS images).
build-asr:
	docker compose --profile asr build asr

# ─── Database ─────────────────────────────────────────────────────────────────
migrate-up:
	cd backend && make migrate-up

migrate-down:
	cd backend && make migrate-down

# ─── Backend ──────────────────────────────────────────────────────────────────
backend-test:
	cd backend && make test

backend-lint:
	cd backend && make lint

backend-build:
	cd backend && make build

# ─── Frontend ─────────────────────────────────────────────────────────────────
frontend-test:
	cd frontend && npm run test

frontend-lint:
	cd frontend && npm run lint

frontend-build:
	cd frontend && npm run build

# ─── ASR (C++) ────────────────────────────────────────────────────────────────
# Set ASR_BUILD=1 to include the C++ service in the aggregate `test`/`lint`
# targets. Without it, CI environments without the C++ toolchain still pass.
ASR_DIR := asr
ASR_BUILD_DIR := $(ASR_DIR)/build
VCPKG_TOOLCHAIN ?= $(VCPKG_ROOT)/scripts/buildsystems/vcpkg.cmake

asr-build:
	cmake -G Ninja -B $(ASR_BUILD_DIR) -S $(ASR_DIR) \
	      -DCMAKE_BUILD_TYPE=Release \
	      $(if $(VCPKG_ROOT),-DCMAKE_TOOLCHAIN_FILE=$(VCPKG_TOOLCHAIN),)
	cmake --build $(ASR_BUILD_DIR) --target rekall-asr -j

asr-test:
	cmake -G Ninja -B $(ASR_BUILD_DIR) -S $(ASR_DIR) \
	      -DCMAKE_BUILD_TYPE=Debug \
	      -DREKALL_ASR_BUILD_TESTS=ON \
	      $(if $(VCPKG_ROOT),-DCMAKE_TOOLCHAIN_FILE=$(VCPKG_TOOLCHAIN),)
	cmake --build $(ASR_BUILD_DIR) -j
	cd $(ASR_BUILD_DIR) && ctest --output-on-failure

asr-lint:
	bash $(ASR_DIR)/scripts/format.sh --check

# Quick syntax-only compile check on every .cpp without actually building.
# Catches missing #include, undeclared identifiers, and the like in seconds —
# no linker, no codegen, no full build. Requires the asr image to have been
# built once so the vcpkg installed/ tree exists.
asr-lint-cc:
	docker run --rm -v $(PWD)/$(ASR_DIR):/src -w /src \
	  -e CXX=g++ rekall-asr:latest sh -c \
	  'find src -name "*.cpp" -print0 | xargs -0 -I{} g++ -std=c++20 -fsyntax-only \
	   -Iinclude -Ibuild/proto/generated \
	   -I/var/lib/rekall-asr/include $(EXTRA_CXXFLAGS) {}'

asr-load:
	cmake -G Ninja -B $(ASR_BUILD_DIR) -S $(ASR_DIR) -DREKALL_ASR_BUILD_LOAD=ON \
	      $(if $(VCPKG_ROOT),-DCMAKE_TOOLCHAIN_FILE=$(VCPKG_TOOLCHAIN),)
	cmake --build $(ASR_BUILD_DIR) --target asr_load -j
	$(ASR_BUILD_DIR)/tests/load/asr_load --concurrency=$(or $(CONCURRENCY),4)

asr-image:
	docker build -f $(ASR_DIR)/docker/Dockerfile -t rekall-asr:dev $(ASR_DIR)

# Generates Go stubs into backend/internal/infrastructure/asr/pb/.
asr-proto-go:
	mkdir -p backend/internal/infrastructure/asr/pb
	protoc -I $(ASR_DIR)/proto \
	       --go_out=backend/internal/infrastructure/asr/pb --go_opt=paths=source_relative \
	       --go-grpc_out=backend/internal/infrastructure/asr/pb --go-grpc_opt=paths=source_relative \
	       $(ASR_DIR)/proto/asr.proto

# ─── All ──────────────────────────────────────────────────────────────────────
test: backend-test frontend-test
ifeq ($(ASR_BUILD),1)
test: asr-test
endif

lint: backend-lint frontend-lint
ifeq ($(ASR_BUILD),1)
lint: asr-lint
endif
