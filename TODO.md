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
* [x] Transfer audit model is good enough for cross-currency operations.
* [x] Idempotency behavior is tested for all financial mutations.
* [ ] E2E tests cover critical user flows.
* [ ] Backup/restore is available before the app is used with real data.
* [ ] Production/self-host deployment path is documented.

## Roadmap order

* [x] v0.5.5 Architecture Stabilization.
* [x] v0.5.6 Financial Auditability & Idempotency.
* [x] v0.5.7 Security Baseline Before Passkeys.
* [x] v0.5.8 Passkey Login / WebAuthn.
* [ ] v0.5.9 Frontend Reference Refactor.
* [ ] v0.6.0 E2E Testing Baseline.
* [ ] v0.6.1 Deposit & Capitalization Engine.
* [ ] v0.6.2 Backup / Restore / Operations.
* [ ] v0.6.3 Performance & Observability.
* [ ] v0.7 Import / Export.
* [ ] v0.8 Budgeting / Goals.
* [ ] v0.9 Analytics / Forecasts.
* [ ] v1.0 Personal CapitalFlow Core Release.
* [ ] v1.x LLM, investments, Telegram bot, advanced multi-currency.

Причина такого порядка: для финансового приложения опасно расширять функциональность поверх неполностью зафиксированных инвариантов. Сначала нужны auditability, idempotency, E2E, backup/restore и эксплуатационная надежность.

---

# v0.5.5 — Architecture Stabilization

## Goal

Зафиксировать архитектурные границы до дальнейшего роста и не допустить превращения сервисного слоя в неструктурированную смесь бизнес-правил.

## Scope

Существующую структуру не обязательно переименовывать сразу, но логические роли должны быть явными:

```text
internal/
  domain/
    money/
    account/
    transaction/
    transfer/
    interest/
    auth/
  services/
  repository/
  postgres/
  http/
```

* [x] `models` содержит данные.
* [x] `domain` содержит правила и инварианты для уже вынесенных областей.
* [x] `services` содержит сценарии использования.
* [x] `repository` содержит контракты доступа к БД.
* [x] `postgres` содержит реализацию доступа к PostgreSQL.
* [x] `handlers` содержит только HTTP-слой.
* [x] Проверки вроде «нельзя перевести деньги на тот же счет» живут в `TransferService` или domain validator, а не только в handler.
* [x] Полный domain scope ещё не завершён: `money`, `interest`, `auth` пока не выделены как отдельные domain packages.

## Architecture invariants

* [x] Любая user-facing финансовая операция принадлежит `user_id`.
* [x] Handler не содержит бизнес-правил.
* [x] Service не знает про HTTP DTO.
* [x] Repository не принимает HTTP DTO.
* [x] Money хранится как `decimal.Decimal` в Go и `NUMERIC` в PostgreSQL.
* [x] Currency scale валидируется на domain/service boundary.
* [x] Sub-minor значения запрещены для user-created financial operations.
* [x] Currency всегда нормализована и валидируется.
* [x] Все write-операции проходят через транзакцию БД.
* [x] Все опасные операции имеют audit/event trail.
* [x] Удаление финансовых данных либо запрещено, либо soft-delete/audit.

## Edge cases

* [x] Account принадлежит другому `user_id`.
* [x] Account archived, но по нему пытаются создать transaction.
* [x] Transaction с `amount = 0`.
* [x] Transaction с отрицательной суммой там, где это запрещено.
* [x] Currency в lowercase: `rub`, `usd`.
* [x] Currency нестандартная: `RUR`, `BTC`, `USDT` отклоняется в stable core.
* [x] Дата операции в будущем.
* [x] Дата операции до даты открытия счета.
* [x] Удаление transaction, которая участвует в transfer.
* [x] Повторный запрос после timeout.
* [x] Одновременное создание двух операций по одному счету.
* [x] Прямая service-level попытка создать `transfer_in` / `transfer_out` transaction вне transfer flow.

## Tests

* [x] Unit tests для domain validators.
* [x] Service tests без HTTP.
* [x] Handler tests только на контракт API.
* [x] Integration tests с PostgreSQL для write-flow.
* [x] Regression tests на найденные audit/concurrency bugs.
* [x] Architecture boundary tests / lint rules, которые не дают handler-слою снова начать решать финансовые правила.

