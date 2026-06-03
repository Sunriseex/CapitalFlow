#!/usr/bin/env sh
set -eu

if [ "${CAPITALFLOW_INTEREST_JOBS_ENABLED:-true}" != "true" ]; then
  echo "interest jobs scheduler disabled"
  sleep infinity
fi

job_time="${CAPITALFLOW_INTEREST_JOBS_TIME:-03:15}"
job_timeout="${CAPITALFLOW_INTEREST_JOB_TIMEOUT:-30m}"

echo "interest jobs scheduler enabled at ${job_time} ${TZ:-UTC}"

while true; do
  now="$(date +%s)"
  today="$(date +%F)"
  target="$(date -d "${today} ${job_time}" +%s)"
  if [ "${target}" -le "${now}" ]; then
    target="$(date -d "${today} ${job_time} + 1 day" +%s)"
  fi

  sleep_seconds="$((target - now))"
  echo "next interest jobs run in ${sleep_seconds}s"
  sleep "${sleep_seconds}"

  for job in daily_interest_accrual_job monthly_interest_accrual_job deposit_maturity_check_job; do
    /usr/local/bin/capitalflow jobs run --name "${job}" --timeout "${job_timeout}"
  done
done
