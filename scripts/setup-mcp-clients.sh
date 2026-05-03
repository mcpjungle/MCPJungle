#!/usr/bin/env bash
# setup-mcp-clients.sh
#
# Wire MCPJungle into AI IDEs and agents on the local machine.
#
# Usage:
#   ./scripts/setup-mcp-clients.sh [OPTIONS]
#
# Options:
#   --claude      Inject into ~/.claude/mcp.json          (Claude Desktop / Claude Code)
#   --cursor      Inject into ~/.cursor/mcp.json          (Cursor)
#   --codex       Inject into ~/.codex/config.toml        (OpenAI Codex CLI)
#   --copilot     Inject into VS Code Copilot mcp.json    (GitHub Copilot / VS Code)
#   --opencode    Inject into ~/.config/opencode/opencode.json (OpenCode)
#   --zed         Inject into ~/.config/zed/settings.json (Zed)
#   --all         All of the above
#   --host URL    MCPJungle base URL  (default: http://localhost:8080)
#   --token TOK   Bearer token for enterprise mode        (omit for dev mode)
#   --name STR    Entry key written into configs          (default: mcpjungle)
#   --dry-run     Print what would change, no writes
#   --remove      Remove the mcpjungle entry instead of adding it
#   --shell SH    Shell profile to source token env var   (bash|zsh|fish|none, default: auto-detect)
#
# The script is idempotent — running it twice does not duplicate entries.
# Existing config files are backed up with a timestamp before modification.
#
# fish users: this is a bash script; run it as:
#   bash ./scripts/setup-mcp-clients.sh --all
#
# Examples:
#   # Dev mode, all clients
#   ./scripts/setup-mcp-clients.sh --all
#
#   # Enterprise mode with token, only Cursor + Claude
#   ./scripts/setup-mcp-clients.sh --cursor --claude --token "MY_TOKEN"
#
#   # Point at a remote gateway
#   ./scripts/setup-mcp-clients.sh --all --host https://mcp.example.com --token "MY_TOKEN"
#
#   # Remove from all
#   ./scripts/setup-mcp-clients.sh --all --remove

set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
HOST="http://localhost:8080"
TOKEN=""
ENTRY_NAME="mcpjungle"
DRY_RUN=0
REMOVE=0
SHELL_PROFILE="auto"

DO_CLAUDE=0
DO_CURSOR=0
DO_CODEX=0
DO_COPILOT=0
DO_OPENCODE=0
DO_ZED=0

# ---------------------------------------------------------------------------
# Config paths
# ---------------------------------------------------------------------------
CLAUDE_MCP="${HOME}/.claude/mcp.json"
CURSOR_MCP="${HOME}/.cursor/mcp.json"
CODEX_TOML="${HOME}/.codex/config.toml"
OPENCODE_JSON="${HOME}/.config/opencode/opencode.json"
ZED_SETTINGS="${HOME}/.config/zed/settings.json"

# VS Code Copilot — platform-specific default location
if [[ "$(uname)" == "Darwin" ]]; then
    COPILOT_MCP="${HOME}/Library/Application Support/Code/User/mcp.json"
else
    COPILOT_MCP="${HOME}/.config/Code/User/mcp.json"
fi

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
if [[ $# -eq 0 ]]; then
    echo "Usage: $0 [--claude] [--cursor] [--codex] [--copilot] [--opencode] [--zed] [--all]"
    echo "          [--host URL] [--token TOKEN] [--name NAME] [--dry-run] [--remove]"
    exit 1
fi

while [[ $# -gt 0 ]]; do
    case "$1" in
        --claude)   DO_CLAUDE=1 ;;
        --cursor)   DO_CURSOR=1 ;;
        --codex)    DO_CODEX=1 ;;
        --copilot)  DO_COPILOT=1 ;;
        --opencode) DO_OPENCODE=1 ;;
        --zed)      DO_ZED=1 ;;
        --all)      DO_CLAUDE=1; DO_CURSOR=1; DO_CODEX=1; DO_COPILOT=1; DO_OPENCODE=1; DO_ZED=1 ;;
        --host)     HOST="${2:?--host requires a value}"; shift ;;
        --token)    TOKEN="${2:?--token requires a value}"; shift ;;
        --name)     ENTRY_NAME="${2:?--name requires a value}"; shift ;;
        --shell)    SHELL_PROFILE="${2:?--shell requires bash|zsh|fish|none}"; shift ;;
        --dry-run)  DRY_RUN=1 ;;
        --remove)   REMOVE=1 ;;
        *)
            echo "Unknown argument: $1"
            echo "Run $0 with no arguments for usage."
            exit 1
            ;;
    esac
    shift
