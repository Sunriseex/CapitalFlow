#!/usr/bin/env bash
set -euo pipefail

VM_HOST="${VM_HOST:-VM}"
REMOTE_DIR="${REMOTE_DIR:-/home/sunriseex/projects/CapitalFlow}"
requested_public_origin="${PUBLIC_ORIGIN-}"
requested_capitalflow_host="${CAPITALFLOW_HOST-}"
requested_proxy_network="${CAPITALFLOW_PROXY_NETWORK-}"
requested_api_image="${CAPITALFLOW_API_IMAGE-}"
requested_web_image="${CAPITALFLOW_WEB_IMAGE-}"
requested_interest_jobs_enabled="${CAPITALFLOW_INTEREST_JOBS_ENABLED-}"
requested_interest_jobs_time="${CAPITALFLOW_INTEREST_JOBS_TIME-}"
requested_interest_job_timeout="${CAPITALFLOW_INTEREST_JOB_TIMEOUT-}"
requested_backups_enabled="${CAPITALFLOW_BACKUPS_ENABLED-}"
requested_backup_time="${CAPITALFLOW_BACKUP_TIME-}"
requested_backup_timeout="${CAPITALFLOW_BACKUP_TIMEOUT-}"
requested_backup_retention_count="${CAPITALFLOW_BACKUP_RETENTION_COUNT-}"
requested_backup_host_dir="${CAPITALFLOW_BACKUP_HOST_DIR-}"
requested_backup_uid="${CAPITALFLOW_BACKUP_UID-}"
requested_backup_gid="${CAPITALFLOW_BACKUP_GID-}"
requested_timezone="${TZ-}"
PUBLIC_ORIGIN="${requested_public_origin:-https://capitalflow.home.arpa}"
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

CAPITALFLOW_HOST="${requested_capitalflow_host:-$(origin_host "${PUBLIC_ORIGIN}")}"
CAPITALFLOW_PROXY_NETWORK="${requested_proxy_network:-proxy}"
CAPITALFLOW_API_IMAGE="${requested_api_image:-capitalflow-api:local}"
CAPITALFLOW_WEB_IMAGE="${requested_web_image:-capitalflow-web:local}"
DEPLOY_MODE="${DEPLOY_MODE:-build}"
DEPLOY_REF="${DEPLOY_REF:-HEAD}"
local_deploy=false
case "${VM_HOST}" in
  local | localhost | 127.0.0.1)
    local_deploy=true
    ;;
esac

case "${DEPLOY_MODE}" in
  build | images) ;;
  *)
    echo "DEPLOY_MODE must be build or images" >&2
    exit 1
    ;;
esac

archive_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${archive_dir}"
}
trap cleanup EXIT

git archive --format=tar "${DEPLOY_REF}" | tar -xf - -C "${archive_dir}"
deploy_commit="$(git rev-parse --short "${DEPLOY_REF}")"

if [ "${local_deploy}" = true ]; then
  mkdir -p "${REMOTE_DIR}"
else
  ssh "${VM_HOST}" "mkdir -p '${REMOTE_DIR}'"
fi

rsync_args=(
  -az
  --delete
  --exclude ".git"
  --exclude ".env"
  --exclude ".env.*"
  --exclude "configs/.env"
  --exclude "deploy/.env"
  --exclude "web/.env"
  --exclude "web/.env.local"
  --exclude "web/.env.*.local"
  --exclude "web/node_modules"
  --exclude "web/dist"
  --exclude "web/test-results"
  --exclude "web/playwright-report"
)

if [ "${local_deploy}" = true ]; then
  rsync "${rsync_args[@]}" "${archive_dir}/" "${REMOTE_DIR}/"
else
  rsync "${rsync_args[@]}" "${archive_dir}/" "${VM_HOST}:${REMOTE_DIR}/"
fi

remote_script="${archive_dir}/deploy-remote.sh"
cat > "${remote_script}" <<'EOF'
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

set_env_var() {
  local key="$1"
  local value="$2"
  local env_file="deploy/.env"
  local tmp
  tmp="$(mktemp)"
  if [ -f "${env_file}" ]; then
    grep -v "^${key}=" "${env_file}" > "${tmp}" || true
  fi
  printf '%s=%s\n' "${key}" "${value}" >> "${tmp}"
  mv "${tmp}" "${env_file}"
  chmod 600 "${env_file}"
}

