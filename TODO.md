# CapitalFlow Roadmap

Этот файл описывает актуальный порядок разработки CapitalFlow. Он не должен быть архивом всех старых идей. Старые завершенные этапы оставлены только как краткий контекст, чтобы было понятно, откуда проект пришел и что уже не нужно планировать заново.

## Product goal

CapitalFlow — self-hosted сервис для личного учета финансов. Главная цель: приватный финансовый центр, который можно запустить у себя в Docker/NixOS, вести счета, карты, наличные, вклады, накопительные счета, переводы, доходы и расходы, а позже расширить до инвестиций, мультивалютности и локального LLM-помощника.

До v1.0 проект должен быть не экспериментом, а приложением, которым можно пользоваться каждый день.

## Development rules

* Финансовая корректность важнее количества фич.
* Баланс должен объясняться операциями, а не храниться как неаудируемое число.
* Все денежные значения — только `amount_minor int64` или `shopspring/decimal` там, где нужен decimal math. Не использовать `float` для денег.
* Ставки хранить в basis points, например `1700` = `17.00%`.
* Любая изменяющая денежная операция должна быть идемпотентной или явно защищенной от повторного выполнения.
* Любой перевод должен быть атомарным: обе стороны перевода созданы или не создано ничего.
* Cross-currency transfer должен сохранять applied exchange rate, provider/date и связь двух transaction legs.
* LLM, Telegram bot, investments и advanced analytics не делать до стабильного core.
* Каждый крупный этап должен иметь tests, docs и acceptance criteria.

## Current status

### Done / mostly done

* [x] Legacy deposit CLI stabilized.
* [x] Core domain added: accounts, transactions, categories, interest rules.
* [x] PostgreSQL storage added.
* [x] Goose migrations added.
* [x] Thin HTTP API added.
* [x] React + Vite + TypeScript WebUI added.
* [x] Basic dashboard/accounts/transactions/transfer flows added.
* [x] Auth setup/login/refresh/logout added.
* [x] Refresh-token rotation and session revocation added.
* [x] Basic auth hardening added: password hashing, rate limits, audit events, sessions.
* [x] OpenAPI contract added.
* [x] CI exists for backend and WebUI checks.

### Needs verification before v1.0

* [ ] README and run docs match the real current auth flow.
* [ ] WebUI dev proxy and API port are documented consistently.
* [ ] Transfer audit model is good enough for cross-currency operations.
* [ ] Idempotency behavior is tested for all financial mutations.
* [ ] E2E tests cover critical user flows.
* [ ] Backup/restore is available before the app is used with real data.
* [ ] Production/self-host deployment path is documented.

---

# v0.6 — Financial Correctness & Auditability

## Goal

Закрыть риск неправильных или неаудируемых денежных операций до добавления новых больших фич.

## Scope

### Transfer audit model

* [ ] Добавить `transfers` table или transfer-group fields.
* [ ] Связать `transfer_out` и `transfer_in` с одним business event.
* [ ] Для cross-currency transfer сохранять:
  * [ ] source account;
  * [ ] destination account;
  * [ ] source amount;
  * [ ] destination amount;
  * [ ] source currency;
  * [ ] destination currency;
  * [ ] applied exchange rate;
  * [ ] rate provider;
  * [ ] rate date/time;
  * [ ] linked transaction IDs.
* [ ] Добавить read endpoint для transfer event:
  * [ ] `GET /api/v1/transfers/{id}`.
* [ ] UI transfer details должен показывать обе стороны перевода и applied rate.

### Idempotency

* [ ] Ввести единый idempotency mechanism для financial mutations.
* [ ] Поддержать `Idempotency-Key` для:
  * [ ] create transaction;
  * [ ] create transfer;
  * [ ] accrue interest;
  * [ ] recalculate interest, если endpoint остается mutating;
  * [ ] import operations, когда они появятся.
* [ ] Persisted idempotency result должен переживать повтор запроса.
* [ ] Не оставлять key в `pending` после успешной записи.
* [ ] Повтор с тем же key и другим body должен возвращать conflict.

### Account and money invariants

* [ ] Запретить изменение currency у account после появления transactions.
* [ ] Проверить overflow/underflow для `amount_minor`.
* [ ] Проверить отрицательные балансы: где разрешены, где запрещены.
* [ ] Унифицировать validation errors для денежных операций.
* [ ] Добавить DB constraints там, где инвариант нельзя оставлять только в сервисе.

### Tests

