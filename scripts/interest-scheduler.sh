#!/usr/bin/env sh
set -eu

if [ "${CAPITALFLOW_INTEREST_JOBS_ENABLED:-true}" != "true" ]; then
  echo "interest jobs scheduler disabled"
  sleep infinity
fi

job_time="${CAPITALFLOW_INTEREST_JOBS_TIME:-03:15}"
job_timeout="${CAPITALFLOW_INTEREST_JOB_TIMEOUT:-30m}"
capitalflow_bin="${CAPITALFLOW_BIN:-/usr/local/bin/capitalflow}"
heartbeat_file="/tmp/capitalflow-interest-scheduler.heartbeat"

if ! date -d "2000-01-01 ${job_time}" +%s >/dev/null 2>&1; then
  echo "invalid CAPITALFLOW_INTEREST_JOBS_TIME: ${job_time}" >&2
  exit 1
fi

echo "interest jobs scheduler enabled at ${job_time} ${TZ:-UTC}"

while true; do
  date -Iseconds > "${heartbeat_file}"
  now="$(date +%s)"
  today="$(date +%F)"
  target="$(date -d "${today} ${job_time}" +%s)"
  if [ "${target}" -le "${now}" ]; then
    target="$(date -d "${today} ${job_time} + 1 day" +%s)"
  fi

  sleep_seconds="$((target - now))"
  if [ "${sleep_seconds}" -lt 0 ]; then
    sleep_seconds=0
  fi
  echo "next interest jobs run in ${sleep_seconds}s"
  sleep "${sleep_seconds}"

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
    echo "one or more interest jobs failed" >&2
  fi
done
