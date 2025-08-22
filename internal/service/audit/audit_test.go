package audit

import (
	"context"
	"testing"
	"time"
)

func TestAuditService_LogToolCall(t *testing.T) {
	// Test with development configuration
	config := Config{
		Environment: "development",
		LogLevel:    "info",
	}

	auditService, err := NewAuditService(config)
	if err != nil {
		t.Fatalf("Failed to create audit service: %v", err)
	}
	defer auditService.Close()

	ctx := context.Background()

	// Test successful tool call
	event := ToolCallEvent{
		RequestID:  "test-123",
		ClientName: "test-client",
		ServerName: "test-server",
		ToolName:   "test-tool",
		Success:    true,
		Duration:   100 * time.Millisecond,
	}

	// This should not panic or error
	auditService.LogToolCall(ctx, event)

	// Test failed tool call
	failedEvent := ToolCallEvent{
		RequestID:    "test-456",
		ClientName:   "test-client",
		ServerName:   "test-server",
		ToolName:     "test-tool",
		Success:      false,
		Duration:     50 * time.Millisecond,
		ErrorMessage: "connection timeout",
	}

	auditService.LogToolCall(ctx, failedEvent)

	// Test start event
	startEvent := ToolCallStartEvent{
		RequestID:  "test-789",
		ClientName: "test-client",
		ServerName: "test-server",
		ToolName:   "test-tool",
	}

	auditService.LogToolCallStart(ctx, startEvent)
}

func TestAuditService_Production(t *testing.T) {
	// Test with production configuration
	config := Config{
		Environment: "production",
		LogLevel:    "warn",
	}

	auditService, err := NewAuditService(config)
	if err != nil {
		t.Fatalf("Failed to create audit service: %v", err)
	}

	// Verify service is healthy
	if !auditService.Health() {
		t.Error("Audit service should be healthy after creation")
	}

	// Test that closing works - ignore sync errors in test environment
	_ = auditService.Close()

	// Verify service is no longer healthy
	if auditService.Health() {
		t.Error("Audit service should not be healthy after close")
	}
}