## Acceptance criteria

* [x] Основные user-facing write-flow имеют понятный service-level сценарий.
* [x] Handler не решает финансовые правила.
* [x] Есть `docs/architecture/layers.md`.
* [x] Есть `docs/architecture/invariants.md`.
* [x] Новая фича добавляется по шаблону: model -> domain rule -> service -> repo -> handler -> tests.
* [x] Hard delete финансовых данных заменён на запрет, soft-delete или audit-backed deletion.
* [x] Legacy/internal write paths либо переведены на транзакции БД, либо явно задокументированы как исключения.

---

# v0.5.6 — Financial Auditability & Idempotency

## Goal

Сделать финансовые write-flow воспроизводимыми, атомарными и безопасными при повторных запросах до passkey, LLM, бюджетов и импорта.

## Transfer model

Текущая модель "перевод = две transaction rows" рабочая для MVP, но слабая для аудита. Нужна отдельная сущность `transfers`, где transfer — это одно business event, а две transaction rows — accounting legs.

```text
transfers
  id
  user_id
  from_account_id
  to_account_id
  from_transaction_id
  to_transaction_id
  from_amount_minor
  to_amount_minor
  from_currency
  to_currency
  exchange_rate
  exchange_rate_scale
  rate_provider
  rate_date
  fee_amount_minor
  fee_currency
  status
  idempotency_key
  created_at
  updated_at
```

### User cases

* [x] Перевод между двумя RUB-счетами.
* [x] Перевод RUB -> USD.
* [x] Перевод USD -> RUB.
* [x] Перевод RUB -> USDT.
* [x] Перевод с комиссией.
* [x] Перевод между своими счетами в разных банках.
* [x] Перевод на брокерский счет.
* [x] Перевод между archived и active account должен быть запрещен или явно ограничен.

### Transfer edge cases

* [x] `from_account_id == to_account_id`.
* [x] `from_amount <= 0`.
* [x] `to_amount <= 0`.
* [x] `exchange_rate` отсутствует при разных валютах.
* [x] `exchange_rate` указан при одинаковых валютах.
* [x] `exchange_rate = 0`.
* [x] `exchange_rate` слишком большой.
* [x] Потеря точности при конвертации.
* [x] Создалась только одна leg из двух.
* [x] Повторный запрос создает дубль.
* [x] Удаление одной leg ломает transfer.
* [x] Один account принадлежит другому `user_id`.

### Transfer tests

* [x] Same-currency transfer persists transfer row.
* [x] Cross-currency transfer persists rate and both legs.
* [x] Transfer rollback: если вторая leg не создалась, первая тоже не сохраняется.
* [x] Idempotent retry returns previous result.
* [x] Same idempotency key + different payload returns conflict.
* [x] Transfer cannot be partially deleted.
* [x] Transfer list shows both business event and legs.

## Idempotency keys

Для финансового приложения idempotency — обязательное свойство, а не nice-to-have.

```text
idempotency_keys
  id
  user_id
  key
  request_hash
  endpoint
  status
  response_status
  response_body
  locked_until
  created_at
  updated_at
  expires_at
```

### Endpoints

* [x] `POST /api/transactions`.
* [x] `POST /api/transfers`.
* [x] `POST /api/accounts/{id}/accrue-interest`.
* [x] `POST /api/accounts/{id}/recalculate-interest`.
* [ ] Future: import.
* [ ] Future: bulk operations.

### Idempotency edge cases

* [x] Клиент отправил один и тот же request дважды.
* [x] Первый request успел записать данные, но клиент получил timeout.
* [x] Два одинаковых request пришли одновременно.
* [x] Один idempotency key используется с другим body.
* [x] Idempotency key истек.
* [x] Request упал до commit.
* [x] Request упал после commit, но до ответа.

## Acceptance criteria

* [x] Повтор POST-запроса не создает дубль.
* [x] Concurrent retry безопасен.
* [x] Idempotency работает на уровне БД, а не только в памяти.
* [x] Cross-currency transfer audit не зависит от текущего курса валют.
* [x] Есть `docs/architecture/idempotency.md`.

