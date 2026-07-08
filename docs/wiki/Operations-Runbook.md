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

## Backup and Restore

The API image includes `pg_dump`, `pg_restore`, and the `capitalflow` admin CLI.
Create backups on the VM in a directory that is copied off-host, for example by
Syncthing:

```bash
cd /home/sunriseex/projects/CapitalFlow/deploy
docker compose --profile tools run -T --rm \
  --user "${CAPITALFLOW_BACKUP_UID:-$(id -u)}:${CAPITALFLOW_BACKUP_GID:-$(id -g)}" \
  -v /srv/backups/capitalflow:/backups \
  job-runner backup --output /backups/capitalflow-$(date -u +%Y%m%dT%H%M%SZ).zip
```

Each archive contains a PostgreSQL custom dump and a manifest with the app
version, schema version, creation time, base currency, format version, and a
SHA-256 checksum. The final archive is written atomically with mode `0600`.

Restore only into a newly created, empty database. The command refuses to
overwrite a database containing public tables:

```bash
docker compose --profile tools run -T --rm \
  --user "${CAPITALFLOW_BACKUP_UID:-$(id -u)}:${CAPITALFLOW_BACKUP_GID:-$(id -g)}" \
  -v /srv/backups/capitalflow:/backups:ro \
  job-runner restore \
  --input /backups/capitalflow-20260708T050000Z.zip \
  --database-url "$RESTORE_DATABASE_URL"
```

After restore, the command verifies that the restored schema version matches
the archive manifest. Test restore regularly; an untested archive is not a
recovery plan.

Production Compose runs `backup-scheduler` daily. Defaults:

* `CAPITALFLOW_BACKUPS_ENABLED=true`
* `CAPITALFLOW_BACKUP_TIME=02:30`
* `CAPITALFLOW_BACKUP_TIMEOUT=30m`
* `CAPITALFLOW_BACKUP_RETENTION_COUNT=14`
* `CAPITALFLOW_BACKUP_HOST_DIR=/srv/backups/capitalflow`
* `CAPITALFLOW_BACKUP_UID` and `CAPITALFLOW_BACKUP_GID` default to the deploy owner

The scheduler writes UTC timestamped archives atomically, keeps the newest
configured number, and exposes a heartbeat health check. Point
`CAPITALFLOW_BACKUP_HOST_DIR` at a directory replicated off-host by Syncthing
or another backup system. Retention only removes files matching
`capitalflow-*.zip`.

Check scheduler state on the VM:

```bash
cd /home/sunriseex/projects/CapitalFlow/deploy
docker compose ps backup-scheduler
docker compose logs --tail 100 backup-scheduler
ls -l "${CAPITALFLOW_BACKUP_HOST_DIR:-/srv/backups/capitalflow}"
```

## Incident Pages

* [Auth Incident Response](Auth-Incident-Response)
* [Auth Observability](Auth-Observability)
