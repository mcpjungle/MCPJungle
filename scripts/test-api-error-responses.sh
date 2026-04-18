#!/usr/bin/env bash

# This script tests that the MCP Jungle server returns appropriate error responses for invalid API requests.
# It starts an isolated server instance, initializes it, and then makes various API calls.
# It refreshes the mcpjungle binary when the checked-in Go sources are newer than the binary.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_PATH="${BIN_PATH:-$ROOT_DIR/bin/mcpjungle}"
PORT="${PORT:-9101}"
BASE_URL="http://127.0.0.1:${PORT}"
API_BASE_URL="${BASE_URL}/api/v0"

TMP_DIR="$(mktemp -d)"
SERVER_LOG="${TMP_DIR}/server.log"
SERVER_PID=""
KEEP_TMP_ON_FAILURE="${KEEP_TMP_ON_FAILURE:-0}"
FAILED=0

log() { printf "\n[API-ERROR-TEST] %s\n" "$*"; }

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: Required command '$1' not found in PATH" >&2
    exit 1
  fi
}

ensure_fresh_binary() {
  local needs_build=0

  if [[ ! -x "$BIN_PATH" ]]; then
    needs_build=1
  elif find "$ROOT_DIR" -name '*.go' -newer "$BIN_PATH" -print -quit | grep -q .; then
    needs_build=1
  fi

  if [[ "$needs_build" -eq 1 ]]; then
    log "Building fresh MCPJungle binary at ${BIN_PATH}"
    mkdir -p "$(dirname "$BIN_PATH")"
    (
      cd "$ROOT_DIR"
      go build -o "$BIN_PATH" .
    )
  fi
}

cleanup() {
  if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" >/dev/null 2>&1; then
    kill "$SERVER_PID" || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi

  if [[ "$FAILED" -eq 1 || "$KEEP_TMP_ON_FAILURE" -eq 1 ]]; then
    log "Temporary files kept at $TMP_DIR"
    log "Server log: $SERVER_LOG"
    return
  fi

  rm -rf "$TMP_DIR"
}

trap cleanup EXIT

