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
OAUTH_MOCK_PORT="${OAUTH_MOCK_PORT:-18081}"
OAUTH_MOCK_URL="http://127.0.0.1:${OAUTH_MOCK_PORT}"
ENTERPRISE_PORT="${ENTERPRISE_PORT:-18082}"
ENTERPRISE_URL="http://127.0.0.1:${ENTERPRISE_PORT}"

# Simple logger for readable output
log() { printf "\n[TEST] %s\n" "$*"; }

# Ensure a command is installed before proceeding
require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: Required command '$1' not found in PATH" >&2
    exit 1
  fi
}

extract_json_string_field() {
  local field=$1
  local body=$2

  printf "%s" "$body" | sed -n "s/.*\"${field}\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p"
}

decode_json_string() {
  local value=$1

  printf "%s" "$value" | sed \
    -e 's#\\/#/#g' \
    -e 's/\\u0026/\&/g' \
    -e 's/\\u003d/=/g' \
    -e 's/\\u003f/?/g'
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

  if [[ -n "${ENTERPRISE_SERVER_PID:-}" ]] && kill -0 "$ENTERPRISE_SERVER_PID" >/dev/null 2>&1; then
    kill "$ENTERPRISE_SERVER_PID" || true
    wait "$ENTERPRISE_SERVER_PID" 2>/dev/null || true
  fi

  if [[ -n "${OAUTH_MOCK_PID:-}" ]] && kill -0 "$OAUTH_MOCK_PID" >/dev/null 2>&1; then
    kill "$OAUTH_MOCK_PID" || true
    wait "$OAUTH_MOCK_PID" 2>/dev/null || true
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

  if [[ -n "${GROUP_CONFIG:-}" ]]; then
    rm -f "$GROUP_CONFIG"
  fi

  if [[ -n "${ENTERPRISE_GROUP_CONFIG:-}" ]]; then
    rm -f "$ENTERPRISE_GROUP_CONFIG"
  fi

  if [[ -n "${ENTERPRISE_FS_CONFIG:-}" ]]; then
    rm -f "$ENTERPRISE_FS_CONFIG"
  fi

  if [[ -n "${ENTERPRISE_EXPORT_DIR:-}" ]]; then
    rm -rf "$ENTERPRISE_EXPORT_DIR"
  fi

  if [[ -n "${REGISTRY_LOG:-}" ]]; then
    rm -f "$REGISTRY_LOG"
  fi

  if [[ -n "${ENTERPRISE_LOG:-}" ]]; then
    rm -f "$ENTERPRISE_LOG"
  fi

  if [[ -n "${OAUTH_MOCK_LOG:-}" ]]; then
    rm -f "$OAUTH_MOCK_LOG"
  fi

  if [[ -n "${OAUTH_MOCK_SOURCE:-}" ]]; then
    rm -f "$OAUTH_MOCK_SOURCE"
  fi

  if [[ -n "${OAUTH_MOCK_DIR:-}" ]]; then
    rm -rf "$OAUTH_MOCK_DIR"
  fi

  if [[ -n "${REGISTRY_RUNTIME_DIR:-}" ]]; then
    rm -rf "$REGISTRY_RUNTIME_DIR"
  fi

  if [[ -n "${ENTERPRISE_RUNTIME_DIR:-}" ]]; then
    rm -rf "$ENTERPRISE_RUNTIME_DIR"
  fi

  if [[ -n "${ENTERPRISE_ADMIN_HOME:-}" ]]; then
    rm -rf "$ENTERPRISE_ADMIN_HOME"
  fi

  if [[ -n "${ENTERPRISE_USER_HOME:-}" ]]; then
    rm -rf "$ENTERPRISE_USER_HOME"
  fi
}