requested_public_origin="${REQUESTED_PUBLIC_ORIGIN:-}"
requested_capitalflow_host="${REQUESTED_CAPITALFLOW_HOST:-}"
requested_proxy_network="${REQUESTED_PROXY_NETWORK:-}"
requested_api_image="${REQUESTED_API_IMAGE:-}"
requested_web_image="${REQUESTED_WEB_IMAGE:-}"
requested_interest_jobs_enabled="${REQUESTED_INTEREST_JOBS_ENABLED:-}"
requested_interest_jobs_time="${REQUESTED_INTEREST_JOBS_TIME:-}"
requested_interest_job_timeout="${REQUESTED_INTEREST_JOB_TIMEOUT:-}"
requested_backups_enabled="${REQUESTED_BACKUPS_ENABLED:-}"
requested_backup_time="${REQUESTED_BACKUP_TIME:-}"
requested_backup_timeout="${REQUESTED_BACKUP_TIMEOUT:-}"
requested_backup_retention_count="${REQUESTED_BACKUP_RETENTION_COUNT:-}"
requested_backup_host_dir="${REQUESTED_BACKUP_HOST_DIR:-}"
requested_backup_uid="${REQUESTED_BACKUP_UID:-}"
requested_backup_gid="${REQUESTED_BACKUP_GID:-}"
requested_timezone="${REQUESTED_TIMEZONE:-}"

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
CAPITALFLOW_INTEREST_JOBS_ENABLED=true
CAPITALFLOW_INTEREST_JOBS_TIME=03:15
CAPITALFLOW_INTEREST_JOB_TIMEOUT=30m
CAPITALFLOW_BACKUPS_ENABLED=true
CAPITALFLOW_BACKUP_TIME=02:30
CAPITALFLOW_BACKUP_TIMEOUT=30m
CAPITALFLOW_BACKUP_RETENTION_COUNT=14
CAPITALFLOW_BACKUP_HOST_DIR=${HOME}/backups/capitalflow
CAPITALFLOW_BACKUP_UID=$(id -u)
CAPITALFLOW_BACKUP_GID=$(id -g)
TZ=Europe/Moscow
ENV
  chmod 600 deploy/.env
fi

set -a
. deploy/.env
set +a

if [ -n "${requested_public_origin}" ]; then
  PUBLIC_ORIGIN="${requested_public_origin}"
fi
if [ -n "${requested_capitalflow_host}" ]; then
  CAPITALFLOW_HOST="${requested_capitalflow_host}"
fi
if [ -n "${requested_proxy_network}" ]; then
  CAPITALFLOW_PROXY_NETWORK="${requested_proxy_network}"
fi
if [ -n "${requested_api_image}" ]; then
  CAPITALFLOW_API_IMAGE="${requested_api_image}"
fi
if [ -n "${requested_web_image}" ]; then
  CAPITALFLOW_WEB_IMAGE="${requested_web_image}"
fi
if [ -n "${requested_interest_jobs_enabled}" ]; then
  CAPITALFLOW_INTEREST_JOBS_ENABLED="${requested_interest_jobs_enabled}"
fi
if [ -n "${requested_interest_jobs_time}" ]; then
  CAPITALFLOW_INTEREST_JOBS_TIME="${requested_interest_jobs_time}"
fi
if [ -n "${requested_interest_job_timeout}" ]; then
  CAPITALFLOW_INTEREST_JOB_TIMEOUT="${requested_interest_job_timeout}"
fi
if [ -n "${requested_backups_enabled}" ]; then
  CAPITALFLOW_BACKUPS_ENABLED="${requested_backups_enabled}"
fi
if [ -n "${requested_backup_time}" ]; then
  CAPITALFLOW_BACKUP_TIME="${requested_backup_time}"
fi
if [ -n "${requested_backup_timeout}" ]; then
  CAPITALFLOW_BACKUP_TIMEOUT="${requested_backup_timeout}"
fi
if [ -n "${requested_backup_retention_count}" ]; then
  CAPITALFLOW_BACKUP_RETENTION_COUNT="${requested_backup_retention_count}"
fi
if [ -n "${requested_backup_host_dir}" ]; then
  CAPITALFLOW_BACKUP_HOST_DIR="${requested_backup_host_dir}"
fi
if [ -n "${requested_backup_uid}" ]; then
  CAPITALFLOW_BACKUP_UID="${requested_backup_uid}"
fi
if [ -n "${requested_backup_gid}" ]; then
  CAPITALFLOW_BACKUP_GID="${requested_backup_gid}"
