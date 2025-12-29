#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/db.sh [--dev|--prod] up
  scripts/db.sh [--dev|--prod] stop
  scripts/db.sh [--dev|--prod] reset

What it does:
  up    : starts only the Postgres service (db) in the selected compose file
  stop  : stops only the Postgres container
  reset : removes the Postgres container + deletes ONLY the Postgres volume so init scripts run again

Notes:
  - prod uses docker-compose.yml + volume name: nodequeue_db_data
  - dev  uses docker-compose.dev.yml + volume name: nodequeue_db_data_dev
EOF
}

MODE="dev"
if [[ "${1:-}" == "--prod" ]]; then MODE="prod"; shift; fi
if [[ "${1:-}" == "--dev" ]]; then MODE="dev"; shift; fi

ACTION="${1:-}"
if [[ -z "${ACTION}" ]]; then usage; exit 1; fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ "${MODE}" == "prod" ]]; then
  COMPOSE_FILE="${ROOT_DIR}/docker-compose.yml"
  DB_VOLUME="nodequeue_db_data"
else
  COMPOSE_FILE="${ROOT_DIR}/docker-compose.dev.yml"
  DB_VOLUME="nodequeue_db_data_dev"
fi

compose() {
  docker compose -f "${COMPOSE_FILE}" "$@"
}

case "${ACTION}" in
  up)
    compose up -d db
    ;;
  stop)
    # stop only the db container; keep volume/data
    compose stop db
    ;;
  reset)
    # remove only the db container then delete only the db volume so init scripts re-run
    compose stop db || true
    compose rm -f -s -v db || true
    docker volume rm -f "${DB_VOLUME}" || true
    compose up -d db
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    echo "Unknown action: ${ACTION}" >&2
    usage
    exit 2
    ;;
esac


