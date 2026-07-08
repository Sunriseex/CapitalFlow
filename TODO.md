# CapitalFlow TODO.md

Status: code-verified TODO / backlog / senior-level implementation plan
Repository checked: `Sunriseex/CapitalFlow`
Repository snapshot observed in GitHub search results: `301eeeb566bfe97f07a892fda684132ae826d2a6`

This file was rebuilt from:

- the new planning TODO created outside the repository;
- current repository `TODO.md`;
- actual code, migrations, routes, services, CI and deployment files.

Important rule used during verification:

```text
A checkbox is marked [x] only when the implementation is visible in code, migrations, tests, CI or deploy files.
Repository TODO checkboxes were not trusted by themselves.
```

Legend:

```text
[x] Done by code.
[ ] Not done.
[ ] PARTIAL — some code exists, but the requirement is not fully closed.
[ ] VERIFY — likely exists or docs say it exists, but code-level confirmation is incomplete.
```

---

## 0. Verification Summary

### Verified done by code

- [x] Core accounts model exists.
- [x] Core categories model exists.
- [x] Core transactions model exists.
- [x] Account balances are calculated from transactions.
- [x] PostgreSQL migrations exist and are checked in CI.
- [x] Money columns were migrated from `amount_minor` to `NUMERIC(38,18)`.
- [x] Go money math uses `shopspring/decimal` in the current transaction/domain paths.
- [x] Transfers have a first-class `transfers` table linked to transaction legs.
- [x] Cross-currency transfer stores applied exchange rate, provider and timestamp.
- [x] Transfer fee is represented separately through `fee_transaction_id` and `fee_amount`.
- [x] Transfer integrity is protected by DB constraints/triggers.
- [x] Idempotency table exists.
- [x] Idempotency middleware exists.
- [x] `POST /api/v1/transactions` requires `Idempotency-Key`.
- [x] `POST /api/v1/transfers` requires `Idempotency-Key`.
- [x] Manual interest accrual/recalculation endpoints require `Idempotency-Key`.
- [x] Setup/login/refresh/logout auth flow exists.
- [x] Refresh tokens are stored hashed.
- [x] Refresh-token rotation/session revocation exists.
- [x] Auth audit events table exists.
- [x] Login lockout fields exist.
- [x] Passkey/WebAuthn tables exist.
- [x] Passkey routes are wired.
- [x] Security middleware exists: rate limit, CORS, host policy, security headers.
- [x] `/health`, `/ready`, `/metrics` routes exist.
- [x] Basic dashboard endpoints exist.
- [x] Basic dashboard UI exists.
- [x] Playwright E2E covers both mocked UI smoke and a real backend + PostgreSQL financial flow.
- [x] CI runs backend tests, race tests, lint, WebUI checks, OpenAPI lint and migration checks.
- [x] Production Docker images are built in CI on release tags.
- [x] Deployment compose exists under `deploy/compose.yaml`.
- [x] VM deploy script exists.
- [x] Interest background jobs exist.
- [x] Interest scheduler script exists.
- [x] Interest job advisory lock exists.

### Not done / not closed by code

- [ ] Subscriptions as first-class entities.
- [ ] Subscription creation/recognition form from `Subscriptions` category.
- [ ] Subscription priority enum.
- [ ] Subscription reminders and missed charge warnings.
- [ ] CSV import pipeline.
- [ ] Import preview/review queue.
- [ ] `Accept all safe` import confirmation.
- [ ] Parser version storage.
- [ ] Manual CSV mapping templates.
- [x] Backup/restore CLI, scheduled backups, retention and pre-migration backups.
- [ ] Encrypted integration secrets backup/restore behavior.
- [ ] Goals/reserved money/emergency fund system goal.
- [ ] Budgeting/safe-to-spend.
- [ ] Money calendar.
- [ ] App-level Telegram integration.
- [ ] Local LLM assistant.
- [ ] User-defined automation rules.
- [ ] Reconciliation.
- [ ] Investments/positions.
- [ ] Server-console password reset recovery.

### Biggest current risks

1. **Financial audit log is incomplete.** Auth audit exists, but financial/settings-wide audit events are not clearly implemented.
2. **Transfer model is strong, but lifecycle is narrow.** DB status currently allows only `completed`; no pending/cancelled/reversed states.
3. **Interest engine exists, but deposit product model is still incomplete.** Rules/accruals/jobs exist; fixed deposits, top-up cutoff, expected-vs-actual interest and imported actual interest matching are not done.
4. **Currency rates are only latest external display rates.** No CBR/manual provider, no hourly sync, no persisted rate history, no historical report rates.
5. **Backup archives are not encrypted.** Financial restore is tested, but future integration-secret recovery still needs an explicit key-loss model.
6. **NixOS service/timer examples are missing.** Production Docker deployment and its backup scheduler are implemented.

---

## Roadmap order

- [ ] v0.5.9 — Frontend Reference Refactor.
- [ ] v0.6.0 — Financial Correctness Foundation.
- [ ] v0.6.1 — Security / Auth / Recovery.
- [x] v0.6.2 — Backup / Restore core (UI and encryption remain follow-ups).
- [ ] v0.6.3 — Subscriptions.
- [ ] v0.6.4 — Deposits / Savings / Interest Engine.
- [ ] v0.6.5 — Goals / Reserves / Emergency Fund.
- [ ] v0.6.6 — CSV Import.
- [ ] v0.6.7 — Review Queue.
- [ ] v0.6.8 — Currencies / Rates.
- [ ] v0.6.9 — Budgeting / Money Calendar.
- [ ] v0.7.0 — Telegram Integration.
- [ ] v0.7.1 — Automation Rules.
- [ ] v0.7.2 — Reconciliation.
- [ ] v0.8.0 — Investments.
- [ ] v0.8.1 — Local LLM Assistant.
- [ ] v0.8.2 — Mobile / PWA.
- [ ] v0.8.3 — Frontend UX Direction.
- [ ] v0.9.0 — Testing / CI Hardening.
- [ ] v0.9.1 — Deployment / Operations.
- [ ] v1.0.0 — Real-money-ready core release.

Reason: the latest repository tag is `v0.5.8`; the next planned release must remain `v0.5.9`, then the verified backlog continues as versioned implementation slices.

# v0.5.9 — Frontend Reference Refactor

## Goal

Привести WebUI к выбранным `.lazyweb` references до расширения E2E и новых фич, чтобы тесты закрепляли уже целевой UX, а не временный интерфейс.

## Scope

