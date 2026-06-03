# TODO Fixes

## v0.6 Remaining Work

- [x] Add production background job wrappers for the PostgreSQL interest engine:
  - `daily_interest_accrual_job`
  - `monthly_interest_accrual_job`
  - `deposit_maturity_check_job`
- [x] Define runtime scheduling and locking for interest jobs in the VM deployment.
- [ ] Add top-up cutoff support to deposit rules if this must be enforced by the engine.
- [ ] Re-run race tests in the VM.