---

# v0.5.7 — Security Baseline Before Passkeys

## Goal

Проверить и задокументировать security baseline до WebAuthn, особенно для self-hosted запуска за Nginx/Traefik.

## Scope

* [x] JWT secret не имеет дефолтного production значения.
* [x] Access token TTL короткий.
* [x] Refresh token хранится только hashed.
* [x] Refresh cookie: Secure, HttpOnly, SameSite, Path.
* [x] Logout отзывает refresh session.
* [x] Password change отзывает все refresh sessions.
* [x] Setup первого пользователя нельзя вызвать повторно.
* [x] Rate limit работает за reverse proxy.
* [x] Реальный client IP корректно определяется через trusted proxy config.
* [x] CORS не разрешает wildcard credentials.
* [x] CSRF модель явно описана.
* [x] Security headers добавлены.

## Self-host configuration

```env
TRUSTED_PROXIES=127.0.0.1,172.16.0.0/12
PUBLIC_ORIGIN=https://capitalflow.example.com
COOKIE_SECURE=true
COOKIE_SAMESITE=strict
WEBAUTHN_RP_ID=capitalflow.example.com
WEBAUTHN_ORIGINS=https://capitalflow.example.com
```

## Edge cases

* [x] Login через reverse proxy.
* [x] Login напрямую по IP должен быть запрещен или явно dev-only.
* [x] Неверный `X-Forwarded-For` не должен обходить rate limit.
* [x] CORS preflight не ломает auth.
* [x] Refresh cookie не отправляется на `/api/*`, если `Path=/auth`.
* [x] Access token expired, refresh успешен.
* [x] Refresh token reused после rotation.
* [x] Пользователь сменил пароль на одном устройстве, остальные сессии умерли.

## Acceptance criteria

* [x] Есть `docs/security/reverse-proxy.md`.
* [x] Есть `docs/security/csrf.md`.
* [x] Есть integration tests для auth за trusted proxy.
* [x] Есть security tests на refresh reuse, logout, password change, CORS, CSRF.

---

# v0.5.8 — Passkey Login / WebAuthn

## Goal

Добавить passkey login как дополнительный способ входа после security baseline. Password login остается fallback до появления нормального recovery flow.

## Backend tasks

* [x] Добавить WebAuthn config.
* [x] Добавить `passkey_credentials`.
* [x] Добавить `webauthn_challenges`.
* [x] Добавить `PasskeyService`.
* [x] Добавить `PasskeyRepository`.
* [x] Добавить registration options endpoint.
* [x] Добавить registration verify endpoint.
* [x] Добавить login options endpoint.
* [x] Добавить login verify endpoint.
* [x] Интегрировать successful passkey login в текущий refresh session flow.

## Frontend tasks

* [x] Login screen: Sign in with passkey.
* [x] Settings -> Security -> Passkeys.
* [x] Add passkey.
* [x] Rename passkey.
* [x] Delete passkey.
* [x] Browser not supported state.
* [x] User cancelled state.
* [x] Safe generic error state.

## Security edge cases

* [x] Replayed challenge rejected.
* [x] Expired challenge rejected.
* [x] Challenge from another user rejected.
* [x] Wrong origin rejected.
* [x] Wrong rpID rejected.
* [x] Revoked credential rejected.
* [x] Credential ID collision rejected.
* [x] Passkey registration without active session rejected.
* [x] First passkey add requires fresh session/password confirmation.
* [x] Deleted passkey cannot login.

## Acceptance criteria

* [x] Пользователь может добавить passkey.
* [x] Пользователь может войти по passkey.
* [x] Password login остается fallback.
* [x] Можно иметь несколько passkeys.
* [x] Можно удалить passkey.
* [x] Passkey login создает обычную refresh session.
* [x] Все passkey-события пишутся в auth audit log.
* [x] Есть unit, handler, security и E2E smoke tests.

---

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

# v0.6.0 — E2E Testing Baseline

## Goal

Добавить end-to-end тесты до активного расширения фич, чтобы проверять реальные пользовательские сценарии через браузер, API, auth, PostgreSQL и routing вместе.

## P0 E2E