* [ ] Login screen по `.lazyweb/quick-references/auth-finance-login-2026-05-10`.
* [ ] Initial setup screen по `.lazyweb/quick-references/auth-finance-login-2026-05-10`.
* [ ] Dashboard по `.lazyweb/quick-references/finance-dashboard-2026-05-10`.
* [ ] Any new pages use the same finance app layout language.
* [ ] Keep existing auth, passkey, dashboard, account, transaction and settings flows working.
* [ ] Avoid large product scope changes during the refactor.

## Acceptance criteria

* [ ] Login and setup are visually separate states, not a cramped mode toggle.
* [ ] Dashboard leads with balance, useful actions, account overview and recent activity.
* [ ] Mobile layout stays usable.
* [ ] Dark theme remains supported.
* [ ] Existing frontend tests pass and are updated for new layout.
* [ ] New UI states have focused tests where behavior changes.

---

# v0.6.0 — Financial Correctness Foundation

## 1. Ledger / Transactions / Balances

### Verified current state

- [x] `transactions` table exists.
- [x] Transaction types exist: `initial_balance`, `income`, `expense`, `transfer_in`, `transfer_out`, `interest_income`, `adjustment`.
- [x] Transactions use `decimal.Decimal` in Go model.
- [x] PostgreSQL money columns were migrated to `NUMERIC(38,18)`.
- [x] Balance service calculates balance from transaction deltas.
- [x] Repository account balance query uses transaction SUM.
- [x] Dashboard balance calculation also derives from transactions.
- [x] Account locking exists for write flows.
- [x] Inactive account writes are guarded in repository flow.
- [x] Transaction-before-account-open is guarded.
- [x] Transaction creation validates account and category existence through service/repository boundaries.
- [x] Transactions persist a validated source type, optional source reference and JSONB metadata.
- [x] Manual, transfer and deposit-interest flows assign their source automatically.

### Still TODO

- [ ] Add transaction lifecycle/status:
  - `pending`;
  - `confirmed`;
  - `cancelled`;
  - `reversed`;
  - `soft_deleted`.
- [x] Add `source_type` to transactions.
- [x] Add `source_ref_id` to transactions.
- [x] Add `source_metadata JSONB` to transactions.
- [x] Add source enum:
  - `manual`;
  - `csv_import`;
  - `transfer`;
  - `deposit_interest`;
  - `savings_allocation`;
  - `subscription`;
  - `reconciliation_adjustment`;
  - `automation_rule`;
  - `llm_draft`;
  - `system`.
- [ ] Add soft-delete/reversal/correction semantics for normal transactions.
- [ ] Add explicit correction event model for changed financial history.
- [ ] Ensure cancelled/reversed/soft-deleted transactions are excluded from current balances.
- [ ] Add transaction history/audit UI for changed records.
- [x] Remove stale README route `DELETE /api/v1/transactions/{id}` and document why hard delete is unavailable.

### Narrow points

```text
The source seam is implemented and exposed through the API.
Future imports, subscriptions and automation must populate it when they create transactions.
```

```text
No hard delete route is better than unsafe delete, but there still needs to be an explicit reversible/correctable history model.
```

---

## 2. Exact Money Representation

### Verified current state

- [x] `transactions.amount` uses `NUMERIC(38,18)` after migration.
- [x] `transfers.from_amount` and `transfers.to_amount` use `NUMERIC(38,18)` after migration.
- [x] `interest_accruals.amount` and `balance` use `NUMERIC(38,18)` after migration.
- [x] Go models use `shopspring/decimal` for money fields.
- [x] Transfer exchange rate uses numeric precision.
- [x] API-facing generated/test data uses string money values in several paths.

### Still TODO

- [ ] Define asset/currency precision table.
- [ ] Support crypto-like precision intentionally instead of treating every asset as normal fiat currency.
- [ ] Decide whether `BTC`, `USDT`, stocks and broker positions are allowed in stable core or future asset module only.
- [ ] Define rounding rules in domain layer.
- [ ] Add tests for:
  - RUB 2-decimal precision;
  - USD/EUR 2-decimal precision;
  - crypto-like 8+ decimal precision;
  - huge amount bounds;
  - sub-minor rejection;
  - cross-currency conversion precision.
- [ ] Avoid converting decimal rates to `float64` in API if precision matters for financial display.

### Narrow points

```text
NUMERIC(38,18) is a strong base, but currency/asset-specific precision is not the same thing as database precision.
The domain still needs explicit asset precision rules.
```

```text
`decimalRatesToFloat` is acceptable for non-authoritative chart/display rates, but not for persisted financial truth.
```

---

## 3. Business Events and Transfers

### Verified current state

- [x] `transfers` table exists.
- [x] Transfer table links `from_transaction_id` and `to_transaction_id`.
- [x] Transaction rows have `transfer_id`.
- [x] Transfer stores from/to account IDs.
- [x] Transfer stores from/to amounts.
- [x] Transfer stores from/to currencies.
- [x] Transfer stores exchange rate.
- [x] Transfer stores exchange rate provider.
- [x] Transfer stores exchange rate date.
- [x] Transfer stores exchange rate scale.
- [x] Transfer supports fee transaction link.
- [x] Transfer supports fee amount and fee currency.
- [x] Transfer creation is atomic in a DB transaction.
- [x] Transfer creation locks source and destination accounts.
- [x] Transfer checks insufficient source balance.
- [x] Transfer validates account ownership through user scope.
- [x] Transfer integrity is enforced by DB function/constraint triggers.
- [x] Transfer list endpoint exists.
- [x] Transfer idempotency key field exists.

### Still TODO

- [ ] Generalize `transfers` into broader `business_events` only if other multi-row events need it.
- [ ] Add transfer lifecycle beyond `completed`:
  - `pending`;
  - `confirmed`;
  - `cancelled`;
  - `reversed`.
- [ ] Store expected rate separately from applied rate.
- [ ] Support `rate_source = manual | cbr | bank | import | other` as enum, not free text.
- [ ] Support imported transfer candidates from CSV Review Queue.
- [ ] Add UI to show transfer details with both legs and fee clearly.
- [ ] Add transfer correction/reversal flow.
- [ ] Decide whether fee transaction should have `source_type = transfer_fee` after source model exists.

### Narrow points

```text
The transfer audit model is mostly closed for MVP.
The weak area is lifecycle/correction and integration with imports/reconciliation.
```

```text
DB-level transfer integrity is good, but if transaction lifecycle statuses are added, the trigger must be updated to ignore reversed/cancelled records correctly.
```

