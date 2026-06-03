# Operations Runbook

This page is the operational entrypoint.

## Health Checks

* `GET /health`: process is alive.
* `GET /ready`: dependencies are ready.
* `GET /metrics`: expvar metrics, including auth counters.

## Normal Deploy Checklist

1. Confirm CI is green.
2. Apply database migrations.
3. Start or roll the backend.
4. Check `/ready`.
5. Check `/metrics`.
6. Review logs for startup errors.

## Interest Jobs On VM

The VM deployment runs interest jobs from Docker Compose. NixOS timers are not used.

`interest-scheduler` runs once per day at `CAPITALFLOW_INTEREST_JOBS_TIME` when
`CAPITALFLOW_INTEREST_JOBS_ENABLED=true`. It runs:

* `daily_interest_accrual_job`
* `monthly_interest_accrual_job`
* `deposit_maturity_check_job`

Manual run from the VM:

```bash
cd /home/sunriseex/projects/CapitalFlow/deploy
docker compose --profile tools run -T --rm job-runner jobs run --name daily_interest_accrual_job
```

Each job uses a PostgreSQL advisory lock by job name, so concurrent duplicate starts
exit with `already running`. Jobs only select interest rules with the matching
`accrual_frequency`.

## Auth Checks

After deploying auth changes:

1. Run setup/login in a test environment.
2. Verify refresh rotation.
3. Verify logout revokes the refresh session.
4. Verify password change revokes all sessions.
5. Verify `auth_audit_events` receives records.
6. Verify `capitalflow_auth_events_total` changes in `/metrics`.

## Incident Pages

* [Auth Incident Response](Auth-Incident-Response)
* [Auth Observability](Auth-Observability)
