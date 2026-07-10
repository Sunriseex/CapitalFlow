#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT

cat > "${work_dir}/capitalflow" <<'EOF'
#!/usr/bin/env sh
set -eu
printf '%s\n' "$*" >> "${CAPITALFLOW_TEST_JOB_LOG}"
case "$*" in
  *monthly_interest_accrual_job*) [ "${CAPITALFLOW_TEST_FAIL:-false}" != true ] ;;
esac
EOF
chmod +x "${work_dir}/capitalflow"

CAPITALFLOW_INTEREST_RUN_ONCE=true \
CAPITALFLOW_BIN="${work_dir}/capitalflow" \
CAPITALFLOW_TEST_JOB_LOG="${work_dir}/jobs.log" \
CAPITALFLOW_INTEREST_HEARTBEAT_FILE="${work_dir}/heartbeat" \
CAPITALFLOW_INTEREST_STARTED_FILE="${work_dir}/started" \
CAPITALFLOW_INTEREST_SUCCESS_FILE="${work_dir}/success" \
CAPITALFLOW_INTEREST_STATUS_FILE="${work_dir}/status" \
  "${script_dir}/interest-scheduler.sh"
test "$(wc -l < "${work_dir}/jobs.log")" -eq 3
test -f "${work_dir}/success"
grep -q '^status=success$' "${work_dir}/status"

rm -f "${work_dir}/success"
if CAPITALFLOW_INTEREST_RUN_ONCE=true \
  CAPITALFLOW_BIN="${work_dir}/capitalflow" \
  CAPITALFLOW_TEST_FAIL=true \
  CAPITALFLOW_TEST_JOB_LOG="${work_dir}/jobs.log" \
  CAPITALFLOW_INTEREST_HEARTBEAT_FILE="${work_dir}/heartbeat" \
  CAPITALFLOW_INTEREST_STARTED_FILE="${work_dir}/started" \
  CAPITALFLOW_INTEREST_SUCCESS_FILE="${work_dir}/success" \
  CAPITALFLOW_INTEREST_STATUS_FILE="${work_dir}/status" \
    "${script_dir}/interest-scheduler.sh" >/dev/null 2>&1; then
  echo "failed interest job must fail one-shot scheduler" >&2
  exit 1
fi
test ! -f "${work_dir}/success"
grep -q '^status=failure$' "${work_dir}/status"
