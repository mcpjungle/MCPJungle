package logger

import (
	"context"
	"strings"
)

// Context keys for common logging fields
const (
	RequestIDKey     = "request_id"
	UserIDKey        = "user_id"
	CorrelationIDKey = "correlation_id"
	TraceIDKey       = "trace_id"
	SpanIDKey        = "span_id"
)

// ContextKey represents a type-safe context key
type ContextKey string

// String returns the string representation of the context key
func (ck ContextKey) String() string {
	return string(ck)
}

// Typed context keys for better type safety
const (
	RequestIDContextKey     = ContextKey(RequestIDKey)
	UserIDContextKey        = ContextKey(UserIDKey)
	CorrelationIDContextKey = ContextKey(CorrelationIDKey)
	TraceIDContextKey       = ContextKey(TraceIDKey)
	SpanIDContextKey        = ContextKey(SpanIDKey)
)

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil || strings.TrimSpace(requestID) == "" {
		return ctx
	}
	return context.WithValue(ctx, RequestIDContextKey, strings.TrimSpace(requestID))
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	if ctx == nil || strings.TrimSpace(userID) == "" {
		return ctx
	}
	return context.WithValue(ctx, UserIDContextKey, strings.TrimSpace(userID))
}

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	if ctx == nil || strings.TrimSpace(correlationID) == "" {
		return ctx
	}
	return context.WithValue(ctx, CorrelationIDContextKey, strings.TrimSpace(correlationID))
}

// WithTraceID adds a trace ID to the context (for OpenTelemetry compatibility)
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil || strings.TrimSpace(traceID) == "" {
		return ctx
	}
	return context.WithValue(ctx, TraceIDContextKey, strings.TrimSpace(traceID))
}

// WithSpanID adds a span ID to the context (for OpenTelemetry compatibility)
func WithSpanID(ctx context.Context, spanID string) context.Context {
	if ctx == nil || strings.TrimSpace(spanID) == "" {
		return ctx
	}
	return context.WithValue(ctx, SpanIDContextKey, strings.TrimSpace(spanID))
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Try typed key first
	if requestID, ok := ctx.Value(RequestIDContextKey).(string); ok && requestID != "" {
		return requestID
	}

	// Fallback to string key for backward compatibility
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
		return requestID
	}

	return ""
}

// GetUserID retrieves the user ID from context
func GetUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Try typed key first
	if userID, ok := ctx.Value(UserIDContextKey).(string); ok && userID != "" {
		return userID
	}

	// Fallback to string key for backward compatibility
	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		return userID
	}

	return ""
}

// GetCorrelationID retrieves the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Try typed key first
	if correlationID, ok := ctx.Value(CorrelationIDContextKey).(string); ok && correlationID != "" {
		return correlationID
	}

	// Fallback to string key for backward compatibility
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok && correlationID != "" {
		return correlationID
	}

	return ""
}

// GetTraceID retrieves the trace ID from context
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Try typed key first
	if traceID, ok := ctx.Value(TraceIDContextKey).(string); ok && traceID != "" {
		return traceID
	}

	// Fallback to string key for backward compatibility
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		return traceID
	}

	return ""
}

// GetSpanID retrieves the span ID from context
func GetSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Try typed key first
	if spanID, ok := ctx.Value(SpanIDContextKey).(string); ok && spanID != "" {
		return spanID
	}

	// Fallback to string key for backward compatibility
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok && spanID != "" {
		return spanID
	}

	return ""
}
