package mcp

import (
	"context"
	"errors"
	"io"
	"strings"
	"syscall"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// connectionErrorPatterns contains common error substrings that indicate a connection problem.
// Used by isConnectionError to determine if a stateful session should be invalidated.
var connectionErrorPatterns = []string{
	"connection refused",
	"connection reset",
	"connection closed",
	"broken pipe",
	"eof",
	"no such host",
	"network is unreachable",
	"timeout",
	"context canceled",
	"context deadline exceeded",
	"transport",
	"dial",
	"i/o timeout",
	"use of closed network connection",
}

// sessionResult holds the result of getting an MCP session.
// It includes the client and whether the caller should close it after use.
type sessionResult struct {
	client      *client.Client
	shouldClose bool // true for stateless sessions, false for stateful sessions

	// For stateful sessions, these are used for reactive invalidation on errors
	serverName     string
	sessionManager *SessionManager
}

// closeIfApplicable closes the session if it should be closed (stateless mode).
func (sr *sessionResult) closeIfApplicable() {
	if sr.shouldClose && sr.client != nil {
		sr.client.Close()
	}
}

// invalidateOnError checks if the error indicates a connection problem and
// invalidates the stateful session so the next call will create a fresh one.
// This should be called when a tool/prompt call fails with an error.
func (sr *sessionResult) invalidateOnError(err error) {
	if err == nil || sr.shouldClose || sr.sessionManager == nil {
		return // Nothing to invalidate for stateless sessions or no error
	}

	// Check if this looks like a connection error
	if isConnectionError(err) {
		sr.sessionManager.InvalidateSession(sr.serverName, err.Error())
	}
}

// getSession returns a session for the given MCP server.
// For stateful servers, it returns a persistent session from the SessionManager.
// For stateless servers, it creates a new session that should be closed after use.
func (m *MCPService) getSession(ctx context.Context, server *model.McpServer) (*sessionResult, error) {
	if server.SessionMode == types.SessionModeStateful {
		// Use the session manager for stateful sessions
		mcpClient, err := m.sessionManager.GetOrCreateSession(ctx, server)
		if err != nil {
			return nil, err
		}
		return &sessionResult{
			client:         mcpClient,
			shouldClose:    false, // Don't close stateful sessions after each call
			serverName:     server.Name,
			sessionManager: m.sessionManager,
		}, nil
	}

	// Default: stateless mode - create a new session for each call
	mcpClient, err := newMcpServerSession(ctx, server, m.mcpServerInitReqTimeoutSec)
	if err != nil {
		return nil, err
	}
	return &sessionResult{
		client:      mcpClient,
		shouldClose: true, // Close stateless sessions after each call
	}, nil
}

// isConnectionError checks if an error indicates a connection problem
// that would warrant invalidating a stateful session.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	for _, pattern := range connectionErrorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Check for specific error types
	if errors.Is(err, io.EOF) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) {
		return true
	}

	return false
}
