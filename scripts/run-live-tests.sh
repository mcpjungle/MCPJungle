#!/usr/bin/env bash
#
# run-live-tests.sh — start all required services and run the TestLive_* suite.
#
# Usage:
#   ./scripts/run-live-tests.sh          # dev tests only (enterprise skipped)
#   ./scripts/run-live-tests.sh --enterprise  # also run enterprise tests
#
# What this script does:
#   1. Builds the MCPJungle binary
#   2. Starts server-everything upstreams (stdio via MCPJungle, SSE on :3001, streamableHttp on :3002)
#   3. Starts MCPJungle dev on :8080 and registers the three upstreams
#   4. (Optional) Starts MCPJungle enterprise on :8081, initialises it, and creates test clients
#   5. Runs:  go test ./internal/e2e/live/ -run TestLive -v -timeout 120s
#   6. Cleans up all background processes on exit
#

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_PATH="$ROOT_DIR/bin/mcpjungle"
WITH_ENTERPRISE=false

for arg in "$@"; do
  case "$arg" in
    --enterprise) WITH_ENTERPRISE=true ;;
  esac
done

log() { printf "\n[LIVE] %s\n" "$*"; }

# Free ports that may be left over from a previous interrupted run
for port in 8080 8081 3001 3002; do
  lsof -ti :"$port" | xargs kill -9 2>/dev/null || true
done
rm -f "$ROOT_DIR/mcpjungle.db" "$ROOT_DIR/mcp.db"

# Cleanup all background processes on exit
PIDS=()
cleanup() {
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" 2>/dev/null || true
  done
  rm -f "$ROOT_DIR/mcpjungle.db" "$ROOT_DIR/mcp.db"
  rm -rf /tmp/ent-live-instance
}
trap cleanup EXIT

wait_for_health() {
  local url=$1
  local attempts=${2:-30}
  local delay=${3:-2}
  for ((i=1; i<=attempts; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done
  echo "ERROR: health check failed for $url" >&2
  return 1
}

# 1) Build binary
log "Building binary"
mkdir -p "$ROOT_DIR/bin"
pushd "$ROOT_DIR" >/dev/null
go build -o "$BIN_PATH" .

# 2) Start server-everything upstreams
log "Starting server-everything SSE on :3001"
npx @modelcontextprotocol/server-everything sse >/tmp/everything-sse.log 2>&1 &
PIDS+=($!)

log "Starting server-everything streamableHttp on :3002"
PORT=3002 npx @modelcontextprotocol/server-everything streamableHttp >/tmp/everything-http.log 2>&1 &
PIDS+=($!)

# 3) Start MCPJungle dev on :8080
log "Starting MCPJungle dev on :8080"
rm -f "$ROOT_DIR/mcpjungle.db" "$ROOT_DIR/mcp.db"
go run . start >/tmp/mcpjungle-dev.log 2>&1 &
PIDS+=($!)

log "Waiting for MCPJungle dev health"
wait_for_health "http://localhost:8080/health"

log "Registering upstreams"
curl -sX POST http://localhost:8080/api/v0/servers -H "Content-Type: application/json" \
  -d '{"name":"everything","transport":"stdio","command":"npx","args":["-y","@modelcontextprotocol/server-everything","stdio"]}' >/dev/null
curl -sX POST http://localhost:8080/api/v0/servers -H "Content-Type: application/json" \
  -d '{"name":"everything-sse","transport":"sse","url":"http://localhost:3001/sse"}' >/dev/null
curl -sX POST http://localhost:8080/api/v0/servers -H "Content-Type: application/json" \
  -d '{"name":"everything-http","transport":"streamable_http","url":"http://localhost:3002/mcp"}' >/dev/null

# 4) (Optional) Enterprise instance on :8081
MCPJUNGLE_ENT_ADMIN_TOKEN=""
MCPJUNGLE_ENT_CLIENT_TOKEN=""
MCPJUNGLE_ENT_BLOCKED_TOKEN=""

if [ "$WITH_ENTERPRISE" = "true" ]; then
  log "Starting MCPJungle enterprise on :8081"
  mkdir -p /tmp/ent-live-instance
  cd /tmp/ent-live-instance
  PORT=8081 "$BIN_PATH" start --enterprise >/tmp/mcpjungle-ent.log 2>&1 &
  PIDS+=($!)
  cd "$ROOT_DIR"

  wait_for_health "http://localhost:8081/health"

  INIT=$(curl -sX POST http://localhost:8081/init -H "Content-Type: application/json" -d '{"mode":"enterprise"}')
  MCPJUNGLE_ENT_ADMIN_TOKEN=$(echo "$INIT" | python3 -c "import sys,json; print(json.load(sys.stdin)['admin_access_token'])")

  curl -sX POST http://localhost:8081/api/v0/servers \
    -H "Content-Type: application/json" -H "Authorization: Bearer $MCPJUNGLE_ENT_ADMIN_TOKEN" \
    -d '{"name":"everything","transport":"stdio","command":"npx","args":["-y","@modelcontextprotocol/server-everything","stdio"]}' >/dev/null

  MCPJUNGLE_ENT_CLIENT_TOKEN=$(curl -sX POST http://localhost:8081/api/v0/clients \
    -H "Content-Type: application/json" -H "Authorization: Bearer $MCPJUNGLE_ENT_ADMIN_TOKEN" \
    -d '{"name":"allowed","allow_list":["everything"]}' | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

  MCPJUNGLE_ENT_BLOCKED_TOKEN=$(curl -sX POST http://localhost:8081/api/v0/clients \
    -H "Content-Type: application/json" -H "Authorization: Bearer $MCPJUNGLE_ENT_ADMIN_TOKEN" \
    -d '{"name":"blocked","allow_list":["other"]}' | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

  log "Enterprise tokens obtained"
fi

popd >/dev/null

# 5) Run the live test suite
log "Running TestLive_* suite"
MCPJUNGLE_ENT_ADMIN_TOKEN="$MCPJUNGLE_ENT_ADMIN_TOKEN" \
MCPJUNGLE_ENT_CLIENT_TOKEN="$MCPJUNGLE_ENT_CLIENT_TOKEN" \
MCPJUNGLE_ENT_BLOCKED_TOKEN="$MCPJUNGLE_ENT_BLOCKED_TOKEN" \
go test "$ROOT_DIR/internal/e2e/live/" -run TestLive -v -timeout 120s

log "All live tests passed ✓"
