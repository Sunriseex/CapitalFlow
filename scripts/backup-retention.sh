#!/usr/bin/env sh
set -eu

backup_dir="${1:?backup directory is required}"
retention_count="${2:?retention count is required}"

case "${retention_count}" in
  *[!0-9]* | 0 | "")
    echo "retention count must be a positive integer" >&2
    exit 1
    ;;
esac

if [ ! -d "${backup_dir}" ]; then
  echo "backup directory does not exist: ${backup_dir}" >&2
  exit 1
fi

find "${backup_dir}" -maxdepth 1 -type f -name 'capitalflow-*.zip' -print \
  | sort -r \
  | awk -v keep="${retention_count}" 'NR > keep' \
  | while IFS= read -r backup; do
      rm -- "${backup}"
      echo "removed expired backup: ${backup}"
    done
