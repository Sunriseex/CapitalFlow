# Architecture layers

CapitalFlow keeps financial logic behind explicit layers. The goal is to make money write-flows reviewable, testable, and hard to bypass from HTTP, CLI, jobs, or future integrations.

## Layers

```text
internal/models       data structs
internal/domain       pure rules and invariants
internal/services     use cases
internal/repository   storage contracts
internal/postgres     PostgreSQL implementation
internal/http         HTTP transport, DTOs, middleware
```

## Responsibilities

### `internal/models`

Models are data containers. They should not open database connections, parse HTTP requests, call services, or make policy decisions.

### `internal/domain`

Domain packages contain pure validation and invariant rules:

- `account`: account type and currency rules.
- `transaction`: transaction amount, type, precision, and date rules.
- `transfer`: transfer request invariants.

Domain code should not know about HTTP DTOs, SQL rows, repositories, or framework types.

### `internal/services`

Services describe application use cases. A service may call domain validators, build model objects, apply defaults, and call repositories. Services must not accept HTTP DTOs.

Examples:

- `AccountService.Create`
- `TransactionService.CreateForUser`
- `TransferService.Create`
- `InterestRuleService.Accrue`

### `internal/repository`

Repository interfaces describe persistence operations in model terms. They must not accept HTTP DTOs.

### `internal/postgres`

PostgreSQL repositories enforce storage-level invariants that must survive concurrent requests:

- owner scoping with `user_id`
- row locks for financial writes
- inactive account rejection
- account opened date checks
- transfer atomicity
- transfer leg integrity
- idempotency persistence

### `internal/http`

Handlers are transport adapters. They decode JSON, validate route/query shapes, load authenticated user context, call services, and encode responses. They should not be the only place where financial rules are enforced.

## New Feature Template

Use this order for new financial features:

1. Model: add or extend `internal/models`.
2. Domain rule: add pure validation in `internal/domain`.
3. Service: add the use case in `internal/services`.
4. Repository contract: add storage interface methods in `internal/repository`.
5. PostgreSQL implementation: implement atomic writes in `internal/postgres`.
6. Handler: expose the use case in `internal/http`.
7. Tests: add domain, service, handler contract, and PostgreSQL integration coverage.