---

## 4. Idempotency for Financial Mutations

### Verified current state

- [x] `idempotency_keys` table exists.
- [x] Idempotency stores key, user, method/path, request hash, response body/status and expiry.
- [x] Later migration adds id, endpoint, pending/completed status, lock time and updated timestamp.
- [x] HTTP middleware supports `Idempotency-Key`.
- [x] Same key + different request body returns conflict.
- [x] Existing completed response is replayed.
- [x] In-progress request returns `idempotency_in_progress`.
- [x] `POST /api/v1/transactions` requires `Idempotency-Key`.
- [x] `POST /api/v1/transfers` requires `Idempotency-Key`.
- [x] `POST /api/v1/accounts/{id}/accrue-interest` requires `Idempotency-Key`.
- [x] `POST /api/v1/accounts/{id}/recalculate-interest` requires `Idempotency-Key`.
- [x] Web client has idempotency-related tests.

### Still TODO

- [ ] Increase or make configurable idempotency retention; current code uses 24h, planning target is safer around 30 days.
- [ ] Add idempotency for future CSV import batch create/apply.
- [ ] Add idempotency for future bulk operations.
- [ ] Add idempotency for future savings allocation confirmation.
- [ ] Consider not storing full response body forever if retention increases.
- [ ] Add cleanup job for expired idempotency records.
- [ ] Add UX toast for “operation is still in progress”.

### Narrow points

```text
Idempotency is strong for current financial mutation endpoints.
The future risk is imports and bulk actions, where batch-level and row-level idempotency will both matter.
```

---

## 5. Financial Audit Log

### Verified current state

- [x] `auth_audit_events` table exists.
- [x] Auth/passkey events can be audited.
- [x] Transfer audit data is persisted in the `transfers` table.
- [x] Interest accruals are persisted and linked to generated transactions.

### Still TODO

- [ ] Add generic `audit_events` table for non-auth events.
- [ ] Audit account create/update/archive.
- [ ] Audit transaction create/change/reverse/soft-delete.
- [ ] Audit transfer create/reverse/correction.
- [ ] Audit interest rule create/update/delete/deactivate.
- [ ] Audit settings/security changes.
- [ ] Audit backup/restore operations.
- [ ] Audit import batch decisions.
- [ ] Store actor/user ID, event type, entity type, entity ID, before/after summary, IP/user-agent where appropriate.
- [ ] Add audit event query UI for security/settings/admin diagnostics.

### Narrow points

```text
Auth audit exists, but “all important actions” is not closed yet.
A finance app needs a generic audit trail beyond login/security events.
```

---

# v0.6.1 — Security / Auth / Recovery

## 6. Auth and Sessions

### Verified current state

- [x] Users table exists.
- [x] Refresh tokens table exists.
- [x] Refresh token hash is stored, not raw token.
- [x] Refresh-token revocation field exists.
- [x] Setup/login/refresh/logout routes exist.
- [x] First-user setup route exists.
- [x] Setup repeat prevention is implemented according to repo TODO/docs and route flow.
- [x] Session list route exists.
- [x] Session revoke route exists.
- [x] Password change route exists.
- [x] Login lockout fields exist.
- [x] Rate limiting middleware is wired.
- [x] Trusted proxy config exists in router config.
- [x] Host policy middleware is wired.
- [x] CORS middleware is wired.
- [x] Security headers middleware is wired.

### Still TODO

- [ ] Verify password hashing parameters and document them in security docs.
- [ ] Add logout-all endpoint if not already covered by password-change/session revocation semantics.
- [ ] Add Telegram notification for new login after app-level Telegram integration exists.
- [ ] Add server-console recovery command for password reset.
- [ ] Add emergency recovery code generated at setup.
- [ ] Add recovery audit events.
- [ ] Add UI for active sessions if current UI is incomplete.
- [ ] Add explicit handling for lost passkey + lost password.

### Narrow points

```text
Security baseline is strong for a trainee/junior project, but recovery is still missing.
For self-hosted software, recovery must be boring and documented before passkey-only login is encouraged.
```

---

## 7. Passkeys / WebAuthn

### Verified current state

- [x] `passkey_credentials` table exists.
- [x] `webauthn_challenges` table exists.
- [x] Credential name exists.
- [x] `last_used_at` exists.
- [x] `revoked_at` exists.
- [x] Passkey routes are wired:
  - login options;
  - login verify;
  - registration options;
  - registration verify;
  - rename;
  - delete;
  - list.
- [x] WebAuthn RP ID/origins are configurable in router config.
- [x] Passkey login is wired into the current auth service flow.

### Still TODO

- [ ] Add real browser E2E with Playwright virtual authenticator.
- [ ] Confirm “fresh password/session required before first passkey add” in code if not already covered by tests.
- [ ] Add recovery UX for deleting last passkey safely.
- [ ] Add passkey management docs for self-host reverse proxy.
- [ ] Add warning when RP ID/origin config is unsafe.

### Narrow points

```text
Passkey backend looks implemented. The remaining weak spot is recovery and real browser E2E, not the DB model.
```

---

## 8. Server Console Recovery

### Verified current state

- [ ] No code-confirmed server console password reset command found.

### TODO

- [ ] Add command:

```bash
capitalflow admin reset-password --email user@example.com
```

- [ ] Generate one-time recovery token/link.
- [ ] Token expires in 15–30 minutes.
- [ ] Revoke active refresh sessions after reset.
- [ ] Write security audit event.
- [ ] Do not automatically delete passkeys during password reset.
- [ ] Add separate dangerous command only if needed:

```bash
capitalflow admin disable-passkeys --email user@example.com
```

### Narrow point

```text
Do not make web-only recovery the only recovery path for a self-hosted app.
If the web account is locked, the server owner needs a local/admin recovery path.
```

---

# v0.6.2 — Backup / Restore

## 9. Backup and Restore

### Verified current state

- [x] Application backup command exists.
- [x] Restore command exists and requires an empty database.
- [ ] No backup UI confirmed.
- [x] Backup/restore CI verification restores financial data into an empty database.
- [ ] No encrypted backup format confirmed.

### TODO

- [x] Add manual backup command.
- [x] Add scheduled backup support.
- [x] Add local backup destination.
- [x] Add Syncthing-friendly backup directory support.
- [x] Add backup before migrations.
- [ ] Add backup before import.
- [ ] Add backup before bulk destructive/reversal actions.
- [x] Add restore into fresh instance.
- [x] Add restore test against empty DB.
- [ ] PARTIAL — add backup metadata:
  - [x] app version;
  - [x] schema version;
  - [x] created_at;
  - [x] base currency;
  - [x] backup format version;
  - [ ] APP_SECRET_KEY fingerprint, not the key itself (key does not exist yet).
