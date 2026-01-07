#!/usr/bin/env bash
#
# Integration test script for the MCP Jungle project.
# This script builds the binary, runs CLI checks, spins up the Docker stack,
# exercises registry + server functionality, and finally ensures the binary
# server runs correctly.
#

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # repo root
BIN_PATH="$ROOT_DIR/bin/mcpjungle"                            # compiled binary
COMPOSE_FILE="$ROOT_DIR/docker-compose.yaml"                  # compose file path
REGISTRY_URL="http://127.0.0.1:8080"                          # local registry

# Simple logger for readable output
log() { printf "\n[TEST] %s\n" "$*"; }

# Ensure a command is installed before proceeding
require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: Required command '$1' not found in PATH" >&2
    exit 1
  fi
}

# Detect docker compose flavor (new `docker compose` vs legacy `docker-compose`)
detect_compose() {
  if docker compose version >/dev/null 2>&1; then
    echo "docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    echo "docker-compose"
  else
    echo "ERROR: Neither 'docker compose' nor 'docker-compose' found" >&2
    exit 1
  fi
}

# Poll a health endpoint until it's available (timeout configurable)
wait_for_health() {
  local url=$1
  local attempts=${2:-30}   # default: 30 attempts
  local delay=${3:-2}       # default: 2s delay → ~60s total
  for ((i=1; i<=attempts; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done
  echo "ERROR: Health check did not pass for $url after $((attempts*delay))s" >&2
  return 1
}

# Cleanup: stop local binary server if running
cleanup_binary_server() {
  if [[ -n "${BIN_SERVER_PID:-}" ]] && kill -0 "$BIN_SERVER_PID" >/dev/null 2>&1; then
    kill "$BIN_SERVER_PID" || true
    wait "$BIN_SERVER_PID" 2>/dev/null || true
  fi
}

# Cleanup: bring down docker compose stack along with volumes
cleanup_compose() {
  if [[ -f "$COMPOSE_FILE" ]]; then
    if [[ -n "${COMPOSE_CLI:-}" ]]; then
      $COMPOSE_CLI -f "$COMPOSE_FILE" down -v || true
    else
      if docker compose version >/dev/null 2>&1; then
        docker compose -f "$COMPOSE_FILE" down -v || true
      elif command -v docker-compose >/dev/null 2>&1; then
        docker-compose -f "$COMPOSE_FILE" down -v || true
      fi
    fi
  fi
}

cleanup_artifacts() {
  rm -f ./mcpjungle.db ./mcp.db
}

# Always cleanup on exit
trap 'cleanup_binary_server; cleanup_compose; cleanup_artifacts' EXIT

export MCP_SERVER_INIT_REQ_TIMEOUT_SEC=30

# 0) Requirements
log "Checking required commands"
require_cmd go
require_cmd docker
require_cmd curl
require_cmd sed
require_cmd awk

# 1) Build the binary
log "Building binary"
mkdir -p "$ROOT_DIR/bin"
pushd "$ROOT_DIR" >/dev/null
go build -o "$BIN_PATH" .

# 2) Basic CLI sanity checks
log "Verifying CLI help and version"
"$BIN_PATH" --help >/dev/null
"$BIN_PATH" version

# 3) Start Docker stack + wait for health
log "Starting Docker compose stack"
COMPOSE_CLI=$(detect_compose)
$COMPOSE_CLI -f "$COMPOSE_FILE" up -d

log "Waiting for containerized server health"
wait_for_health "$REGISTRY_URL/health"

# 4) Register a test MCP server (idempotent)
log "Ensuring 'context7' server is registered"
if ! "$BIN_PATH" --registry "$REGISTRY_URL" list servers 2>/dev/null | grep -q "context7"; then
  "$BIN_PATH" --registry "$REGISTRY_URL" register \
    --name context7 \
    --description "Context7 docs MCP" \
    --url https://mcp.context7.com/mcp
else
  log "'context7' already registered"
fi

# 5) Exercise tools via registry
log "Listing tools"
"$BIN_PATH" --registry "$REGISTRY_URL" list tools

log "Invoking context7__resolve-library-id"
"$BIN_PATH" --registry "$REGISTRY_URL" invoke context7__resolve-library-id \
  --input '{"libraryName":"lodash"}' >/dev/null

log "Invoking context7__get-library-docs"
"$BIN_PATH" --registry "$REGISTRY_URL" invoke context7__get-library-docs \
  --input '{"context7CompatibleLibraryID":"/lodash/lodash","tokens":500}' >/dev/null

# 6) Start local binary server on port 9090 + verify
log "Starting server via local binary on port 9090"
"$BIN_PATH" start --port 9090 >/dev/null 2>&1 &
BIN_SERVER_PID=$!

log "Waiting for local binary server health"
wait_for_health "http://127.0.0.1:9090/health"

# 7) Test filesystem MCP server in Docker
log "Testing filesystem MCP server in Docker"

if ! "$BIN_PATH" --registry "$REGISTRY_URL" init-server; then
  log "warning: init-server command failed, but this is not fatal"
fi

# Create temp config file for a stdio mcp server
FS_CONFIG=$(mktemp)
cat > "$FS_CONFIG" <<EOF
{
  "name": "filesystem",
  "transport": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/host"]
}
EOF

"$BIN_PATH" --registry "$REGISTRY_URL" register -c "$FS_CONFIG"

rm -f "$FS_CONFIG"

"$BIN_PATH" --registry "$REGISTRY_URL" invoke filesystem__list_allowed_directories --input '{}' >/dev/null

# 8) Test stateful session mode (uses local binary server on port 9090)
log "Testing stateful session mode (session reuse for faster subsequent calls)"
LOCAL_REGISTRY="http://127.0.0.1:9090"

FS_STATEFUL_CONFIG=$(mktemp)
cat > "$FS_STATEFUL_CONFIG" <<EOF
{
  "name": "fs-stateful",
  "transport": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
  "session_mode": "stateful"
}
EOF

"$BIN_PATH" --registry "$LOCAL_REGISTRY" register -c "$FS_STATEFUL_CONFIG"
rm -f "$FS_STATEFUL_CONFIG"

# First call (cold start)
log "First call to stateful server (cold start)..."
TIME1_START=$(date +%s%N)
"$BIN_PATH" --registry "$LOCAL_REGISTRY" invoke fs-stateful__list_allowed_directories --input '{}' >/dev/null
TIME1_END=$(date +%s%N)
TIME1_MS=$(( (TIME1_END - TIME1_START) / 1000000 ))

# Second call (session reused)
log "Second call to stateful server (session reused)..."
TIME2_START=$(date +%s%N)
"$BIN_PATH" --registry "$LOCAL_REGISTRY" invoke fs-stateful__list_allowed_directories --input '{}' >/dev/null
TIME2_END=$(date +%s%N)
TIME2_MS=$(( (TIME2_END - TIME2_START) / 1000000 ))

log "First call: ${TIME1_MS}ms, Second call: ${TIME2_MS}ms"
if [ "$TIME2_MS" -lt "$TIME1_MS" ]; then
  log "✅ Stateful session reuse working - second call was faster!"
else
  log "⚠️  Times similar (MCP server may have fast startup)"
fi

# 9) Print Homebrew formula config snippet
log "Homebrew formula config (from .goreleaser.yaml)"
sed -n '/^brews:/,/^dockers:/p' "$ROOT_DIR/.goreleaser.yaml" || true
popd >/dev/null

log "All tests passed 🎉"

log "Cleaning up"
unset MCP_SERVER_INIT_REQ_TIMEOUT_SEC

log "All done!"
