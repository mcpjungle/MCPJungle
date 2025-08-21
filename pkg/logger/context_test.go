package logger

import (
	"context"
	"strings"
	"testing"
)

func TestContextKeyString(t *testing.T) {
	key := ContextKey("test_key")
	if key.String() != "test_key" {
		t.Errorf("Expected 'test_key', got '%s'", key.String())
	}
}

func TestWithRequestID(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		requestID string
		wantNil   bool
	}{
		{
			name:      "nil context",
			ctx:       nil,
			requestID: "req-123",
			wantNil:   true,
		},
		{
			name:      "empty request ID",
			ctx:       context.Background(),
			requestID: "",
			wantNil:   true,
		},
		{
			name:      "whitespace request ID",
			ctx:       context.Background(),
			requestID: "   ",
			wantNil:   true,
		},
		{
			name:      "valid request ID",
			ctx:       context.Background(),
			requestID: "req-123",
			wantNil:   false,
		},
		{
			name:      "request ID with whitespace",
			ctx:       context.Background(),
			requestID: "  req-123  ",
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithRequestID(tt.ctx, tt.requestID)
			if tt.wantNil {
				if result != tt.ctx {
					t.Errorf("Expected original context, got different context")
				}
			} else {
				if result == tt.ctx {
					t.Error("Expected new context, got same context")
				}
				// Verify the value was set
				value := GetRequestID(result)
				expected := strings.TrimSpace(tt.requestID)
				if value != expected {
					t.Errorf("Expected request ID '%s', got '%s'", expected, value)
				}
			}
		})
	}
}

func TestWithUserID(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		userID  string
		wantNil bool
	}{
		{
			name:    "nil context",
			ctx:     nil,
			userID:  "user-123",
			wantNil: true,
		},
		{
			name:    "empty user ID",
			ctx:     context.Background(),
			userID:  "",
			wantNil: true,
		},
		{
			name:    "whitespace user ID",
			ctx:     context.Background(),
			userID:  "   ",
			wantNil: true,
		},
		{
			name:    "valid user ID",
			ctx:     context.Background(),
			userID:  "user-123",
			wantNil: false,
		},
		{
			name:    "user ID with whitespace",
			ctx:     context.Background(),
			userID:  "  user-123  ",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithUserID(tt.ctx, tt.userID)
			if tt.wantNil {
				if result != tt.ctx {
					t.Errorf("Expected original context, got different context")
				}
			} else {
				if result == tt.ctx {
					t.Error("Expected new context, got same context")
				}
				// Verify the value was set
				value := GetUserID(result)
				expected := strings.TrimSpace(tt.userID)
				if value != expected {
					t.Errorf("Expected user ID '%s', got '%s'", expected, value)
				}
			}
		})
	}
}

func TestWithCorrelationID(t *testing.T) {
	tests := []struct {
		name          string
		ctx           context.Context
		correlationID string
		wantNil       bool
	}{
		{
			name:          "nil context",
			ctx:           nil,
			correlationID: "corr-123",
			wantNil:       true,
		},
		{
			name:          "empty correlation ID",
			ctx:           context.Background(),
			correlationID: "",
			wantNil:       true,
		},
		{
			name:          "whitespace correlation ID",
			ctx:           context.Background(),
			correlationID: "   ",
			wantNil:       true,
		},
		{
			name:          "valid correlation ID",
			ctx:           context.Background(),
			correlationID: "corr-123",
			wantNil:       false,
		},
		{
			name:          "correlation ID with whitespace",
			ctx:           context.Background(),
			correlationID: "  corr-123  ",
			wantNil:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithCorrelationID(tt.ctx, tt.correlationID)
			if tt.wantNil {
				if result != tt.ctx {
					t.Errorf("Expected original context, got different context")
				}
			} else {
				if result == tt.ctx {
					t.Error("Expected new context, got same context")
				}
				// Verify the value was set
				value := GetCorrelationID(result)
				expected := strings.TrimSpace(tt.correlationID)
				if value != expected {
					t.Errorf("Expected correlation ID '%s', got '%s'", expected, value)
				}
			}
		})
	}
}