* [ ] First setup user.
* [ ] Login.
* [ ] Session bootstrap after reload.
* [ ] Logout.
* [ ] Create account.
* [ ] Create income transaction.
* [ ] Create expense transaction.
* [ ] Create transfer.
* [ ] Dashboard updates after operations.
* [ ] Manual interest accrual.
* [ ] Duplicate interest accrual does not duplicate data.

## P1 E2E

* [ ] Passkey add/login/delete через virtual authenticator.
* [ ] Theme persistence.
* [ ] Mobile dashboard smoke.
* [ ] Empty state without accounts.
* [ ] Archived account behavior.
* [ ] Transaction filters.
* [ ] Date filters.
* [ ] Account details chart.

## Test infrastructure

* [ ] `docker-compose.e2e.yml`.
* [ ] `web/playwright.config.ts`.
* [ ] `web/tests/e2e/`.
* [ ] `docs/testing/e2e.md`.

## CI

* [ ] `backend-tests`.
* [ ] `frontend-lint-build`.
* [ ] `migration-check`.
* [ ] `e2e`.

## Acceptance criteria

* [ ] `npm run test:e2e` работает локально.
* [ ] E2E использует отдельную PostgreSQL DB.
* [ ] CI сохраняет trace/screenshot/video только при падении.
* [ ] E2E не зависит от порядка тестов.
* [ ] Деньги проверяются точными minor units.
* [ ] Даты фиксируются через controlled clock/test helpers.

---

# v0.6.1 — Deposit & Capitalization Engine

## Goal

Сделать объяснимый engine для процентов, вкладов, капитализации, пересчета и background jobs.

## Domain cases

* [ ] Накопительный счет с daily accrual + daily capitalization.
* [ ] Накопительный счет с monthly capitalization.
* [ ] Срочный вклад без пополнения.
* [ ] Срочный вклад с пополнением до даты.
* [ ] Срочный вклад с выплатой процентов в конце срока.
* [ ] Промо-ставка до даты, потом базовая.
* [ ] Закрытие вклада в дату окончания.
* [ ] Forecast без записи в БД.
* [ ] Recalculate с удалением/пересозданием generated accruals.

## Edge cases

* [ ] Leap year: 2024/2028.
* [ ] `actual_365` vs `actual_366` vs `actual_actual`.
* [ ] Promo end date совпадает с accrual date.
* [ ] Rule start date в будущем.
* [ ] Rule end date раньше start date.
* [ ] Несколько active rules пересекаются.
* [ ] Баланс отрицательный.
* [ ] Balance changed after interest was already accrued.
* [ ] Recalculate после удаления transaction.
* [ ] Повторный запуск daily job.
* [ ] Два job запущены одновременно.

## Job architecture

```text
job_runs
  id
  job_name
  run_date
  status
  started_at
  finished_at
  error

job_locks
  job_name
  locked_until
  locked_by
```

## Acceptance criteria

* [ ] `daily_interest_accrual_job` idempotent.
* [ ] `monthly_interest_accrual_job` idempotent.
* [ ] `deposit_maturity_check_job` idempotent.
* [ ] Повторный запуск job безопасен.
* [ ] Concurrent job не создает дубли.
* [ ] Recalculate объясним и обратим.
* [ ] Есть `docs/domain/interest-engine.md`.

---

# v0.6.2 — Backup / Restore / Operations

## Goal

Поднять backup/restore и эксплуатационные задачи выше бюджетов, LLM и инвестиций. Финансовый сервис без restore-теста нельзя считать готовым к реальным данным.

## Tasks

* [ ] `pg_dump` backup command.
* [ ] Restore command на отдельную test DB.
* [ ] Manual backup button in Settings.
* [ ] Backup before import.
* [ ] Backup before bulk delete.
* [ ] Backup before restore.
* [ ] Retention policy.
* [ ] Docker volumes documented.
* [ ] NixOS systemd service example.
* [ ] NixOS backup timer example.
* [ ] Production docker-compose.
* [ ] Healthcheck for backend container.

## Edge cases

* [ ] Backup directory is not writable.
* [ ] PostgreSQL unavailable.
* [ ] Backup file corrupted.
* [ ] Restore version older than current migrations.
* [ ] Restore into non-empty DB.
* [ ] Backup contains secrets.
* [ ] Backup contains personal financial data.
* [ ] Disk full during backup.

