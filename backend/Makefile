.PHONY: build run test test-integration test-hook lint fmt tidy docs docs-check migrate-up migrate-down clean

BINARY       := bin/rekall-server
MAIN         := ./cmd/server
SWAG_VERSION := latest
MIGRATE      := $(shell which migrate 2>/dev/null)
DB_URL       ?= postgres://rekall:rekall_secret@localhost:5432/rekall_db?sslmode=disable
TEST_FLAGS   ?= -v -race -count=1

# ─── Documentation ───────────────────────────────────────────────────────────
docs: ## Generate Swagger/OpenAPI spec from handler annotations
	@which swag > /dev/null 2>&1 || go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
	swag init \
	  --generalInfo  cmd/server/main.go \
	  --output        docs \
	  --parseDependency \
	  --parseInternal

docs-check: ## Fail if committed docs/ is out of date with current annotations
	@which swag > /dev/null 2>&1 || go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
	@cp -r docs docs_backup_check 2>/dev/null || true
	@$(MAKE) --no-print-directory docs
	@diff -rq docs docs_backup_check > /dev/null 2>&1 \
	  && echo "docs/ is up to date." \
	  || (echo "ERROR: docs/ is out of date. Run 'make docs' and commit the result." && rm -rf docs_backup_check && exit 1)
	@rm -rf docs_backup_check

# ─── Build ────────────────────────────────────────────────────────────────────
build: docs
	go build -ldflags="-s -w" -o $(BINARY) $(MAIN)

run: build
	./$(BINARY)

# ─── Test ─────────────────────────────────────────────────────────────────────
test:
	go test $(TEST_FLAGS) ./tests/unit/...

test-integration:
	go test $(TEST_FLAGS) ./tests/integration/...

test-all:
	go test $(TEST_FLAGS) ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Integration test for the Go pre-commit hook (Linux/macOS only).
# Skips cleanly when gofmt/goimports/golangci-lint are not on $PATH.
test-hook:
	bash tests/integration/pre-commit-hook.sh

# ─── Lint & Format ────────────────────────────────────────────────────────────
lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

vet:
	go vet ./...

# ─── Dependencies ─────────────────────────────────────────────────────────────
tidy:
	go mod tidy

# ─── Migrations ───────────────────────────────────────────────────────────────
migrate-up:
	migrate -path ./migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DB_URL)" down 1

migrate-drop:
	migrate -path ./migrations -database "$(DB_URL)" drop -f

# ─── Clean ────────────────────────────────────────────────────────────────────
clean:
	rm -rf bin/ coverage.out coverage.html
