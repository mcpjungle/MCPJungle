package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	labelMCPServerName   = "mcp_server_name"
	labelToolName        = "tool_name"
	labelToolCallOutcome = "outcome"
)

const (
	// attrValueMaxLen is the maximum length for string attribute values.
	attrValueMaxLen  = 64
	attrValueUnknown = "unknown"
)

// OtelCustomMetrics bundles all the OpenTelemetry metric instruments used for MCPJungle.
// It implements the CustomMetrics interface.
type OtelCustomMetrics struct {
	toolCalls       metric.Int64Counter
	toolCallLatency metric.Float64Histogram
}

// NewOtelCustomMetrics initializes all metric instruments required by MCPJungle.
// Returns an CustomMetrics instance ready for use, or an error if any instrument
// could not be created.
func NewOtelCustomMetrics(meter metric.Meter) (CustomMetrics, error) {
	if meter == nil {
		return nil, fmt.Errorf("meter cannot be nil")
	}

	toolInv, err := meter.Int64Counter(
		"mcpjungle_tool_calls_total",
		metric.WithDescription("Total number of tool calls"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool calls counter: %w", err)
	}

	toolLat, err := meter.Float64Histogram(
		"mcpjungle_tool_call_latency_seconds",
		metric.WithDescription("Latency of tool calls in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 20, 30),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool latency histogram: %w", err)
	}

	return &OtelCustomMetrics{
		toolCalls:       toolInv,
		toolCallLatency: toolLat,
	}, nil
}

func (m *OtelCustomMetrics) RecordToolCall(
	ctx context.Context, mcpServerName, toolName string, outcome ToolCallOutcome, elapsedTime time.Duration,
) {
	attrs := []attribute.KeyValue{
		attribute.String(labelMCPServerName, boundString(mcpServerName)),
		attribute.String(labelToolName, boundString(toolName)),
		attribute.String(labelToolCallOutcome, string(outcome)),
	}
	m.toolCalls.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.toolCallLatency.Record(ctx, elapsedTime.Seconds(), metric.WithAttributes(attrs...))
}

// boundString ensures strings are capped at maxLen and not empty.
func boundString(s string) string {
	if s == "" {
		return attrValueUnknown
	}
	if len(s) > attrValueMaxLen {
		return s[:attrValueMaxLen]
	}
	return s
}
