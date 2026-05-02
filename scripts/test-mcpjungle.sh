#!/usr/bin/env bash
#
# Integration test script for the MCP Jungle project.
# This script builds the binary, runs CLI checks, starts a local server from
# the freshly built binary, and exercises registry + server functionality
# against that local process.
#

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # repo root
BIN_PATH="$ROOT_DIR/bin/mcpjungle"                            # compiled binary
REGISTRY_PORT="${REGISTRY_PORT:-18080}"
REGISTRY_URL="http://127.0.0.1:${REGISTRY_PORT}"              # local registry

# Simple logger for readable output
log() { printf "\n[TEST] %s\n" "$*"; }

# Ensure a command is installed before proceeding
require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: Required command '$1' not found in PATH" >&2
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

cleanup_binary_servers() {
  if [[ -n "${REGISTRY_SERVER_PID:-}" ]] && kill -0 "$REGISTRY_SERVER_PID" >/dev/null 2>&1; then
    kill "$REGISTRY_SERVER_PID" || true
    wait "$REGISTRY_SERVER_PID" 2>/dev/null || true
  fi

  if [[ -n "${BIN_SERVER_PID:-}" ]] && kill -0 "$BIN_SERVER_PID" >/dev/null 2>&1; then
    kill "$BIN_SERVER_PID" || true
    wait "$BIN_SERVER_PID" 2>/dev/null || true
  fi
}

cleanup_temp_files() {
  if [[ -n "${FS_CONFIG:-}" ]]; then
    rm -f "$FS_CONFIG"
  fi

  if [[ -n "${FS_STATEFUL_CONFIG:-}" ]]; then
    rm -f "$FS_STATEFUL_CONFIG"
  fi

  if [[ -n "${REGISTRY_LOG:-}" ]]; then
    rm -f "$REGISTRY_LOG"
  fi
}

cleanup_runtime_state() {
  rm -f "$ROOT_DIR/mcpjungle.db" "$ROOT_DIR/mcp.db"
  rm -rf "$ROOT_DIR/mcpjungle_data"
  rm -f "$HOME/.mcpjungle.conf"
}

cleanup() {
  cleanup_binary_servers
  cleanup_temp_files
  cleanup_runtime_state
}

reset_runtime_state() {
  cleanup_runtime_state
  mkdir -p "$ROOT_DIR/mcpjungle_data"
}

start_local_server() {
  local port=$1
  local log_file=$2

  "$BIN_PATH" start --port "$port" >"$log_file" 2>&1 &
  local pid=$!

  if [[ "$port" == "$REGISTRY_PORT" ]]; then
    REGISTRY_SERVER_PID=$pid
  else
    BIN_SERVER_PID=$pid
  fi
}

# Always cleanup on exit
trap cleanup EXIT

export MCP_SERVER_INIT_REQ_TIMEOUT_SEC=30

# 0) Requirements
log "Checking required commands"
require_cmd go
require_cmd curl
require_cmd sed
require_cmd awk
require_cmd mktemp

# 1) Build the binary
log "Building binary"
mkdir -p "$ROOT_DIR/bin"
pushd "$ROOT_DIR" >/dev/null
go build -o "$BIN_PATH" .

# 2) Basic CLI sanity checks
log "Verifying CLI help and version"
"$BIN_PATH" --help >/dev/null
"$BIN_PATH" version

# 3) Start local server + wait for health
log "Starting server via local binary on port ${REGISTRY_PORT}"
reset_runtime_state
REGISTRY_LOG=$(mktemp)
start_local_server "$REGISTRY_PORT" "$REGISTRY_LOG"

log "Waiting for local registry server health"
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

# 6) Test filesystem MCP server on the local host
log "Testing filesystem MCP server on the local host"

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
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "${ROOT_DIR}"]
}
EOF

"$BIN_PATH" --registry "$REGISTRY_URL" register -c "$FS_CONFIG"

rm -f "$FS_CONFIG"
unset FS_CONFIG

"$BIN_PATH" --registry "$REGISTRY_URL" invoke filesystem__list_allowed_directories --input '{}' >/dev/null

# 7) Test stateful session mode
log "Testing stateful session mode (session reuse for faster subsequent calls)"
LOCAL_REGISTRY="$REGISTRY_URL"

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
unset FS_STATEFUL_CONFIG

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

# 8) Print Homebrew formula config snippet
log "Homebrew formula config (from .goreleaser.yaml)"
sed -n '/^brews:/,/^dockers:/p' "$ROOT_DIR/.goreleaser.yaml" || true

# 9) Run manual API error response verification against an isolated server
log "Running manual API error response verification"
BIN_PATH="$BIN_PATH" "$ROOT_DIR/scripts/test-api-error-responses.sh"
popd >/dev/null

log "All tests passed 🎉"

log "Cleaning up"
unset MCP_SERVER_INIT_REQ_TIMEOUT_SEC

log "All done!"