func TestWithTraceID(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		traceID string
		wantNil bool
	}{
		{
			name:    "nil context",
			ctx:     nil,
			traceID: "trace-123",
			wantNil: true,
		},
		{
			name:    "empty trace ID",
			ctx:     context.Background(),
			traceID: "",
			wantNil: true,
		},
		{
			name:    "whitespace trace ID",
			ctx:     context.Background(),
			traceID: "   ",
			wantNil: true,
		},
		{
			name:    "valid trace ID",
			ctx:     context.Background(),
			traceID: "trace-123",
			wantNil: false,
		},
		{
			name:    "trace ID with whitespace",
			ctx:     context.Background(),
			traceID: "  trace-123  ",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithTraceID(tt.ctx, tt.traceID)
			if tt.wantNil {
				if result != tt.ctx {
					t.Errorf("Expected original context, got different context")
				}
			} else {
				if result == tt.ctx {
					t.Error("Expected new context, got same context")
				}
				// Verify the value was set
				value := GetTraceID(result)
				expected := strings.TrimSpace(tt.traceID)
				if value != expected {
					t.Errorf("Expected trace ID '%s', got '%s'", expected, value)
				}
			}
		})
	}
}

func TestWithSpanID(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		spanID  string
		wantNil bool
	}{
		{
			name:    "nil context",
			ctx:     nil,
			spanID:  "span-123",
			wantNil: true,
		},
		{
			name:    "empty span ID",
			ctx:     context.Background(),
			spanID:  "",
			wantNil: true,
		},
		{
			name:    "whitespace span ID",
			ctx:     context.Background(),
			spanID:  "   ",
			wantNil: true,
		},
		{
			name:    "valid span ID",
			ctx:     context.Background(),
			spanID:  "span-123",
			wantNil: false,
		},
		{
			name:    "span ID with whitespace",
			ctx:     context.Background(),
			spanID:  "  span-123  ",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithSpanID(tt.ctx, tt.spanID)
			if tt.wantNil {
				if result != tt.ctx {
					t.Errorf("Expected original context, got different context")
				}
			} else {
				if result == tt.ctx {
					t.Error("Expected new context, got same context")
				}
				// Verify the value was set
				value := GetSpanID(result)
				expected := strings.TrimSpace(tt.spanID)
				if value != expected {
					t.Errorf("Expected span ID '%s', got '%s'", expected, value)
				}
			}
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "context with typed key",
			ctx:      WithRequestID(context.Background(), "req-123"),
			expected: "req-123",
		},
		{
			name:     "context with string key",
			ctx:      context.WithValue(context.Background(), RequestIDKey, "req-456"),
			expected: "req-456",
		},
		{
			name:     "context with empty value",
			ctx:      context.WithValue(context.Background(), RequestIDContextKey, ""),
			expected: "",
		},
		{
			name:     "context with wrong type",
			ctx:      context.WithValue(context.Background(), RequestIDContextKey, 123),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRequestID(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "context with typed key",
			ctx:      WithUserID(context.Background(), "user-123"),
			expected: "user-123",
		},
		{
			name:     "context with string key",
			ctx:      context.WithValue(context.Background(), UserIDKey, "user-456"),
			expected: "user-456",
		},
		{
			name:     "context with empty value",
			ctx:      context.WithValue(context.Background(), UserIDContextKey, ""),
			expected: "",
		},
		{
			name:     "context with wrong type",
			ctx:      context.WithValue(context.Background(), UserIDContextKey, 123),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserID(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetCorrelationID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "context with typed key",
			ctx:      WithCorrelationID(context.Background(), "corr-123"),
			expected: "corr-123",
		},
		{
			name:     "context with string key",
			ctx:      context.WithValue(context.Background(), CorrelationIDKey, "corr-456"),
			expected: "corr-456",
		},
		{
			name:     "context with empty value",
			ctx:      context.WithValue(context.Background(), CorrelationIDContextKey, ""),
			expected: "",
		},
		{
			name:     "context with wrong type",
			ctx:      context.WithValue(context.Background(), CorrelationIDContextKey, 123),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCorrelationID(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetTraceID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "context with typed key",
			ctx:      WithTraceID(context.Background(), "trace-123"),
			expected: "trace-123",
		},
		{
			name:     "context with string key",
			ctx:      context.WithValue(context.Background(), TraceIDKey, "trace-456"),
			expected: "trace-456",
		},
		{
			name:     "context with empty value",
			ctx:      context.WithValue(context.Background(), TraceIDContextKey, ""),
			expected: "",
		},
		{
			name:     "context with wrong type",
			ctx:      context.WithValue(context.Background(), TraceIDContextKey, 123),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTraceID(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetSpanID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "context with typed key",
			ctx:      WithSpanID(context.Background(), "span-123"),
			expected: "span-123",
		},
		{
			name:     "context with string key",
			ctx:      context.WithValue(context.Background(), SpanIDKey, "span-456"),
			expected: "span-456",
		},
		{
			name:     "context with empty value",
			ctx:      context.WithValue(context.Background(), SpanIDContextKey, ""),
			expected: "",
		},
		{
			name:     "context with wrong type",
			ctx:      context.WithValue(context.Background(), SpanIDContextKey, 123),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSpanID(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestContextChaining(t *testing.T) {
	// Test that multiple context operations can be chained
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")
	ctx = WithCorrelationID(ctx, "corr-789")
	ctx = WithTraceID(ctx, "trace-abc")
	ctx = WithSpanID(ctx, "span-def")

	// Verify all values are preserved
	if GetRequestID(ctx) != "req-123" {
		t.Errorf("Expected request ID 'req-123', got '%s'", GetRequestID(ctx))
	}
	if GetUserID(ctx) != "user-456" {
		t.Errorf("Expected user ID 'user-456', got '%s'", GetUserID(ctx))
	}
	if GetCorrelationID(ctx) != "corr-789" {
		t.Errorf("Expected correlation ID 'corr-789', got '%s'", GetCorrelationID(ctx))
	}
	if GetTraceID(ctx) != "trace-abc" {
		t.Errorf("Expected trace ID 'trace-abc', got '%s'", GetTraceID(ctx))
	}
	if GetSpanID(ctx) != "span-def" {
		t.Errorf("Expected span ID 'span-def', got '%s'", GetSpanID(ctx))
	}
}

func TestContextConstants(t *testing.T) {
	// Test that the context key constants are properly defined
	if RequestIDKey != "request_id" {
		t.Errorf("Expected RequestIDKey to be 'request_id', got '%s'", RequestIDKey)
	}
	if UserIDKey != "user_id" {
		t.Errorf("Expected UserIDKey to be 'user_id', got '%s'", UserIDKey)
	}
	if CorrelationIDKey != "correlation_id" {
		t.Errorf("Expected CorrelationIDKey to be 'correlation_id', got '%s'", CorrelationIDKey)
	}
	if TraceIDKey != "trace_id" {
		t.Errorf("Expected TraceIDKey to be 'trace_id', got '%s'", TraceIDKey)
	}
	if SpanIDKey != "span_id" {
		t.Errorf("Expected SpanIDKey to be 'span_id', got '%s'", SpanIDKey)
	}
}

func TestContextKeyConstants(t *testing.T) {
	// Test that the typed context key constants are properly defined
	if RequestIDContextKey != ContextKey("request_id") {
		t.Errorf("Expected RequestIDContextKey to be ContextKey('request_id')")
	}
	if UserIDContextKey != ContextKey("user_id") {
		t.Errorf("Expected UserIDContextKey to be ContextKey('user_id')")
	}
	if CorrelationIDContextKey != ContextKey("correlation_id") {
		t.Errorf("Expected CorrelationIDContextKey to be ContextKey('correlation_id')")
	}
	if TraceIDContextKey != ContextKey("trace_id") {
		t.Errorf("Expected TraceIDContextKey to be ContextKey('trace_id')")
	}
	if SpanIDContextKey != ContextKey("span_id") {
		t.Errorf("Expected SpanIDContextKey to be ContextKey('span_id')")
	}
}

func BenchmarkContextOperations(b *testing.B) {
	ctx := context.Background()

	b.Run("WithRequestID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			WithRequestID(ctx, "benchmark-request")
		}
	})

	b.Run("GetRequestID", func(b *testing.B) {
		ctxWithValue := WithRequestID(ctx, "benchmark-request")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			GetRequestID(ctxWithValue)
		}
	})

	b.Run("MultipleContextOperations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			ctx = WithRequestID(ctx, "req-123")
			ctx = WithUserID(ctx, "user-456")
			ctx = WithCorrelationID(ctx, "corr-789")
			ctx = WithTraceID(ctx, "trace-abc")
			ctx = WithSpanID(ctx, "span-def")
			_ = ctx
		}
	})
}
