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
started_file="${CAPITALFLOW_BACKUP_STARTED_FILE:-/tmp/capitalflow-backup-scheduler.started}"
success_file="${CAPITALFLOW_BACKUP_SUCCESS_FILE:-/tmp/capitalflow-backup-scheduler.last-success}"
status_file="${CAPITALFLOW_BACKUP_STATUS_FILE:-/tmp/capitalflow-backup-scheduler.status}"
replication_command="${CAPITALFLOW_BACKUP_REPLICATION_COMMAND:-}"
replication_required="${CAPITALFLOW_BACKUP_REPLICATION_REQUIRED:-false}"
restore_drill_enabled="${CAPITALFLOW_RESTORE_DRILL_ENABLED:-false}"
restore_drill_interval_days="${CAPITALFLOW_RESTORE_DRILL_INTERVAL_DAYS:-7}"
restore_drill_bin="${CAPITALFLOW_RESTORE_DRILL_BIN:-/usr/local/bin/capitalflow-restore-drill}"
restore_drill_marker="${CAPITALFLOW_RESTORE_DRILL_MARKER:-/backups/.restore-drill-last-success}"
disk_guard_bin="${CAPITALFLOW_DISK_GUARD_BIN:-/usr/local/bin/capitalflow-disk-guard}"
disk_max_used_percent="${CAPITALFLOW_DISK_MAX_USED_PERCENT:-90}"
disk_min_free_mb="${CAPITALFLOW_DISK_MIN_FREE_MB:-1024}"

if ! date -d "2000-01-01 ${backup_time}" +%s >/dev/null 2>&1; then
  echo "invalid CAPITALFLOW_BACKUP_TIME: ${backup_time}" >&2
  exit 1
fi
if [ "${replication_required}" = "true" ] && [ -z "${replication_command}" ]; then
  echo "off-host replication is required but CAPITALFLOW_BACKUP_REPLICATION_COMMAND is empty" >&2
  exit 1
fi
case "${retention_count}" in
  *[!0-9]* | 0 | "")
    echo "CAPITALFLOW_BACKUP_RETENTION_COUNT must be a positive integer" >&2
    exit 1
    ;;
esac
case "${restore_drill_interval_days}" in
  *[!0-9]* | 0 | "")
    echo "CAPITALFLOW_RESTORE_DRILL_INTERVAL_DAYS must be a positive integer" >&2
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
date -Iseconds > "${started_file}"
date -Iseconds > "${heartbeat_file}"

set_status() {
  status="$1"
  detail="${2:-}"
  printf 'status=%s\ntimestamp=%s\ndetail=%s\n' "${status}" "$(date -Iseconds)" "${detail}" > "${status_file}"
}

wait_with_heartbeat() {
  remaining="$1"
  while [ "${remaining}" -gt 60 ]; do
    sleep 60
    date -Iseconds > "${heartbeat_file}"
    remaining=$((remaining - 60))
  done
  if [ "${remaining}" -gt 0 ]; then
    sleep "${remaining}"
  fi
  date -Iseconds > "${heartbeat_file}"
}

restore_drill_due() {
  [ "${restore_drill_enabled}" = "true" ] || return 1
  [ -x "${restore_drill_bin}" ] || {
    echo "restore drill binary not found or not executable: ${restore_drill_bin}" >&2
    return 0
  }
  [ -f "${restore_drill_marker}" ] || return 0
  now="$(date +%s)"
  last="$(stat -c %Y "${restore_drill_marker}")"
  [ $((now - last)) -ge $((restore_drill_interval_days * 86400)) ]
}

run_backup() {
  if [ -x "${disk_guard_bin}" ]; then
    "${disk_guard_bin}" "${backup_dir}" "${disk_max_used_percent}" "${disk_min_free_mb}"
  fi
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
  if [ -n "${replication_command}" ]; then
    echo "replicating backup off-host: ${output}"
    if ! sh -c "${replication_command}" capitalflow-backup-replication "${output}"; then
      echo "off-host backup replication failed: ${output}" >&2
      return 1
    fi
  fi
  if restore_drill_due; then
    echo "running scheduled restore drill"
    if ! CAPITALFLOW_BACKUP_DIR="${backup_dir}" "${restore_drill_bin}"; then
      echo "scheduled restore drill failed" >&2
      return 1
    fi
    touch "${restore_drill_marker}"
  fi
  date -Iseconds > "${success_file}"
  set_status success "${output}"
  date -Iseconds > "${heartbeat_file}"
  echo "backup complete: ${output}"
}

if [ "${run_once}" = "true" ]; then
  if ! run_backup; then
    set_status failure backup
    exit 1
  fi
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
  wait_with_heartbeat "${sleep_seconds}"
  if ! run_backup; then
    set_status failure backup
    date -Iseconds > "${heartbeat_file}"
  fi
done
