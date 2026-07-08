# CapitalFlow

CapitalFlow is a self-hosted personal finance tracker built with Go, PostgreSQL and React.

The project is intended to become a private financial center for tracking accounts, transactions, transfers, savings accounts, deposits, interest rules and future investment/multi-currency data. It is also a practical backend learning project focused on production-oriented habits: layered architecture, PostgreSQL migrations, financial correctness, secure auth, tests, CI and self-host deployment.

## Product direction

CapitalFlow should help answer:

* How much money do I have now?
* Where is the money stored: cash, cards, savings, deposits, broker accounts or other accounts?
* What changed recently?
* How much did I earn or spend this month?
* How much interest did my savings/deposits generate?
* Can I audit how every balance was produced?

The project prioritizes explainable balances: balances are derived from transactions, interest accruals are stored as operations, and transfers should be represented as auditable business events.

## Features

Implemented or mostly implemented:

* Account management: cash, card, savings, term deposit, broker and other account types.
* Transaction tracking: income, expense, transfer, initial balance, adjustment and interest income transactions.
* Transfers between accounts.
* Account balance calculation from transactions.
* Interest rules for savings and deposits.
* Manual interest accrual with duplicate-accrual protection.
* PostgreSQL persistence.
* Legacy JSON data migration.
* HTTP API with validation and unified error responses.
* Auth setup/login/refresh/logout flow.
* Refresh-token rotation and session revocation.
* React + Vite + TypeScript WebUI.
* Health and readiness endpoints.
* CI with backend and WebUI checks.

Planned before v1.0:

* Stronger financial auditability for transfers.
* Idempotency for financial mutations.
* E2E tests for critical user flows.
* Optional passkey/WebAuthn login.
* Backup/restore.
* Production Docker/self-host documentation.

Post-v1.0:

* Budgets and goals.
* Multi-currency rates.
* Investments and portfolio tracking.
* Telegram bot.
* Local LLM/Ollama assistant based on safe aggregated summaries.

## Tech Stack

* Go
* PostgreSQL
* Goose migrations
* Chi router
* React + Vite + TypeScript
* TanStack Query
* Recharts
* golangci-lint
* GitHub Actions

## Project Structure

```text
.
├── cmd/
│   └── server/              # HTTP API entrypoint
├── configs/
│   └── example.env          # Example local configuration
├── docs/
│   ├── RUNNING.md           # Local startup guide
│   ├── openapi.yaml         # OpenAPI contract
│   └── wiki/                # Canonical GitHub Wiki source
├── internal/
│   ├── auth/                # Auth token helpers
│   ├── config/              # Application configuration
│   ├── http/
│   │   ├── dto/             # HTTP request/response DTOs
│   │   ├── handlers/        # HTTP handlers and routing
│   │   └── middleware/      # HTTP middleware
│   ├── jobs/                # Background job logic
│   ├── legacyjson/          # Read-only legacy import adapter
│   ├── models/              # Domain models
│   ├── postgres/            # PostgreSQL repositories/store
│   ├── repository/          # Repository interfaces/contracts
│   └── services/            # Business logic
├── migrations/              # PostgreSQL migrations
├── web/                     # React WebUI
└── .github/workflows/       # CI configuration
```

## Documentation

Start here:

* [Running locally](docs/RUNNING.md)
* [Project roadmap](TODO.md)
* [API contract](docs/openapi.yaml)
* [Auth Security Model](docs/wiki/Auth-Security-Model.md)
* [Operations Runbook](docs/wiki/Operations-Runbook.md)
* [Auth Incident Response](docs/wiki/Auth-Incident-Response.md)
* [Auth Security ADR](docs/wiki/ADR-0001-Auth-Security-Hardening.md)

The documentation portal source lives in [docs/wiki/Home.md](docs/wiki/Home.md). Mirror these pages to the GitHub Wiki when publishing user-facing docs.

## Requirements

* Go 1.26.2 or newer, based on the project `go.mod`.
* PostgreSQL 17 or compatible.
* Node.js and npm for the WebUI.
* Docker and Docker Compose, optional but recommended for local PostgreSQL.
* `goose` for running migrations.
* `golangci-lint` for local linting.

## Quick start

Clone the project:

```bash
git clone https://github.com/Sunriseex/CapitalFlow.git
cd CapitalFlow
```

Create local config:

```bash
cp configs/example.env configs/.env
```

Edit `configs/.env` and set at minimum:

```env
DATABASE_URL=postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable
API_AUTH_TOKEN=<long-random-token>
JWT_SECRET=<long-random-secret>
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
```

Generate local secrets:

```bash
openssl rand -hex 32
openssl rand -hex 64
```

Start PostgreSQL, apply migrations, then start the API:

```bash
# PostgreSQL, if using docker compose
docker compose up -d postgres

# migrations
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 \
  -dir migrations \
  postgres "postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable" \
  up

# API; recommended local port for WebUI proxy
go run ./cmd/server --addr :18080
```

Start WebUI in a second terminal:

