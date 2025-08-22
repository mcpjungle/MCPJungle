package audit

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger interface defines the contract for audit logging
type Logger interface {
	LogToolCall(ctx context.Context, event ToolCallEvent)
	LogToolCallStart(ctx context.Context, event ToolCallStartEvent)
	Close() error
}

// ToolCallEvent represents a completed MCP tool call for audit logging
type ToolCallEvent struct {
	RequestID    string
	ClientName   string // Name of the MCP client making the request [for future use]
	ServerName   string
	ToolName     string
	Success      bool
	Duration     time.Duration
	ErrorMessage string
}

// ToolCallStartEvent represents the start of an MCP tool call
type ToolCallStartEvent struct {
	RequestID  string
	ClientName string 
	ServerName string
	ToolName   string
}

type AuditService struct {
	logger *zap.Logger
	mu     sync.RWMutex
	closed bool
}

// Config holds configuration for the audit service
type Config struct {
	Environment string // "production" or "development"
	LogLevel    string // "debug", "info", "warn", "error"
}

func NewAuditService(config Config) (*AuditService, error) {
	var zapConfig zap.Config

	if config.Environment == "production" {
		zapConfig = zap.NewProductionConfig()
		zapConfig.EncoderConfig.TimeKey = "timestamp"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		// Enable sampling in production to prevent log flooding
		zapConfig.Sampling = &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		}
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	if config.LogLevel != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(config.LogLevel)); err != nil {
			return nil, err
		}
		zapConfig.Level = zap.NewAtomicLevelAt(level)
	}

	logger, err := zapConfig.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(
			zap.String("service", "mcpjungle"),
			zap.String("component", "audit"),
		),
	)
	if err != nil {
		return nil, err
	}

	return &AuditService{
		logger: logger,
	}, nil
}

// LogToolCall logs a completed MCP tool call with all relevant metadata
func (a *AuditService) LogToolCall(ctx context.Context, event ToolCallEvent) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.closed {
		return
	}

	fields := []zap.Field{
		zap.String("event_type", "mcp_tool_call"),
		zap.String("request_id", event.RequestID),
		zap.String("client_name", event.ClientName),
		zap.String("server_name", event.ServerName),
		zap.String("tool_name", event.ToolName),
		zap.Bool("success", event.Success),
		zap.Int64("duration_ms", event.Duration.Milliseconds()),
	}

	if event.ErrorMessage != "" {
		fields = append(fields, zap.String("error", event.ErrorMessage))
	}

	// Add correlation ID from context if available
	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		fields = append(fields, zap.String("correlation_id", correlationID))
	}

	if event.Success {
		a.logger.Info("MCP tool call completed", fields...)
	} else {
		a.logger.Error("MCP tool call failed", fields...)
	}
}

// LogToolCallStart logs the initiation of an MCP tool call
func (a *AuditService) LogToolCallStart(ctx context.Context, event ToolCallStartEvent) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.closed {
		return
	}

	fields := []zap.Field{
		zap.String("event_type", "mcp_tool_call_start"),
		zap.String("request_id", event.RequestID),
		zap.String("client_name", event.ClientName),
		zap.String("server_name", event.ServerName),
		zap.String("tool_name", event.ToolName),
	}

	// Add correlation ID from context if available
	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		fields = append(fields, zap.String("correlation_id", correlationID))
	}

	a.logger.Info("MCP tool call started", fields...)
}

func (a *AuditService) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return nil
	}

	a.closed = true
	return a.logger.Sync()
}

func (a *AuditService) Health() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return !a.closed
}
