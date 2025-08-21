package logger

import (
	"context"
	"sync/atomic"
)

var (
	globalLogger atomic.Value // Logger
)

// SetGlobalLogger sets the global logger instance (thread-safe)
func SetGlobalLogger(logger Logger) {
	if logger != nil {
		globalLogger.Store(logger)
	}
}

// GetGlobalLogger returns the global logger instance (thread-safe)
func GetGlobalLogger() Logger {
	if logger, ok := globalLogger.Load().(Logger); ok {
		return logger
	}
	return nil
}

// InitGlobalLogger initializes the global logger with the given configuration
func InitGlobalLogger(config *Config) error {
	logger, err := New(config)
	if err != nil {
		return err
	}
	SetGlobalLogger(logger)
	return nil
}

// InitGlobalDevelopmentLogger initializes the global logger with development configuration
func InitGlobalDevelopmentLogger() error {
	return InitGlobalLogger(DefaultConfig())
}

// InitGlobalProductionLogger initializes the global logger with production configuration
func InitGlobalProductionLogger() error {
	return InitGlobalLogger(ProductionConfig())
}

// Global convenience functions

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...Field) {
	if logger := GetGlobalLogger(); logger != nil {
		logger.Debug(msg, fields...)
	}
}

// Info logs an info message using the global logger
func Info(msg string, fields ...Field) {
	if logger := GetGlobalLogger(); logger != nil {
		logger.Info(msg, fields...)
	}
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...Field) {
	if logger := GetGlobalLogger(); logger != nil {
		logger.Warn(msg, fields...)
	}
}

// Error logs an error message using the global logger
func Error(msg string, fields ...Field) {
	if logger := GetGlobalLogger(); logger != nil {
		logger.Error(msg, fields...)
	}
}

// WithContext creates a new logger with context fields using the global logger
func WithContext(ctx context.Context) Logger {
	if logger := GetGlobalLogger(); logger != nil {
		return logger.WithContext(ctx)
	}
	return nil
}

// WithFields creates a new logger with additional fields using the global logger
func WithFields(fields ...Field) Logger {
	if logger := GetGlobalLogger(); logger != nil {
		return logger.WithFields(fields...)
	}
	return nil
}

// SyncGlobal flushes any buffered log entries from the global logger
func SyncGlobal() error {
	if logger := GetGlobalLogger(); logger != nil {
		return logger.Sync()
	}
	return nil
}
