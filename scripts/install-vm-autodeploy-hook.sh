#!/usr/bin/env bash
set -euo pipefail

hook_path="$(git rev-parse --git-path hooks/post-commit)"
mkdir -p "$(dirname "${hook_path}")"

if [ -e "${hook_path}" ] &&
  ! grep -q "CapitalFlow VM autodeploy hook" "${hook_path}" &&
  ! grep -q "./scripts/deploy-vm.sh" "${hook_path}"; then
  echo "post-commit hook already exists: ${hook_path}" >&2
  echo "Move its logic into another script or chain ./scripts/deploy-vm.sh manually, then rerun this installer." >&2
  exit 1
fi

cat > "${hook_path}" <<'HOOK'
#!/usr/bin/env bash
set -euo pipefail

# CapitalFlow VM autodeploy hook
repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

VM_HOST="${VM_HOST:-VM}" ./scripts/deploy-vm.sh
HOOK

chmod +x "${hook_path}"
echo "Installed post-commit VM autodeploy hook."
