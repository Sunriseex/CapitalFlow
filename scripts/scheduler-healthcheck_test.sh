#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT

heartbeat="${work_dir}/heartbeat"
success="${work_dir}/success"
started="${work_dir}/started"
touch "${heartbeat}" "${started}"

"${script_dir}/scheduler-healthcheck.sh" "${heartbeat}" "${success}" "${started}" 5 5
touch "${success}"
"${script_dir}/scheduler-healthcheck.sh" "${heartbeat}" "${success}" "${started}" 5 5

touch -d '10 seconds ago' "${heartbeat}"
if "${script_dir}/scheduler-healthcheck.sh" "${heartbeat}" "${success}" "${started}" 5 5 >/dev/null 2>&1; then
  echo "stale heartbeat must fail" >&2
  exit 1
fi

touch "${heartbeat}"
touch -d '10 seconds ago' "${success}"
if "${script_dir}/scheduler-healthcheck.sh" "${heartbeat}" "${success}" "${started}" 5 5 >/dev/null 2>&1; then
  echo "stale success marker must fail" >&2
  exit 1
fi
