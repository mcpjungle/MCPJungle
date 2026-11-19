package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// outcomeCapture is a telemetry.CustomMetrics that records the last tool-call
// outcome, so a test can assert what was recorded.
type outcomeCapture struct {
	tool telemetry.ToolCallOutcome
}

func (o *outcomeCapture) RecordToolCall(_ context.Context, _, _ string, outcome telemetry.ToolCallOutcome, _ time.Duration) {
	o.tool = outcome
}

func (o *outcomeCapture) RecordPromptCall(_ context.Context, _, _ string, _ telemetry.PromptCallOutcome, _ time.Duration) {
}

// A tool that completes without a transport error but returns a tool-level error
// (CallToolResult.IsError) must be recorded as an "error" outcome, not "success".
func TestMCPProxyToolCallHandler_RecordsToolLevelErrorAsErrorOutcome(t *testing.T) {
	db := setupTestDBForProxyAdditional(t)

	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0", mcpserver.WithToolCapabilities(true))
	upstream.AddTool(
		mcp.NewTool("boom"),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultError("kaboom"), nil
		},
	)

	httpServer := newUpstreamStreamableHTTPServer(t, upstream)
	defer httpServer.Close()

	srv := createStreamableHTTPTestServer(t, "tool-server", httpServer.URL)
	require.NoError(t, db.Create(srv).Error)

	metrics := &outcomeCapture{}
	service := &MCPService{
		db:                         db,
		metrics:                    metrics,
		mcpServerInitReqTimeoutSec: 5,
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = "tool-server__boom"

	res, err := service.MCPProxyToolCallHandler(context.WithValue(context.Background(), "mode", model.ModeDev), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.IsError)
	assert.Equal(t, telemetry.ToolCallOutcomeError, metrics.tool)
}

// The REST InvokeTool path has the same contract: a completed call whose result
// is a tool-level error (CallToolResult.IsError) must record an "error" outcome.
func TestInvokeTool_RecordsToolLevelErrorAsErrorOutcome(t *testing.T) {
	db := setupTestDBForProxyAdditional(t)

	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0", mcpserver.WithToolCapabilities(true))
	upstream.AddTool(
		mcp.NewTool("boom"),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultError("kaboom"), nil
		},
	)

	httpServer := newUpstreamStreamableHTTPServer(t, upstream)
	defer httpServer.Close()

	srv := createStreamableHTTPTestServer(t, "tool-server", httpServer.URL)
	require.NoError(t, db.Create(srv).Error)

	metrics := &outcomeCapture{}
	service := &MCPService{
		db:                         db,
		metrics:                    metrics,
		mcpServerInitReqTimeoutSec: 5,
	}

	result, err := service.InvokeTool(context.Background(), "tool-server__boom", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Equal(t, telemetry.ToolCallOutcomeError, metrics.tool)
}
