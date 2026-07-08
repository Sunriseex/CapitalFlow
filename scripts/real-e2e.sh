#!/usr/bin/env bash
set -euo pipefail

readonly db_name="capitalflow_e2e"
readonly db_user="${POSTGRES_USER:-capitalflow}"
readonly db_password="${POSTGRES_PASSWORD:-capitalflow}"
readonly db_port="${POSTGRES_PORT:-5432}"
readonly database_url="postgres://${db_user}:${db_password}@127.0.0.1:${db_port}/${db_name}?sslmode=disable"

cleanup() {
  docker compose exec -T postgres dropdb --if-exists --force -U "${db_user}" "${db_name}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker compose up -d postgres

for _ in {1..30}; do
  if docker compose exec -T postgres pg_isready -U "${db_user}" -d postgres >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! docker compose exec -T postgres pg_isready -U "${db_user}" -d postgres >/dev/null 2>&1; then
  echo "PostgreSQL did not become ready" >&2
  exit 1
fi

cleanup
docker compose exec -T postgres createdb -U "${db_user}" "${db_name}"
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 \
  -dir migrations \
  postgres "${database_url}" \
  up

TEST_DATABASE_URL="${database_url}" npm --prefix web run test:e2e:real
