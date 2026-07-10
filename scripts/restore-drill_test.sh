#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT
mkdir -p "${work_dir}/backups"
touch "${work_dir}/backups/capitalflow-20260710T000000Z.zip"

cat > "${work_dir}/psql" <<'EOF'
#!/usr/bin/env sh
set -eu
printf '%s\n' "$*" >> "${CAPITALFLOW_TEST_PSQL_LOG}"
case "$*" in
  *"SELECT 1 FROM goose_db_version"*) printf '1\n' ;;
esac
EOF
cat > "${work_dir}/capitalflow" <<'EOF'
#!/usr/bin/env sh
set -eu
printf '%s\n' "$*" > "${CAPITALFLOW_TEST_RESTORE_ARGS}"
EOF
chmod +x "${work_dir}/psql" "${work_dir}/capitalflow"

DATABASE_URL='postgres://user:pass@postgres:5432/capitalflow?sslmode=disable' \
CAPITALFLOW_BACKUP_DIR="${work_dir}/backups" \
CAPITALFLOW_BIN="${work_dir}/capitalflow" \
CAPITALFLOW_PSQL_BIN="${work_dir}/psql" \
CAPITALFLOW_TEST_PSQL_LOG="${work_dir}/psql.log" \
CAPITALFLOW_TEST_RESTORE_ARGS="${work_dir}/restore.args" \
  "${script_dir}/restore-drill.sh"

grep -q '^restore --input .*/capitalflow-20260710T000000Z.zip --database-url postgres://user:pass@postgres:5432/capitalflow_restore_drill_.*?sslmode=disable --timeout 30m$' "${work_dir}/restore.args"
grep -q 'CREATE DATABASE' "${work_dir}/psql.log"
grep -q 'DROP DATABASE IF EXISTS' "${work_dir}/psql.log"
