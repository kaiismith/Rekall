.PHONY: up down logs restart build migrate \
        backend-test backend-lint backend-build \
        frontend-test frontend-lint frontend-build

# ─── Docker Compose ───────────────────────────────────────────────────────────
up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

restart:
	docker compose down && docker compose up -d --build

build:
	docker compose build

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

# ─── All ──────────────────────────────────────────────────────────────────────
test: backend-test frontend-test

lint: backend-lint frontend-lint
