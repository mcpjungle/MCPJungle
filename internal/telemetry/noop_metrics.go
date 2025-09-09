package telemetry

import (
	"context"
	"time"
)

// NoopCustomMetrics is a no-op implementation of CustomMetrics that does nothing and adds minimal overhead.
// This should be used when metrics are disabled or not needed.
type NoopCustomMetrics struct{}

// NewNoopCustomMetrics returns a no-op implementation of CustomMetrics.
func NewNoopCustomMetrics() CustomMetrics {
	return &NoopCustomMetrics{}
}

func (m *NoopCustomMetrics) RecordToolCall(
	ctx context.Context, serverName, toolName string, outcome ToolCallOutcome, elapsedTime time.Duration,
) {
	// No-op
}