fi
if [ -n "${requested_timezone}" ]; then
  TZ="${requested_timezone}"
fi

CAPITALFLOW_API_PORT="${CAPITALFLOW_API_PORT:-18080}"
CAPITALFLOW_WEB_PORT="${CAPITALFLOW_WEB_PORT:-18081}"
CAPITALFLOW_PROXY_NETWORK="${CAPITALFLOW_PROXY_NETWORK:-proxy}"
CAPITALFLOW_HOST="${CAPITALFLOW_HOST:-$(origin_host "${PUBLIC_ORIGIN}")}"
CAPITALFLOW_API_IMAGE="${CAPITALFLOW_API_IMAGE:-capitalflow-api:local}"
CAPITALFLOW_WEB_IMAGE="${CAPITALFLOW_WEB_IMAGE:-capitalflow-web:local}"
CAPITALFLOW_INTEREST_JOBS_ENABLED="${CAPITALFLOW_INTEREST_JOBS_ENABLED:-true}"
CAPITALFLOW_INTEREST_JOBS_TIME="${CAPITALFLOW_INTEREST_JOBS_TIME:-03:15}"
CAPITALFLOW_INTEREST_JOB_TIMEOUT="${CAPITALFLOW_INTEREST_JOB_TIMEOUT:-30m}"
CAPITALFLOW_BACKUPS_ENABLED="${CAPITALFLOW_BACKUPS_ENABLED:-true}"
CAPITALFLOW_BACKUP_TIME="${CAPITALFLOW_BACKUP_TIME:-02:30}"
CAPITALFLOW_BACKUP_TIMEOUT="${CAPITALFLOW_BACKUP_TIMEOUT:-30m}"
CAPITALFLOW_BACKUP_RETENTION_COUNT="${CAPITALFLOW_BACKUP_RETENTION_COUNT:-14}"
CAPITALFLOW_BACKUP_HOST_DIR="${CAPITALFLOW_BACKUP_HOST_DIR:-${HOME}/backups/capitalflow}"
if [ -z "${requested_backup_host_dir}" ] && [ "${CAPITALFLOW_BACKUP_HOST_DIR}" = "/srv/backups/capitalflow" ]; then
  CAPITALFLOW_BACKUP_HOST_DIR="${HOME}/backups/capitalflow"
fi
CAPITALFLOW_BACKUP_UID="${CAPITALFLOW_BACKUP_UID:-$(id -u)}"
CAPITALFLOW_BACKUP_GID="${CAPITALFLOW_BACKUP_GID:-$(id -g)}"
TZ="${TZ:-Europe/Moscow}"
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
set_env_var PUBLIC_ORIGIN "${PUBLIC_ORIGIN}"
set_env_var CAPITALFLOW_HOST "${CAPITALFLOW_HOST}"
set_env_var CAPITALFLOW_PROXY_NETWORK "${CAPITALFLOW_PROXY_NETWORK}"
set_env_var CAPITALFLOW_API_IMAGE "${CAPITALFLOW_API_IMAGE}"
set_env_var CAPITALFLOW_WEB_IMAGE "${CAPITALFLOW_WEB_IMAGE}"
set_env_var DATABASE_URL "${DATABASE_URL}"
set_env_var CAPITALFLOW_INTEREST_JOBS_ENABLED "${CAPITALFLOW_INTEREST_JOBS_ENABLED}"
set_env_var CAPITALFLOW_INTEREST_JOBS_TIME "${CAPITALFLOW_INTEREST_JOBS_TIME}"
set_env_var CAPITALFLOW_INTEREST_JOB_TIMEOUT "${CAPITALFLOW_INTEREST_JOB_TIMEOUT}"
set_env_var CAPITALFLOW_BACKUPS_ENABLED "${CAPITALFLOW_BACKUPS_ENABLED}"
set_env_var CAPITALFLOW_BACKUP_TIME "${CAPITALFLOW_BACKUP_TIME}"
set_env_var CAPITALFLOW_BACKUP_TIMEOUT "${CAPITALFLOW_BACKUP_TIMEOUT}"
set_env_var CAPITALFLOW_BACKUP_RETENTION_COUNT "${CAPITALFLOW_BACKUP_RETENTION_COUNT}"
set_env_var CAPITALFLOW_BACKUP_HOST_DIR "${CAPITALFLOW_BACKUP_HOST_DIR}"
set_env_var CAPITALFLOW_BACKUP_UID "${CAPITALFLOW_BACKUP_UID}"
set_env_var CAPITALFLOW_BACKUP_GID "${CAPITALFLOW_BACKUP_GID}"
set_env_var TZ "${TZ}"

