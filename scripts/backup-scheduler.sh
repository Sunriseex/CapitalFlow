#!/usr/bin/env sh
set -eu

if [ "${CAPITALFLOW_BACKUPS_ENABLED:-true}" != "true" ]; then
  echo "backup scheduler disabled"
  sleep infinity
fi

backup_time="${CAPITALFLOW_BACKUP_TIME:-02:30}"
backup_timeout="${CAPITALFLOW_BACKUP_TIMEOUT:-30m}"
backup_dir="${CAPITALFLOW_BACKUP_DIR:-/backups}"
retention_count="${CAPITALFLOW_BACKUP_RETENTION_COUNT:-14}"
run_once="${CAPITALFLOW_BACKUP_RUN_ONCE:-false}"
capitalflow_bin="${CAPITALFLOW_BIN:-/usr/local/bin/capitalflow}"
schedule_bin="${CAPITALFLOW_BACKUP_SCHEDULE_BIN:-/usr/local/bin/capitalflow-backup-schedule}"
retention_bin="${CAPITALFLOW_BACKUP_RETENTION_BIN:-/usr/local/bin/capitalflow-backup-retention}"
heartbeat_file="${CAPITALFLOW_BACKUP_HEARTBEAT_FILE:-/tmp/capitalflow-backup-scheduler.heartbeat}"

if ! date -d "2000-01-01 ${backup_time}" +%s >/dev/null 2>&1; then
  echo "invalid CAPITALFLOW_BACKUP_TIME: ${backup_time}" >&2
  exit 1
fi
case "${retention_count}" in
  *[!0-9]* | 0 | "")
    echo "CAPITALFLOW_BACKUP_RETENTION_COUNT must be a positive integer" >&2
    exit 1
    ;;
esac
if [ ! -x "${capitalflow_bin}" ]; then
  echo "capitalflow binary not found or not executable: ${capitalflow_bin}" >&2
  exit 1
fi
if [ ! -x "${retention_bin}" ]; then
  echo "backup retention binary not found or not executable: ${retention_bin}" >&2
  exit 1
fi

mkdir -p "${backup_dir}"
chmod 700 "${backup_dir}"

run_backup() {
  timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
  output="${backup_dir}/capitalflow-${timestamp}.zip"
  date -Iseconds > "${heartbeat_file}"
  echo "creating backup: ${output}"
  if ! timeout "${backup_timeout}" "${capitalflow_bin}" backup \
    --output "${output}" \
    --timeout "${backup_timeout}"; then
    echo "backup failed: ${output}" >&2
    return 1
  fi
  "${retention_bin}" "${backup_dir}" "${retention_count}"
  date -Iseconds > "${heartbeat_file}"
  echo "backup complete: ${output}"
}

if [ "${run_once}" = "true" ]; then
  run_backup
  exit 0
fi

if [ ! -x "${schedule_bin}" ]; then
  echo "backup schedule binary not found or not executable: ${schedule_bin}" >&2
  exit 1
fi

echo "backup scheduler enabled at ${backup_time} ${TZ:-UTC}; keeping ${retention_count} archives"
while true; do
  date -Iseconds > "${heartbeat_file}"
  sleep_seconds="$("${schedule_bin}" "${backup_time}")"
  echo "next backup in ${sleep_seconds}s"
  sleep "${sleep_seconds}"
  if ! run_backup; then
    date -Iseconds > "${heartbeat_file}"
  fi
done