- [ ] Include original CSV import files in backups.
- [ ] Include encrypted integration secrets by default.
- [ ] Restore financial data even when `APP_SECRET_KEY` is missing/different.
- [ ] Restore encrypted integration secrets only when `APP_SECRET_KEY` matches.
- [ ] Mark integrations as `reconnect_required` when secrets cannot be decrypted.
- [x] Add backup integrity checks/checksums.
- [x] Add retention policy.
- [ ] Add Settings → Backups UI page.
- [ ] Add Telegram alert for failed backup after Telegram integration exists.

### Narrow points

```text
The core backup/restore path is tested against an empty database.
Encryption and future integration-secret recovery remain separate follow-ups.
```

```text
Financial data must never depend on APP_SECRET_KEY.
Only integration secrets may become unrecoverable when APP_SECRET_KEY is lost.
```

---

# v0.6.3 — Subscriptions

## 10. Subscription Tracking

### Verified current state

- [ ] No `subscriptions` table found.
- [ ] No subscription service found.
- [ ] No subscription routes found.
- [ ] No subscription UI found.
- [ ] Existing categories can represent a “Subscriptions” category, but this is not a first-class subscription model.

### Decisions

- Category `Subscriptions` is the trigger.
- Creating or importing an expense with category `Subscriptions` opens a details form.
- The app should create/recognize a first-class subscription entity after confirmation.
- Subscriptions are linked through transactions; fixed `account_id` is not required.
- Yearly subscriptions are spread into monthly equivalent for planning.
- Actual ledger keeps the real charge date.
- Missed expected charge shows warning.
- `unused` is a priority value.

### TODO

- [ ] Add `subscriptions` table.
- [ ] Add `subscription_transactions` table.
- [ ] Add priority enum:
  - `essential`;
  - `important`;
  - `optional`;
  - `unused`;
  - `paused`.
- [ ] Add subscription status:
  - `active`;
  - `paused`;
  - `cancelled`;
  - `archived`.
- [ ] Add fields:
  - name;
  - amount;
  - currency;
  - billing period;
  - billing interval;
  - next charge date;
  - last charge date;
  - priority;
  - want_to_cancel;
  - auto_detected;
  - detection confidence.
- [ ] Add form triggered by transaction category `Subscriptions`.
- [ ] Form minimum fields:
  - name;
  - amount;
  - currency;
  - billing period;
  - next charge date;
  - priority.
- [ ] Link current transaction to created/recognized subscription.
- [ ] Detect subscription candidates from CSV import.
- [ ] Show monthly equivalent for yearly subscriptions.
- [ ] Show yearly total cost.
- [ ] Show upcoming subscription charges on dashboard.
- [ ] Show subscriptions in Money Calendar.
- [ ] Add missed-charge warning.
- [ ] Add “want to cancel” marker.
- [ ] Add “possibly unused” detection.
- [ ] Add Telegram reminder before charge.
- [ ] Add tests for create, link, missed charge, yearly monthly equivalent and unused priority.

### Narrow points

```text
A category is not enough.
Transaction = real charge.
Subscription = recurring obligation and planning object.
```

```text
Do not silently create subscriptions from a single transaction.
Use a form because period, next charge date and priority cannot be inferred reliably.
```

---

# v0.6.4 — Deposits / Savings / Interest Engine

## 11. Current Interest Engine

### Verified current state

- [x] `interest_rules` table exists.
- [x] `interest_accruals` table exists.
- [x] Interest rules support annual rate, promo rate/end date, accrual frequency, capitalization frequency and day count convention.
- [x] Interest accrual creates linked transaction.
- [x] Duplicate interest accrual is protected by unique constraint.
- [x] Interest accrual service exists.
- [x] Background interest job exists.
- [x] Daily/monthly/end-of-term job names exist.
- [x] Interest scheduler script exists.
- [x] Deploy compose includes interest scheduler container.
- [x] Advisory lock infrastructure exists.
- [x] Production interest wrappers are confirmed by code.

### Still TODO

- [ ] Add first-class fixed deposit entity if term deposits need separate product data beyond account type.
- [ ] Add deposit open/close/maturity metadata beyond generic account opened date.
- [ ] Add top-up deposit support as explicit product behavior.
- [ ] Add top-up cutoff support.
- [ ] Add partial withdrawal flag even if not needed initially.
- [ ] Add expected vs actual interest model.
- [ ] Add imported actual interest linking to expected interest event.
- [ ] Add interest discrepancy warning/review item.
- [ ] Add expected interest forecast without writing DB records.
- [ ] Add recalculation preview before destructive regeneration.
- [ ] Add generated/accrued/corrected statuses for interest events.
- [ ] Add docs/domain/interest-engine.md.
- [ ] Add job run persistence if operational history is required.
- [ ] Re-run race tests in VM, from existing ``.

### Decisions

- Savings account is an account with interest rules.
- Fixed deposit is a separate product-like model if account-level fields are not enough.
- Daily accrued estimate for savings accounts is not needed initially.
- Imported bank interest should be linked to expected interest when possible.

### Narrow points

```text
The current engine covers interest rules and generated accruals.
It does not yet model the real banking product lifecycle: top-ups, maturity, expected-vs-actual and imported bank interest matching.
```

```text
Savings account should appear in Accounts as real money, and in Interest/Deposits as an interest-bearing product.
```

---

# v0.6.5 — Goals / Reserves / Emergency Fund

## 12. Goals and Reserved Money

### Verified current state

- [ ] No `goals` table found.
- [ ] No `goal_contributions` table found.
- [ ] No reserved-money planning layer found.
- [ ] No goals UI found.

### Decisions

- Goal reserve is a global planning layer.
- Goal reserve should not mutate real account balance.
- Emergency fund is a system goal.
- Goal reserve may consider deposits through liquidity/eligibility rules.
- Cross-currency reserves display value can recalculate daily.
- Funding history should be visible.
- Goal templates are useful.
- Goal completion should be manual through a button.

### TODO

- [ ] Add `goals` table.
- [ ] Add `goal_contributions` or `goal_allocations` table.
- [ ] Add priority enum:
  - `critical`;
  - `high`;
  - `medium`;
  - `low`;
  - `paused`.
- [ ] Add goal status:
  - `active`;
  - `completed`;
  - `paused`;
  - `archived`.
