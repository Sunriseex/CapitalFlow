#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
work_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

fake_psql="${work_dir}/psql"
cat > "${fake_psql}" <<'EOF'
#!/usr/bin/env sh
printf '%s\n' "${CAPITALFLOW_TEST_HAS_SCHEMA:-t}"
EOF
chmod +x "${fake_psql}"

fake_capitalflow="${work_dir}/capitalflow"
cat > "${fake_capitalflow}" <<'EOF'
#!/usr/bin/env sh
set -eu
printf '%s\n' "$*" > "${CAPITALFLOW_TEST_ARGS}"
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output" ]; then
    shift
    printf 'backup' > "$1"
    exit 0
  fi
  shift
done
exit 1
EOF
chmod +x "${fake_capitalflow}"

backup_dir="${work_dir}/backups"
mkdir -p "${backup_dir}"
DATABASE_URL='postgres://backup.test/database' \
CAPITALFLOW_BIN="${fake_capitalflow}" \
CAPITALFLOW_PSQL_BIN="${fake_psql}" \
CAPITALFLOW_BACKUP_DIR="${backup_dir}" \
CAPITALFLOW_BACKUP_TIMEOUT=10m \
CAPITALFLOW_BACKUP_RETENTION_BIN="${script_dir}/backup-retention.sh" \
CAPITALFLOW_BACKUP_RETENTION_COUNT=14 \
CAPITALFLOW_TEST_ARGS="${work_dir}/args" \
  "${script_dir}/pre-migration-backup.sh"

grep -q '^backup --output .*/capitalflow-[0-9]\{8\}T[0-9]\{6\}Z-pre-migration.zip --timeout 10m$' "${work_dir}/args"
test "$(find "${backup_dir}" -type f -name 'capitalflow-*-pre-migration.zip' | wc -l)" -eq 1

rm "${work_dir}/args"
DATABASE_URL='postgres://backup.test/database' \
CAPITALFLOW_BIN="${fake_capitalflow}" \
CAPITALFLOW_PSQL_BIN="${fake_psql}" \
CAPITALFLOW_BACKUP_DIR="${backup_dir}" \
CAPITALFLOW_BACKUP_RETENTION_BIN="${script_dir}/backup-retention.sh" \
CAPITALFLOW_TEST_HAS_SCHEMA=f \
CAPITALFLOW_TEST_ARGS="${work_dir}/args" \
  "${script_dir}/pre-migration-backup.sh"

test ! -e "${work_dir}/args"
test "$(find "${backup_dir}" -type f -name 'capitalflow-*-pre-migration.zip' | wc -l)" -eq 1
