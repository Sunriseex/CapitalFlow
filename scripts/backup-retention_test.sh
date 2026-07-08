#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
work_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

touch "${work_dir}/capitalflow-20260701T000000Z.zip"
touch "${work_dir}/capitalflow-20260702T000000Z.zip"
touch "${work_dir}/capitalflow-20260703T000000Z.zip"
touch "${work_dir}/capitalflow-20260704T000000Z.zip"
touch "${work_dir}/keep-me.txt"

"${script_dir}/backup-retention.sh" "${work_dir}" 2

test ! -e "${work_dir}/capitalflow-20260701T000000Z.zip"
test ! -e "${work_dir}/capitalflow-20260702T000000Z.zip"
test -e "${work_dir}/capitalflow-20260703T000000Z.zip"
test -e "${work_dir}/capitalflow-20260704T000000Z.zip"
test -e "${work_dir}/keep-me.txt"

if "${script_dir}/backup-retention.sh" "${work_dir}" 0 >/dev/null 2>&1; then
  echo "retention count 0 must fail" >&2
  exit 1
fi