wait_for_health() {
  local url=$1
  local attempts=${2:-30}
  local delay=${3:-1}

  for ((i=1; i<=attempts; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done

  echo "ERROR: Health check did not pass for $url after $((attempts * delay))s" >&2
  return 1
}

extract_json_string_field() {
  local field=$1
  local body=$2

  printf "%s" "$body" | sed -n "s/.*\"${field}\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p"
}

request() {
  local method=$1
  local path=$2
  local auth_token=${3:-}
  local data=${4:-}

  local body_file
  body_file="$(mktemp "$TMP_DIR/body.XXXXXX")"

  local curl_args=(
    -sS
    -o "$body_file"
    -w "%{http_code}"
    -X "$method"
  )

  if [[ -n "$auth_token" ]]; then
    curl_args+=(-H "Authorization: Bearer ${auth_token}")
  fi

  if [[ -n "$data" ]]; then
    curl_args+=(-H "Content-Type: application/json" --data "$data")
  fi

  local status
  status="$(curl "${curl_args[@]}" "${BASE_URL}${path}")"
  local body
  body="$(cat "$body_file")"
  rm -f "$body_file"

  printf "%s\n%s" "$status" "$body"
}

assert_status() {
  local name=$1
  local method=$2
  local path=$3
  local expected_status=$4
  local expected_body_fragment=$5
  local auth_token=${6:-}
  local data=${7:-}

  log "$name"

  local result
  result="$(request "$method" "$path" "$auth_token" "$data")"
  local status
  status="$(printf "%s" "$result" | sed -n '1p')"
  local body
  body="$(printf "%s" "$result" | sed -n '2,$p')"

  if [[ "$status" != "$expected_status" ]]; then
    FAILED=1
    echo "ERROR: ${name} expected status ${expected_status}, got ${status}" >&2
    echo "Body: ${body}" >&2
    exit 1
  fi

  if [[ -n "$expected_body_fragment" ]] && [[ "$body" != *"$expected_body_fragment"* ]]; then
    FAILED=1
    echo "ERROR: ${name} expected body to contain '${expected_body_fragment}'" >&2
    echo "Body: ${body}" >&2
    exit 1
  fi

  printf "[OK] %s -> %s\n" "$name" "$status"
}

log "Checking required commands"
require_cmd curl
require_cmd sed
require_cmd mktemp
require_cmd find
require_cmd grep
require_cmd go

ensure_fresh_binary

log "Starting isolated MCPJungle server on port ${PORT}"
(
  cd "$TMP_DIR"
  exec "$BIN_PATH" start --enterprise --port "$PORT"
) >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!

wait_for_health "${BASE_URL}/health"

log "Initializing isolated server in enterprise mode"
init_result="$(request "POST" "/init" "" '{"mode":"production"}')"
init_status="$(printf "%s" "$init_result" | sed -n '1p')"
init_body="$(printf "%s" "$init_result" | sed -n '2,$p')"
if [[ "$init_status" != "200" ]]; then
  FAILED=1
  echo "ERROR: failed to initialize isolated server, status=${init_status}" >&2
  echo "Body: ${init_body}" >&2
  exit 1
fi

ADMIN_TOKEN="$(extract_json_string_field "admin_access_token" "$init_body")"
if [[ -z "$ADMIN_TOKEN" ]]; then
  FAILED=1
  echo "ERROR: init response did not contain admin_access_token" >&2
  echo "Body: ${init_body}" >&2
  exit 1
fi

# Test cases for API responses

assert_status \
  "register server rejects invalid server name" \
  "POST" \
  "/api/v0/servers" \
  "400" \
  "invalid server name" \
  "$ADMIN_TOKEN" \
  '{"name":"bad__server","transport":"stdio","command":"echo"}'

assert_status \
  "register server rejects invalid http url" \
  "POST" \
  "/api/v0/servers" \
  "400" \
  "must be a valid http or https url" \
  "$ADMIN_TOKEN" \
  '{"name":"bad-http-url","transport":"streamable_http","url":"http:///missing-host"}'

assert_status \
  "get tool rejects invalid canonical name" \
  "GET" \
  "/api/v0/tool?name=invalid-name" \
  "400" \
  "does not contain a __ separator" \
  "$ADMIN_TOKEN"

assert_status \
  "get tool returns not found for valid canonical name" \
  "GET" \
  "/api/v0/tool?name=noserver__notool" \
  "404" \
  "not found" \
  "$ADMIN_TOKEN"

assert_status \
  "invoke tool rejects invalid canonical name" \
  "POST" \
  "/api/v0/tools/invoke" \
  "400" \
  "does not contain a __ separator" \
  "$ADMIN_TOKEN" \
  '{"name":"invalid-name"}'

assert_status \
  "get prompt rejects invalid canonical name" \
  "GET" \
  "/api/v0/prompt?name=invalid-name" \
  "400" \
  "does not contain a __ separator" \
  "$ADMIN_TOKEN"

assert_status \
  "get prompt returns not found for valid canonical name" \
  "GET" \
  "/api/v0/prompt?name=noserver__noprompt" \
  "404" \
  "not found" \
  "$ADMIN_TOKEN"

assert_status \
  "get resource rejects invalid mcpj uri" \
  "POST" \
  "/api/v0/resources/get" \
  "400" \
  "not a valid MCPJungle resource URI" \
  "$ADMIN_TOKEN" \
  '{"uri":"not-a-mcpj-uri"}'

assert_status \
  "get resource returns not found for valid mcpj uri" \
  "POST" \
  "/api/v0/resources/get" \
  "404" \
  "not found" \
  "$ADMIN_TOKEN" \
  '{"uri":"mcpj://res/ghost-server/ZmlsZTovL2Zvby50eHQ"}'

assert_status \
  "read resource rejects invalid mcpj uri" \
  "POST" \
  "/api/v0/resources/read" \
  "400" \
  "not a valid MCPJungle resource URI" \
  "$ADMIN_TOKEN" \
  '{"uri":"not-a-mcpj-uri"}'

assert_status \
  "read resource returns not found for valid mcpj uri" \
  "POST" \
  "/api/v0/resources/read" \
  "404" \
  "not found" \
  "$ADMIN_TOKEN" \
  '{"uri":"mcpj://res/ghost-server/ZmlsZTovL2Zvby50eHQ"}'

assert_status \
  "create tool group rejects invalid group name" \
  "POST" \
  "/api/v0/tool-groups" \
  "400" \
  "invalid group name" \
  "$ADMIN_TOKEN" \
  '{"name":"-bad-group","included_tools":["ghost__tool"]}'

assert_status \
  "create tool group rejects empty effective tool set" \
  "POST" \
  "/api/v0/tool-groups" \
  "400" \
  "at least one tool" \
  "$ADMIN_TOKEN" \
  '{"name":"empty-group"}'

assert_status \
  "get tool group returns not found when group is missing" \
  "GET" \
  "/api/v0/tool-groups/ghost-group" \
  "404" \
  "not found" \
  "$ADMIN_TOKEN"

assert_status \
  "update tool group returns not found when group is missing" \
  "PUT" \
  "/api/v0/tool-groups/ghost-group" \
  "404" \
  "not found" \
  "$ADMIN_TOKEN" \
  '{"name":"ghost-group","description":"updated"}'

assert_status \
  "create client rejects invalid custom access token" \
  "POST" \
  "/api/v0/clients" \
  "400" \
  "invalid access token" \
  "$ADMIN_TOKEN" \
  '{"name":"bad-client","access_token":"invalid token with spaces"}'

assert_status \
  "create valid client succeeds" \
  "POST" \
  "/api/v0/clients" \
  "201" \
  "" \
  "$ADMIN_TOKEN" \
  '{"name":"good-client"}'

assert_status \
  "update client rejects invalid custom access token" \
  "PUT" \
  "/api/v0/clients/good-client" \
  "400" \
  "invalid access token" \
  "$ADMIN_TOKEN" \
  '{"access_token":"invalid token with spaces"}'

assert_status \
  "update missing client returns not found" \
  "PUT" \
  "/api/v0/clients/ghost-client" \
  "404" \
  "not found" \
  "$ADMIN_TOKEN" \
  '{"access_token":"validtoken12345"}'

assert_status \
  "create user rejects invalid custom access token" \
  "POST" \
  "/api/v0/users" \
  "400" \
  "invalid access token" \
  "$ADMIN_TOKEN" \
  '{"username":"bad-user","access_token":"bad token"}'

assert_status \
  "create valid user succeeds" \
  "POST" \
  "/api/v0/users" \
  "201" \
  "" \
  "$ADMIN_TOKEN" \
  '{"username":"alice","access_token":"validtoken12345"}'

assert_status \
  "update user rejects invalid custom access token" \
  "PUT" \
  "/api/v0/users/alice" \
  "400" \
  "invalid access token" \
  "$ADMIN_TOKEN" \
  '{"username":"alice","access_token":"bad token"}'

assert_status \
  "update missing user returns not found" \
  "PUT" \
  "/api/v0/users/ghost" \
  "404" \
  "not found" \
  "$ADMIN_TOKEN" \
  '{"username":"ghost","access_token":"validtoken67890"}'

assert_status \
  "delete missing user returns not found" \
  "DELETE" \
  "/api/v0/users/ghost" \
  "404" \
  "not found" \
  "$ADMIN_TOKEN"

assert_status \
  "delete admin user rejects invalid operation" \
  "DELETE" \
  "/api/v0/users/admin" \
  "400" \
  "cannot delete an admin user" \
  "$ADMIN_TOKEN"

log "All API error response checks passed"
