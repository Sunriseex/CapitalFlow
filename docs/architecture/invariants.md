# Financial invariants

These invariants protect CapitalFlow from silent balance corruption and audit gaps.

## Ownership

- Every user-facing financial write must be scoped to `user_id`.
- Repositories must return `not found` when an account belongs to another user.
- HTTP handlers must use the authenticated user from request context.

## Accounts

- Financial writes to archived accounts are rejected.
- Currency is normalized to uppercase before persistence.
- Only supported currencies are accepted in the stable core.
- Account currency cannot change after transactions exist.

## Transactions

- Transaction amount cannot be zero.
- Transaction amount cannot exceed storage bounds.
- Income, expense, transfer, initial balance, and interest income amounts must be positive.
- Negative amounts are allowed only for adjustment transactions.
- Amount scale must match the account currency.
- User-created transaction dates must not be in the future.
- Transaction date must be on or after account opened date.
- Direct creation of transfer transactions is forbidden outside the transfer flow.

## Transfers

- A transfer is one business event and two accounting legs.
- Source and destination accounts must be different.
- Both accounts must belong to the same user.
- Both accounts must be active.
- Both legs are created in one database transaction.
- Transfer rows keep the exchange rate and linked transaction IDs.
- Transfer fees are stored on the transfer audit row and linked to a source-account expense transaction.
- Transfer transaction legs cannot be deleted through the transaction endpoint.
- Database constraints validate that transfer legs match the transfer audit row.

## Idempotency

- Financial POST endpoints require an `Idempotency-Key`.
- A repeated request with the same key and body replays the stored response.
- A repeated key with a different body returns conflict.
- Idempotency state is stored in PostgreSQL.

## Deletion

- User-facing transaction deletion is forbidden.
- Corrections must be modeled as new financial events, not by erasing posted transactions.
- Generated interest recalculation may replace generated accrual transactions only inside one database transaction.
- Transfer foreign keys use restrictive deletes.
- Future destructive financial operations must be soft-delete or audit-backed before they become user-facing.

## Concurrency

- Financial writes lock related account rows in deterministic order.
- Transfer creation checks source balance inside the same transaction that writes both legs.
- Concurrent transfer requests cannot partially apply one leg.
