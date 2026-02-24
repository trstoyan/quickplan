#!/usr/bin/env bash
set -e
set -o pipefail
set -E

trap 'echo "ERROR: line ${LINENO}: ${BASH_COMMAND}" >&2' ERR

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLI_DIR="${ROOT_DIR}/quick-plan-cli"
WEB_DIR="${ROOT_DIR}/quickplan-web"
LOG_DIR="${ROOT_DIR}/artifacts/e2e-alpha-rc1-$(date +%s)"
WEB_BLUEPRINTS_DIR="${WEB_DIR}/data/blueprints"

mkdir -p "${LOG_DIR}"

DATA_DIR_DEFAULT="${HOME}/.local/share/quickplan"
LEGACY_DATA_DIR="${HOME}/.quickplan"
DATA_DIR="${QUICKPLAN_DATADIR:-${DATA_DIR_DEFAULT}}"
IDENTITY_DIR="${HOME}/.config/quickplan"
IDENTITY_FILE="${IDENTITY_DIR}/identity.json"

BACKUP_DATA_DIR=""
BACKUP_LEGACY_DIR=""
BACKUP_IDENTITY_DIR=""
BACKUP_BLUEPRINTS_DIR=""

SERVER_PID=""
WEB_LOG="${LOG_DIR}/quickplan-web.log"

