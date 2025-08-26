package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	// LabelMCPServerName is the name of the MCP server
	LabelMCPServerName = "mcp_server_name"
	// LabelToolName is the name of the tool
	LabelToolName = "tool_name"
	// LabelStatus is the status of the operation
	LabelStatus = "status"
	// LabelErrorType is the type of error
	LabelErrorType = "error_type"
	// LabelEventType is the type of event
	LabelEventType = "event_type"
	// LabelMCPMethod is the method of the MCP request
	LabelMCPMethod = "mcp_method"
)

const (
	// ErrorTypeTimeout is a timeout error
	ErrorTypeTimeout = "timeout"
	// ErrorTypeConnection is a connection error
	ErrorTypeConnection = "connection"
	// ErrorTypePermission is a permission error
	ErrorTypePermission = "permission"
	// ErrorTypeValidation is a validation error
	ErrorTypeValidation = "validation"
	// ErrorTypeRegistration is a registration error
	ErrorTypeRegistration = "registration"
	// ErrorTypeUnknown is an unknown error
	ErrorTypeUnknown = "unknown"
)

const (
	// EventTypeRegistered is a registered event
	EventTypeRegistered = "registered"
	// EventTypeDeregistered is a deregistered event
	EventTypeDeregistered = "deregistered"
)

const (
	// StatusSuccess is a successful operation
	StatusSuccess MetricStatus = "success"
	// StatusError is an error operation
	StatusError MetricStatus = "error"
)

// MetricStatus represents the status of an operation.
type MetricStatus string

// MCPMetrics bundles all the OpenTelemetry metric instruments used for MCPJungle.
// It provides convenience methods for recording tool usage, requests, errors,
// server lifecycle events, and tool availability.
type MCPMetrics struct {
	// Tool-related metrics
	ToolInvocations metric.Int64Counter
	ToolLatency     metric.Float64Histogram
	ToolErrors      metric.Int64Counter

	// Request-related metrics
	RequestsTotal  metric.Int64Counter
	RequestLatency metric.Float64Histogram
	RequestErrors  metric.Int64Counter

	// Session-related metrics
	Sessions metric.Int64UpDownCounter

	// Server-related metrics
	ServerRegistrations   metric.Int64Counter
	ServerDeregistrations metric.Int64Counter
	ServerTransport       metric.Int64Counter

	// Tool availability metrics
	ToolAvailability metric.Int64UpDownCounter
	ToolDiscovery    metric.Int64Counter

	// Enhanced error metrics
	ErrorsByType metric.Int64Counter
}

