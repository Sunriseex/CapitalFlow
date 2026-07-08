#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

check_delay() {
  now_text="$1"
  expected="$2"
  now="$(TZ=Europe/Moscow date -d "${now_text}" +%s)"
  actual="$(TZ=Europe/Moscow "${script_dir}/backup-schedule.sh" 02:30 "${now}")"
  if [ "${actual}" -ne "${expected}" ]; then
    echo "next backup from ${now_text} = ${actual}s, want ${expected}s" >&2
    exit 1
  fi
}

check_delay '2026-06-30 01:00:00' 5400
check_delay '2026-06-30 20:00:00' 23400
