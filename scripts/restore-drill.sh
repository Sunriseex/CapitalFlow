#!/usr/bin/env sh
set -eu

database_url="${DATABASE_URL:?DATABASE_URL is required}"
backup_dir="${CAPITALFLOW_BACKUP_DIR:-/backups}"
capitalflow_bin="${CAPITALFLOW_BIN:-/usr/local/bin/capitalflow}"
psql_bin="${CAPITALFLOW_PSQL_BIN:-psql}"
timeout_value="${CAPITALFLOW_RESTORE_DRILL_TIMEOUT:-30m}"

archive="$(find "${backup_dir}" -maxdepth 1 -type f -name 'capitalflow-*.zip' -print | sort | tail -n 1)"
if [ -z "${archive}" ]; then
  echo "restore drill failed: no backup archive found" >&2
  exit 1
fi

database_name="capitalflow_restore_drill_$(date -u +%Y%m%d%H%M%S)_$$"
base_url="${database_url%%\?*}"
query=""
if [ "${database_url}" != "${base_url}" ]; then
  query="?${database_url#*\?}"
fi
target_url="${base_url%/*}/${database_name}${query}"

cleanup() {
  "${psql_bin}" "${database_url}" -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS \"${database_name}\" WITH (FORCE)" >/dev/null 2>&1 || true
}
trap cleanup EXIT HUP INT TERM

"${psql_bin}" "${database_url}" -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"${database_name}\"" >/dev/null
timeout "${timeout_value}" "${capitalflow_bin}" restore \
  --input "${archive}" \
  --database-url "${target_url}" \
  --timeout "${timeout_value}"
"${psql_bin}" "${target_url}" -v ON_ERROR_STOP=1 -Atc \
  "SELECT 1 FROM goose_db_version WHERE is_applied ORDER BY version_id DESC LIMIT 1" | grep -qx 1
echo "restore drill complete: ${archive}"
