# Operations Runbook

This page is the operational entrypoint.

## Health Checks

* `GET /health`: process is alive.
* `GET /ready`: dependencies are ready.
* `GET /metrics`: allowlisted expvar metrics for auth, HTTP traffic and the DB pool.

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
  -v "${CAPITALFLOW_BACKUP_HOST_DIR:-${HOME}/backups/capitalflow}:/backups" \
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
  -v "${CAPITALFLOW_BACKUP_HOST_DIR:-${HOME}/backups/capitalflow}:/backups:ro" \
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
* `CAPITALFLOW_BACKUP_HOST_DIR=$HOME/backups/capitalflow`
* `CAPITALFLOW_BACKUP_UID` and `CAPITALFLOW_BACKUP_GID` default to the deploy owner

The scheduler writes UTC timestamped archives atomically, keeps the newest
configured number, and is healthy only while both its heartbeat and last
successful backup are fresh. The interest scheduler uses the same rule. A
permanent job failure therefore makes its container unhealthy after
`CAPITALFLOW_BACKUP_SUCCESS_MAX_AGE` or
`CAPITALFLOW_INTEREST_SUCCESS_MAX_AGE` (30 hours by default).

Off-host replication is provider-neutral. Configure an executable shell command
that receives the completed local archive as `$1`. The backup is not marked
successful when that command fails. For example:

```env
CAPITALFLOW_BACKUP_REPLICATION_COMMAND='/replication/rclone --config /replication/rclone.conf copyto "$1" "offsite:capitalflow/$(basename "$1")"'
```

The read-only `CAPITALFLOW_BACKUP_REPLICATION_HOST_DIR` mount is available as
`/replication`; place the chosen provider's executable and protected config
there. Leaving the command empty keeps local-only backups and
must be treated as an unresolved disaster-recovery risk.
Replication is required by default in production. The scheduler fails fast
(and the container alerts through its health/restart state) when the command is
absent. An operator may set `CAPITALFLOW_BACKUP_REPLICATION_REQUIRED=false`
only for an explicitly accepted temporary local-only deployment.

Production also runs a restore drill every seven days by default
(`CAPITALFLOW_RESTORE_DRILL_ENABLED=true`). It restores the newest archive into
a temporary database, verifies the schema, and drops that database. A failed
drill fails the backup cycle and eventually makes the scheduler unhealthy.
The durable `.restore-drill-last-success` marker is stored beside the archives.
Retention only removes files matching `capitalflow-*.zip`.

Every VM deploy also creates
`capitalflow-<UTC timestamp>-pre-migration.zip` before Goose runs. A backup
failure stops the deploy before the schema changes. Fresh installations skip
this step until `goose_db_version` exists. Pre-migration archives participate
in the same retention policy.

Check scheduler state on the VM:

```bash
cd /home/sunriseex/projects/CapitalFlow/deploy
docker compose ps backup-scheduler
docker compose logs --tail 100 backup-scheduler
ls -l "${CAPITALFLOW_BACKUP_HOST_DIR:-${HOME}/backups/capitalflow}"
docker compose exec backup-scheduler cat /operations/capitalflow-backup-scheduler.status
docker compose exec interest-scheduler cat /operations/capitalflow-interest-scheduler.status
```

Deploy and backup runs stop before writes when disk usage reaches
`CAPITALFLOW_DISK_MAX_USED_PERCENT` (90 by default) or free space falls below
`CAPITALFLOW_DISK_MIN_FREE_MB`. Treat a failed disk guard or an unhealthy
scheduler container as an alert. External monitoring should alert on container
health, `/ready`, HTTP 5xx growth, DB pool saturation, and filesystem usage.

Deploys validate that the API image label, Web image label, CLI version and
`/health` version are identical. If a migration command fails after services
were stopped, the deploy trap restarts exactly the services that were running
before migration.

## Incident Pages

* [Auth Incident Response](Auth-Incident-Response)
* [Auth Observability](Auth-Observability)
