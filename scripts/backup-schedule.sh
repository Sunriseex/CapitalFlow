#!/usr/bin/env sh
set -eu

backup_time="${1:?backup time is required}"
now="${2:-$(date +%s)}"
today="$(date -d "@${now}" +%F)"
target="$(date -d "${today} ${backup_time}" +%s)"
if [ "${target}" -le "${now}" ]; then
  tomorrow="$(date -d "${today} 1 day" +%F)"
  target="$(date -d "${tomorrow} ${backup_time}" +%s)"
fi

sleep_seconds="$((target - now))"
if [ "${sleep_seconds}" -lt 0 ]; then
  sleep_seconds=0
fi
printf '%s\n' "${sleep_seconds}"
