#!/usr/bin/env bash
set -euo pipefail

VM_HOST="${VM_HOST:-VM}"
REMOTE_DIR="${REMOTE_DIR:-/home/sunriseex/projects/CapitalFlow}"
PUBLIC_ORIGIN="${PUBLIC_ORIGIN:-https://capitalflow.home.arpa}"
CAPITALFLOW_HOST="${CAPITALFLOW_HOST:-capitalflow.home.arpa}"
CAPITALFLOW_PROXY_NETWORK="${CAPITALFLOW_PROXY_NETWORK:-proxy}"
DEPLOY_REF="${DEPLOY_REF:-HEAD}"

archive_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${archive_dir}"
}
trap cleanup EXIT

git archive --format=tar "${DEPLOY_REF}" | tar -xf - -C "${archive_dir}"
deploy_commit="$(git rev-parse --short "${DEPLOY_REF}")"

rsync -az --delete \
  --exclude ".git" \
  --exclude ".env" \
  --exclude ".env.*" \
  --exclude "configs/.env" \
  --exclude "deploy/.env" \
  --exclude "web/.env" \
  --exclude "web/.env.local" \
  --exclude "web/.env.*.local" \
  --exclude "web/node_modules" \
  --exclude "web/dist" \
  --exclude "web/test-results" \
  --exclude "web/playwright-report" \
  "${archive_dir}/" "${VM_HOST}:${REMOTE_DIR}/"

ssh "${VM_HOST}" "REMOTE_DIR='${REMOTE_DIR}' PUBLIC_ORIGIN='${PUBLIC_ORIGIN}' CAPITALFLOW_HOST='${CAPITALFLOW_HOST}' CAPITALFLOW_PROXY_NETWORK='${CAPITALFLOW_PROXY_NETWORK}' DEPLOY_COMMIT='${deploy_commit}' bash -s" <<'EOF'
set -euo pipefail

cd "$REMOTE_DIR"
mkdir -p deploy
echo "Deploying CapitalFlow ${DEPLOY_COMMIT}"

if [ ! -f deploy/.env ]; then
  db_password="$(openssl rand -hex 24)"
  jwt_secret="$(openssl rand -hex 64)"
  api_auth_token="$(openssl rand -hex 32)"
  cat > deploy/.env <<ENV
POSTGRES_DB=capitalflow
POSTGRES_USER=capitalflow
POSTGRES_PASSWORD=${db_password}
JWT_SECRET=${jwt_secret}
API_AUTH_TOKEN=${api_auth_token}
PUBLIC_ORIGIN=${PUBLIC_ORIGIN}
CAPITALFLOW_HOST=${CAPITALFLOW_HOST}
CAPITALFLOW_PROXY_NETWORK=${CAPITALFLOW_PROXY_NETWORK}
CAPITALFLOW_API_PORT=18080
CAPITALFLOW_WEB_PORT=18081
TRUSTED_PROXIES=127.0.0.1/32,172.16.0.0/12
LOG_LEVEL=info
ENV
  chmod 600 deploy/.env
fi

set -a
. deploy/.env
set +a

CAPITALFLOW_API_PORT="${CAPITALFLOW_API_PORT:-18080}"
CAPITALFLOW_WEB_PORT="${CAPITALFLOW_WEB_PORT:-18081}"
CAPITALFLOW_PROXY_NETWORK="${CAPITALFLOW_PROXY_NETWORK:-proxy}"

if ! docker network inspect "${CAPITALFLOW_PROXY_NETWORK}" >/dev/null 2>&1; then
  docker network create "${CAPITALFLOW_PROXY_NETWORK}" >/dev/null
fi

docker compose -f deploy/compose.yaml up -d --build postgres
docker compose -f deploy/compose.yaml --profile tools build migrate
docker compose -f deploy/compose.yaml --profile tools run -T --rm migrate </dev/null
docker compose -f deploy/compose.yaml up -d --build api web
docker compose -f deploy/compose.yaml ps

curl -fsS "http://127.0.0.1:${CAPITALFLOW_API_PORT}/health" >/dev/null
curl -fsS "http://127.0.0.1:${CAPITALFLOW_API_PORT}/ready" >/dev/null
curl -fsS "http://127.0.0.1:${CAPITALFLOW_WEB_PORT}/health" >/dev/null
EOF
