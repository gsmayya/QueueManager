#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/metrics_smoke.sh [BASE_URL]

Defaults:
  BASE_URL defaults to http://localhost:8080

What it does:
  Calls the metrics endpoints and prints:
  - HTTP status
  - total request time
  - response size (bytes)
  - (optional) JSON validity check if jq is installed

Endpoints:
  GET /resources/metrics
  GET /nodes/metrics
  GET /schedules/metrics
EOF
}

BASE_URL="${1:-${BASE_URL:-http://localhost:8080}}"
BASE_URL="${BASE_URL%/}"

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

have_jq() {
  command -v jq >/dev/null 2>&1
}

call() {
  local path="$1"
  local url="${BASE_URL}${path}"

  local tmp
  tmp="$(mktemp)"
  local meta

  # Write body to tmp; write status + time + size to stdout via -w.
  # shellcheck disable=SC2086
  meta="$(curl -sS -o "${tmp}" -w "%{http_code} %{time_total} %{size_download}" "${url}" || true)"

  local code time_total size
  code="$(awk '{print $1}' <<<"${meta}")"
  time_total="$(awk '{print $2}' <<<"${meta}")"
  size="$(awk '{print $3}' <<<"${meta}")"

  local ok="no"
  if [[ "${code}" =~ ^2[0-9][0-9]$ ]]; then
    ok="yes"
  fi

  printf "%-22s status=%s ok=%s time=%ss bytes=%s\n" "${path}" "${code}" "${ok}" "${time_total}" "${size}"

  if have_jq; then
    if jq -e . >/dev/null 2>&1 <"${tmp}"; then
      printf "%-22s json=valid\n" "${path}"
    else
      printf "%-22s json=INVALID\n" "${path}"
    fi
  fi

  echo "---- ${path} json ----"
  if have_jq; then
    # Pretty print if possible; otherwise fall back to raw.
    jq . <"${tmp}" 2>/dev/null || cat "${tmp}"
  else
    cat "${tmp}"
  fi
  echo
  echo "----------------------"

  rm -f "${tmp}"
}

echo "Base URL: ${BASE_URL}"
call "/resources/metrics"
call "/nodes/metrics"
call "/schedules/metrics"