mkdir -p "${CAPITALFLOW_BACKUP_HOST_DIR}"
chown "${CAPITALFLOW_BACKUP_UID}:${CAPITALFLOW_BACKUP_GID}" "${CAPITALFLOW_BACKUP_HOST_DIR}"
chmod 700 "${CAPITALFLOW_BACKUP_HOST_DIR}"

if ! docker network inspect "${CAPITALFLOW_PROXY_NETWORK}" >/dev/null 2>&1; then
  docker network create "${CAPITALFLOW_PROXY_NETWORK}" >/dev/null
fi

cd deploy

if [ "${DEPLOY_MODE}" = "images" ]; then
  docker compose --profile tools pull api web migrate
else
  docker compose --profile tools build api web migrate
fi
docker compose up -d --wait postgres
docker compose stop api web interest-scheduler backup-scheduler >/dev/null 2>&1 || true
docker compose --profile tools run -T --rm migrate </dev/null
docker compose up -d --wait --no-build api web
if [ "${CAPITALFLOW_INTEREST_JOBS_ENABLED}" = "true" ]; then
  docker compose up -d --wait --no-build interest-scheduler
else
  docker compose rm -sf interest-scheduler >/dev/null 2>&1 || true
fi
if [ "${CAPITALFLOW_BACKUPS_ENABLED}" = "true" ]; then
  docker compose up -d --wait --no-build backup-scheduler
else
  docker compose rm -sf backup-scheduler >/dev/null 2>&1 || true
fi
docker compose ps

curl -fsS "http://127.0.0.1:${CAPITALFLOW_API_PORT}/health" >/dev/null
curl -fsS "http://127.0.0.1:${CAPITALFLOW_API_PORT}/ready" >/dev/null
curl -fsS "http://127.0.0.1:${CAPITALFLOW_WEB_PORT}/health" >/dev/null
EOF

deploy_env=(
  "REMOTE_DIR=${REMOTE_DIR}"
  "PUBLIC_ORIGIN=${PUBLIC_ORIGIN}"
  "CAPITALFLOW_HOST=${CAPITALFLOW_HOST}"
  "CAPITALFLOW_PROXY_NETWORK=${CAPITALFLOW_PROXY_NETWORK}"
  "CAPITALFLOW_API_IMAGE=${CAPITALFLOW_API_IMAGE}"
  "CAPITALFLOW_WEB_IMAGE=${CAPITALFLOW_WEB_IMAGE}"
  "REQUESTED_PUBLIC_ORIGIN=${requested_public_origin}"
  "REQUESTED_CAPITALFLOW_HOST=${requested_capitalflow_host}"
  "REQUESTED_PROXY_NETWORK=${requested_proxy_network}"
  "REQUESTED_API_IMAGE=${requested_api_image}"
  "REQUESTED_WEB_IMAGE=${requested_web_image}"
  "REQUESTED_INTEREST_JOBS_ENABLED=${requested_interest_jobs_enabled}"
  "REQUESTED_INTEREST_JOBS_TIME=${requested_interest_jobs_time}"
  "REQUESTED_INTEREST_JOB_TIMEOUT=${requested_interest_job_timeout}"
  "REQUESTED_BACKUPS_ENABLED=${requested_backups_enabled}"
  "REQUESTED_BACKUP_TIME=${requested_backup_time}"
  "REQUESTED_BACKUP_TIMEOUT=${requested_backup_timeout}"
  "REQUESTED_BACKUP_RETENTION_COUNT=${requested_backup_retention_count}"
  "REQUESTED_BACKUP_HOST_DIR=${requested_backup_host_dir}"
  "REQUESTED_BACKUP_UID=${requested_backup_uid}"
  "REQUESTED_BACKUP_GID=${requested_backup_gid}"
  "REQUESTED_TIMEZONE=${requested_timezone}"
  "DEPLOY_MODE=${DEPLOY_MODE}"
  "DEPLOY_COMMIT=${deploy_commit}"
)

if [ "${local_deploy}" = true ]; then
  env "${deploy_env[@]}" bash "${remote_script}"
else
  remote_command=""
  for assignment in "${deploy_env[@]}"; do
    remote_command+=" $(printf "%q" "${assignment}")"
  done
  ssh "${VM_HOST}" "${remote_command# } bash -s" < "${remote_script}"
fi