cleanup_runtime_state() {
  rm -f "$ROOT_DIR/mcpjungle.db" "$ROOT_DIR/mcp.db"
  rm -rf "$ROOT_DIR/mcpjungle_data"
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
  local workdir=$3
  shift 3

  (
    cd "$workdir"
    exec "$BIN_PATH" start --port "$port" "$@"
  ) >"$log_file" 2>&1 &
  local pid=$!

  if [[ "$port" == "$REGISTRY_PORT" ]]; then
    REGISTRY_SERVER_PID=$pid
  elif [[ "$port" == "$ENTERPRISE_PORT" ]]; then
    ENTERPRISE_SERVER_PID=$pid
  else
    BIN_SERVER_PID=$pid
  fi
}

start_mock_oauth_upstream() {
  OAUTH_MOCK_DIR=$(mktemp -d "$ROOT_DIR/.mock-oauth-mcp.XXXXXX")
  OAUTH_MOCK_SOURCE="$OAUTH_MOCK_DIR/main.go"
  OAUTH_MOCK_LOG=$(mktemp)

  cat >"$OAUTH_MOCK_SOURCE" <<EOF
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	baseURL := os.Getenv("OAUTH_MOCK_BASE_URL")
	accessToken := "mock-access-token"

	upstreamMCP := mcpserver.NewMCPServer("oauth-upstream", "0.1.0")
	upstreamMCP.AddTool(
		mcp.NewTool("echo", mcp.WithDescription("Echoes the msg argument"), mcp.WithString("msg", mcp.Required())),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, _ := req.GetArguments()["msg"].(string)
			return mcp.NewToolResultText("oauth echo: " + msg), nil
		},
	)
	streamable := mcpserver.NewStreamableHTTPServer(upstreamMCP)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource/mcp", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"authorization_servers": []string{baseURL},
			"resource":              baseURL + "/mcp",
		})
	})
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 baseURL,
			"authorization_endpoint": baseURL + "/authorize",
			"token_endpoint":         baseURL + "/token",
			"registration_endpoint":  baseURL + "/register",
			"response_types_supported": []string{
				"code",
			},
		})
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"client_id": "mock-client-id",
		})
	})
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")
		u, err := url.Parse(redirectURI)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		q := u.Query()
		q.Set("code", "mock-auth-code")
		q.Set("state", state)
		u.RawQuery = q.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  accessToken,
			"token_type":    "Bearer",
			"refresh_token": "mock-refresh-token",
			"expires_in":    3600,
			"scope":         "mcp.read",
		})
	})
	mux.Handle("/mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+accessToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}
		streamable.ServeHTTP(w, r)
	}))

	if err := http.ListenAndServe("127.0.0.1:"+os.Getenv("OAUTH_MOCK_PORT"), mux); err != nil {
		panic(err)
	}
}
EOF

  OAUTH_MOCK_PORT="$OAUTH_MOCK_PORT" OAUTH_MOCK_BASE_URL="$OAUTH_MOCK_URL" \
    go run "$OAUTH_MOCK_SOURCE" >"$OAUTH_MOCK_LOG" 2>&1 &
  OAUTH_MOCK_PID=$!
}

