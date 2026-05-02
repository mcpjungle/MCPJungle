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

  if [[ -n "${REGISTRY_LOG:-}" ]]; then
    rm -f "$REGISTRY_LOG"
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
REGISTRY_LOG=$(mktemp)
start_local_server "$REGISTRY_PORT" "$REGISTRY_LOG"

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

OAUTH_REGISTER_BODY=$(cat <<EOF
{"name":"oauthsrv","description":"OAuth-protected upstream MCP server","transport":"streamable_http","url":"${OAUTH_MOCK_URL}/mcp","oauth_redirect_uri":"http://127.0.0.1:9999/oauth/callback","oauth_scopes":["mcp.read"]}
EOF
)

OAUTH_REGISTER_RESPONSE_FILE=$(mktemp)
OAUTH_REGISTER_STATUS=$(curl -sS -o "$OAUTH_REGISTER_RESPONSE_FILE" -w "%{http_code}" \
  -X POST \
  -H "Content-Type: application/json" \
  --data "$OAUTH_REGISTER_BODY" \
  "${REGISTRY_URL}/api/v0/servers")
OAUTH_REGISTER_BODY_CONTENT="$(cat "$OAUTH_REGISTER_RESPONSE_FILE")"
rm -f "$OAUTH_REGISTER_RESPONSE_FILE"

if [[ "$OAUTH_REGISTER_STATUS" != "202" ]]; then
  echo "ERROR: expected OAuth registration to return 202, got ${OAUTH_REGISTER_STATUS}" >&2
  echo "Body: ${OAUTH_REGISTER_BODY_CONTENT}" >&2
  exit 1
fi

OAUTH_SESSION_ID=$(extract_json_string_field "session_id" "$OAUTH_REGISTER_BODY_CONTENT")
OAUTH_AUTH_URL=$(extract_json_string_field "authorization_url" "$OAUTH_REGISTER_BODY_CONTENT")
OAUTH_AUTH_URL=$(decode_json_string "$OAUTH_AUTH_URL")
if [[ -z "$OAUTH_SESSION_ID" || -z "$OAUTH_AUTH_URL" ]]; then
  echo "ERROR: OAuth registration response did not include session_id and authorization_url" >&2
  echo "Body: ${OAUTH_REGISTER_BODY_CONTENT}" >&2
  exit 1
fi

OAUTH_AUTH_HEADERS=$(mktemp)
curl -sS -D "$OAUTH_AUTH_HEADERS" -o /dev/null "$OAUTH_AUTH_URL"
OAUTH_CALLBACK_URL=$(sed -n 's/^[Ll]ocation: \(.*\)\r$/\1/p' "$OAUTH_AUTH_HEADERS" | tail -n 1)
rm -f "$OAUTH_AUTH_HEADERS"
if [[ -z "$OAUTH_CALLBACK_URL" ]]; then
  echo "ERROR: OAuth authorize endpoint did not return a callback redirect" >&2
  exit 1
fi

OAUTH_CODE=$(printf "%s" "$OAUTH_CALLBACK_URL" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')
OAUTH_STATE=$(printf "%s" "$OAUTH_CALLBACK_URL" | sed -n 's/.*[?&]state=\([^&]*\).*/\1/p')
if [[ -z "$OAUTH_CODE" || -z "$OAUTH_STATE" ]]; then
  echo "ERROR: OAuth callback URL did not include both code and state" >&2
  echo "Callback URL: ${OAUTH_CALLBACK_URL}" >&2
  exit 1
fi

OAUTH_COMPLETE_BODY=$(cat <<EOF
{"code":"${OAUTH_CODE}","state":"${OAUTH_STATE}"}
EOF
)

OAUTH_COMPLETE_RESPONSE_FILE=$(mktemp)
OAUTH_COMPLETE_STATUS=$(curl -sS -o "$OAUTH_COMPLETE_RESPONSE_FILE" -w "%{http_code}" \
  -X POST \
  -H "Content-Type: application/json" \
  --data "$OAUTH_COMPLETE_BODY" \
  "${REGISTRY_URL}/api/v0/upstream_oauth/sessions/${OAUTH_SESSION_ID}/complete")
OAUTH_COMPLETE_BODY_CONTENT="$(cat "$OAUTH_COMPLETE_RESPONSE_FILE")"
rm -f "$OAUTH_COMPLETE_RESPONSE_FILE"

if [[ "$OAUTH_COMPLETE_STATUS" != "201" ]]; then
  echo "ERROR: expected OAuth completion to return 201, got ${OAUTH_COMPLETE_STATUS}" >&2
  echo "Body: ${OAUTH_COMPLETE_BODY_CONTENT}" >&2
  exit 1
fi

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

# 10) Print Homebrew formula config snippet
log "Homebrew formula config (from .goreleaser.yaml)"
sed -n '/^brews:/,/^dockers:/p' "$ROOT_DIR/.goreleaser.yaml" || true

# 11) Run manual API error response verification against an isolated server
log "Running manual API error response verification"
BIN_PATH="$BIN_PATH" "$ROOT_DIR/scripts/test-api-error-responses.sh"
popd >/dev/null

log "All tests passed 🎉"

log "Cleaning up"
unset MCP_SERVER_INIT_REQ_TIMEOUT_SEC

log "All done!"
