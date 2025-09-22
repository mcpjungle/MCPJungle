package telemetry

import (
	"context"
	"time"
)

// ToolCallOutcome represents the outcome of a tool call, either success or error.
type ToolCallOutcome string

const (
	// ToolCallOutcomeSuccess indicates a successful tool call
	ToolCallOutcomeSuccess ToolCallOutcome = "success"
	// ToolCallOutcomeError indicates a failed tool call
	ToolCallOutcomeError ToolCallOutcome = "error"
)

// CustomMetrics defines the interface for recording custom metrics from mcpjungle.
// It provides convenience methods for recording metrics related to http server, mcp servers, tools, usage, etc.
type CustomMetrics interface {
	// RecordToolCall records a tool invocation, its latency, and its outcome (success or error).
	RecordToolCall(ctx context.Context, serverName, toolName string, outcome ToolCallOutcome, elapsedTime time.Duration)
}
