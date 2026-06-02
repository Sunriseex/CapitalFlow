#!/usr/bin/env bash
set -euo pipefail

git_dir="$(git rev-parse --git-dir)"
mkdir -p "${git_dir}/hooks"

cat > "${git_dir}/hooks/post-commit" <<'HOOK'
#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

VM_HOST="${VM_HOST:-VM}" ./scripts/deploy-vm.sh
HOOK

chmod +x "${git_dir}/hooks/post-commit"
echo "Installed post-commit VM autodeploy hook."
