#!/usr/bin/env sh
set -eu

path="${1:-/}"
max_used_percent="${2:-90}"
min_free_mb="${3:-1024}"

case "${max_used_percent}:${min_free_mb}" in
  *[!0-9:]* | 0:* | 100:* | 1??:* | *:0 | :*)
    echo "disk thresholds are invalid" >&2
    exit 2
    ;;
esac

set -- $(df -Pk "${path}" | awk 'NR == 2 {gsub(/%/, "", $5); print $4, $5}')
free_kb="$1"
used_percent="$2"
free_mb=$((free_kb / 1024))

if [ "${used_percent}" -ge "${max_used_percent}" ]; then
  echo "disk guard failed for ${path}: ${used_percent}% used (limit ${max_used_percent}%)" >&2
  exit 1
fi
if [ "${free_mb}" -lt "${min_free_mb}" ]; then
  echo "disk guard failed for ${path}: ${free_mb} MiB free (minimum ${min_free_mb} MiB)" >&2
  exit 1
fi
echo "disk guard ok for ${path}: ${used_percent}% used, ${free_mb} MiB free"