- [ ] Add fields:
  - target amount;
  - target currency;
  - current/reserved amount;
  - deadline date;
  - monthly required amount;
  - template type;
  - completion metadata.
- [ ] Add goal funding history UI.
- [ ] Add goal archive behavior.
- [ ] Add manual complete button.
- [ ] Add progress calculation.
- [ ] Add risk marker: not enough monthly saving to reach deadline.
- [ ] Add deterministic sorting:
  - active first;
  - priority;
  - deadline;
  - lower completion percentage;
  - created date.
- [ ] Exclude `paused` goals from allocation suggestions.
- [ ] Add sub-goals later, not MVP.

### Narrow points

```text
Do not subtract goal reserve from real account balance.
Instead calculate free-to-spend separately.
```

```text
Do not allow actual reserved amount to exceed eligible liquid/reservable assets.
Show deficit separately.
```

---

## 13. Emergency Fund

### Verified current state

- [ ] No emergency fund system goal found.
- [ ] No required monthly expenses calculation found.

### Decisions

- Default target = 6 months.
- Settings allow 3 / 6 / 12 / custom.
- If target = 3 months, minimum threshold = 1 month.
- If target > 3 months, minimum threshold = 3 months.
- Required monthly expenses should support adaptive mode or configurable mode.

### TODO

- [ ] Add emergency fund settings.
- [ ] Calculate required monthly expenses automatically.
- [ ] Allow manual override/config mode.
- [ ] Add target month settings.
- [ ] Add minimum threshold logic.
- [ ] Add emergency fund dashboard card.
- [ ] Add tests for 3/6/12/custom target behavior.
- [ ] Add tests for minimum threshold behavior.

### Narrow point

```text
Automatic required-expense calculation will be wrong until categories/subscriptions/budgeting are mature.
Give the user an override.
```

---

## 14. Liquidity Rules for Goal Reserves

### Verified current state

- [ ] No liquidity rule model found.

### TODO

- [ ] Add liquidity classification:
  - `immediate`;
  - `short_term`;
  - `locked`;
  - `investment_risk`.
- [ ] Add goal reserve eligibility:
  - `cash_only`;
  - `cash_and_savings`;
  - `include_deposits`;
  - `include_investments`.
- [ ] Default emergency fund eligibility should exclude risky/locked assets.
- [ ] Allow user override for goal-specific reserve eligibility.
- [ ] Add tests for free-to-spend and reserve calculations.

### Narrow point

```text
A locked fixed deposit should not count the same way as cash in an emergency fund.
```

---

# v0.6.6 — CSV Import

## 15. CSV Import

### Verified current state

- [ ] No import batch table found.
- [ ] No import rows table found.
- [ ] No CSV upload route found.
- [ ] No parser version model found.
- [ ] No import preview UI found.

### Decisions

- CSV first.
- Priority banks/sources: Alfa, Sber, T-Bank, Yandex.
- Original CSV files are always stored and included in backups.
- Preview is required before write.
- `Accept all safe` is allowed but must show final confirmation summary.
- Parser version must be stored.
- Unknown CSV format falls back to manual mapping.
- Manual mapping must be saved as reusable template.
- Duplicate detection should be strict but configurable.
- Duplicate detection across different accounts is not required initially.
- Undo accepted import batch should exist, but must be safe.

### TODO

- [ ] Add `import_batches` table.
- [ ] Add `import_rows` table.
- [ ] Add `import_mapping_templates` table.
- [ ] Add original file storage strategy.
- [ ] Add parser version field.
- [ ] Add file hash field.
- [ ] Add drag-and-drop CSV upload UI.
- [ ] Add bank detection.
- [ ] Add Alfa parser.
- [ ] Add Sber parser.
- [ ] Add T-Bank parser.
- [ ] Add Yandex parser.
- [ ] Add generic CSV parser.
- [ ] Add manual mapping UI.
- [ ] Save manual mapping as reusable template.
- [ ] Add import preview.
- [ ] Split rows into:
  - safe;
  - needs review;
  - duplicate;
  - error.
- [ ] Add `Accept all safe` action.
- [ ] Add final confirmation summary before creating transactions.
- [ ] Store raw row and normalized row.
- [ ] Link created transactions to import row through `source_type/source_ref_id` or typed FK.
- [ ] Add import batch undo/revert.
- [ ] Full revert when imported transactions were not manually edited.
- [ ] Partial/safe revert when rows were edited or linked to subscriptions/transfers/goals.
- [ ] Add import audit events.
- [ ] Add tests for CSV encodings, decimal comma/dot, date formats, duplicate detection and safe accept.

### Narrow points

```text
Import must not silently modify balances.
Preview and final confirmation are mandatory.
```

```text
Original CSV must be preserved; normalized rows are not enough for audit.
```

```text
Undo import must not physically delete history.
Use soft delete/reversal and audit trail.
```

---

# v0.6.7 — Review Queue

## 16. Review Queue

### Verified current state

- [ ] No review queue table found.
- [ ] No review item routes found.
- [ ] No review queue UI found.

### TODO

- [ ] Add `review_items` table.
- [ ] Add types:
  - import row needs review;
  - duplicate candidate;
  - subscription candidate;
  - missed subscription charge;
  - failed rate sync;
  - unmatched transfer candidate;
  - interest discrepancy;
  - reconciliation difference.
- [ ] Add severity:
  - info;
  - warning;
  - critical.
- [ ] Add actions:
  - accept;
  - ignore;
  - edit;
  - link;
  - split;
  - mark duplicate;
  - reconnect integration.
- [ ] Add dashboard “needs attention” card.
- [ ] Add tests for review item lifecycle.

### Narrow point

```text
Review Queue is where the app stays safe while still being helpful.
Avoid auto-mutating money when confidence is low.
```

---

# v0.6.8 — Currencies / Rates

## 17. Multi-Currency and Rates

### Verified current state

- [x] Currency validation exists for 3-letter uppercase codes.
- [x] `/api/v1/currency-rates` route exists.
- [x] Currency service exists.
- [x] HTTP provider exists.
- [x] Latest exchange rate cache exists.
- [ ] PARTIAL: Current provider is `open.er-api.com`, not the planned manual/CBR-first model.
- [ ] PARTIAL: Current cache TTL is 6 hours, not hourly sync.
- [ ] PARTIAL: Rates are not persisted as historical rate records.

### Decisions

