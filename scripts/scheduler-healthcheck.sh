#!/usr/bin/env sh
set -eu

heartbeat_file="${1:?heartbeat file is required}"
success_file="${2:?success file is required}"
started_file="${3:?started file is required}"
heartbeat_max_age="${4:-300}"
success_max_age="${5:-108000}"

case "${heartbeat_max_age}:${success_max_age}" in
  *[!0-9:]* | *:0 | 0:* | :*)
    echo "healthcheck ages must be positive integers" >&2
    exit 2
    ;;
esac

now="$(date +%s)"
file_age() {
  file="$1"
  test -f "${file}" || return 1
  modified="$(stat -c %Y "${file}")"
  age=$((now - modified))
  test "${age}" -ge 0 || age=0
  printf '%s\n' "${age}"
}

heartbeat_age="$(file_age "${heartbeat_file}")" || {
  echo "scheduler heartbeat is missing" >&2
  exit 1
}
if [ "${heartbeat_age}" -gt "${heartbeat_max_age}" ]; then
  echo "scheduler heartbeat is stale: ${heartbeat_age}s" >&2
  exit 1
fi

if success_age="$(file_age "${success_file}")"; then
  if [ "${success_age}" -gt "${success_max_age}" ]; then
    echo "last successful run is stale: ${success_age}s" >&2
    exit 1
  fi
  exit 0
fi

started_age="$(file_age "${started_file}")" || {
  echo "scheduler start marker is missing" >&2
  exit 1
}
if [ "${started_age}" -gt "${success_max_age}" ]; then
  echo "scheduler has never completed successfully in ${started_age}s" >&2
  exit 1
fi