* [ ] Cross-currency transfer test проверяет persisted rate/group data.
* [ ] Transfer rollback test: если вторая leg падает, первая не остается в БД.
* [ ] Idempotency retry test возвращает тот же результат.
* [ ] Idempotency body mismatch test возвращает conflict.
* [ ] Account currency invariant test.
* [ ] Overflow/large amount tests.

## Acceptance criteria

* [ ] По любой операции перевода можно восстановить весь business event.
* [ ] Cross-currency transfer audit не зависит от текущего курса валют.
* [ ] Повторный financial mutation request не создает дублей.
* [ ] Все P0/P1 financial correctness tests проходят в CI.

---

# v0.7 — E2E Testing Baseline

## Goal

Добавить end-to-end тесты, которые проверяют реальные пользовательские сценарии через браузер: WebUI, API, auth, PostgreSQL и routing вместе.

## Scope

* [ ] Настроить Playwright.
* [ ] Добавить отдельную test database.
* [ ] Добавить docker compose profile или отдельный compose-файл для E2E.
* [ ] Перед E2E автоматически применять migrations.
* [ ] Добавить stable seed для тестов.
* [ ] Не использовать production secrets в E2E.

## Critical flows

* [ ] First setup.
* [ ] Login.
* [ ] Logout.
* [ ] Session bootstrap after page reload.
* [ ] Create account.
* [ ] Create income transaction.
* [ ] Create expense transaction.
* [ ] Create transfer.
* [ ] Dashboard updates after mutation.
* [ ] Create interest rule.
* [ ] Manual interest accrual.
* [ ] Theme persistence.

## CI

* [ ] Добавить `npm run test:e2e` в CI.
* [ ] Сохранять Playwright report как artifact.
* [ ] E2E job должен быть отдельным check, не смешанным с unit tests.

## Acceptance criteria

* [ ] Один локальный command запускает E2E окружение.
* [ ] Critical user flows покрыты E2E.
* [ ] E2E стабильно проходит в CI.
* [ ] В документации описано, как запускать E2E локально.

---

# v0.8 — Passkey / WebAuthn

## Goal

Добавить passkey login как optional secure login method поверх уже стабильной password auth.

Passkey не должен заменять password login до появления нормального recovery flow. Password остается fallback.

## Scope

### Backend

* [ ] Добавить WebAuthn config:
  * [ ] `WEBAUTHN_RP_ID`;
  * [ ] `WEBAUTHN_RP_NAME`;
  * [ ] `WEBAUTHN_ALLOWED_ORIGINS`.
* [ ] Добавить `passkey_credentials` table.
* [ ] Добавить одноразовые challenges с TTL.
* [ ] Добавить registration flow.
* [ ] Добавить login flow.
* [ ] Интегрировать успешный passkey login в текущий access/refresh token flow.
* [ ] Писать passkey events в audit log.
* [ ] Rate limit для passkey endpoints.

### API

* [ ] `POST /auth/passkeys/register/options`.
* [ ] `POST /auth/passkeys/register/verify`.
* [ ] `POST /auth/passkeys/login/options`.
* [ ] `POST /auth/passkeys/login/verify`.
* [ ] `GET /auth/passkeys`.
* [ ] `PATCH /auth/passkeys/{id}`.
* [ ] `DELETE /auth/passkeys/{id}`.

### Frontend

* [ ] `Sign in with passkey` на login screen.
* [ ] `Settings -> Security -> Passkeys`.
* [ ] Add/rename/delete passkey.
* [ ] Browser-not-supported fallback.
* [ ] Безопасные ошибки без раскрытия sensitive details.

### Tests

* [ ] Challenge replay rejected.
* [ ] Expired challenge rejected.
* [ ] Wrong origin rejected.
* [ ] Wrong rpID rejected.
* [ ] Revoked credential rejected.
* [ ] Credential from another user rejected.
* [ ] E2E smoke через virtual authenticator.

## Acceptance criteria

* [ ] Пользователь может добавить passkey в Settings.
* [ ] Пользователь может войти через passkey.
* [ ] Password login остается рабочим fallback.
* [ ] Один пользователь может иметь несколько passkeys.
* [ ] Passkey flow покрыт unit, handler, security и E2E smoke tests.

---

# v0.9 — Backup, Restore & Import

## Goal

Сделать безопасное использование приложения с реальными данными.

## Scope

### Backup/restore

* [ ] CLI command для backup PostgreSQL.
* [ ] CLI command для restore в новую/пустую БД.
* [ ] UI action для manual backup.
* [ ] Backup перед опасными операциями.
* [ ] Retention policy.
* [ ] Restore verification на test database.
* [ ] Документация: где лежат backup, как восстановиться, как проверить backup.