## Acceptance criteria

* [ ] Можно создать backup одной командой.
* [ ] Можно восстановить backup на чистую DB.
* [ ] Restore регулярно проверяется в CI или локальном script.
* [ ] Есть `docs/operations/backup-restore.md`.
* [ ] Production docker-compose не хранит secrets в image.

---

# v0.6.3 — Performance & Observability

## Goal

Не оптимизировать преждевременно, но заранее защититься от плохих query, list и observability-паттернов.

## Performance risks

* [ ] Dashboard может начать делать много тяжелых SUM-запросов.
* [ ] Account balance может пересчитываться из всех transactions каждый раз.
* [ ] Long transaction history может замедлить UI.
* [ ] Recalculate interest может блокировать account.
* [ ] Import может создать много duplicate checks.
* [ ] FX conversion может дергать внешний provider слишком часто.

## Scope

* [ ] Cursor pagination везде, где есть списки.
* [ ] Индексы под реальные query patterns.
* [ ] `EXPLAIN ANALYZE` для dashboard queries.
* [ ] Balance snapshots для тяжелых периодов.
* [ ] Ограничение max `limit`.
* [ ] Timeout на DB queries.
* [ ] Context propagation.
* [ ] Metrics: request duration, DB duration, auth failures, job duration.
* [ ] Structured logs с `request_id`.
* [ ] Audit logs отдельно от application logs.

## Load tests

* [ ] 10 accounts, 1_000 transactions.
* [ ] 50 accounts, 10_000 transactions.
* [ ] 100 accounts, 100_000 transactions.
* [ ] Dashboard p95 latency.
* [ ] Transaction list p95 latency.
* [ ] Balance calculation p95 latency.
* [ ] Import 10_000 rows.

## Acceptance criteria

* [ ] Dashboard не деградирует резко на 10k transactions.
* [ ] Все list endpoints имеют pagination.
* [ ] Есть `docs/performance/query-patterns.md`.
* [ ] Есть Prometheus metrics или хотя бы `/metrics`-compatible design.

---

# v0.7 — Import / Export

## Goal

Добавить import/export раньше budgeting и analytics, чтобы новые продуктовые функции работали на реальных данных.

## Import cases

* [ ] CSV from bank.
* [ ] Manual CSV.
* [ ] Custom column mapping.
* [ ] Preview before import.
* [ ] Category auto-suggestion.
* [ ] Duplicate detection.
* [ ] Import rollback.
* [ ] Import report.

## Import edge cases

* [ ] Different date formats.
* [ ] Decimal comma: `123,45`.
* [ ] Decimal dot: `123.45`.
* [ ] Negative expense format.
* [ ] Separate debit/credit columns.
* [ ] Currency missing.
* [ ] Unknown account.
* [ ] Duplicate operation.
* [ ] Encoding: UTF-8 / Windows-1251.
* [ ] Huge file.

## Export cases

* [ ] CSV transactions export.
* [ ] JSON full export.
* [ ] Markdown monthly report.
* [ ] Backup export.

## Acceptance criteria

* [ ] Import always has preview.
* [ ] Import can be cancelled before write.
* [ ] Import creates audit event.
* [ ] Import can be traced by `import_batch_id`.
* [ ] Duplicate detection works.
* [ ] Export does not leak secrets.

---

# v0.8 — Budgeting / Goals

## Goal

Сначала реализовать обычные budgets/goals, а smart recommendations оставить на потом.

## Entities

* [ ] `budgets`.
* [ ] `budget_categories`.
* [ ] `goals`.
* [ ] `goal_contributions`.

## User cases

* [ ] Месячный бюджет по категориям.
* [ ] Лимит на еду.
* [ ] Лимит на транспорт.
* [ ] Финансовая подушка.
* [ ] Цель "квартира".
* [ ] Цель "обучение".
* [ ] Связать goal с account.
* [ ] Видеть progress на dashboard.

## Edge cases

