## Summary

<!-- Briefly describe what changed and why. Keep it practical. -->

## Scope

<!-- Mark the areas touched by this PR. -->

* [ ] Backend / API
* [ ] Database / migrations
* [ ] Web UI
* [ ] Auth / security
* [ ] Imports / parsers
* [ ] Money calculations / currency handling
* [ ] Docker / self-hosting / infra
* [ ] Tests only
* [ ] Docs only

## Type of change

* [ ] Feature
* [ ] Bug fix
* [ ] Refactor
* [ ] UI/UX improvement
* [ ] i18n / localization
* [ ] Test coverage
* [ ] Documentation
* [ ] CI / build / tooling

## What changed

<!-- List the main changes. Prefer concrete bullets. -->

*

## Why this approach

<!-- Explain important decisions, trade-offs, or rejected alternatives. For financial logic, this section matters. -->

*

## CapitalFlow safety checklist

<!-- Check only what applies. Leave non-applicable items unchecked or explain below. -->

* [ ] Money calculations keep decimal precision and do not use unsafe floating-point logic for stored amounts.
* [ ] Currency codes remain stable in API/DB/storage; display-only formatting is isolated to UI.
* [ ] Transaction, transfer, subscription, or account invariants are preserved.
* [ ] Import/backfill behavior is safe and does not silently duplicate or mutate financial history.
* [ ] Sensitive data is not logged or exposed in API responses.
* [ ] Auth/session behavior is not weakened.
* [ ] Self-hosted deployment behavior remains compatible with Docker/reverse proxy usage.

## Web UI checklist

* [ ] UI works in light and dark themes.
* [ ] UI works on mobile, tablet, and desktop widths.
* [ ] Interactive controls have accessible names.
* [ ] Keyboard navigation still works where relevant.
* [ ] Loading, empty, and error states are handled.
* [ ] RU/EN text is added or updated when visible UI text changes.
* [ ] Screenshots or short recordings are attached for visible UI changes.

## Backend checklist

* [ ] `go test ./...` passes.
* [ ] Code is formatted with `gofmt` / `goimports`.
* [ ] API changes are reflected in OpenAPI/types where needed.
* [ ] Migrations are forward-safe and reviewed.
* [ ] New or changed business rules have tests.
* [ ] Logs use appropriate levels and do not expose secrets.

## Local checks

<!-- Mark what you actually ran. -->

* [ ] Backend tests: `go test ./...`
* [ ] Web tests: `cd web && npm run test`
* [ ] Web build: `cd web && npm run build`
* [ ] Lint/typecheck:
* [ ] Manual browser check:
* [ ] Other:

## How to test manually

<!-- Give short, reproducible steps. -->

1.
2.
3.

## Screenshots / recordings

<!-- Required for visible Web UI changes. Add before/after when useful. -->

## Notes for reviewer

<!-- Mention known limitations, follow-up PRs, or anything intentionally left out. -->
