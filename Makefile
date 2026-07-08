SHELL := bash
.DEFAULT_GOAL := help

MAKEFLAGS += --no-print-directory

DEV_DIR := .dev
DEV_BIN_DIR := $(DEV_DIR)/bin
DEV_LOG_DIR := $(DEV_DIR)/logs
DEV_PID_DIR := $(DEV_DIR)/pids

API_PORT ?= 18080
API_ADDR ?= :$(API_PORT)
WEB_PORT ?= 5173

DATABASE_URL ?= postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable

API_BIN := $(DEV_BIN_DIR)/capitalflow-server
API_PID := $(DEV_PID_DIR)/backend.pid
WEB_PID := $(DEV_PID_DIR)/web.pid

API_LOG := $(DEV_LOG_DIR)/backend.log
WEB_LOG := $(DEV_LOG_DIR)/web.log

WEB_API_PROXY_TARGET := http://127.0.0.1:$(API_PORT)

.PHONY: help test race lint check check-race check-all web-api-types web-lint web-test web-build web-check real-e2e db-up db-down db-migrate db-rollback demo-seed demo-reset run-dev stop-dev reset-dev dev-dirs dev-db-wait dev-backend dev-web dev-status dev-logs deploy-vm fix fix-check

help:
	@echo "Targets:"
	@echo "  test          - run Go tests"
	@echo "  lint          - run golangci-lint"
	@echo "  check         - run Go tests and Go lint"
	@echo "  check-all     - run Go checks and WebUI checks"
	@echo "  check-race    - run Go tests, lint, and race tests"
	@echo "  web-api-types - check generated WebUI API types"
	@echo "  web-lint      - run WebUI lint"
	@echo "  web-test      - run WebUI tests"
	@echo "  web-build     - build WebUI"
	@echo "  web-check     - run WebUI lint, tests, and build"
	@echo "  real-e2e      - run browser tests against a clean PostgreSQL database"
	@echo "  db-up         - start local PostgreSQL"
	@echo "  db-down       - stop local PostgreSQL"
	@echo "  db-migrate    - run PostgreSQL migrations"
	@echo "  db-rollback   - rollback one PostgreSQL migration"
	@echo "  demo-seed    - replace local demo user data (requires DEMO_PASSWORD)"
	@echo "  demo-reset   - remove local demo user data"
	@echo "  run-dev       - start local dev stack: PostgreSQL, migrations, backend, WebUI"
	@echo "  stop-dev      - stop local dev stack"
	@echo "  reset-dev     - wipe local dev database and start dev stack"
	@echo "  dev-status    - show local dev URLs"
	@echo "  dev-logs      - follow backend and WebUI logs"
	@echo "  deploy-vm     - manually sync, migrate, and run on VM"

test:
	@go list ./... | grep -v '/web/' | xargs go test

lint:
	@gofumpt -w .
	@golangci-lint run ./...

race:
	@go list ./... | grep -v '/web/' | xargs go test -race

check-race: lint test race

check: fix lint test

check-all: fix check web-check

web-api-types:
	@cd web && npm run check:api-types

web-lint:
	@cd web && npm run lint

web-test:
	@cd web && npm test

web-build:
	@cd web && npm run build

web-check: web-api-types web-lint web-test web-build

real-e2e:
	@./scripts/real-e2e.sh

db-up:
	@docker compose up -d postgres

db-down:
	@docker compose down

db-migrate:
	@go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$(DATABASE_URL)" up

db-rollback:
	@go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$(DATABASE_URL)" down

demo-seed:
	@test -n "$(DEMO_PASSWORD)" || (echo "DEMO_PASSWORD is required" >&2; exit 1)
	@APP_ENV=development DATABASE_URL="$(DATABASE_URL)" DEMO_PASSWORD="$(DEMO_PASSWORD)" go run ./cmd/seed-demo

demo-reset:
	@APP_ENV=development DATABASE_URL="$(DATABASE_URL)" go run ./cmd/seed-demo --reset

run-dev:
	@$(MAKE) stop-dev
	@$(MAKE) dev-dirs
	@$(MAKE) db-up
	@$(MAKE) dev-db-wait
	@$(MAKE) db-migrate
	@$(MAKE) dev-backend
	@$(MAKE) dev-web
	@$(MAKE) dev-status

