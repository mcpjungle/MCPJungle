package api

import (
    "bytes"
    "log"
    "net/http"
    "strings"
    "testing"
)

func TestDiagnostic404NeverLogsRequestData(t *testing.T) {
    request, _ := http.NewRequest(http.MethodPost, "/v0/groups/demo/mcp?private=query",
        strings.NewReader(`{"params":{"arguments":{"private":"body"}}}`))
    request.Header.Set("Mcp-Session-Id", "secret-session-id")
    request.Header.Set("Authorization", "Bearer private-auth")
    request.Header.Set("MCP-Protocol-Version", "2026-07-28")
    request.Header.Set("Mcp-Method", "tools/call")

    event, emit := sessionDiagnosticEvent(request, http.StatusNotFound)
    if !emit {
        t.Fatal("missing diagnostic on 404")
    }
    for _, expected := range []string{"verb=POST", "protocol=modern", "session_present=true", "method=tools/call", "status=404"} {
        if !strings.Contains(event, expected) {
            t.Fatalf("missing safe classification %q in %q", expected, event)
        }
    }
    for _, private := range []string{"secret-session-id", "private-auth", "private=query", "private", "body"} {
        if strings.Contains(event, private) {
            t.Fatalf("diagnostic leaked request data")
        }
    }
}

func TestDiagnosticOptInAndUnknownHeaders(t *testing.T) {
    req, _ := http.NewRequest(http.MethodPost, "/v0/groups/demo/mcp", nil)
    req.Header.Set("MCP-Protocol-Version", "unsafe-header")
    req.Header.Set("Mcp-Method", "unsafe-method")
    req.Header.Set("Mcp-Session-Id", "unlogged-session")
    var out bytes.Buffer
    logger := log.New(&out, "", 0)

    t.Setenv("MCP_GROUP_SESSION_DIAGNOSTICS", "")
    emitSessionDiagnostic(logger, req, http.StatusNotFound)
    if out.Len() != 0 { t.Fatal("diagnostic did not default off") }

    t.Setenv("MCP_GROUP_SESSION_DIAGNOSTICS", "1")
    emitSessionDiagnostic(logger, req, http.StatusOK)
    if out.Len() != 0 { t.Fatal("success response emitted a diagnostic") }
    emitSessionDiagnostic(logger, req, http.StatusNotFound)
    record := out.String()
    if !strings.Contains(record, "protocol=other") || !strings.Contains(record, "method=other") || strings.Contains(record, "unsafe") || strings.Contains(record, "unlogged-session") {
        t.Fatalf("invalid categorization/redaction: %q", record)
    }
}
