# Idempotency

CapitalFlow treats idempotency as part of financial correctness. Retryable financial mutations must not create duplicate transactions, transfers, or generated interest records.

## Scope

The HTTP API requires `Idempotency-Key` for these endpoints:

- `POST /api/v1/transactions`
- `POST /api/v1/transfers`
- `POST /api/v1/accounts/{id}/accrue-interest`
- `POST /api/v1/accounts/{id}/recalculate-interest`

Other mutations may also pass through the idempotency middleware, but financial endpoints must reject requests without a valid key.

## Key Rules

- Keys are scoped by `user_id`, HTTP method, and path.
- Keys are trimmed at the HTTP boundary.
- Blank keys are rejected.
- Keys longer than 255 bytes are rejected.
- A key may be reused only with the same request body for the same endpoint.
- A key reused with a different body returns `409 idempotency_key_reused`.
- A retry while the first request is still pending returns `409 idempotency_in_progress`.
- A completed retry returns the stored status code and response body.
- Records expire after 24 hours.

## Request Hash

The middleware hashes the raw request body with SHA-256 before the handler runs. The hash is stored with the idempotency record and compared on retry.

The current scope intentionally excludes headers and query parameters from the hash because the protected financial write endpoints use JSON bodies for mutation input. If a future write endpoint accepts meaningful query parameters, include them in the idempotency fingerprint before enabling retries.

## Storage Model

Idempotency state is stored in PostgreSQL in `idempotency_keys`.

The primary key is:

```text
key, user_id, method, path
```

This prevents one user's retry key from affecting another user and allows the same client-generated key to be used on different endpoints without collision.

## Handler Contract

Handlers must be safe to run once after the middleware creates a pending idempotency record.

For successful first execution:

1. Middleware creates a pending record.
2. Handler performs the financial mutation.
3. Middleware stores the response status and body.
4. Middleware flushes the captured response to the client.

If storing the completed response fails after a successful mutation, the API returns `409 idempotency_completion_unknown`. The client must retry later with the same key and must not retry with a new key.

## Service Contract

Service code must still enforce financial invariants. Idempotency middleware is not a replacement for:

- account ownership checks
- positive amount validation
- currency validation
- transaction boundaries
- transfer atomicity
- duplicate generated-accrual protection

For transfers, the transfer business event also stores its own idempotency key. That protects the transfer audit model even if code is later called outside HTTP.

## Tests

Required coverage:

- missing key is rejected for financial mutation endpoints
- blank key is rejected
- overlong key is rejected
- same key and same body replays the stored response
- same key and different body returns conflict
- in-progress request returns conflict
- transfer creation remains atomic under retry
- generated interest accruals are not duplicated by retry