stop-dev:
	@echo "Stopping CapitalFlow dev stack..."
	@if [ -f "$(WEB_PID)" ]; then \
		pid=$$(cat "$(WEB_PID)"); \
		if kill -0 "$$pid" 2>/dev/null; then \
			kill "$$pid" 2>/dev/null || true; \
		fi; \
		rm -f "$(WEB_PID)"; \
	fi
	@if [ -f "$(API_PID)" ]; then \
		pid=$$(cat "$(API_PID)"); \
		if kill -0 "$$pid" 2>/dev/null; then \
			kill "$$pid" 2>/dev/null || true; \
		fi; \
		rm -f "$(API_PID)"; \
	fi
	@pkill -TERM -f "[g]o run ./cmd/server --addr :$(API_PORT)" 2>/dev/null || true
	@pkill -TERM -f "[c]apitalflow-server --addr :$(API_PORT)" 2>/dev/null || true
	@pkill -TERM -f "[v]ite.*$(WEB_PORT)" 2>/dev/null || true
	@docker compose down
	@echo "Dev stack stopped."

reset-dev:
	@echo "Resetting CapitalFlow dev stack and PostgreSQL volume..."
	@$(MAKE) stop-dev
	@docker compose down -v --remove-orphans
	@rm -rf "$(DEV_DIR)"
	@$(MAKE) run-dev

dev-dirs:
	@mkdir -p "$(DEV_BIN_DIR)" "$(DEV_LOG_DIR)" "$(DEV_PID_DIR)"

dev-db-wait:
	@echo "Waiting for PostgreSQL..."
	@for i in {1..30}; do \
		if docker compose exec -T postgres pg_isready -U "$${POSTGRES_USER:-capitalflow}" -d "$${POSTGRES_DB:-capitalflow}" >/dev/null 2>&1; then \
			echo "PostgreSQL is ready."; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "PostgreSQL did not become ready."; \
	docker compose logs postgres; \
	exit 1

dev-backend: dev-dirs
	@echo "Building backend..."
	@go build -o "$(API_BIN)" ./cmd/server
	@echo "Starting backend on http://127.0.0.1:$(API_PORT) ..."
	@DATABASE_URL="$(DATABASE_URL)" nohup "$(API_BIN)" --addr "$(API_ADDR)" > "$(API_LOG)" 2>&1 & echo $$! > "$(API_PID)"
	@for i in {1..30}; do \
		if curl -fsS "http://127.0.0.1:$(API_PORT)/health" >/dev/null 2>&1; then \
			echo "Backend is ready."; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Backend failed to start. Last logs:"; \
	tail -n 80 "$(API_LOG)"; \
	exit 1

dev-web: dev-dirs
	@if [ ! -d web/node_modules ]; then \
		echo "Installing WebUI dependencies..."; \
		cd web && npm install; \
	fi
	@echo "Starting WebUI on http://127.0.0.1:$(WEB_PORT) ..."
	@cd web; VITE_API_PROXY_TARGET="$(WEB_API_PROXY_TARGET)" nohup npm run dev -- --port "$(WEB_PORT)" --strictPort > "../$(WEB_LOG)" 2>&1 & echo $$! > "../$(WEB_PID)"
	@for i in {1..30}; do \
		if curl -fsS "http://127.0.0.1:$(WEB_PORT)" >/dev/null 2>&1; then \
			echo "WebUI is ready."; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "WebUI failed to start. Last logs:"; \
	tail -n 80 "$(WEB_LOG)"; \
	exit 1

dev-status:
	@echo ""
	@echo "CapitalFlow dev stack:"
	@echo "  WebUI:   http://127.0.0.1:$(WEB_PORT)"
	@echo "  API:     http://127.0.0.1:$(API_PORT)"
	@echo "  Health:  http://127.0.0.1:$(API_PORT)/health"
	@echo "  Ready:   http://127.0.0.1:$(API_PORT)/ready"
	@echo ""
	@echo "Useful commands:"
	@echo "  make dev-logs"
	@echo "  make stop-dev"
	@echo "  make reset-dev"

dev-logs:
	@tail -f "$(API_LOG)" "$(WEB_LOG)"

deploy-vm:
	@./scripts/deploy-vm.sh

fix-check:
	@echo "Checking if 'go fix' suggests changes..."
	@output=$$(go fix -diff ./... 2>&1); \
	if [ -n "$$output" ]; then \
		echo "❌ 'go fix' suggests changes:"; \
		echo "$$output"; \
		echo "Run 'make fix' locally and commit the changes."; \
		exit 1; \
	else \
		echo "✅ 'go fix' found no suggestions."; \
	fi

fix:
	@echo "Applying 'go fix'..."
	@go fix ./...
	@echo "Done. Please review and commit changes."