- Base currency is selected at setup but can be changed.
- Old reports should not be retroactively recalculated just because base currency changes.
- First planned providers: manual and CBR.
- Hourly updates are desired.
- Rate history is required.
- Failed sync should create review item.
- Manual override is required.
- Historical reports should use historical rates.

### TODO

- [ ] Add `currency_rates` table.
- [ ] Add `rate_provider` enum:
  - `manual`;
  - `cbr`;
  - `open_er_api` if kept;
  - `bank`;
  - `import`;
  - `other`.
- [ ] Add manual rate entry UI.
- [ ] Add CBR provider.
- [ ] Add hourly sync job.
- [ ] Persist rate history.
- [ ] Add failed sync review item.
- [ ] Add manual override support.
- [ ] Add historical report rate lookup.
- [ ] Add current dashboard display conversion separate from historical applied rates.
- [ ] Add tests for missing historical FX rate.

### Narrow point

```text
Do not confuse display rates with applied transaction/transfer rates.
Historical applied rate is financial truth; latest rate is only display/estimate.
```

---

# v0.6.9 — Budgeting / Money Calendar

## 18. Budgeting and Safe-to-Spend

### Verified current state

- [ ] No budgets table found.
- [ ] No budget categories table found.
- [ ] No safe-to-spend calculation found.

### Decisions

- Monthly category budgets are needed.
- Planned vs actual is needed.
- Rollover should exist.
- `safe_to_spend` should account for:
  - liquid balance;
  - upcoming mandatory payments;
  - active goal reservations;
  - minimum emergency reserve;
  - planned subscriptions.

### TODO

- [ ] Add `budgets` table.
- [ ] Add `budget_categories` table.
- [ ] Add monthly budget UI.
- [ ] Add planned vs actual calculations.
- [ ] Add rollover behavior.
- [ ] Add safe-to-spend service.
- [ ] Add safe-to-spend dashboard card.
- [ ] Add tests for transfers not counted as expenses.
- [ ] Add tests for refunds.
- [ ] Add tests for month boundaries and timezones.

### Narrow point

```text
Budgeting depends on correct categories, subscriptions and imports.
Avoid smart recommendations before basic budget truth is stable.
```

---

## 19. Money Calendar

### Verified current state

- [ ] No money calendar model or UI found.

### TODO

- [ ] Add planned events model or generated calendar query.
- [ ] Show salary/income events.
- [ ] Show rent/utilities/subscriptions.
- [ ] Show goal deadlines.
- [ ] Show expected deposit interest.
- [ ] Show monthly cashflow forecast.
- [ ] Add `.ics` export if practical.
- [ ] Add Telegram reminders for future events.

### Narrow point

```text
Money Calendar should read from planned sources first.
Do not duplicate subscription/deposit/goal data unless a snapshot is needed.
```

---

# v0.7.0 — Telegram Integration

## 20. Telegram Integration

### Verified current state

- [x] CI release Telegram notification exists.
- [ ] App-level Telegram integration is not implemented.
- [ ] No Telegram settings table found.
- [ ] No encrypted Telegram token storage found.
- [ ] No app Telegram commands found.

### Decisions

- Telegram integration is built into CapitalFlow settings.
- User enters bot token and chat IDs in settings.
- Multiple chat IDs should be supported.
- Financial amounts should be hidden under Telegram spoiler by default.
- Notification types should be configurable.
- Token should be encrypted in DB using `APP_SECRET_KEY`.
- Backup should include encrypted integration secrets by default.

### TODO

- [ ] Add `APP_SECRET_KEY` config for app-managed integration secret encryption.
- [ ] Add encrypted secret helper.
- [ ] Add `telegram_settings` table.
- [ ] Store bot token encrypted.
- [ ] Store chat IDs encrypted or protected.
- [ ] Add notification preferences:
  - backup failed;
  - weekly digest;
  - large transaction;
  - deposit interest;
  - goal deadline;
  - subscription upcoming charge;
  - new login.
- [ ] Add amount visibility setting:
  - hidden;
  - spoiler;
  - visible.
- [ ] Add test message button.
- [ ] Add `/summary` command.
- [ ] Add `/backup` command.
- [ ] Add `/goals` command.
- [ ] Keep dangerous mutation commands disabled or confirmation-only.

### Narrow point

```text
Release workflow Telegram notification is not the same as app-level Telegram integration.
Do not mark Telegram product feature done because CI can notify releases.
```

---

# v0.7.1 — Automation Rules

## 21. Automation Rules

### Verified current state

- [x] Safe system automation exists for interest jobs.
- [x] Interest jobs use locking/duplicate protection.
- [ ] User-defined automation rules are not implemented.
- [ ] Rule execution log is not implemented.
- [ ] Dry-run UI is not implemented.

### Decisions

- Savings allocations should be suggested, not auto-executed.
- Only safe system calculations like deposit interest can auto-execute initially.
- Rule execution log is needed.
- Dry-run is needed.

### TODO

- [ ] Add `automation_rules` table.
- [ ] Add `automation_rule_runs` table.
- [ ] Add modes:
  - `suggest`;
  - `auto_apply_safe`;
  - `requires_confirmation`.
- [ ] Add rule dry-run for historical period.
- [ ] Add rule execution log UI.
- [ ] Add guardrails:
  - max amount per execution;
  - max executions per day;
  - confirmation above amount;
  - allowed action types.
- [ ] Add category suggestion rule.
- [ ] Add salary → savings allocation suggestion rule.
- [ ] Add large transaction notice rule.
- [ ] Add crypto price threshold notice later.

### Narrow point

```text
Automation should never become a hidden actor.
Suggestions are safe; financial writes require confirmation.
```

---

# v0.7.2 — Reconciliation

## 22. Reconciliation

### Verified current state

- [ ] No reconciliation model found.
- [ ] No reconciliation routes found.
- [ ] No reconciliation UI found.

### TODO

- [ ] Add `reconciliations` table.
- [ ] Compare ledger balance with user-entered actual balance.
- [ ] Show account balance difference.
- [ ] Allow adjustment transaction with required reason.
- [ ] Store reconciliation history by account.
- [ ] Show badge: balance unchecked / last checked.
- [ ] Do not block reports initially when account is not reconciled.
- [ ] Add tests for adjustment and audit trail.

### Narrow point

```text
Reconciliation depends on transaction-derived balances and audit/correction model.
Do not implement it as a direct balance override.
```

---

# v0.8.0 — Investments

## 23. Crypto / Stocks / Investments

### Verified current state

