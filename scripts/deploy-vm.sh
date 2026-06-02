#!/usr/bin/env bash
set -euo pipefail

VM_HOST="${VM_HOST:-VM}"
REMOTE_DIR="${REMOTE_DIR:-/home/sunriseex/projects/CapitalFlow}"
PUBLIC_ORIGIN="${PUBLIC_ORIGIN:-https://capitalflow.home.arpa}"
CAPITALFLOW_HOST="${CAPITALFLOW_HOST:-capitalflow.home.arpa}"

rsync -az --delete \
  --exclude ".git" \
  --exclude "configs/.env" \
  --exclude "deploy/.env" \
  --exclude "web/node_modules" \
  --exclude "web/dist" \
  --exclude "web/test-results" \
  --exclude "web/playwright-report" \
  ./ "${VM_HOST}:${REMOTE_DIR}/"

ssh "${VM_HOST}" "REMOTE_DIR='${REMOTE_DIR}' PUBLIC_ORIGIN='${PUBLIC_ORIGIN}' CAPITALFLOW_HOST='${CAPITALFLOW_HOST}' bash -s" <<'EOF'
set -euo pipefail

cd "$REMOTE_DIR"
mkdir -p deploy

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
CAPITALFLOW_API_PORT=18080
CAPITALFLOW_WEB_PORT=18081
LOG_LEVEL=info
ENV
  chmod 600 deploy/.env
fi

docker compose -f deploy/compose.yaml up -d --build postgres
docker compose -f deploy/compose.yaml --profile tools build migrate
docker compose -f deploy/compose.yaml --profile tools run -T --rm migrate </dev/null
docker compose -f deploy/compose.yaml up -d --build api web
docker compose -f deploy/compose.yaml ps

curl -fsS http://127.0.0.1:18080/health >/dev/null
curl -fsS http://127.0.0.1:18080/ready >/dev/null
curl -fsS http://127.0.0.1:18081/health >/dev/null
EOF
