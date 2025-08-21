package logger

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// clearGlobalLogger is a helper function to ensure the global logger is cleared
func clearGlobalLogger() {
	// We can't store nil in atomic.Value, so we'll use a dummy logger
	dummyLogger := &zapLogger{Logger: nil}
	globalLogger.Store(dummyLogger)
}

func TestSetGlobalLogger(t *testing.T) {
	// Test setting nil logger
	SetGlobalLogger(nil)
	logger := GetGlobalLogger()
	if logger != nil {
		t.Error("Expected nil logger after setting nil")
	}

	// Test setting valid logger
	testLogger, err := NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	SetGlobalLogger(testLogger)
	retrievedLogger := GetGlobalLogger()
	if retrievedLogger == nil {
		t.Error("Expected non-nil logger after setting valid logger")
	}
}

func TestGetGlobalLogger(t *testing.T) {
	// Clear any existing global logger first
	clearGlobalLogger()

	// Should return the dummy logger (not nil)
	logger := GetGlobalLogger()
	if logger == nil {
		t.Error("Expected non-nil logger (dummy logger)")
	}

	// Set a logger and retrieve it
	testLogger, err := NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	SetGlobalLogger(testLogger)
	retrievedLogger := GetGlobalLogger()
	if retrievedLogger == nil {
		t.Error("Expected non-nil logger after setting")
	}
}

func TestInitGlobalLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid development config",
			config: &Config{
				Level:       "debug",
				Development: true,
			},
			wantErr: false,
		},
		{
			name: "valid production config",
			config: &Config{
				Level:       "info",
				Development: false,
			},
			wantErr: false,
		},
		{
			name: "invalid config",
			config: &Config{
				Level:       "invalid",
				Development: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitGlobalLogger(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitGlobalLogger() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				logger := GetGlobalLogger()
				if logger == nil {
					t.Error("Expected non-nil logger after successful initialization")
				}
			}
		})
	}
}

func TestInitGlobalDevelopmentLogger(t *testing.T) {
	err := InitGlobalDevelopmentLogger()
	if err != nil {
		t.Errorf("InitGlobalDevelopmentLogger() error = %v", err)
	}

	logger := GetGlobalLogger()
	if logger == nil {
		t.Error("Expected non-nil logger after development initialization")
	}
}

func TestInitGlobalProductionLogger(t *testing.T) {
	err := InitGlobalProductionLogger()
	if err != nil {
		t.Errorf("InitGlobalProductionLogger() error = %v", err)
	}

	logger := GetGlobalLogger()
	if logger == nil {
		t.Error("Expected non-nil logger after production initialization")
	}
}

func TestGlobalLoggingFunctions(t *testing.T) {
	// Initialize global logger
	err := InitGlobalDevelopmentLogger()
	if err != nil {
		t.Fatalf("Failed to initialize global logger: %v", err)
	}

	// Test all global logging functions
	Debug("debug message", String("key", "value"))
	Info("info message", Int("count", 42))
	Warn("warn message", Bool("flag", true))
	Error("error message", ErrorField(context.DeadlineExceeded))
}

func TestGlobalLoggingFunctionsWithNilLogger(t *testing.T) {
	// Clear global logger
	SetGlobalLogger(nil)

	// These should not panic
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")
}

func TestWithContextGlobal(t *testing.T) {
	// Initialize global logger
	err := InitGlobalDevelopmentLogger()
	if err != nil {
		t.Fatalf("Failed to initialize global logger: %v", err)
	}

	// Test WithContext with valid context
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")

	logger := WithContext(ctx)
	if logger == nil {
		t.Error("Expected non-nil logger from WithContext")
	}

	// Test WithContext with nil context
	logger = WithContext(nil)
	if logger == nil {
		t.Error("Expected non-nil logger from WithContext with nil context")
	}
}

func TestWithContextGlobalWithNilLogger(t *testing.T) {
	// Clear global logger
	clearGlobalLogger()

	// Should return a logger (dummy logger) when global logger is set to dummy
	ctx := context.Background()
	logger := WithContext(ctx)
	if logger == nil {
		t.Error("Expected non-nil logger (dummy logger)")
	}
}

func TestWithFieldsGlobal(t *testing.T) {
	// Initialize global logger
	err := InitGlobalDevelopmentLogger()
	if err != nil {
		t.Fatalf("Failed to initialize global logger: %v", err)
	}

	// Test WithFields with valid fields
	fields := []Field{
		String("field1", "value1"),
		Int("field2", 42),
		Bool("field3", true),
	}

	logger := WithFields(fields...)
	if logger == nil {
		t.Error("Expected non-nil logger from WithFields")
	}

	// Test WithFields with empty fields
	logger = WithFields()
	if logger == nil {
		t.Error("Expected non-nil logger from WithFields with empty fields")
	}
}

func TestWithFieldsGlobalWithNilLogger(t *testing.T) {
	// Clear global logger
	clearGlobalLogger()

	// Should return a logger (dummy logger) when global logger is set to dummy
	fields := []Field{String("key", "value")}
	logger := WithFields(fields...)
	if logger == nil {
		t.Error("Expected non-nil logger (dummy logger)")
	}
}

func TestSyncGlobal(t *testing.T) {
	// Initialize global logger
	err := InitGlobalDevelopmentLogger()
	if err != nil {
		t.Fatalf("Failed to initialize global logger: %v", err)
	}

	// Test SyncGlobal - ignore sync errors on stdout in test environment
	err = SyncGlobal()
	// Sync errors on stdout are common in test environments, so we don't fail the test
	if err != nil && !strings.Contains(err.Error(), "sync") {
		t.Errorf("SyncGlobal() unexpected error = %v", err)
	}
}

func TestSyncGlobalWithNilLogger(t *testing.T) {
	// Clear global logger
	clearGlobalLogger()

	// Should return nil error when global logger is dummy
	err := SyncGlobal()
	if err != nil {
		t.Errorf("SyncGlobal() should return nil error when global logger is dummy, got %v", err)
	}
}

func TestGlobalLoggerThreadSafety(t *testing.T) {
	// This test verifies that the global logger operations are thread-safe
	// by running concurrent operations

	// Initialize global logger
	err := InitGlobalDevelopmentLogger()
	if err != nil {
		t.Fatalf("Failed to initialize global logger: %v", err)
	}

	// Run concurrent operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Test concurrent logging
			Info("concurrent message", Int("goroutine", id))

			// Test concurrent context operations
			ctx := context.Background()
			ctx = WithRequestID(ctx, fmt.Sprintf("req-%d", id))
			logger := WithContext(ctx)
			if logger != nil {
				logger.Info("context message", Int("goroutine", id))
			}

			// Test concurrent field operations
			logger = WithFields(Int("goroutine", id))
			if logger != nil {
				logger.Info("field message", Int("goroutine", id))
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkGlobalLoggerOperations(b *testing.B) {
	// Initialize global logger
	err := InitGlobalDevelopmentLogger()
	if err != nil {
		b.Fatalf("Failed to initialize global logger: %v", err)
	}

	b.Run("Info", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Info("benchmark message", Int("iteration", i))
		}
	})

	b.Run("WithFields", func(b *testing.B) {
		fields := []Field{
			String("field1", "value1"),
			Int("field2", 42),
		}
		for i := 0; i < b.N; i++ {
			WithFields(fields...)
		}
	})

	b.Run("WithContext", func(b *testing.B) {
		ctx := context.Background()
		ctx = WithRequestID(ctx, "benchmark-request")
		for i := 0; i < b.N; i++ {
			WithContext(ctx)
		}
	})
}
