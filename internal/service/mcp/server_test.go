package mcp

import (
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestGetMcpServerNotFound(t *testing.T) {
	setup := testhelpers.SetupMCPTest(t)
	defer setup.Cleanup()

	proxyServer := server.NewMCPServer("test-proxy", "0.1.0")
	svc, err := NewMCPService(&ServiceConfig{
		DB:                      setup.DB,
		McpProxyServer:          proxyServer,
		SseMcpProxyServer:       proxyServer,
		Metrics:                 telemetry.NewNoopCustomMetrics(),
		McpServerInitReqTimeout: 10,
	})
	if err != nil {
		t.Fatalf("failed to create mcp service: %v", err)
	}

	_, err = svc.GetMcpServer("missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrMCPServerNotFound) {
		t.Fatalf("expected ErrMCPServerNotFound, got: %v", err)
	}
}
