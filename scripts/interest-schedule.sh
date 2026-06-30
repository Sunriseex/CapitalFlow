#!/usr/bin/env sh
set -eu

job_time="${1:?job time is required}"
now="${2:-$(date +%s)}"
today="$(date -d "@${now}" +%F)"
target="$(date -d "${today} ${job_time}" +%s)"
if [ "${target}" -le "${now}" ]; then
  tomorrow="$(date -d "${today} 1 day" +%F)"
  target="$(date -d "${tomorrow} ${job_time}" +%s)"
fi

sleep_seconds="$((target - now))"
if [ "${sleep_seconds}" -lt 0 ]; then
  sleep_seconds=0
fi
printf '%s\n' "${sleep_seconds}"