* [ ] Budget category deleted.
* [ ] Category renamed.
* [ ] Transaction moved to another category.
* [ ] Month boundary.
* [ ] Timezone issue near midnight.
* [ ] Refund reduces expense.
* [ ] Transfer не считается расходом.
* [ ] Goal linked account archived.

## Acceptance criteria

* [ ] Можно создать бюджет на месяц.
* [ ] Можно задать лимиты по категориям.
* [ ] Dashboard показывает budget progress.
* [ ] Transfer не ломает spending analytics.
* [ ] Goal progress считается объяснимо.

---

# v0.9 — Analytics / Forecasts

## Goal

Добавить отчеты и forecasts после budgets/goals/import. Analytics must be reproducible: отчет за май, построенный 1 июня, должен совпадать 15 июня, если пользователь не менял данные вручную.

## Reports

* [ ] Income vs expense by month.
* [ ] Category spending.
* [ ] Net worth over time.
* [ ] Savings rate.
* [ ] Interest income.
* [ ] Goal forecast.
* [ ] Emergency fund coverage.
* [ ] Deposit forecast.

## Edge cases

* [ ] Cross-currency totals.
* [ ] Historical FX rate missing.
* [ ] Transaction backdated.
* [ ] Category changed after report.
* [ ] Archived account.
* [ ] Investment account excluded/included.
* [ ] Refund.
* [ ] Internal transfer.

## Acceptance criteria

* [ ] Отчеты отделяют income/expense от transfers.
* [ ] Base currency conversion использует persisted historical rates.
* [ ] Есть monthly report snapshot или reproducible query strategy.
* [ ] User может понять, из каких данных получился отчет.

---

# v1.0 — Personal CapitalFlow Core Release

## Goal

v1.0 — стабильная версия для ежедневного личного использования. Она не должна включать все будущие идеи.

## Minimum scope

* [ ] Accounts.
* [ ] Transactions.
* [ ] Transfers with auditability.
* [ ] Categories.
* [ ] Interest rules.
* [ ] Deposit/capitalization engine.
* [ ] Dashboard.
* [ ] Reports basic.
* [ ] Auth.
* [ ] Optional passkey.
* [ ] E2E critical flows.
* [ ] Backup/restore.
* [ ] Docker production compose.
* [ ] NixOS-friendly service example.
* [ ] Reverse proxy docs.
* [ ] Operations docs.

## Out of v1.0 scope

* [ ] LLM assistant.
* [ ] Telegram bot.
* [ ] Advanced investments.
* [ ] Smart budget AI-like recommendations.
* [ ] Public multi-user SaaS behavior.
* [ ] Complex broker integrations.

---

# v1.x — LLM, investments, Telegram bot, advanced multi-currency

## Goal

Развивать расширенные возможности только после auditability, backup, E2E и reports.

## LLM assistant foundation

LLM не должна иметь прямой доступ к сырой БД. Она должна получать подготовленный safe summary. Пользователь должен явно включать интеграцию, cloud-модели не должны получать sensitive data без предупреждения, любые изменения требуют подтверждения.

```text
LLMProvider
  OllamaProvider
  OpenAICompatibleProvider
  MockProvider

ContextBuilder
  MonthlyFinancialSummaryBuilder
  CategorySpendingSummaryBuilder
  GoalProgressSummaryBuilder
  DepositForecastSummaryBuilder
```

### Forbidden

* [ ] LLM напрямую вызывает repository.
* [ ] LLM сама создает transaction.
* [ ] LLM видит raw transaction descriptions без privacy mode.
* [ ] LLM отправляет данные в cloud без явного consent.

### Allowed

* [ ] Explain my month.
* [ ] Explain spending growth.
* [ ] Suggest next budget.
* [ ] Generate monthly report.
* [ ] Find anomalies.
* [ ] Explain deposit forecast.

### Acceptance criteria

* [ ] Mock provider работает в tests.
* [ ] Ollama local работает без cloud.
* [ ] Cloud provider требует explicit opt-in.
* [ ] LLM получает summary, а не raw DB dump.
* [ ] Любое write-action требует user confirmation.

## Other v1.x directions

* [ ] Investments and portfolio tracking.
* [ ] Advanced multi-currency and historical FX management.
* [ ] Telegram bot for daily digest, notifications and quick entry.
