#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
work_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

fake_capitalflow="${work_dir}/capitalflow"
cat > "${fake_capitalflow}" <<'EOF'
#!/usr/bin/env sh
set -eu
if [ "${CAPITALFLOW_TEST_FAIL:-false}" = "true" ]; then
  exit 42
fi
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
touch "${backup_dir}/capitalflow-20260701T000000Z.zip"
touch "${backup_dir}/capitalflow-20260702T000000Z.zip"

CAPITALFLOW_BACKUP_RUN_ONCE=true \
CAPITALFLOW_BACKUP_DIR="${backup_dir}" \
CAPITALFLOW_BACKUP_RETENTION_COUNT=2 \
CAPITALFLOW_BACKUP_TIMEOUT=5m \
CAPITALFLOW_BIN="${fake_capitalflow}" \
CAPITALFLOW_TEST_ARGS="${work_dir}/args" \
CAPITALFLOW_BACKUP_RETENTION_BIN="${script_dir}/backup-retention.sh" \
  "${script_dir}/backup-scheduler.sh"

grep -q '^backup --output .*/capitalflow-[0-9]\{8\}T[0-9]\{6\}Z.zip --timeout 5m$' "${work_dir}/args"
test "$(find "${backup_dir}" -maxdepth 1 -type f -name 'capitalflow-*.zip' | wc -l)" -eq 2

if CAPITALFLOW_BACKUP_RUN_ONCE=true \
  CAPITALFLOW_BACKUP_DIR="${backup_dir}" \
  CAPITALFLOW_BACKUP_RETENTION_COUNT=2 \
  CAPITALFLOW_BACKUP_TIMEOUT=5m \
  CAPITALFLOW_BIN="${fake_capitalflow}" \
  CAPITALFLOW_TEST_FAIL=true \
  CAPITALFLOW_TEST_ARGS="${work_dir}/args" \
  CAPITALFLOW_BACKUP_RETENTION_BIN="${script_dir}/backup-retention.sh" \
    "${script_dir}/backup-scheduler.sh" >/dev/null 2>&1; then
  echo "failed backup must make one-shot scheduler fail" >&2
  exit 1
fi
test "$(find "${backup_dir}" -maxdepth 1 -type f -name 'capitalflow-*.zip' | wc -l)" -eq 2
