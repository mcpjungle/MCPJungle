package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger interface defines the minimal required logging methods
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithFields(fields ...Field) Logger
	Sync() error
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value any
}

// Config holds minimal logger configuration
type Config struct {
	Level       string `json:"level"`       // "debug", "info", "warn", "error"
	Development bool   `json:"development"` // true for development mode (console), false for production (json)
}

// zapLogger implements the Logger interface using uber/zap
type zapLogger struct {
	*zap.Logger
}

// DefaultConfig returns a default configuration for development
func DefaultConfig() *Config {
	return &Config{
		Level:       zap.InfoLevel.String(),
		Development: true,
	}
}

// ProductionConfig returns a configuration for production
func ProductionConfig() *Config {
	return &Config{
		Level:       zap.InfoLevel.String(),
		Development: false,
	}
}

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// ErrorField creates an error field
func ErrorField(err error) Field {
	return Field{Key: "error", Value: err}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate level
	if _, err := zapcore.ParseLevel(c.Level); err != nil {
		return fmt.Errorf("invalid log level: %s", c.Level)
	}

	return nil
}

// New creates a new logger with the given configuration
func New(config *Config) (Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Parse log level
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log level: %w", err)
	}

	// Create encoder config
	var encoderConfig zapcore.EncoderConfig
	var encoder zapcore.Encoder

	if config.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create core with stdout
	writeSyncer := zapcore.AddSync(os.Stdout)
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// Create zap logger
	zapLog := zap.New(core)

	return &zapLogger{Logger: zapLog}, nil
}

// NewDevelopment creates a logger with development configuration
func NewDevelopment() (Logger, error) {
	return New(DefaultConfig())
}

// NewProduction creates a logger with production configuration
func NewProduction() (Logger, error) {
	return New(ProductionConfig())
}

// Convert fields to zap fields
func fieldsToZap(fields []Field) []zap.Field {
	if len(fields) == 0 {
		return nil
	}

	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	return zapFields
}

// Debug logs a debug message
func (l *zapLogger) Debug(msg string, fields ...Field) {
	if l.Logger != nil {
		l.Logger.Debug(msg, fieldsToZap(fields)...)
	}
}

// Info logs an info message
func (l *zapLogger) Info(msg string, fields ...Field) {
	if l.Logger != nil {
		l.Logger.Info(msg, fieldsToZap(fields)...)
	}
}

// Warn logs a warning message
func (l *zapLogger) Warn(msg string, fields ...Field) {
	if l.Logger != nil {
		l.Logger.Warn(msg, fieldsToZap(fields)...)
	}
}

// Error logs an error message
func (l *zapLogger) Error(msg string, fields ...Field) {
	if l.Logger != nil {
		l.Logger.Error(msg, fieldsToZap(fields)...)
	}
}

// WithFields creates a new logger with additional fields
func (l *zapLogger) WithFields(fields ...Field) Logger {
	if len(fields) == 0 || l.Logger == nil {
		return l
	}

	zapFields := fieldsToZap(fields)
	newLogger := l.With(zapFields...)

	return &zapLogger{Logger: newLogger}
}

// Sync flushes any buffered log entries
func (l *zapLogger) Sync() error {
	if l.Logger != nil {
		return l.Logger.Sync()
	}
	return nil
}