log() {
  printf "\n==> %s\n" "$*"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

cleanup() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    kill -9 "${SERVER_PID}" >/dev/null 2>&1 || true
  fi

  if [[ -n "${BACKUP_DATA_DIR}" ]]; then
    rm -rf "${DATA_DIR}" >/dev/null 2>&1 || true
    mv "${BACKUP_DATA_DIR}" "${DATA_DIR}" >/dev/null 2>&1 || true
  fi

  if [[ -n "${BACKUP_LEGACY_DIR}" ]]; then
    rm -rf "${LEGACY_DATA_DIR}" >/dev/null 2>&1 || true
    mv "${BACKUP_LEGACY_DIR}" "${LEGACY_DATA_DIR}" >/dev/null 2>&1 || true
  fi

  if [[ -n "${BACKUP_IDENTITY_DIR}" ]]; then
    rm -rf "${IDENTITY_DIR}" >/dev/null 2>&1 || true
    mv "${BACKUP_IDENTITY_DIR}" "${IDENTITY_DIR}" >/dev/null 2>&1 || true
  fi

  if [[ -n "${BACKUP_BLUEPRINTS_DIR}" ]]; then
    rm -rf "${WEB_BLUEPRINTS_DIR}" >/dev/null 2>&1 || true
    mv "${BACKUP_BLUEPRINTS_DIR}" "${WEB_BLUEPRINTS_DIR}" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

resolve_quickplan_bin() {
  if [[ -n "${QUICKPLAN_BIN:-}" ]]; then
    echo "${QUICKPLAN_BIN}"
    return
  fi

  echo "==> Building quickplan binary" >&2
  (cd "${CLI_DIR}" && make build) >&2
  echo "${CLI_DIR}/build/quickplan"
}

backup_and_clear_state() {
  log "Phase 1: Backup and clear local state"

  if [[ -d "${DATA_DIR}" ]]; then
    BACKUP_DATA_DIR="${DATA_DIR}.backup.$(date +%s)"
    mv "${DATA_DIR}" "${BACKUP_DATA_DIR}"
  fi

  if [[ -d "${LEGACY_DATA_DIR}" ]]; then
    BACKUP_LEGACY_DIR="${LEGACY_DATA_DIR}.backup.$(date +%s)"
    mv "${LEGACY_DATA_DIR}" "${BACKUP_LEGACY_DIR}"
  fi

  if [[ -d "${IDENTITY_DIR}" ]]; then
    BACKUP_IDENTITY_DIR="${IDENTITY_DIR}.backup.$(date +%s)"
    mv "${IDENTITY_DIR}" "${BACKUP_IDENTITY_DIR}"
  fi

  if [[ -d "${WEB_BLUEPRINTS_DIR}" ]]; then
    BACKUP_BLUEPRINTS_DIR="${WEB_BLUEPRINTS_DIR}.backup.$(date +%s)"
    mv "${WEB_BLUEPRINTS_DIR}" "${BACKUP_BLUEPRINTS_DIR}"
  fi

  mkdir -p "${DATA_DIR}"
  mkdir -p "${IDENTITY_DIR}"
  mkdir -p "${WEB_BLUEPRINTS_DIR}"
}

test_identity() {
  log "Phase 1: Identity & Zero-State"

  "${QUICKPLAN_BIN}" --non-interactive keygen
  "${QUICKPLAN_BIN}" create "E2E_DeepTest"

  if [[ ! -f "${IDENTITY_FILE}" ]]; then
    echo "Identity file not found at ${IDENTITY_FILE}" >&2
    exit 1
  fi

  jq -e '.ed25519_pub != "" and .ed25519_priv != ""' "${IDENTITY_FILE}" >/dev/null
}

test_m2m() {
  log "Phase 2: M2M Interoperability (Headless Agent)"

  "${QUICKPLAN_BIN}" add "Local Setup" --json
  "${QUICKPLAN_BIN}" add "Daytona Compute" --json

  FIRST_ID=$("${QUICKPLAN_BIN}" list --json | jq -r '.[0].id')
  if [[ -z "${FIRST_ID}" || "${FIRST_ID}" == "null" ]]; then
    echo "Failed to extract first task ID from JSON output" >&2
    exit 1
  fi

  "${QUICKPLAN_BIN}" complete "${FIRST_ID}" --non-interactive --json

  "${QUICKPLAN_BIN}" list --json | jq -e '.[0].done == true or .[0].status == "DONE"' >/dev/null
}

test_runners() {
  log "Phase 3: Pluggable Compute Routing"

  "${QUICKPLAN_BIN}" migrate v1.1 --project "E2E_DeepTest" --force

  PROJECT_FILE="${DATA_DIR}/E2E_DeepTest/project.yaml"
  if [[ ! -f "${PROJECT_FILE}" ]]; then
    echo "project.yaml not found at ${PROJECT_FILE}" >&2
    exit 1
  fi

  if ! grep -q "provider: daytona" "${PROJECT_FILE}"; then
    sed -i '/name: Daytona Compute/a\
      behavior:\
        environment:\
          provider: daytona' "${PROJECT_FILE}"
  fi

  SWARM_OUT_FILE="${LOG_DIR}/swarm_phase3.log"
  SWARM_OUT=$("${QUICKPLAN_BIN}" swarm start --project "E2E_DeepTest" --workers 1 2>&1 || true)
  printf "%s\n" "${SWARM_OUT}" >"${SWARM_OUT_FILE}"

  echo "${SWARM_OUT}" | grep -E "Daytona:|Daytona provider requested|runner setup failed: Daytona provider requested" >/dev/null
}

start_registry_server() {
  log "Phase 4: Registry Persistence (start server)"
  pushd "${WEB_DIR}" >/dev/null
  PORT=8080 \
    GIT_REPO_URL="http://127.0.0.1/quickplan.git" \
    QUICKPLAN_REPO_PATH="${LOG_DIR}/registry-repo" \
    go run ./cmd/server >"${WEB_LOG}" 2>&1 &
  SERVER_PID=$!
  popd >/dev/null

  for _ in $(seq 1 30); do
    if curl -sf "http://127.0.0.1:8080/health" >/dev/null 2>&1; then
      return
    fi
    sleep 1
  done

  echo "Registry server failed to start. See ${WEB_LOG}" >&2
  exit 1
}

test_persistence() {
  log "Phase 4: Registry Persistence (Ghost Hunt)"

  start_registry_server

  QUICKPLAN_REGISTRY_URL="http://127.0.0.1:8080" "${QUICKPLAN_BIN}" sync push --project "E2E_DeepTest"

  kill -9 "${SERVER_PID}"
  SERVER_PID=""

  start_registry_server

  rm -rf "${DATA_DIR}/E2E_DeepTest"

  QUICKPLAN_REGISTRY_URL="http://127.0.0.1:8080" "${QUICKPLAN_BIN}" sync pull --project "E2E_DeepTest"

  if [[ ! -f "${DATA_DIR}/E2E_DeepTest/tasks.yaml" ]]; then
    echo "Restored tasks.yaml not found after pull" >&2
    exit 1
  fi

  grep -q "Local Setup" "${DATA_DIR}/E2E_DeepTest/tasks.yaml"
  grep -q "Daytona Compute" "${DATA_DIR}/E2E_DeepTest/tasks.yaml"
}

require_cmd jq
require_cmd curl

QUICKPLAN_BIN="$(resolve_quickplan_bin)"

backup_and_clear_state
test_identity
test_m2m
test_runners
test_persistence

log "E2E Alpha RC1 test suite completed successfully"
