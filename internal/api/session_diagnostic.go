package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// sessionDiagnosticEvent returns a bounded, allowlisted summary. It never reads
// request bodies, raw path/query, session IDs, credentials, or identity headers.
func sessionDiagnosticEvent(req *http.Request, status int) (string, bool) {
	if status != http.StatusNotFound {
		return "", false
	}
	verb := "other"
	switch req.Method {
	case http.MethodPost, http.MethodGet, http.MethodDelete:
		verb = req.Method
	}
	protocol := "other"
	switch req.Header.Get("MCP-Protocol-Version") {
	case "2026-07-28":
		protocol = "modern"
	case "2025-03-26", "2025-06-18", "2025-11-25":
		protocol = "legacy"
	case "":
		protocol = "missing"
	}
	method := "other"
	switch req.Header.Get("Mcp-Method") {
	case "tools/call", "tools/list", "initialize", "server/discover":
		method = req.Header.Get("Mcp-Method")
	}
	present := strings.TrimSpace(req.Header.Get("Mcp-Session-Id")) != ""
	return fmt.Sprintf("verb=%s protocol=%s method=%s session_present=%t status=%d",
		verb, protocol, method, present, status), true
}

// The diagnostic is opt-in; normal HTTP responses and session state are never modified.
// The injected logger permits verification without replacing the process-global log sink.
func emitSessionDiagnostic(logger *log.Logger, req *http.Request, status int) {
	if os.Getenv("MCP_GROUP_SESSION_DIAGNOSTICS") != "1" {
		return
	}
	if event, ok := sessionDiagnosticEvent(req, status); ok {
		logger.Printf("mcp_group_session_diagnostic %s", event)
	}
}