// NewMCPMetrics initializes all metric instruments required by MCPJungle.
// Returns an MCPMetrics instance ready for use, or an error if any instrument
// could not be created.
func NewMCPMetrics(meter metric.Meter) (*MCPMetrics, error) {
	if meter == nil {
		return nil, fmt.Errorf("meter cannot be nil")
	}

	// Tool metrics
	toolInv, err := meter.Int64Counter("mcpjungle_tool_invocations_total",
		metric.WithDescription("Total count of tool invocation attempts"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create tool invocations counter: %w", err)
	}

	// Tool latency metrics
	toolLat, err := meter.Float64Histogram("mcpjungle_tool_latency_seconds",
		metric.WithDescription("Latency of tool calls in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05,
			0.1, 0.25, 0.5, 1, 2, 5, 10))
	if err != nil {
		return nil, fmt.Errorf("failed to create tool latency histogram: %w", err)
	}

	// Tool error metrics
	toolErrors, err := meter.Int64Counter("mcpjungle_tool_errors_total",
		metric.WithDescription("Total count of tool invocation errors"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create tool errors counter: %w", err)
	}

	// Request metrics
	reqTotal, err := meter.Int64Counter("mcpjungle_client_requests_total",
		metric.WithDescription("Total count of MCP client RPC requests"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create requests counter: %w", err)
	}

	// Request latency metrics
	reqLat, err := meter.Float64Histogram("mcpjungle_request_duration_seconds",
		metric.WithDescription("Duration of MCP RPC handlers in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05,
			0.1, 0.25, 0.5, 1, 2, 5, 10))
	if err != nil {
		return nil, fmt.Errorf("failed to create request latency histogram: %w", err)
	}

	// Request error metrics
	reqErrors, err := meter.Int64Counter("mcpjungle_request_errors_total",
		metric.WithDescription("Total count of MCP request errors"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create request errors counter: %w", err)
	}

	// Session metrics
	sessions, err := meter.Int64UpDownCounter("mcpjungle_connected_sessions",
		metric.WithDescription("Current number of active MCP client sessions"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create sessions counter: %w", err)
	}

	// Server metrics
	serverReg, err := meter.Int64Counter("mcpjungle_server_registrations_total",
		metric.WithDescription("Total count of MCP server registrations"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create server registrations counter: %w", err)
	}

	// Server deregistration metrics
	serverDereg, err := meter.Int64Counter("mcpjungle_server_deregistrations_total",
		metric.WithDescription("Total count of MCP server deregistrations"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create server deregistrations counter: %w", err)
	}

	// Server transport metrics
	serverTransport, err := meter.Int64Counter("mcpjungle_server_transport_total",
		metric.WithDescription("Total count of MCP server transport events"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create server transport counter: %w", err)
	}

	// Tool availability metrics
	toolAvailability, err := meter.Int64UpDownCounter("mcpjungle_tool_availability",
		metric.WithDescription("Current availability of MCP tools (1=available, 0=unavailable)"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create tool availability counter: %w", err)
	}

	// Tool discovery metrics
	toolDiscovery, err := meter.Int64Counter("mcpjungle_tool_discovery_total",
		metric.WithDescription("Total count of MCP tool discovery attempts"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create tool discovery counter: %w", err)
	}

	// Error metrics
	errorsByType, err := meter.Int64Counter("mcpjungle_errors_by_type_total",
		metric.WithDescription("Total count of errors by type"),
		metric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create enhanced error counter: %w", err)
	}

	return &MCPMetrics{
		ToolInvocations:       toolInv,
		ToolLatency:           toolLat,
		ToolErrors:            toolErrors,
		RequestsTotal:         reqTotal,
		RequestLatency:        reqLat,
		RequestErrors:         reqErrors,
		Sessions:              sessions,
		ServerRegistrations:   serverReg,
		ServerDeregistrations: serverDereg,
		ServerTransport:       serverTransport,
		ToolAvailability:      toolAvailability,
		ToolDiscovery:         toolDiscovery,
		ErrorsByType:          errorsByType,
	}, nil
}

// RecordTool records a tool invocation, its latency, and errors if any.
func (m *MCPMetrics) RecordTool(ctx context.Context, mcpServerName, toolName string, status MetricStatus, started time.Time, err error) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String(LabelMCPServerName, boundString(mcpServerName, 64, "unknown")),
		attribute.String(LabelToolName, boundString(toolName, 64, "unknown")),
		attribute.String(LabelStatus, string(status)),
	}
	if err != nil {
		attrs = append(attrs, attribute.String(LabelErrorType, getErrorType(err)))
	}

	m.ToolInvocations.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.ToolLatency.Record(ctx, time.Since(started).Seconds(), metric.WithAttributes(attrs...))

	if status == StatusError {
		m.ToolErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordRequest records an MCP request, including latency and errors.
func (m *MCPMetrics) RecordRequest(ctx context.Context, mcpMethod string, status MetricStatus, started time.Time, err error) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String(LabelMCPMethod, boundString(mcpMethod, 64, "unknown")),
		attribute.String(LabelStatus, string(status)),
	}
	if err != nil {
		attrs = append(attrs, attribute.String(LabelErrorType, getErrorType(err)))
	}

	m.RequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.RequestLatency.Record(ctx, time.Since(started).Seconds(), metric.WithAttributes(attrs...))

	if status == StatusError {
		m.RequestErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordServerRegistration records a server registration attempt.
func (m *MCPMetrics) RecordServerRegistration(ctx context.Context, serverName string, success bool) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String(LabelMCPServerName, boundString(serverName, 64, "unknown")),
		attribute.Bool("success", success),
	}
	m.ServerRegistrations.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordServerDeregistration records a server deregistration event.
func (m *MCPMetrics) RecordServerDeregistration(ctx context.Context, serverName string) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String(LabelMCPServerName, boundString(serverName, 64, "unknown")),
	}
	m.ServerDeregistrations.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordServerTransport records a server transport lifecycle event.
func (m *MCPMetrics) RecordServerTransport(ctx context.Context, serverName, eventType string) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String(LabelMCPServerName, boundString(serverName, 64, "unknown")),
		attribute.String(LabelEventType, eventType),
	}
	m.ServerTransport.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordToolAvailability records tool availability as an up/down counter (1=available, 0=unavailable).
func (m *MCPMetrics) RecordToolAvailability(ctx context.Context, toolName string, available bool) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String(LabelToolName, boundString(toolName, 64, "unknown")),
	}
	var value int64
	if available {
		value = 1
	}
	m.ToolAvailability.Add(ctx, value, metric.WithAttributes(attrs...))
}

// RecordToolDiscovery records a tool discovery attempt.
func (m *MCPMetrics) RecordToolDiscovery(ctx context.Context, toolName string) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String(LabelToolName, boundString(toolName, 64, "unknown")),
	}
	m.ToolDiscovery.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordEnhancedError records an error occurrence by its categorized type.
func (m *MCPMetrics) RecordEnhancedError(ctx context.Context, errorType string) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String(LabelErrorType, errorType),
	}
	m.ErrorsByType.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// boundString ensures strings are capped at maxLen and not empty.
func boundString(s string, maxLen int, fallback string) string {
	if s == "" {
		return fallback
	}
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// getErrorType normalizes error messages into a small set of categories
// to avoid high-cardinality labels.
func getErrorType(err error) string {
	if err == nil {
		return "none"
	}

	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "timeout"):
		return ErrorTypeTimeout
	case strings.Contains(errStr, "connection"):
		return ErrorTypeConnection
	case strings.Contains(errStr, "permission"):
		return ErrorTypePermission
	case strings.Contains(errStr, "not found"):
		return "not_found"
	case strings.Contains(errStr, "invalid"):
		return "invalid"
	case strings.Contains(errStr, "cancelled"):
		return "cancelled"
	default:
		return ErrorTypeUnknown
	}
}