done

# Derived
MCP_URL="${HOST}/mcp"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log()  { echo "  $*"; }
ok()   { echo "  ✓ $*"; }
skip() { echo "  - $*"; }
warn() { echo "  ! $*"; }
die()  { echo "ERROR: $*" >&2; exit 1; }

require_python() {
    command -v python3 &>/dev/null || die "python3 is required but not found on PATH"
}

backup_file() {
    local path="$1"
    if [[ -f "$path" && $DRY_RUN -eq 0 ]]; then
        cp "$path" "${path}.bak-$(date +%Y%m%d_%H%M%S)"
        log "backed up → ${path}.bak-*"
    fi
}

write_file() {
    local path="$1" content="$2"
    if [[ $DRY_RUN -eq 1 ]]; then
        echo "    [dry-run] would write: ${path}"
    else
        mkdir -p "$(dirname "$path")"
        printf '%s\n' "$content" > "$path"
    fi
}

# ---------------------------------------------------------------------------
# JSON helpers (pure python3, no jq dependency)
# ---------------------------------------------------------------------------

# Upsert mcpServers.<name> with a URL entry (Cursor, Copilot style)
json_upsert_url_entry() {
    python3 - "$1" "$2" "$3" "$4" <<'PYEOF'
import json, sys

file_path   = sys.argv[1]
entry_name  = sys.argv[2]
mcp_url     = sys.argv[3]
token       = sys.argv[4]  # empty string = no auth

try:
    with open(file_path) as f:
        data = json.load(f)
except FileNotFoundError:
    data = {}

entry = {"url": mcp_url}
if token:
    entry["headers"] = {"Authorization": f"Bearer {token}"}

data.setdefault("mcpServers", {})[entry_name] = entry

with open(file_path, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")

print(json.dumps(entry, indent=2))
PYEOF
}

# Upsert mcpServers.<name> with npx mcp-remote (Claude Desktop style)
json_upsert_npx_entry() {
    python3 - "$1" "$2" "$3" "$4" <<'PYEOF'
import json, sys

file_path   = sys.argv[1]
entry_name  = sys.argv[2]
mcp_url     = sys.argv[3]
token       = sys.argv[4]

try:
    with open(file_path) as f:
        data = json.load(f)
except FileNotFoundError:
    data = {}

args = ["mcp-remote", mcp_url]
# mcp-remote accepts --allow-http for non-TLS local URLs
if mcp_url.startswith("http://"):
    args.append("--allow-http")
if token:
    args += ["--header", f"Authorization: Bearer {token}"]

entry = {"command": "npx", "args": args}
data.setdefault("mcpServers", {})[entry_name] = entry

with open(file_path, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")

print(json.dumps(entry, indent=2))
PYEOF
}

# Upsert servers.<name> (GitHub Copilot mcp.json uses "servers" not "mcpServers")
json_upsert_copilot_entry() {
    python3 - "$1" "$2" "$3" "$4" <<'PYEOF'
import json, sys

file_path   = sys.argv[1]
entry_name  = sys.argv[2]
mcp_url     = sys.argv[3]
token       = sys.argv[4]

try:
    with open(file_path) as f:
        data = json.load(f)
except FileNotFoundError:
    data = {}

entry = {"url": mcp_url}
if token:
    entry["headers"] = {"Authorization": f"Bearer {token}"}

data.setdefault("servers", {})[entry_name] = entry

with open(file_path, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")

print(json.dumps(entry, indent=2))
PYEOF
}

# Remove mcpServers.<name>
json_remove_mcp_entry() {
    python3 - "$1" "$2" <<'PYEOF'
import json, sys

file_path  = sys.argv[1]
entry_name = sys.argv[2]

try:
    with open(file_path) as f:
        data = json.load(f)
except FileNotFoundError:
    sys.exit(0)

removed = data.get("mcpServers", {}).pop(entry_name, None)
if removed:
    with open(file_path, "w") as f:
        json.dump(data, f, indent=2)
        f.write("\n")
    print(f"removed '{entry_name}' from mcpServers")
else:
    print(f"'{entry_name}' was not present in mcpServers")
PYEOF
}

# Remove servers.<name> (Copilot)
json_remove_copilot_entry() {
    python3 - "$1" "$2" <<'PYEOF'
import json, sys

file_path  = sys.argv[1]
entry_name = sys.argv[2]

try:
    with open(file_path) as f:
        data = json.load(f)
except FileNotFoundError:
    sys.exit(0)

removed = data.get("servers", {}).pop(entry_name, None)
if removed:
    with open(file_path, "w") as f:
        json.dump(data, f, indent=2)
        f.write("\n")
    print(f"removed '{entry_name}' from servers")
else:
    print(f"'{entry_name}' was not present in servers")
PYEOF
}

# Upsert Zed context_servers entry
json_upsert_zed_entry() {
    python3 - "$1" "$2" "$3" "$4" <<'PYEOF'
import json, sys

file_path  = sys.argv[1]
entry_name = sys.argv[2]
mcp_url    = sys.argv[3]
token      = sys.argv[4]

try:
    with open(file_path) as f:
        data = json.load(f)
except FileNotFoundError:
    data = {}

entry = {
    "command": {"path": "npx", "args": ["mcp-remote", mcp_url]
                + (["--allow-http"] if mcp_url.startswith("http://") else [])
                + (["--header", f"Authorization: Bearer {token}"] if token else [])},
    "settings": {}
}

data.setdefault("context_servers", {})[entry_name] = entry

with open(file_path, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")

print(json.dumps(entry, indent=2))
PYEOF
}

# Remove Zed context_servers.<name>
json_remove_zed_entry() {
    python3 - "$1" "$2" <<'PYEOF'
import json, sys

file_path  = sys.argv[1]
entry_name = sys.argv[2]

try:
    with open(file_path) as f:
        data = json.load(f)
except FileNotFoundError:
    sys.exit(0)

removed = data.get("context_servers", {}).pop(entry_name, None)
if removed:
    with open(file_path, "w") as f:
        json.dump(data, f, indent=2)
        f.write("\n")
    print(f"removed '{entry_name}' from context_servers")
else:
    print(f"'{entry_name}' was not present in context_servers")
PYEOF
}

# Upsert OpenCode mcp entry
json_upsert_opencode_entry() {
    python3 - "$1" "$2" "$3" "$4" <<'PYEOF'
import json, sys

file_path  = sys.argv[1]
entry_name = sys.argv[2]
mcp_url    = sys.argv[3]
token      = sys.argv[4]

try:
    with open(file_path) as f:
        data = json.load(f)
except FileNotFoundError:
    data = {}

entry = {"type": "remote", "url": mcp_url, "enabled": True}
if token:
    entry["headers"] = {"Authorization": f"Bearer {token}"}

data.setdefault("mcp", {})[entry_name] = entry

with open(file_path, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")

print(json.dumps(entry, indent=2))
PYEOF
}

json_remove_opencode_entry() {
    python3 - "$1" "$2" <<'PYEOF'
import json, sys

file_path  = sys.argv[1]
entry_name = sys.argv[2]

try:
    with open(file_path) as f:
        data = json.load(f)
except FileNotFoundError:
    sys.exit(0)

removed = data.get("mcp", {}).pop(entry_name, None)
if removed:
    with open(file_path, "w") as f:
        json.dump(data, f, indent=2)
        f.write("\n")
    print(f"removed '{entry_name}' from mcp")
else:
    print(f"'{entry_name}' was not present in mcp")
PYEOF
}

# ---------------------------------------------------------------------------
# Codex TOML helpers
# ---------------------------------------------------------------------------
toml_has_entry() {
    grep -q "^\[mcp_servers\.${ENTRY_NAME}\]" "$1" 2>/dev/null
}

toml_add_entry() {
    local file="$1"
    {
        echo ""
        echo "[mcp_servers.${ENTRY_NAME}]"
        echo "command = \"npx\""
        if [[ -n "$TOKEN" ]]; then
            echo "args = [\"mcp-remote\", \"${MCP_URL}\", \"--header\", \"Authorization: Bearer ${TOKEN}\"]"
        elif [[ "$MCP_URL" == http://* ]]; then
            echo "args = [\"mcp-remote\", \"${MCP_URL}\", \"--allow-http\"]"
        else
            echo "args = [\"mcp-remote\", \"${MCP_URL}\"]"
        fi
    } >> "$file"
}

toml_remove_entry() {
    python3 - "$1" "$2" <<'PYEOF'
import sys, re

path       = sys.argv[1]
entry_name = sys.argv[2]

try:
    text = open(path).read()
except FileNotFoundError:
    sys.exit(0)

pattern = rf'\n\[mcp_servers\.{re.escape(entry_name)}\][^\[]*'
cleaned, n = re.subn(pattern, '', text, flags=re.DOTALL)
open(path, "w").write(cleaned)
print(f"removed {n} block(s) for '{entry_name}'")
PYEOF
}

# ---------------------------------------------------------------------------
# Per-client install/remove
# ---------------------------------------------------------------------------
install_claude() {
    local file="$CLAUDE_MCP"
    echo "[claude] ${file}"
    if [[ $REMOVE -eq 1 ]]; then
        [[ ! -f "$file" ]] && { skip "file not found, nothing to remove"; return; }
        backup_file "$file"
        [[ $DRY_RUN -eq 0 ]] && ok "$(json_remove_mcp_entry "$file" "$ENTRY_NAME")" \
            || echo "    [dry-run] would remove '${ENTRY_NAME}'"
        return
    fi
    backup_file "$file"
    if [[ $DRY_RUN -eq 0 ]]; then
        result=$(json_upsert_npx_entry "$file" "$ENTRY_NAME" "$MCP_URL" "$TOKEN")
        ok "wrote entry:"; echo "$result" | sed 's/^/      /'
    else
        echo "    [dry-run] would upsert '${ENTRY_NAME}' (npx mcp-remote) in ${file}"
    fi
    echo ""
}

install_cursor() {
    local file="$CURSOR_MCP"
    echo "[cursor] ${file}"
    if [[ $REMOVE -eq 1 ]]; then
        [[ ! -f "$file" ]] && { skip "file not found, nothing to remove"; return; }
        backup_file "$file"
        [[ $DRY_RUN -eq 0 ]] && ok "$(json_remove_mcp_entry "$file" "$ENTRY_NAME")" \
            || echo "    [dry-run] would remove '${ENTRY_NAME}'"
        return
    fi
    backup_file "$file"
    if [[ $DRY_RUN -eq 0 ]]; then
        result=$(json_upsert_url_entry "$file" "$ENTRY_NAME" "$MCP_URL" "$TOKEN")
        ok "wrote entry:"; echo "$result" | sed 's/^/      /'
    else
        echo "    [dry-run] would upsert '${ENTRY_NAME}' (url) in ${file}"
    fi
    echo ""
}

install_codex() {
    local file="$CODEX_TOML"
    echo "[codex] ${file}"
    if [[ $REMOVE -eq 1 ]]; then
        [[ ! -f "$file" ]] && { skip "file not found, nothing to remove"; return; }
        backup_file "$file"
        [[ $DRY_RUN -eq 0 ]] && ok "$(toml_remove_entry "$file" "$ENTRY_NAME")" \
            || echo "    [dry-run] would remove [mcp_servers.${ENTRY_NAME}]"
        echo ""; return
    fi
    if toml_has_entry "$file"; then
        skip "entry already present — use --remove first to replace"
        echo ""; return
    fi
    backup_file "$file"
    if [[ $DRY_RUN -eq 0 ]]; then
        toml_add_entry "$file"
        ok "appended [mcp_servers.${ENTRY_NAME}] block"
    else
        echo "    [dry-run] would append [mcp_servers.${ENTRY_NAME}] to ${file}"
    fi
    echo ""
}

install_copilot() {
    local file="$COPILOT_MCP"
    echo "[copilot] ${file}"
    if [[ $REMOVE -eq 1 ]]; then
        [[ ! -f "$file" ]] && { skip "file not found, nothing to remove"; return; }
        backup_file "$file"
        [[ $DRY_RUN -eq 0 ]] && ok "$(json_remove_copilot_entry "$file" "$ENTRY_NAME")" \
            || echo "    [dry-run] would remove '${ENTRY_NAME}'"
        echo ""; return
    fi
    backup_file "$file"
    if [[ $DRY_RUN -eq 0 ]]; then
        result=$(json_upsert_copilot_entry "$file" "$ENTRY_NAME" "$MCP_URL" "$TOKEN")
        ok "wrote entry:"; echo "$result" | sed 's/^/      /'
    else
        echo "    [dry-run] would upsert '${ENTRY_NAME}' in ${file}"
    fi
    echo ""
}

install_opencode() {
    local file="$OPENCODE_JSON"
    echo "[opencode] ${file}"
    if [[ $REMOVE -eq 1 ]]; then
        [[ ! -f "$file" ]] && { skip "file not found, nothing to remove"; return; }
        backup_file "$file"
        [[ $DRY_RUN -eq 0 ]] && ok "$(json_remove_opencode_entry "$file" "$ENTRY_NAME")" \
            || echo "    [dry-run] would remove '${ENTRY_NAME}'"
        echo ""; return
    fi
    backup_file "$file"
    if [[ $DRY_RUN -eq 0 ]]; then
        result=$(json_upsert_opencode_entry "$file" "$ENTRY_NAME" "$MCP_URL" "$TOKEN")
        ok "wrote entry:"; echo "$result" | sed 's/^/      /'
    else
        echo "    [dry-run] would upsert '${ENTRY_NAME}' in ${file}"
    fi
    echo ""
}

install_zed() {
    local file="$ZED_SETTINGS"
    echo "[zed] ${file}"
    if [[ $REMOVE -eq 1 ]]; then
        [[ ! -f "$file" ]] && { skip "file not found, nothing to remove"; return; }
        backup_file "$file"
        [[ $DRY_RUN -eq 0 ]] && ok "$(json_remove_zed_entry "$file" "$ENTRY_NAME")" \
            || echo "    [dry-run] would remove '${ENTRY_NAME}'"
        echo ""; return
    fi
    backup_file "$file"
    if [[ $DRY_RUN -eq 0 ]]; then
        result=$(json_upsert_zed_entry "$file" "$ENTRY_NAME" "$MCP_URL" "$TOKEN")
        ok "wrote entry:"; echo "$result" | sed 's/^/      /'
    else
        echo "    [dry-run] would upsert '${ENTRY_NAME}' in ${file}"
    fi
    echo ""
}

# ---------------------------------------------------------------------------
# Shell profile helper — emit a MCPJUNGLE_TOKEN export snippet
# ---------------------------------------------------------------------------
print_shell_hint() {
    [[ -z "$TOKEN" ]] && return

    local detected_shell="$SHELL_PROFILE"
    if [[ "$detected_shell" == "auto" ]]; then
        local shell_bin
        shell_bin="$(basename "${SHELL:-bash}")"
        detected_shell="$shell_bin"
    fi

    echo "-----------------------------------------------------------------------"
    echo "Token hint — add this to your shell profile to avoid passing --token:"
    echo ""

    case "$detected_shell" in
        fish)
            echo "  # ~/.config/fish/config.fish"
            echo "  set -gx MCPJUNGLE_TOKEN \"${TOKEN}\""
            ;;
        zsh)
            echo "  # ~/.zshrc"
            echo "  export MCPJUNGLE_TOKEN=\"${TOKEN}\""
            ;;
        bash)
            echo "  # ~/.bashrc or ~/.bash_profile"
            echo "  export MCPJUNGLE_TOKEN=\"${TOKEN}\""
            ;;
        *)
            echo "  export MCPJUNGLE_TOKEN=\"${TOKEN}\"   # add to your shell profile"
            ;;
    esac
    echo ""
    echo "  Then pass via: --token \"\$MCPJUNGLE_TOKEN\""
    echo "-----------------------------------------------------------------------"
    echo ""
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
require_python

echo ""
echo "MCPJungle client setup"
echo "----------------------"
echo "  gateway : ${HOST}"
echo "  mcp url : ${MCP_URL}"
echo "  entry   : ${ENTRY_NAME}"
echo "  auth    : ${TOKEN:+enterprise (token set)}"
echo "  auth    : ${TOKEN:-development (no token)}"
echo "  mode    : $( [[ $REMOVE -eq 1 ]] && echo remove || echo install )$( [[ $DRY_RUN -eq 1 ]] && echo " (dry-run)" )"
echo ""

[[ $DO_CLAUDE   -eq 1 ]] && install_claude
[[ $DO_CURSOR   -eq 1 ]] && install_cursor
[[ $DO_CODEX    -eq 1 ]] && install_codex
[[ $DO_COPILOT  -eq 1 ]] && install_copilot
[[ $DO_OPENCODE -eq 1 ]] && install_opencode
[[ $DO_ZED      -eq 1 ]] && install_zed

print_shell_hint

echo "Done."
echo ""
if [[ $REMOVE -eq 0 && $DRY_RUN -eq 0 ]]; then
    echo "Restart / reload each client to pick up the MCPJungle MCP server."
    echo ""
fi
