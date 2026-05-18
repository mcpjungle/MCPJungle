#!/usr/bin/env bash

# Runs focused Go tests for the streamable_http tools/list_changed sync feature.
# This keeps the shell-level entrypoint lightweight while still exercising the
# watcher sync paths, proxy replacement behavior, disabled-server DB-only sync,
# and upstream deletion semantics.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

log() { printf "\n[TOOL-LIST-SYNC] %s\n" "$*"; }

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: Required command '$1' not found in PATH" >&2
    exit 1
  fi
}

log "Checking required commands"
require_cmd go

cd "$ROOT_DIR"

log "Running focused MCP watcher sync tests"
go test ./internal/service/mcp -run 'TestStreamableHTTPWatcherClientCanResyncTools|TestStreamableHTTPWatcherSyncsDisabledServerToDBOnly|TestSyncServerToolsPreservesDisabledToolUntilUpstreamDeletion|TestSyncServerToolsReplacesExistingProxyToolInPlace'

log "Running focused tool group sync behavior tests"
go test ./internal/service/toolgroup -run 'TestNewToolGroupService_DegradedPersistedGroupDoesNotFailStartup|TestToolGroupProxyServers_AdvertiseToolListChanged'

log "Tool list sync tests passed"
