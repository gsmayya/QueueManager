#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/db.sh [--dev|--prod] up
  scripts/db.sh [--dev|--prod] stop
  scripts/db.sh [--dev|--prod] reset
  scripts/db.sh [--dev|--prod] wait
  scripts/db.sh [--dev|--prod] psql [nodequeue|master_db]
  scripts/db.sh [--dev|--prod] env [queue-service|queue-admin]

What it does:
  up    : starts only the Postgres service (db) in the selected compose file
  stop  : stops only the Postgres container
  reset : removes the Postgres container + deletes ONLY the Postgres volume so init scripts run again
  wait  : waits until Postgres is ready (pg_isready)
  psql  : opens psql inside the db container (default DB: nodequeue)
  env   : prints DB env var exports for running services locally

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
shift || true

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

db_container_id() {
  compose ps -q db
}

db_exec() {
  local cid
  cid="$(db_container_id)"
  if [[ -z "${cid}" ]]; then
    echo "db container is not running (try: scripts/db.sh up)" >&2
    exit 2
  fi
  docker exec -it "${cid}" "$@"
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
  wait)
    # Wait for health (pg_isready) inside container.
    for i in {1..60}; do
      if db_exec pg_isready -U nodequeue -d nodequeue >/dev/null 2>&1; then
        echo "db is ready"
        exit 0
      fi
      sleep 1
    done
    echo "db did not become ready in time" >&2
    exit 1
    ;;
  psql)
    DB_NAME="${1:-nodequeue}"
    if [[ "${DB_NAME}" != "nodequeue" && "${DB_NAME}" != "master_db" ]]; then
      echo "Unknown DB name: ${DB_NAME} (expected nodequeue|master_db)" >&2
      exit 2
    fi
    db_exec psql -U nodequeue -d "${DB_NAME}"
    ;;
  env)
    TARGET="${1:-}"
    case "${TARGET}" in
      queue-service)
        cat <<'EOF'
export MAIN_DB_HOST=localhost
export MAIN_DB_PORT=5432
export MAIN_DB_NAME=nodequeue
export MAIN_DB_USER=nodequeue
export MAIN_DB_PASSWORD=nodequeue
export MAIN_DB_SSLMODE=disable
EOF
        ;;
      queue-admin)
        cat <<'EOF'
export MAIN_DB_HOST=localhost
export MAIN_DB_PORT=5432
export MAIN_DB_NAME=master_db
export MAIN_DB_USER=nodequeue
export MAIN_DB_PASSWORD=nodequeue
export MAIN_DB_SSLMODE=disable

export NODEQUEUE_DB_HOST=localhost
export NODEQUEUE_DB_PORT=5432
export NODEQUEUE_DB_NAME=nodequeue
export NODEQUEUE_DB_USER=nodequeue
export NODEQUEUE_DB_PASSWORD=nodequeue
export NODEQUEUE_DB_SSLMODE=disable
EOF
        ;;
      "")
        cat <<'EOF'
# queue-service (nodequeue DB):
scripts/db.sh env queue-service

# queue-admin (master_db + nodequeue DB):
scripts/db.sh env queue-admin
EOF
        ;;
      *)
        echo "Unknown env target: ${TARGET} (expected queue-service|queue-admin)" >&2
        exit 2
        ;;
    esac
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


