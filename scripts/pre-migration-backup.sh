#!/usr/bin/env sh
set -eu

database_url="${DATABASE_URL:?DATABASE_URL is required}"
backup_dir="${CAPITALFLOW_BACKUP_DIR:-/backups}"
backup_timeout="${CAPITALFLOW_BACKUP_TIMEOUT:-30m}"
capitalflow_bin="${CAPITALFLOW_BIN:-/usr/local/bin/capitalflow}"
psql_bin="${CAPITALFLOW_PSQL_BIN:-psql}"
retention_bin="${CAPITALFLOW_BACKUP_RETENTION_BIN:-/usr/local/bin/capitalflow-backup-retention}"
retention_count="${CAPITALFLOW_BACKUP_RETENTION_COUNT:-14}"

has_schema="$("${psql_bin}" "${database_url}" -v ON_ERROR_STOP=1 -Atc \
  "SELECT to_regclass('public.goose_db_version') IS NOT NULL")"
if [ "${has_schema}" != "t" ]; then
  echo "pre-migration backup skipped: database schema is empty"
  exit 0
fi

mkdir -p "${backup_dir}"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
output="${backup_dir}/capitalflow-${timestamp}-pre-migration.zip"
echo "creating pre-migration backup: ${output}"
timeout "${backup_timeout}" "${capitalflow_bin}" backup \
  --output "${output}" \
  --timeout "${backup_timeout}"
"${retention_bin}" "${backup_dir}" "${retention_count}"
echo "pre-migration backup complete: ${output}"
