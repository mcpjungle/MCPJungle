package logger

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	if config.Level != "info" {
		t.Errorf("Expected level 'info', got '%s'", config.Level)
	}
	if !config.Development {
		t.Error("Expected development mode to be true")
	}
}

func TestProductionConfig(t *testing.T) {
	config := ProductionConfig()
	if config == nil {
		t.Fatal("ProductionConfig() returned nil")
	}
	if config.Level != "info" {
		t.Errorf("Expected level 'info', got '%s'", config.Level)
	}
	if config.Development {
		t.Error("Expected development mode to be false")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid debug level",
			config: &Config{
				Level:       "debug",
				Development: true,
			},
			wantErr: false,
		},
		{
			name: "valid info level",
			config: &Config{
				Level:       "info",
				Development: true,
			},
			wantErr: false,
		},
		{
			name: "valid warn level",
			config: &Config{
				Level:       "warn",
				Development: true,
			},
			wantErr: false,
		},
		{
			name: "valid error level",
			config: &Config{
				Level:       "error",
				Development: true,
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: &Config{
				Level:       "invalid",
				Development: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFieldHelpers(t *testing.T) {
	// Test String field
	strField := String("key", "value")
	if strField.Key != "key" || strField.Value != "value" {
		t.Errorf("String() failed: got %+v", strField)
	}

	// Test Int field
	intField := Int("key", 42)
	if intField.Key != "key" || intField.Value != 42 {
		t.Errorf("Int() failed: got %+v", intField)
	}

	// Test Int64 field
	int64Field := Int64("key", 123456789)
	if int64Field.Key != "key" || int64Field.Value != int64(123456789) {
		t.Errorf("Int64() failed: got %+v", int64Field)
	}

	// Test Float64 field
	floatField := Float64("key", 3.14)
	if floatField.Key != "key" || floatField.Value != 3.14 {
		t.Errorf("Float64() failed: got %+v", floatField)
	}

	// Test Bool field
	boolField := Bool("key", true)
	if boolField.Key != "key" || boolField.Value != true {
		t.Errorf("Bool() failed: got %+v", boolField)
	}

	// Test Any field
	anyField := Any("key", "any value")
	if anyField.Key != "key" || anyField.Value != "any value" {
		t.Errorf("Any() failed: got %+v", anyField)
	}

	// Test ErrorField
	testErr := fmt.Errorf("test error")
	errField := ErrorField(testErr)
	if errField.Key != "error" || errField.Value != testErr {
		t.Errorf("ErrorField() failed: got %+v", errField)
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config uses default",
			config:  nil,
			wantErr: false,
		},
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
			name: "invalid level",
			config: &Config{
				Level:       "invalid",
				Development: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("New() returned nil logger when no error expected")
			}
		})
	}
}

func TestNewDevelopment(t *testing.T) {
	logger, err := NewDevelopment()
	if err != nil {
		t.Errorf("NewDevelopment() error = %v", err)
	}
	if logger == nil {
		t.Error("NewDevelopment() returned nil logger")
	}
}

func TestNewProduction(t *testing.T) {
	logger, err := NewProduction()
	if err != nil {
		t.Errorf("NewProduction() error = %v", err)
	}
	if logger == nil {
		t.Error("NewProduction() returned nil logger")
	}
}

func TestZapLoggerMethods(t *testing.T) {
	logger, err := NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test all logging methods
	logger.Debug("debug message", String("key", "value"))
	logger.Info("info message", Int("count", 42))
	logger.Warn("warn message", Bool("flag", true))
	logger.Error("error message", ErrorField(fmt.Errorf("test error")))

	// Test WithFields
	loggerWithFields := logger.WithFields(String("field1", "value1"), Int("field2", 123))
	if loggerWithFields == nil {
		t.Error("WithFields() returned nil")
	}

	// Test Sync - ignore sync errors on stdout in test environment
	err = logger.Sync()
	// Sync errors on stdout are common in test environments, so we don't fail the test
	if err != nil && !strings.Contains(err.Error(), "sync") {
		t.Errorf("Sync() unexpected error = %v", err)
	}
}

func TestWithFieldsEmpty(t *testing.T) {
	logger, err := NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	loggerWithFields := logger.WithFields()
	if loggerWithFields == nil {
		t.Error("WithFields() returned nil for empty fields")
	}
}

func TestFieldsToZap(t *testing.T) {
	fields := []Field{
		String("str", "value"),
		Int("int", 42),
		Bool("bool", true),
	}

	zapFields := fieldsToZap(fields)
	if len(zapFields) != len(fields) {
		t.Errorf("Expected %d zap fields, got %d", len(fields), len(zapFields))
	}

	// Test empty fields
	emptyZapFields := fieldsToZap(nil)
	if emptyZapFields != nil {
		t.Error("fieldsToZap(nil) should return nil")
	}

	emptyZapFields = fieldsToZap([]Field{})
	if emptyZapFields != nil {
		t.Error("fieldsToZap([]Field{}) should return nil")
	}
}

func TestLoggerInterface(t *testing.T) {
	// Test that zapLogger implements Logger interface
	var _ Logger = (*zapLogger)(nil)
}

func TestLoggerWithNilZapLogger(t *testing.T) {
	// Create a zapLogger with nil zap.Logger
	logger := &zapLogger{Logger: nil}

	// These should not panic
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	// Sync should return nil error when logger is nil
	err := logger.Sync()
	if err != nil {
		t.Errorf("Sync() should return nil error for nil logger, got %v", err)
	}
}

func TestLoggerOutput(t *testing.T) {
	// Redirect stdout to capture log output
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
		w.Close()
	}()

	// Create logger and log a message
	logger, err := NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Info("test message", String("key", "value"))

	// Close write end and read output
	w.Close()
	output := make([]byte, 1024)
	n, err := r.Read(output)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Failed to read output: %v", err)
	}

	outputStr := string(output[:n])
	if !strings.Contains(outputStr, "test message") {
		t.Errorf("Expected log output to contain 'test message', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "key") {
		t.Errorf("Expected log output to contain 'key', got: %s", outputStr)
	}
}

func BenchmarkLoggerCreation(b *testing.B) {
	config := DefaultConfig()
	for i := 0; i < b.N; i++ {
		_, err := New(config)
		if err != nil {
			b.Fatalf("Failed to create logger: %v", err)
		}
	}
}

func BenchmarkLoggerInfo(b *testing.B) {
	logger, err := NewDevelopment()
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", Int("iteration", i))
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	logger, err := NewDevelopment()
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}

	fields := []Field{
		String("field1", "value1"),
		Int("field2", 42),
		Bool("field3", true),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(fields...)
	}
}