```bash
cd web
npm install
npm run dev
```

Open:

```text
http://127.0.0.1:5173
```

The Vite dev server proxies `/api/v1` and `/auth` to `http://127.0.0.1:18080` by default. Full startup notes are in [docs/RUNNING.md](docs/RUNNING.md).

## Configuration

The application reads environment variables from:

```text
configs/.env
```

You can override the env file path:

```bash
CAPITALFLOW_ENV_FILE=/path/to/.env go run ./cmd/server --addr :18080
```

Important variables:

```env
APP_VERSION=v0.5.8
LOG_LEVEL=debug

DATABASE_URL=postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable
JWT_SECRET=<generated-secret>
API_AUTH_TOKEN=<generated-token>

ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h

CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
RATE_LIMIT_REQUESTS=120
RATE_LIMIT_WINDOW=1m
AUTH_RATE_LIMIT_REQUESTS=5
AUTH_RATE_LIMIT_WINDOW=1m
MUTATION_RATE_LIMIT_REQUESTS=60
MUTATION_RATE_LIMIT_WINDOW=1m

```

`JWT_SECRET` is required by the HTTP server and must be at least 32 characters. `API_AUTH_TOKEN` is used by the bearer-token fallback mode and must also be at least 32 characters when that mode is enabled.
Do not commit real secrets. Keep local secrets in `configs/.env` and commit only `configs/example.env`.

Generate a strong token on Linux/macOS:

```bash
openssl rand -hex 32
```

Generate a strong token on Windows PowerShell:

```powershell
[Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).ToLower()
```

Put generated values into `configs/.env`:

```env
JWT_SECRET=<generated-secret>
API_AUTH_TOKEN=<generated-token>
```

## Database Setup

Create a local PostgreSQL database and user manually if you are not using Docker Compose:

```sql
CREATE USER capitalflow WITH PASSWORD 'capitalflow';
CREATE DATABASE capitalflow OWNER capitalflow;
```

Run migrations:

```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 \
  -dir migrations \
  postgres "postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable" \
  up
```

Check migration status:

```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 \
  -dir migrations \
  postgres "postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable" \
  status
```

### One-time legacy deposit import

Legacy JSON is accepted only as read-only migration input. It is never used as
a runtime ledger and the old deposit/payment CLI tools are not supported.

```bash
go run ./cmd/capitalflow migrate-json \
  --deposits ~/.config/waybar/deposits.json \
  --database-url "$DATABASE_URL"
```

The import is idempotent by legacy deposit ID. After a successful import,
PostgreSQL is the only source of truth.

## Running the API

Before starting the API, make sure `configs/.env` exists and contains `DATABASE_URL` and a strong `JWT_SECRET`. If you run the bearer-token fallback mode, also set a strong `API_AUTH_TOKEN`.

Recommended local command:

```bash
go run ./cmd/server --addr :18080
```

Alternative port:

```bash
go run ./cmd/server --addr :8080
```

Public endpoints:

```text
GET /health
GET /ready
GET /auth/status
```

Protected API routes use the current auth flow. Login/setup returns an access token and sets a refresh cookie.

Example health check:

```bash
curl http://127.0.0.1:18080/health
curl http://127.0.0.1:18080/ready
```

## Running the WebUI

Start the API on `:18080` first, then run:

```bash
cd web
npm install
npm run dev
```

Open:

```text
http://127.0.0.1:5173
```

If the API is running on a different port:

```bash
cd web
VITE_API_PROXY_TARGET=http://127.0.0.1:8080 npm run dev
```

## API Overview

Auth:

```text
GET  /auth/status
POST /auth/setup
POST /auth/login
POST /auth/refresh
POST /auth/logout
```

Accounts:

```text
GET    /api/v1/accounts
POST   /api/v1/accounts
GET    /api/v1/accounts/{id}
PATCH  /api/v1/accounts/{id}
POST   /api/v1/accounts/{id}/archive
GET    /api/v1/accounts/{id}/balance
```

Categories:

```text
GET /api/v1/categories
```

Transactions:

```text
GET    /api/v1/transactions
POST   /api/v1/transactions
GET    /api/v1/transactions/{id}
```

Transactions cannot be hard-deleted. Correction and reversal semantics are
planned so financial history remains auditable.

Transfers:

```text
POST /api/v1/transfers
```

Interest rules and accruals:

```text
GET   /api/v1/accounts/{id}/interest-rules
POST  /api/v1/accounts/{id}/interest-rules
PATCH /api/v1/interest-rules/{id}
POST  /api/v1/accounts/{id}/accrue-interest
POST  /api/v1/accounts/{id}/recalculate-interest
```

Dashboard:

```text
GET /api/v1/dashboard/summary
GET /api/v1/dashboard/net-worth
GET /api/v1/dashboard/cashflow
GET /api/v1/dashboard/interest-income
```

## Interest Rules

Interest rules define how interest should be accrued for an account.

Important behavior:

