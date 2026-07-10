#!/usr/bin/env sh
set -eu

if [ "${CAPITALFLOW_INTEREST_JOBS_ENABLED:-true}" != "true" ]; then
  echo "interest jobs scheduler disabled"
  sleep infinity
fi

job_time="${CAPITALFLOW_INTEREST_JOBS_TIME:-03:15}"
job_timeout="${CAPITALFLOW_INTEREST_JOB_TIMEOUT:-30m}"
capitalflow_bin="${CAPITALFLOW_BIN:-/usr/local/bin/capitalflow}"
schedule_bin="${CAPITALFLOW_INTEREST_SCHEDULE_BIN:-/usr/local/bin/capitalflow-interest-schedule}"
heartbeat_file="${CAPITALFLOW_INTEREST_HEARTBEAT_FILE:-/tmp/capitalflow-interest-scheduler.heartbeat}"
started_file="${CAPITALFLOW_INTEREST_STARTED_FILE:-/tmp/capitalflow-interest-scheduler.started}"
success_file="${CAPITALFLOW_INTEREST_SUCCESS_FILE:-/tmp/capitalflow-interest-scheduler.last-success}"
status_file="${CAPITALFLOW_INTEREST_STATUS_FILE:-/tmp/capitalflow-interest-scheduler.status}"
run_once="${CAPITALFLOW_INTEREST_RUN_ONCE:-false}"

if ! date -d "2000-01-01 ${job_time}" +%s >/dev/null 2>&1; then
  echo "invalid CAPITALFLOW_INTEREST_JOBS_TIME: ${job_time}" >&2
  exit 1
fi

echo "interest jobs scheduler enabled at ${job_time} ${TZ:-UTC}"
date -Iseconds > "${started_file}"
date -Iseconds > "${heartbeat_file}"

set_status() {
  printf 'status=%s\ntimestamp=%s\ndetail=%s\n' "$1" "$(date -Iseconds)" "${2:-}" > "${status_file}"
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

while true; do
  if [ "${run_once}" != "true" ]; then
    sleep_seconds="$("${schedule_bin}" "${job_time}")"
    echo "next interest jobs run in ${sleep_seconds}s"
    wait_with_heartbeat "${sleep_seconds}"
  fi

  if [ ! -x "${capitalflow_bin}" ]; then
    echo "capitalflow binary not found or not executable: ${capitalflow_bin}" >&2
    sleep 60
    continue
  fi

  failed=0
  for job in daily_interest_accrual_job monthly_interest_accrual_job deposit_maturity_check_job; do
    echo "running ${job}"
    if ! "${capitalflow_bin}" jobs run --name "${job}" --timeout "${job_timeout}"; then
      echo "job failed: ${job}" >&2
      failed=1
    fi
  done
  date -Iseconds > "${heartbeat_file}"
  if [ "${failed}" -ne 0 ]; then
    set_status failure interest-jobs
    echo "one or more interest jobs failed" >&2
  else
    date -Iseconds > "${success_file}"
    set_status success interest-jobs
  fi
  if [ "${run_once}" = "true" ]; then
    exit "${failed}"
  fi
done
