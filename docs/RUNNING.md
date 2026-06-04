# Running CapitalFlow locally

This guide describes the expected local development startup path.

## Requirements

Install:

* Go 1.26.2 or newer.
* PostgreSQL 17 or compatible.
* Node.js and npm.
* Docker and Docker Compose, if you want PostgreSQL from compose.
* `make`, optional but recommended.

## 1. Clone repository

```bash
git clone https://github.com/Sunriseex/CapitalFlow.git
cd CapitalFlow
```

## 2. Create local environment file

```bash
cp configs/example.env configs/.env
```

Then edit `configs/.env`.

Minimal local values:

```env
APP_VERSION=v0.5.8
LOG_LEVEL=debug
APP_ENV=development

DATABASE_URL=postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable
API_AUTH_TOKEN=change-me-to-a-long-random-token
JWT_SECRET=change-me-to-a-long-random-secret

ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
PUBLIC_ORIGIN=
WEBAUTHN_RP_DISPLAY_NAME=CapitalFlow
WEBAUTHN_RP_ID=localhost
WEBAUTHN_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
COOKIE_SECURE=true
COOKIE_SAMESITE=Strict
ALLOW_DIRECT_IP_LOGIN=true

CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
RATE_LIMIT_REQUESTS=120
RATE_LIMIT_WINDOW=1m
AUTH_RATE_LIMIT_REQUESTS=5
AUTH_RATE_LIMIT_WINDOW=1m
PASSKEY_OPTIONS_RATE_LIMIT_REQUESTS=3
PASSKEY_OPTIONS_RATE_LIMIT_WINDOW=1m
MUTATION_RATE_LIMIT_REQUESTS=60
MUTATION_RATE_LIMIT_WINDOW=1m

DATA_PATH=~/.config/capitalflow/payments.json
DEPOSITS_DATA_PATH=~/.config/capitalflow/deposits.json

TELEGRAM_BOT_TOKEN=
TELEGRAM_USER_ID=0
```

Generate secrets:

```bash
openssl rand -hex 32
openssl rand -hex 64
```

Use the first value for `API_AUTH_TOKEN` and the second value for `JWT_SECRET`.

Do not commit `configs/.env`.

For self-hosted production behind a reverse proxy, use a real origin and disable direct IP login:

```env
APP_ENV=production
PUBLIC_ORIGIN=https://capitalflow.home.arpa
WEBAUTHN_RP_ID=capitalflow.home.arpa
WEBAUTHN_ORIGINS=https://capitalflow.home.arpa
COOKIE_SECURE=true
COOKIE_SAMESITE=Strict
ALLOW_DIRECT_IP_LOGIN=false
TRUSTED_PROXIES=127.0.0.1/32,172.16.0.0/12
CORS_ALLOWED_ORIGINS=https://capitalflow.home.arpa
```

See `docs/security/reverse-proxy.md` and `docs/security/csrf.md`.

## 3. Start PostgreSQL

### Option A: Docker Compose

If the repository compose file is configured for local PostgreSQL, run:

```bash
docker compose up -d postgres
```

Check that PostgreSQL is reachable:

```bash
docker compose ps
```

### Option B: Existing local PostgreSQL

Create user and database manually:

```sql
CREATE USER capitalflow WITH PASSWORD 'capitalflow';
CREATE DATABASE capitalflow OWNER capitalflow;
```

## 4. Apply migrations

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

## 5. Start API server

Recommended local port for WebUI proxy compatibility:

```bash
go run ./cmd/server --addr :18080
```

The Vite dev server proxies `/api/v1` and `/auth` to `http://127.0.0.1:18080` by default.

Health checks:

```bash
curl http://127.0.0.1:18080/health
curl http://127.0.0.1:18080/ready
```

If you want another API port, set `VITE_API_PROXY_TARGET` before running the WebUI.

Example:

```bash
VITE_API_PROXY_TARGET=http://127.0.0.1:8080 npm run dev
```

## 6. Start WebUI

Open a second terminal:

```bash
cd web
npm install
npm run dev
```

Open:

```text
http://127.0.0.1:5173
```

On first launch, create the first user through setup. After setup, use login.

## 7. Auth notes for local development

The current auth flow uses:

* access token in API responses;
* refresh token in `__Secure-capitalflow_refresh` cookie;
* refresh and logout through `/auth/*`;
* protected app API under `/api/v1/*`.

The refresh cookie is `Secure`, `HttpOnly`, `SameSite=Strict`, and scoped to `/auth`.

If browser refresh does not persist locally, check:

* frontend is opened through `http://127.0.0.1:5173` or `http://localhost:5173`;
* backend is reachable through the Vite proxy;
* `CORS_ALLOWED_ORIGINS` contains the frontend origin;
* browser devtools shows the refresh cookie after setup/login;
* local HTTPS/reverse proxy is used if your browser refuses secure cookies on plain HTTP.

## 8. Run checks

Backend:

```bash
go test ./...
go test -race ./...
go build ./cmd/...
```

With make:

```bash
make check
make check-race
```

WebUI:

```bash
cd web
npm run lint
npm run build
npm test
```

## 9. Common problems

### `JWT_SECRET is required`

Add `JWT_SECRET` to `configs/.env`.

### `connect: connection refused` for PostgreSQL

PostgreSQL is not running or `DATABASE_URL` points to the wrong host/port.

### WebUI calls wrong backend port

Default proxy target is `http://127.0.0.1:18080`.

Either start API with:

```bash
go run ./cmd/server --addr :18080
```

or override the proxy target:

```bash
cd web
VITE_API_PROXY_TARGET=http://127.0.0.1:8080 npm run dev
```

### Setup/login works but session disappears after reload

Check that the refresh cookie is set. If it is missing, inspect cookie rules, frontend origin, proxy target and HTTPS behavior.

### Migrations fail because tables already exist

Use a fresh local database for development, or inspect migration status before applying migrations again.

## 10. Local development order

Use this order for daily work:

```bash
# terminal 1
# start db first
docker compose up -d postgres

# terminal 2
# apply migrations when schema changed
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$DATABASE_URL" up

# terminal 3
# start backend
go run ./cmd/server --addr :18080

# terminal 4
# start frontend
cd web && npm run dev
```

Then open `http://127.0.0.1:5173`.