* The rule must belong to the target account.
* When no `rule_id` is provided for manual accrual, the API selects the latest active rule that applies to the requested accrual date.
* Balance used for accrual is calculated only from transactions with `occurred_at` on or before the accrual date.
* Duplicate accruals for the same account, rule and date are skipped.
* Promo rate and promo end date must be set together.
* Existing promo settings can be cleared with `null` or an empty promo end date.

## Validation Notes

The HTTP API validates user input before writing to PostgreSQL:

* Currency must be exactly three uppercase Latin letters, for example `RUB`, `USD` or `EUR`.
* Unknown JSON fields are rejected.
* Trailing JSON data is rejected.
* Invalid enum values are rejected.
* Invalid interest rule date ranges are rejected.
* Missing resources return `404` instead of silently returning empty data.

## Development

Download dependencies:

```bash
go mod download
```

Run backend checks:

```bash
make check
make check-race
```

Or manually:

```bash
go test ./...
go test -race ./...
go build ./cmd/...
```

Run WebUI checks:

```bash
cd web
npm run lint
npm run build
npm test
```

Check formatting:

```bash
gofmt -l $(git ls-files '*.go')
```

## CI

The GitHub Actions CI pipeline runs backend and WebUI checks, including tests, race tests, linting, builds, migration checks, `go mod tidy` verification and frontend build checks.

Pushes to `master` run checks only. Production images and GitHub Releases are created only from release tags that point to commits already on `master`:

```bash
git checkout master
git pull
git tag -a v0.5.8 -m "v0.5.8"
git push origin v0.5.8
```

If a `v*` tag points to a commit that is not on `master`, the release guard fails before images are published. A successful release tag builds and pushes API and Web images to GHCR with these tags:

```text
ghcr.io/<owner>/capitalflow-api:<tag>
ghcr.io/<owner>/capitalflow-web:<tag>
ghcr.io/<owner>/capitalflow-api:sha-<commit>
ghcr.io/<owner>/capitalflow-web:sha-<commit>
```

Use the release tag, such as `v0.5.8`, for normal deploys. Use the immutable `sha-<commit>` tags for pinned rollback/debug deploys.

Release Telegram notifications use `appleboy/telegram-action`. Add these repository secrets in GitHub:

```text
TELEGRAM_TOKEN=<BotFather token>
TELEGRAM_TO=<chat id>
```

To get `TELEGRAM_TO`, send any message to the bot and open:

```text
https://api.telegram.org/bot<TELEGRAM_TOKEN>/getUpdates
```

Use `message.chat.id` from the response. Notifications are sent only for `v*` release tag workflows, including failed release guards or failed image builds.

Deploy is intentionally manual and instance-owned. From a checkout that can reach the target VM, run:

```bash
DEPLOY_MODE=images \
CAPITALFLOW_API_IMAGE=ghcr.io/sunriseex/capitalflow-api:v0.5.8 \
CAPITALFLOW_WEB_IMAGE=ghcr.io/sunriseex/capitalflow-web:v0.5.8 \
./scripts/deploy-vm.sh
```

When running the script directly on the target VM, add `VM_HOST=local`.

Optional deploy settings:

```text
VM_HOST=VM
REMOTE_DIR=/home/sunriseex/projects/CapitalFlow
PUBLIC_ORIGIN=https://capitalflow.home.arpa
CAPITALFLOW_PROXY_NETWORK=proxy
CAPITALFLOW_INTEREST_JOBS_ENABLED=true
CAPITALFLOW_INTEREST_JOBS_TIME=03:15
CAPITALFLOW_INTEREST_JOB_TIMEOUT=30m
TZ=Europe/Moscow
```

The VM deploy runs interest jobs inside Docker Compose, not through NixOS timers. When
`CAPITALFLOW_INTEREST_JOBS_ENABLED=true`, the `interest-scheduler` container runs once
per day at `CAPITALFLOW_INTEREST_JOBS_TIME` and starts:

```text
daily_interest_accrual_job
monthly_interest_accrual_job
deposit_maturity_check_job
```

Manual VM run:

```bash
cd /home/sunriseex/projects/CapitalFlow/deploy
docker compose --profile tools run -T --rm job-runner jobs run --name daily_interest_accrual_job
```

Each job takes a PostgreSQL advisory lock by job name. A second concurrent run exits
successfully with `already running`. Daily, monthly and end-of-term jobs only load
rules with their matching `accrual_frequency`.

## Security

Auth responses return access-token metadata and set a `__Secure-capitalflow_refresh` cookie for refresh-token rotation. The cookie is scoped to `/auth` and uses `Secure`, `HttpOnly`, and `SameSite=Strict`. `/auth/refresh` and `/auth/logout` use the refresh cookie.

Protected routes:

```text
/api/v1/*
```

Public routes:

```text
GET /health
GET /ready
GET /auth/status
POST /auth/setup
POST /auth/login
POST /auth/refresh
POST /auth/logout
```

Do not commit real secrets. Keep local secrets in `configs/.env` and commit only `configs/example.env`.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).
