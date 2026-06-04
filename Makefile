SHELL := bash
.DEFAULT_GOAL := help

.PHONY: help test race lint check check-race check-all web-api-types web-lint web-test web-build web-check db-up db-down db-migrate db-rollback deploy-vm fix fix-check

help:
	@echo "Targets:"
	@echo "  test        - run Go tests"
	@echo "  lint        - run golangci-lint"
	@echo "  check       - run Go tests and Go lint"
	@echo "  check-all   - run Go checks and WebUI checks"
	@echo "  check-race  - run Go tests, lint, and race tests"
	@echo "  web-api-types - check generated WebUI API types"
	@echo "  web-lint    - run WebUI lint"
	@echo "  web-test    - run WebUI tests"
	@echo "  web-build   - build WebUI"
	@echo "  web-check   - run WebUI lint, tests, and build"
	@echo "  db-up       - start local PostgreSQL"
	@echo "  db-down     - stop local PostgreSQL"
	@echo "  db-migrate  - run PostgreSQL migrations"
	@echo "  db-rollback - rollback one PostgreSQL migration"
	@echo "  deploy-vm   - manually sync, migrate, and run on VM"

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

db-up:
	@docker compose up -d postgres

db-down:
	@docker compose down

db-migrate:
	@go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$${DATABASE_URL:-postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable}" up

db-rollback:
	@go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$${DATABASE_URL:-postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable}" down

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