- [x] Account type `broker` exists.
- [ ] No asset position table found.
- [ ] No investment transaction model found.
- [ ] No portfolio summary found.
- [ ] No PnL model found.

### TODO

- [ ] Add `assets` table.
- [ ] Add `asset_positions` table.
- [ ] Add manual position entry.
- [ ] Add symbol, amount, average price, currency, rate source.
- [ ] Add crypto support foundation.
- [ ] Add Russian stocks support foundation.
- [ ] Add current value summary.
- [ ] Add portfolio allocation chart.
- [ ] Add realized/unrealized gains later.
- [ ] Add dividends/coupons later.
- [ ] Add broker imports later.

### Narrow point

```text
The presence of `broker` account type is not portfolio tracking.
It only prepares navigation/account classification.
```

---

# v0.8.1 — Local LLM Assistant

## 24. Ollama / Local LLM Assistant

### Verified current state

- [ ] No LLM/Ollama service found.
- [ ] No assistant routes found.
- [ ] No assistant UI found.

### Decisions

- First scenario: budget allocation advice based on priorities.
- Subscription priorities must be included in advice.
- Text-only at first.
- No raw transaction access by default.
- Confirmation before data access.
- Chat history is desired.
- Sensitive fields should be redacted.
- Explain-only mode is needed.
- Disclaimer is needed.
- Draft actions require confirmation.

### TODO

- [ ] Add assistant settings.
- [ ] Add provider interface:
  - Ollama provider;
  - mock provider for tests;
  - OpenAI-compatible provider only with explicit opt-in, if ever added.
- [ ] Add safe context endpoints:
  - budget summary;
  - goals summary;
  - subscriptions summary;
  - deposits summary;
  - upcoming cashflow.
- [ ] Add assistant context audit: what data was used.
- [ ] Add chat history table.
- [ ] Add redaction/privacy layer.
- [ ] Add explain-only mode.
- [ ] Add disclaimer.
- [ ] Add draft action model.
- [ ] Require confirmation for any write action.

### Narrow point

```text
LLM must not call repositories directly.
It should consume prepared summaries and show which data it used.
```

---

# v0.8.2 — Mobile / PWA

## 25. Mobile and PWA

### Verified current state

- [ ] No PWA manifest confirmed.
- [ ] No offline cache confirmed.
- [ ] No mobile-specific E2E except general responsive intent.

### Decisions

- iPhone 13/13 Pro should be a priority viewport.
- Quick add transaction is useful.
- Telegram is enough for push notifications initially.
- Biometric/passkey-friendly login matters.

### TODO

- [ ] Add mobile layout audit for iPhone 13/13 Pro.
- [ ] Add mobile dashboard smoke test.
- [ ] Add quick-add transaction UX.
- [ ] Add installable PWA manifest if useful.
- [ ] Add offline read-only cache only after core data sync model is clear.
- [ ] Do not add offline write queue until conflict/retry/idempotency model is ready.

### Narrow point

```text
Offline writes are dangerous for finance apps unless conflict resolution and idempotency are mature.
```

---

# v0.8.3 — Frontend UX Direction

## 26. UI / UX

### Verified current state

- [x] React + Vite + TypeScript WebUI exists.
- [x] Chakra UI is used.
- [x] TanStack Query is installed.
- [x] Recharts is installed.
- [x] Dashboard view exists.
- [x] Accounts/transactions/transfer flows exist.
- [x] Login/setup/auth flow exists.
- [ ] PARTIAL: compact professional dashboard exists, but it is not yet aligned with the new subscription/deposit/budget-first product direction.
- [ ] Privacy mode / hide amounts hotkey not found.
- [ ] Subscriptions dashboard card not implemented.
- [ ] Upcoming expenses/income planning card not implemented.
- [ ] Budget/safe-to-spend card not implemented.

### Decisions

- Both light and dark themes are desired.
- UI should be compact and professional.
- Dashboard top cards:
  - total balance;
  - income/expense graph;
  - upcoming expenses;
  - last 5 transactions;
  - subscriptions/budget soon.
- Privacy mode is needed.
- Hotkey to hide amounts is needed.
- Onboarding should explain basic functionality.

### TODO

- [ ] Add privacy mode state.
- [ ] Add hotkey to hide amounts.
- [ ] Add total capital card with clear hierarchy.
- [ ] Add upcoming expenses/income card.
- [ ] Add subscriptions-this-month card.
- [ ] Add budget/safe-to-spend card after budgeting module exists.
- [ ] Add “needs attention” card after Review Queue exists.
- [ ] Add empty/loading/error/warning states consistently for new pages.
- [ ] Add mobile bottom navigation only if sidebar is poor on phone.

### Narrow point

```text
Do not over-design before subscriptions/imports/goals exist.
But do keep dashboard slots ready for those modules.
```

---

# v0.9.0 — Testing / CI Hardening

## 27. Test Strategy

### Verified current state

- [x] GitHub Actions CI exists.
- [x] Backend tests run on Ubuntu and Windows.
- [x] Race tests run.
- [x] golangci-lint runs.
- [x] WebUI lint/tests/build run.
- [x] OpenAPI lint runs.
- [x] Migration up/status check runs against PostgreSQL service.
- [x] Production images are built in CI for PR validation.
- [x] Release tag guard exists.
- [x] Basic Playwright config exists.
- [x] `npm run test:e2e` script exists.
- [x] Basic E2E smoke test exists.
- [x] Real backend + PostgreSQL E2E stack exists.
- [x] Real setup, account, transaction, transfer and dashboard flow runs in CI.
- [x] Backup/restore smoke test runs in CI against an empty database.

### Still TODO

- [x] Add real backend + PostgreSQL E2E stack.
- [x] Add isolated test DB reset strategy.
- [x] Add setup flow E2E with real backend.
- [ ] Add login/logout/refresh E2E with real backend.
- [x] Add transaction/transfer/dashboard E2E with real backend.
- [ ] Add passkey virtual authenticator E2E.
- [x] Add backup/restore smoke test.
- [ ] Add import preview/apply E2E after import module exists.
- [ ] Add Playwright traces/screenshots/videos only on failure.
- [ ] Add controlled clock/test helpers for date-sensitive flows.
- [ ] Add migration down test where safe.
- [ ] Add dashboard performance baseline.
- [ ] Re-run race tests in the VM after roadmap cleanup.

### Narrow point

```text
The mocked UI smoke test remains useful for broad flows.
The real E2E additionally proves the core browser + backend + PostgreSQL financial path.
```

---

# v0.9.1 — Deployment / Operations

