#!/usr/bin/env bash
set -euo pipefail

VM_HOST="${VM_HOST:-VM}"
REMOTE_DIR="${REMOTE_DIR:-/home/sunriseex/projects/CapitalFlow}"
PUBLIC_ORIGIN="${PUBLIC_ORIGIN:-https://capitalflow.home.arpa}"
origin_host() {
  local origin="$1"
  origin="${origin#*://}"
  origin="${origin%%/*}"
  origin="${origin%%:*}"
  if [ -z "${origin}" ]; then
    return 1
  fi
  printf '%s\n' "${origin}"
}

CAPITALFLOW_HOST="${CAPITALFLOW_HOST:-$(origin_host "${PUBLIC_ORIGIN}")}"
CAPITALFLOW_PROXY_NETWORK="${CAPITALFLOW_PROXY_NETWORK:-proxy}"
DEPLOY_REF="${DEPLOY_REF:-HEAD}"

archive_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${archive_dir}"
}
trap cleanup EXIT

git archive --format=tar "${DEPLOY_REF}" | tar -xf - -C "${archive_dir}"
deploy_commit="$(git rev-parse --short "${DEPLOY_REF}")"

ssh "${VM_HOST}" "mkdir -p '${REMOTE_DIR}'"

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

origin_host() {
  local origin="$1"
  origin="${origin#*://}"
  origin="${origin%%/*}"
  origin="${origin%%:*}"
  if [ -z "${origin}" ]; then
    return 1
  fi
  printf '%s\n' "${origin}"
}

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
CAPITALFLOW_HOST="${CAPITALFLOW_HOST:-$(origin_host "${PUBLIC_ORIGIN}")}"
PUBLIC_ORIGIN_HOST="$(origin_host "${PUBLIC_ORIGIN}")"

if [ "${CAPITALFLOW_HOST}" != "${PUBLIC_ORIGIN_HOST}" ]; then
  echo "CAPITALFLOW_HOST (${CAPITALFLOW_HOST}) must match PUBLIC_ORIGIN host (${PUBLIC_ORIGIN_HOST})" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required to encode DATABASE_URL credentials" >&2
  exit 1
fi

DATABASE_URL="$(POSTGRES_USER="${POSTGRES_USER}" POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" POSTGRES_DB="${POSTGRES_DB}" python3 - <<'PY'
import os
from urllib.parse import quote

user = quote(os.environ["POSTGRES_USER"], safe="")
password = quote(os.environ["POSTGRES_PASSWORD"], safe="")
database = quote(os.environ["POSTGRES_DB"], safe="")
print(f"postgres://{user}:{password}@postgres:5432/{database}?sslmode=disable")
PY
)"
export DATABASE_URL

if ! docker network inspect "${CAPITALFLOW_PROXY_NETWORK}" >/dev/null 2>&1; then
  docker network create "${CAPITALFLOW_PROXY_NETWORK}" >/dev/null
fi

docker compose -f deploy/compose.yaml --profile tools build api web migrate
docker compose -f deploy/compose.yaml up -d postgres
docker compose -f deploy/compose.yaml --profile tools run -T --rm migrate </dev/null
docker compose -f deploy/compose.yaml up -d --no-build api web
docker compose -f deploy/compose.yaml ps

curl -fsS "http://127.0.0.1:${CAPITALFLOW_API_PORT}/health" >/dev/null
curl -fsS "http://127.0.0.1:${CAPITALFLOW_API_PORT}/ready" >/dev/null
curl -fsS "http://127.0.0.1:${CAPITALFLOW_WEB_PORT}/health" >/dev/null
EOF