start_oauth_registration() {
  local query=${1:-}
  local request_body
  request_body=$(cat <<EOF
{"name":"oauthsrv","description":"OAuth-protected upstream MCP server","transport":"streamable_http","url":"${OAUTH_MOCK_URL}/mcp","oauth_redirect_uri":"http://127.0.0.1:9999/oauth/callback","oauth_scopes":["mcp.read"]}
EOF
)

  local response_file
  response_file=$(mktemp)
  local status
  status=$(curl -sS -o "$response_file" -w "%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    --data "$request_body" \
    "${REGISTRY_URL}/api/v0/servers${query}")
  local body
  body="$(cat "$response_file")"
  rm -f "$response_file"

  if [[ "$status" != "202" ]]; then
    echo "ERROR: expected OAuth registration to return 202, got ${status}" >&2
    echo "Body: ${body}" >&2
    exit 1
  fi

  OAUTH_SESSION_ID=$(extract_json_string_field "session_id" "$body")
  OAUTH_AUTH_URL=$(extract_json_string_field "authorization_url" "$body")
  OAUTH_AUTH_URL=$(decode_json_string "$OAUTH_AUTH_URL")
  if [[ -z "$OAUTH_SESSION_ID" || -z "$OAUTH_AUTH_URL" ]]; then
    echo "ERROR: OAuth registration response did not include session_id and authorization_url" >&2
    echo "Body: ${body}" >&2
    exit 1
  fi
}

complete_oauth_registration() {
  local auth_headers
  auth_headers=$(mktemp)
  curl -sS -D "$auth_headers" -o /dev/null "$OAUTH_AUTH_URL"
  local callback_url
  callback_url=$(sed -n 's/^[Ll]ocation: \(.*\)\r$/\1/p' "$auth_headers" | tail -n 1)
  rm -f "$auth_headers"
  if [[ -z "$callback_url" ]]; then
    echo "ERROR: OAuth authorize endpoint did not return a callback redirect" >&2
    exit 1
  fi

  local oauth_code
  oauth_code=$(printf "%s" "$callback_url" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')
  local oauth_state
  oauth_state=$(printf "%s" "$callback_url" | sed -n 's/.*[?&]state=\([^&]*\).*/\1/p')
  if [[ -z "$oauth_code" || -z "$oauth_state" ]]; then
    echo "ERROR: OAuth callback URL did not include both code and state" >&2
    echo "Callback URL: ${callback_url}" >&2
    exit 1
  fi

  local complete_body
  complete_body=$(cat <<EOF
{"code":"${oauth_code}","state":"${oauth_state}"}
EOF
)

  local response_file
  response_file=$(mktemp)
  local status
  status=$(curl -sS -o "$response_file" -w "%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    --data "$complete_body" \
    "${REGISTRY_URL}/api/v0/upstream_oauth/sessions/${OAUTH_SESSION_ID}/complete")
  local body
  body="$(cat "$response_file")"
  rm -f "$response_file"

  if [[ "$status" != "201" ]]; then
    echo "ERROR: expected OAuth completion to return 201, got ${status}" >&2
    echo "Body: ${body}" >&2
    exit 1
  fi
}

register_stateful_oauth_server() {
  local request_body
  request_body=$(cat <<EOF
{"name":"oauthstateful","description":"OAuth-protected stateful upstream MCP server","transport":"streamable_http","url":"${OAUTH_MOCK_URL}/mcp","session_mode":"stateful","oauth_redirect_uri":"http://127.0.0.1:9998/oauth/callback","oauth_scopes":["mcp.read"]}
EOF
)

  local response_file
  response_file=$(mktemp)
  local status
  status=$(curl -sS -o "$response_file" -w "%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    --data "$request_body" \
    "${REGISTRY_URL}/api/v0/servers")
  local body
  body="$(cat "$response_file")"
  rm -f "$response_file"

  if [[ "$status" != "202" ]]; then
    echo "ERROR: expected stateful OAuth registration to return 202, got ${status}" >&2
    echo "Body: ${body}" >&2
    exit 1
  fi

  OAUTH_STATEFUL_SESSION_ID=$(extract_json_string_field "session_id" "$body")
  OAUTH_STATEFUL_AUTH_URL=$(extract_json_string_field "authorization_url" "$body")
  OAUTH_STATEFUL_AUTH_URL=$(decode_json_string "$OAUTH_STATEFUL_AUTH_URL")
  if [[ -z "$OAUTH_STATEFUL_SESSION_ID" || -z "$OAUTH_STATEFUL_AUTH_URL" ]]; then
    echo "ERROR: stateful OAuth registration response did not include session_id and authorization_url" >&2
    echo "Body: ${body}" >&2
    exit 1
  fi
}

complete_stateful_oauth_registration() {
  local auth_headers
  auth_headers=$(mktemp)
  curl -sS -D "$auth_headers" -o /dev/null "$OAUTH_STATEFUL_AUTH_URL"
  local callback_url
  callback_url=$(sed -n 's/^[Ll]ocation: \(.*\)\r$/\1/p' "$auth_headers" | tail -n 1)
  rm -f "$auth_headers"
  if [[ -z "$callback_url" ]]; then
    echo "ERROR: stateful OAuth authorize endpoint did not return a callback redirect" >&2
    exit 1
  fi

  local oauth_code
  oauth_code=$(printf "%s" "$callback_url" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')
  local oauth_state
  oauth_state=$(printf "%s" "$callback_url" | sed -n 's/.*[?&]state=\([^&]*\).*/\1/p')
  if [[ -z "$oauth_code" || -z "$oauth_state" ]]; then
    echo "ERROR: stateful OAuth callback URL did not include both code and state" >&2
    echo "Callback URL: ${callback_url}" >&2
    exit 1
  fi

  local complete_body
  complete_body=$(cat <<EOF
{"code":"${oauth_code}","state":"${oauth_state}"}
EOF
)

  local response_file
  response_file=$(mktemp)
  local status
  status=$(curl -sS -o "$response_file" -w "%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    --data "$complete_body" \
    "${REGISTRY_URL}/api/v0/upstream_oauth/sessions/${OAUTH_STATEFUL_SESSION_ID}/complete")
  local body
  body="$(cat "$response_file")"
  rm -f "$response_file"

  if [[ "$status" != "201" ]]; then
    echo "ERROR: expected stateful OAuth completion to return 201, got ${status}" >&2
    echo "Body: ${body}" >&2
    exit 1
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

# 2) Start local server + wait for health
log "Starting server via local binary on port ${REGISTRY_PORT}"
reset_runtime_state
REGISTRY_RUNTIME_DIR=$(mktemp -d)
REGISTRY_LOG=$(mktemp)
start_local_server "$REGISTRY_PORT" "$REGISTRY_LOG" "$REGISTRY_RUNTIME_DIR"

log "Waiting for local registry server health"
wait_for_health "$REGISTRY_URL/health"

# 3) Basic CLI sanity checks
log "Verifying CLI help and version"
"$BIN_PATH" --help >/dev/null
"$BIN_PATH" --registry "$REGISTRY_URL" version

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

# 6) Test upstream OAuth registration and later authenticated tool calls
log "Testing upstream OAuth registration flow with a local mock MCP server"
start_mock_oauth_upstream
wait_for_health "${OAUTH_MOCK_URL}/healthz"
start_oauth_registration
complete_oauth_registration

OAUTH_TOOLS_OUTPUT=$("$BIN_PATH" --registry "$REGISTRY_URL" list tools 2>&1)
if [[ "$OAUTH_TOOLS_OUTPUT" != *"oauthsrv__echo"* ]]; then
  echo "ERROR: expected oauthsrv__echo to be registered after OAuth completion" >&2
  echo "$OAUTH_TOOLS_OUTPUT" >&2
  exit 1
fi

OAUTH_INVOKE_OUTPUT=$("$BIN_PATH" --registry "$REGISTRY_URL" invoke oauthsrv__echo --input '{"msg":"hello oauth"}' 2>&1)
if [[ "$OAUTH_INVOKE_OUTPUT" != *"oauth echo: hello oauth"* ]]; then
  echo "ERROR: expected oauth-protected tool invocation to succeed after OAuth completion" >&2
  echo "$OAUTH_INVOKE_OUTPUT" >&2
  exit 1
fi

OAUTH_SECOND_INVOKE_OUTPUT=$("$BIN_PATH" --registry "$REGISTRY_URL" invoke oauthsrv__echo --input '{"msg":"hello oauth again"}' 2>&1)
if [[ "$OAUTH_SECOND_INVOKE_OUTPUT" != *"oauth echo: hello oauth again"* ]]; then
  echo "ERROR: expected second oauth-protected tool invocation to reuse stored credentials successfully" >&2
  echo "$OAUTH_SECOND_INVOKE_OUTPUT" >&2
  exit 1
fi

log "Testing OAuth re-registration with force=true"
start_oauth_registration "?force=true"
complete_oauth_registration

OAUTH_SERVERS_BODY=$(curl -sS "${REGISTRY_URL}/api/v0/servers")
OAUTH_SERVER_COUNT=$(printf "%s" "$OAUTH_SERVERS_BODY" | grep -o '"name":"oauthsrv"' | wc -l | tr -d ' ')
if [[ "$OAUTH_SERVER_COUNT" != "1" ]]; then
  echo "ERROR: expected exactly one oauthsrv entry after force re-registration, got ${OAUTH_SERVER_COUNT}" >&2
  echo "Body: ${OAUTH_SERVERS_BODY}" >&2
  exit 1
fi

OAUTH_FORCE_INVOKE_OUTPUT=$("$BIN_PATH" --registry "$REGISTRY_URL" invoke oauthsrv__echo --input '{"msg":"hello after force"}' 2>&1)
if [[ "$OAUTH_FORCE_INVOKE_OUTPUT" != *"oauth echo: hello after force"* ]]; then
  echo "ERROR: expected oauth-protected tool invocation to succeed after force re-registration" >&2
  echo "$OAUTH_FORCE_INVOKE_OUTPUT" >&2
  exit 1
fi

log "Testing upstream OAuth flow with session_mode=stateful"
register_stateful_oauth_server
complete_stateful_oauth_registration

OAUTH_STATEFUL_TOOLS_OUTPUT=$("$BIN_PATH" --registry "$REGISTRY_URL" list tools 2>&1)
if [[ "$OAUTH_STATEFUL_TOOLS_OUTPUT" != *"oauthstateful__echo"* ]]; then
  echo "ERROR: expected oauthstateful__echo to be registered after stateful OAuth completion" >&2
  echo "$OAUTH_STATEFUL_TOOLS_OUTPUT" >&2
  exit 1
fi

OAUTH_STATEFUL_FIRST_INVOKE=$("$BIN_PATH" --registry "$REGISTRY_URL" invoke oauthstateful__echo --input '{"msg":"hello stateful oauth"}' 2>&1)
if [[ "$OAUTH_STATEFUL_FIRST_INVOKE" != *"oauth echo: hello stateful oauth"* ]]; then
  echo "ERROR: expected first stateful OAuth tool invocation to succeed" >&2
  echo "$OAUTH_STATEFUL_FIRST_INVOKE" >&2
  exit 1
fi

OAUTH_STATEFUL_SECOND_INVOKE=$("$BIN_PATH" --registry "$REGISTRY_URL" invoke oauthstateful__echo --input '{"msg":"hello stateful oauth again"}' 2>&1)
if [[ "$OAUTH_STATEFUL_SECOND_INVOKE" != *"oauth echo: hello stateful oauth again"* ]]; then
  echo "ERROR: expected second stateful OAuth tool invocation to succeed" >&2
  echo "$OAUTH_STATEFUL_SECOND_INVOKE" >&2
  exit 1
fi

# 7) Test filesystem MCP server on the local host
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

# 8) Test stateful session mode
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

# 9) Test tool groups
log "Testing tool groups"

GROUP_CONFIG=$(mktemp)
cat > "$GROUP_CONFIG" <<EOF
{
  "name": "test-group",
  "description": "Curated integration test tools",
  "included_tools": [
    "context7__resolve-library-id",
    "oauthsrv__echo"
  ]
}
EOF

"$BIN_PATH" --registry "$REGISTRY_URL" create group -c "$GROUP_CONFIG" >/dev/null
rm -f "$GROUP_CONFIG"
unset GROUP_CONFIG

GROUPS_OUTPUT=$("$BIN_PATH" --registry "$REGISTRY_URL" list groups 2>&1)
if [[ "$GROUPS_OUTPUT" != *"test-group"* ]]; then
  echo "ERROR: expected test-group to be listed after creation" >&2
  echo "$GROUPS_OUTPUT" >&2
  exit 1
fi

GROUP_DETAILS=$("$BIN_PATH" --registry "$REGISTRY_URL" get group test-group 2>&1)
if [[ "$GROUP_DETAILS" != *"context7__resolve-library-id"* || "$GROUP_DETAILS" != *"oauthsrv__echo"* ]]; then
  echo "ERROR: expected test-group details to include both configured tools" >&2
  echo "$GROUP_DETAILS" >&2
  exit 1
fi

GROUP_TOOLS_OUTPUT=$("$BIN_PATH" --registry "$REGISTRY_URL" list tools --group test-group 2>&1)
if [[ "$GROUP_TOOLS_OUTPUT" != *"context7__resolve-library-id"* || "$GROUP_TOOLS_OUTPUT" != *"oauthsrv__echo"* ]]; then
  echo "ERROR: expected grouped tool listing to include configured tools" >&2
  echo "$GROUP_TOOLS_OUTPUT" >&2
  exit 1
fi

GROUP_INVOKE_OUTPUT=$("$BIN_PATH" --registry "$REGISTRY_URL" invoke oauthsrv__echo --group test-group --input '{"msg":"hello group"}' 2>&1)
if [[ "$GROUP_INVOKE_OUTPUT" != *"oauth echo: hello group"* ]]; then
  echo "ERROR: expected grouped tool invocation to succeed" >&2
  echo "$GROUP_INVOKE_OUTPUT" >&2
  exit 1
fi

# 10) Test enterprise-only user and MCP client features
log "Testing enterprise users and MCP clients"

ENTERPRISE_RUNTIME_DIR=$(mktemp -d)
ENTERPRISE_LOG=$(mktemp)
ENTERPRISE_ADMIN_HOME=$(mktemp -d)
ENTERPRISE_USER_HOME=$(mktemp -d)
start_local_server "$ENTERPRISE_PORT" "$ENTERPRISE_LOG" "$ENTERPRISE_RUNTIME_DIR" --enterprise
wait_for_health "${ENTERPRISE_URL}/health"

HOME="$ENTERPRISE_ADMIN_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" init-server >/dev/null

ENTERPRISE_FS_CONFIG=$(mktemp)
cat > "$ENTERPRISE_FS_CONFIG" <<EOF
{
  "name": "enterprise-fs",
  "transport": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "${ROOT_DIR}"]
}
EOF

HOME="$ENTERPRISE_ADMIN_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" register -c "$ENTERPRISE_FS_CONFIG" >/dev/null
rm -f "$ENTERPRISE_FS_CONFIG"
unset ENTERPRISE_FS_CONFIG

ENTERPRISE_USERS_OUTPUT=$(HOME="$ENTERPRISE_ADMIN_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" list users 2>&1)
if [[ "$ENTERPRISE_USERS_OUTPUT" != *"admin"* ]]; then
  echo "ERROR: expected enterprise user list to include admin after initialization" >&2
  echo "$ENTERPRISE_USERS_OUTPUT" >&2
  exit 1
fi

ENTERPRISE_USER_TOKEN="enterprise_user_token_12345"
HOME="$ENTERPRISE_ADMIN_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" create user enterprise-user --access-token "$ENTERPRISE_USER_TOKEN" >/dev/null

ENTERPRISE_USERS_OUTPUT=$(HOME="$ENTERPRISE_ADMIN_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" list users 2>&1)
if [[ "$ENTERPRISE_USERS_OUTPUT" != *"enterprise-user"* ]]; then
  echo "ERROR: expected enterprise user list to include enterprise-user" >&2
  echo "$ENTERPRISE_USERS_OUTPUT" >&2
  exit 1
fi

HOME="$ENTERPRISE_USER_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" login "$ENTERPRISE_USER_TOKEN" >/dev/null
ENTERPRISE_SERVERS_OUTPUT=$(HOME="$ENTERPRISE_USER_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" list servers 2>&1)
if [[ "$ENTERPRISE_SERVERS_OUTPUT" != *"enterprise-fs"* ]]; then
  echo "ERROR: expected logged-in enterprise user to be able to list registered servers" >&2
  echo "$ENTERPRISE_SERVERS_OUTPUT" >&2
  exit 1
fi

ENTERPRISE_CLIENT_TOKEN="enterprise_client_token_12345"
HOME="$ENTERPRISE_ADMIN_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" create mcp-client enterprise-client --allow "enterprise-fs" --access-token "$ENTERPRISE_CLIENT_TOKEN" >/dev/null

ENTERPRISE_CLIENTS_OUTPUT=$(HOME="$ENTERPRISE_ADMIN_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" list mcp-clients 2>&1)
if [[ "$ENTERPRISE_CLIENTS_OUTPUT" != *"enterprise-client"* || "$ENTERPRISE_CLIENTS_OUTPUT" != *"enterprise-fs"* ]]; then
  echo "ERROR: expected enterprise MCP client list to include enterprise-client and its allow-list" >&2
  echo "$ENTERPRISE_CLIENTS_OUTPUT" >&2
  exit 1
fi

ENTERPRISE_GROUP_CONFIG=$(mktemp)
cat > "$ENTERPRISE_GROUP_CONFIG" <<EOF
{
  "name": "enterprise-group",
  "description": "Enterprise integration test group",
  "included_servers": ["enterprise-fs"]
}
EOF

HOME="$ENTERPRISE_ADMIN_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" create group -c "$ENTERPRISE_GROUP_CONFIG" >/dev/null
rm -f "$ENTERPRISE_GROUP_CONFIG"
unset ENTERPRISE_GROUP_CONFIG

# 11) Test enterprise export for currently supported entities
log "Testing enterprise export"

ENTERPRISE_EXPORT_DIR=$(mktemp -d)
rm -rf "$ENTERPRISE_EXPORT_DIR"
HOME="$ENTERPRISE_ADMIN_HOME" "$BIN_PATH" --registry "$ENTERPRISE_URL" export --dir "$ENTERPRISE_EXPORT_DIR" >/dev/null

if [[ ! -f "$ENTERPRISE_EXPORT_DIR/servers/enterprise-fs.json" ]]; then
  echo "ERROR: expected enterprise export to include server config for enterprise-fs" >&2
  exit 1
fi

if [[ ! -f "$ENTERPRISE_EXPORT_DIR/groups/enterprise-group.json" ]]; then
  echo "ERROR: expected enterprise export to include tool group config for enterprise-group" >&2
  exit 1
fi

if ! grep -q '"name": "enterprise-fs"' "$ENTERPRISE_EXPORT_DIR/servers/enterprise-fs.json"; then
  echo "ERROR: expected exported server config to preserve the server name" >&2
  cat "$ENTERPRISE_EXPORT_DIR/servers/enterprise-fs.json" >&2
  exit 1
fi

if ! grep -q '"name": "enterprise-group"' "$ENTERPRISE_EXPORT_DIR/groups/enterprise-group.json" || \
   ! grep -q '"enterprise-fs"' "$ENTERPRISE_EXPORT_DIR/groups/enterprise-group.json"; then
  echo "ERROR: expected exported tool group config to preserve the group definition" >&2
  cat "$ENTERPRISE_EXPORT_DIR/groups/enterprise-group.json" >&2
  exit 1
fi

# 12) Print Homebrew formula config snippet
log "Homebrew formula config (from .goreleaser.yaml)"
sed -n '/^brews:/,/^dockers:/p' "$ROOT_DIR/.goreleaser.yaml" || true

# 13) Run manual API error response verification against an isolated server
log "Running manual API error response verification"
BIN_PATH="$BIN_PATH" "$ROOT_DIR/scripts/test-api-error-responses.sh"
popd >/dev/null

log "All tests passed 🎉"

log "Cleaning up"
unset MCP_SERVER_INIT_REQ_TIMEOUT_SEC

log "All done!"
