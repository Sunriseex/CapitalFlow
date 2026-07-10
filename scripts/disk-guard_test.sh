#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
used="$(df -Pk "${TMPDIR:-/tmp}" | awk 'NR == 2 {gsub(/%/, "", $5); print $5}')"
passing_limit=$((used + 1))
[ "${passing_limit}" -lt 100 ] || passing_limit=99
"${script_dir}/disk-guard.sh" "${TMPDIR:-/tmp}" "${passing_limit}" 1
if "${script_dir}/disk-guard.sh" "${TMPDIR:-/tmp}" 1 1 >/dev/null 2>&1; then
  echo "disk usage above threshold must fail" >&2
  exit 1
fi
if "${script_dir}/disk-guard.sh" "${TMPDIR:-/tmp}" "${passing_limit}" 999999999 >/dev/null 2>&1; then
  echo "insufficient free space must fail" >&2
  exit 1
fi