### Import/export

* [ ] CSV import preview.
* [ ] Column mapping.
* [ ] Deduplication rules.
* [ ] Import report.
* [ ] Export accounts/transactions to CSV.

## Acceptance criteria

* [ ] Можно сделать backup одной командой.
* [ ] Можно восстановить данные из backup по инструкции.
* [ ] Import не пишет данные без preview/confirm step.
* [ ] Перед import/restore создается safety backup.

---

# v0.10 — Self-host Release Candidate

## Goal

Подготовить приложение к нормальному запуску на своем сервере за reverse proxy.

## Scope

* [ ] Dockerfile для backend.
* [ ] Dockerfile для WebUI.
* [ ] `docker-compose.prod.yml`.
* [ ] Healthcheck для backend.
* [ ] Healthcheck для WebUI/reverse proxy.
* [ ] `.env.production.example`.
* [ ] Reverse proxy docs:
  * [ ] Nginx;
  * [ ] Traefik;
  * [ ] HTTPS;
  * [ ] forwarded headers;
  * [ ] trusted proxies;
  * [ ] CORS_ALLOWED_ORIGINS.
* [ ] NixOS-friendly service example.
* [ ] Basic Prometheus metrics endpoint или documented future decision.
* [ ] Operations runbook обновлен под self-host.

## Acceptance criteria

* [ ] Новый пользователь может запустить проект через Docker Compose по документации.
* [ ] Проект можно безопасно поставить за reverse proxy.
* [ ] Secrets не попадают в git.
* [ ] Health/readiness checks работают.

---

# v0.11 — v1.0 Polish

## Goal

Довести core до ежедневного использования.

## Scope

* [ ] Пройти полный ручной сценарий: setup -> login -> account -> transaction -> transfer -> interest -> backup.
* [ ] Убрать устаревшие legacy CLI/docs, если они больше не являются основным путем.
* [ ] Проверить empty states в WebUI.
* [ ] Проверить mobile layout.
* [ ] Проверить accessibility basics: labels, focus states, keyboard flow.
* [ ] Добавить базовые reports:
  * [ ] monthly income/expense;
  * [ ] interest earned;
  * [ ] account balances;
  * [ ] recent activity.
* [ ] Финально сверить README, RUNNING, SECURITY docs, TODO.

## Acceptance criteria

* [ ] Приложением можно пользоваться для личного учета без ручных SQL-команд.
* [ ] Основной happy path покрыт E2E.
* [ ] Backup/restore задокументирован и проверен.
* [ ] Финансовые операции аудитируемы.
* [ ] v1.0 release notes готовы.

---

# v1.0 — Personal CapitalFlow Core Release

## Definition of done

* [ ] WebUI.
* [ ] PostgreSQL.
* [ ] Accounts.
* [ ] Transactions.
* [ ] Transfers.
* [ ] Categories.
* [ ] Savings/deposits.
* [ ] Interest rules.
* [ ] Interest accrual.
* [ ] Reports.
* [ ] Secure auth.
* [ ] Optional passkey login.
* [ ] Backup/restore.
* [ ] Docker/self-host docs.
* [ ] E2E tests for critical flows.
* [ ] NixOS-friendly deployment notes.

---

# After v1.0

Эти задачи важны, но не должны блокировать v1.0 core.

## Budgets and goals

* [ ] Budgets by category.
* [ ] Monthly limits.
* [ ] Goals: emergency fund, apartment, investments.
* [ ] Allocation calculator for income distribution.

## Multi-currency

* [ ] Base currency in user settings.
* [ ] Manual exchange rates.
* [ ] Historical FX rates.
* [ ] Fiat provider integration.
* [ ] Crypto rate provider.
* [ ] Converted dashboard totals.

## Investments

* [ ] Broker account details.
* [ ] Assets: stocks, funds, crypto.
* [ ] Buy/sell transactions.
* [ ] Dividends.
* [ ] Portfolio performance.

## LLM assistant

* [ ] Provider interface: Ollama/local first.
* [ ] Safe financial summary builder.
* [ ] Monthly report prompt.
* [ ] Spending explanation prompt.
* [ ] Budget suggestion prompt.
* [ ] No direct raw DB access for LLM.
* [ ] No data mutation without explicit user confirmation.

## Telegram bot

* [ ] Daily digest.
* [ ] Interest accrual notification.
* [ ] Quick expense entry.
* [ ] Budget warnings.