## 28. Local and Self-Hosted Deployment

### Verified current state

- [x] Local run docs exist.
- [x] Docker Compose for local PostgreSQL is referenced by Makefile.
- [x] Production compose exists under `deploy/compose.yaml`.
- [x] API container healthcheck exists.
- [x] Web container healthcheck exists.
- [x] Postgres healthcheck exists.
- [x] Interest scheduler container exists.
- [x] Traefik labels exist for web service.
- [x] Deploy VM script exists.
- [x] CI can build and push GHCR images on release tags.
- [x] `/health`, `/ready`, `/metrics` routes exist.
- [x] `slog` request logging exists.
- [ ] PARTIAL: Request logging does not include request ID or user ID.
- [ ] PARTIAL: `/metrics` route exists, but Prometheus-quality metrics need verification.
- [ ] NixOS service/timer examples not confirmed.
- [x] Backup scheduler is part of the production compose deployment.

### TODO

- [ ] Add or verify `docker-compose.yml` for local development and make docs match actual file names.
- [ ] Add nginx reverse-proxy example.
- [ ] Keep Traefik example from deploy compose and document it.
- [ ] Add NixOS systemd service example.
- [ ] Add NixOS backup timer example for non-Docker deployments.
- [ ] Add production `.env.example` with safe comments.
- [ ] Add request ID to logs.
- [ ] Add user ID to logs when available.
- [ ] Add route name/status/latency consistently.
- [ ] Separate audit logs from application logs.
- [ ] Add admin diagnostics page.
- [ ] Add graceful shutdown tests if not already covered.

### Narrow points

```text
Deployment is ahead of product modules, which is good.
Backup/restore and scheduled retention now cover the Docker deployment path.
Native NixOS examples remain operational documentation work.
```

```text
Current deploy script can write deploy/.env on the server.
The web app itself should not casually rewrite env files.
```

---

# Deduplicated Issue Backlog

## P0 — Must close before real money usage

- [ ] `feat(audit): add generic financial/settings audit_events table`
- [x] `feat(transactions): add source_type/source_ref_id/source_metadata`
- [ ] `feat(transactions): add lifecycle status and correction/reversal model`
- [x] `feat(backups): add backup command and restore into fresh DB`
- [x] `test(backups): verify restore path against empty DB`
- [ ] `feat(recovery): add server-console password reset command`
- [ ] `docs(security): document recovery, APP_SECRET_KEY and secret-loss behavior`
- [x] `fix(docs): remove stale transaction DELETE route or implement safe endpoint`
- [x] `test(e2e): add real backend + PostgreSQL critical flow`

## P1 — Daily personal value

- [ ] `feat(subscriptions): add subscription domain model`
- [ ] `feat(subscriptions): create/recognize subscription from Subscriptions-category expense form`
- [ ] `feat(subscriptions): link subscriptions to transactions`
- [ ] `feat(subscriptions): add priority essential/important/optional/unused/paused`
- [ ] `feat(subscriptions): show monthly equivalent and yearly total`
- [ ] `feat(subscriptions): warn when expected charge is missed`
- [ ] `feat(imports): add CSV import batches and rows`
- [ ] `feat(imports): store original CSV and parser_version`
- [ ] `feat(imports): add preview and final accept-all-safe confirmation`
- [ ] `feat(imports): add manual mapping fallback and reusable templates`
- [ ] `feat(review): add review queue for imports/subscriptions/rates/interest discrepancies`
- [ ] `feat(goals): add goals and global reserved money planning layer`
- [ ] `feat(goals): add emergency fund system goal`
- [ ] `feat(budget): add monthly budgets and safe-to-spend`
- [ ] `feat(interest): add expected-vs-actual interest matching`
- [ ] `feat(rates): add persisted rate history and CBR/manual provider`

## P2 — Product quality and automation

- [ ] `feat(telegram): add app-level encrypted Telegram settings and notifications`
- [ ] `feat(calendar): add money calendar`
- [ ] `feat(reconciliation): add reconciliation checks and adjustment flow`
- [ ] `feat(automation): add suggestion-only rule engine with dry-run`
- [ ] `feat(observability): add request_id/user_id to logs and improve metrics`
- [ ] `docs(ops): add nginx/Traefik/NixOS examples`

## P3 — Later expansion

- [ ] `feat(investments): add manual asset positions`
- [ ] `feat(llm): add local Ollama assistant with safe summaries`
- [ ] `feat(pwa): add installable/mobile-friendly PWA shell`
- [ ] `feat(analytics): add advanced forecasts and report snapshots`

---

# v1.0.0 — Real-money-ready Core Release

## Goal

Make CapitalFlow safe enough for daily personal finance use with real data.

## Acceptance criteria

- [x] Backup and restore are tested against a fresh database.
- [x] Critical browser flows are covered by real backend + PostgreSQL E2E tests.
- [ ] Financial audit, correction and recovery paths are documented.
- [ ] README, deployment docs and API docs match the implemented product.
- [ ] No P0 issue remains in the deduplicated backlog.

---

# Files checked during this verification

This list is not exhaustive, but these files were directly inspected and influenced checkbox status:

```text
TODO.md
README.md
Makefile
.github/workflows/ci.yml
migrations/000001_create_auth.sql
migrations/000002_create_accounts.sql
migrations/000004_create_transactions.sql
migrations/000005_create_interest_rules.sql
migrations/000006_create_interest_accruals.sql
migrations/000007_create_balance_snapshots.sql
migrations/000010_create_auth_audit_events.sql
migrations/000013_create_idempotency_keys.sql
migrations/000015_add_user_login_lockout.sql
migrations/000018_create_transfers.sql
migrations/000019_add_financial_invariants_and_indexes.sql
migrations/000021_money_amounts_numeric.sql
migrations/000022_complete_v056_transfer_idempotency.sql
migrations/000025_create_passkeys.sql
internal/models/transaction.go
internal/services/account_service.go
internal/services/balance_service.go
internal/services/transaction_service.go
internal/services/currency_service.go
internal/jobs/interest.go
internal/postgres/store.go
internal/postgres/transactions.go
internal/http/handlers/router.go
internal/http/handlers/health.go
internal/http/handlers/dashboard.go
internal/http/handlers/currencies.go
internal/http/handlers/response.go
internal/http/middleware/idempotency.go
internal/http/middleware/logging.go
web/package.json
web/playwright.config.ts
web/e2e/core-flow.spec.ts
scripts/interest-scheduler.sh
scripts/deploy-vm.sh
deploy/compose.yaml
```
