# Finance Tracker

Imported from `finance-manager`.

This will become the new finance tracker.
Target features include a web UI and daily-capitalization deposits.

## PostgreSQL

Start local PostgreSQL:

```sh
docker compose up -d postgres
```

If local port `5432` is busy, use another host port:

```sh
POSTGRES_PORT=55432 docker compose up -d postgres
```

PowerShell:

```powershell
$env:POSTGRES_PORT = "55432"; docker compose up -d postgres
```

Run migrations:

```sh
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "${DATABASE_URL:-postgres://finance_tracker:finance_tracker@localhost:5432/finance_tracker?sslmode=disable}" up
```

PowerShell:

```powershell
$env:DATABASE_URL = "postgres://finance_tracker:finance_tracker@localhost:55432/finance_tracker?sslmode=disable"
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres $env:DATABASE_URL up
```

Migrate legacy deposits JSON:

```sh
go run ./cmd/finance-manager migrate-json --deposits ./deposits.json --database-url "${DATABASE_URL:-postgres://finance_tracker:finance_tracker@localhost:5432/finance_tracker?sslmode=disable}"
```

PowerShell:

```powershell
go run .\cmd\finance-manager migrate-json --deposits .\deposits.json --database-url $env:DATABASE_URL
```

Use PostgreSQL as the working storage:

```powershell
go run .\cmd\finance-manager doctor --database-url $env:DATABASE_URL
go run .\cmd\finance-manager accounts create --name "Main Savings" --type savings --bank "Bank" --currency RUB --database-url $env:DATABASE_URL
go run .\cmd\finance-manager accounts list --database-url $env:DATABASE_URL
go run .\cmd\finance-manager transactions create --account <account-id> --type initial_balance --amount 1000.00 --database-url $env:DATABASE_URL
go run .\cmd\finance-manager balance --account <account-id> --database-url $env:DATABASE_URL
```

## HTTP API

Start the API:

```powershell
go run .\cmd\server --addr 127.0.0.1:8080 --database-url $env:DATABASE_URL
```

Core endpoints:

```text
GET    /health
GET    /ready
GET    /api/accounts
POST   /api/accounts
GET    /api/accounts/{id}
PATCH  /api/accounts/{id}
POST   /api/accounts/{id}/archive
GET    /api/accounts/{id}/balance
GET    /api/transactions
POST   /api/transactions
GET    /api/transactions/{id}
DELETE /api/transactions/{id}
POST   /api/transfers
GET    /api/accounts/{id}/interest-rules
POST   /api/accounts/{id}/interest-rules
PATCH  /api/interest-rules/{id}
POST   /api/accounts/{id}/accrue-interest
```
