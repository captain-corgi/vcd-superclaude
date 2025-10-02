package logger

import (
	"context"
	"os"
	"testing"

	"github.com/captain-corgi/multi-agent-system/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func TestNewDatadogLogger(t *testing.T) {
	// Set up test environment
	os.Setenv("DD_ENV", "test")
	os.Setenv("DD_VERSION", "1.0.0")
	defer func() {
		os.Unsetenv("DD_ENV")
		os.Unsetenv("DD_VERSION")
	}()

	// Test configuration
	logCfg := config.LoggingConfig{
		Level:       "info",
		Format:      "json",
		Output:      []string{"stdout"},
		Development: false,
		Datadog: config.DatadogLogConfig{
			Enabled:   true,
			InjectAPM: true,
			Service:   "test-service",
			Env:       "test",
			Version:   "1.0.0",
		},
	}

	appCfg := config.AppConfig{
		Name:    "test-app",
		Version: "1.0.0",
	}

	// Create logger
	logger := NewDatadogLogger(logCfg, appCfg)
	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	// Test that logger has the expected fields
	// We can't easily test the internal structure, but we can test that it logs correctly
	logger.Info("Test message",
		zap.String("test_field", "test_value"))

	// Test error logging
	err := &testError{message: "test error"}
	logger.LogError(err, "Test error message",
		zap.String("additional_field", "value"))
}

func TestDatadogLogger_WithAPMContext(t *testing.T) {
	// Create a mock tracer
	mt := mocktracer.Start()
	defer mt.Stop()

	// Create a span
	span := tracer.StartSpan("test.operation")
	ctx := tracer.ContextWithSpan(context.Background(), span)

	// Create logger
	logCfg := config.LoggingConfig{
		Level:       "info",
		Format:      "json",
		Output:      []string{"stdout"},
		Development: false,
		Datadog: config.DatadogLogConfig{
			Enabled:   true,
			InjectAPM: true,
		},
	}

	appCfg := config.AppConfig{
		Name: "test-app",
	}

	logger := NewDatadogLogger(logCfg, appCfg)

	// Test APM context
	apmLogger := logger.WithAPMContext(ctx)
	if apmLogger == nil {
		t.Fatal("Expected APM logger to be created")
	}

	// Log with APM context
	apmLogger.Info("Test message with APM context")

	// Finish span
	span.Finish()

	// Verify span was created
	spans := mt.FinishedSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}
}

func TestDatadogLogger_WithAPMFields(t *testing.T) {
	logCfg := config.LoggingConfig{
		Level:       "info",
		Format:      "json",
		Output:      []string{"stdout"},
		Development: false,
		Datadog: config.DatadogLogConfig{
			Enabled:   true,
			InjectAPM: true,
		},
	}

	appCfg := config.AppConfig{
		Name: "test-app",
	}

	logger := NewDatadogLogger(logCfg, appCfg)

	// Test APM fields
	apmLogger := logger.WithAPMFields(12345, 67890)
	if apmLogger == nil {
		t.Fatal("Expected APM logger to be created")
	}

	// Log with APM fields
	apmLogger.Info("Test message with APM fields")
}

func TestDatadogLogger_LogError(t *testing.T) {
	logCfg := config.LoggingConfig{
		Level:       "info",
		Format:      "json",
		Output:      []string{"stdout"},
		Development: false,
		Datadog: config.DatadogLogConfig{
			Enabled:   true,
			InjectAPM: true,
		},
	}

	appCfg := config.AppConfig{
		Name: "test-app",
	}

	logger := NewDatadogLogger(logCfg, appCfg)

	// Test error logging
	err := &testError{message: "test error message"}
	logger.LogError(err, "Test error occurred",
		zap.String("context", "test"))

	// Test nil error
	logger.LogError(nil, "Test with nil error")
}

func TestGetZapLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"critical", zapcore.FatalLevel},
		{"alert", zapcore.FatalLevel},
		{"emergency", zapcore.FatalLevel},
		{"unknown", zapcore.InfoLevel},
		{"", zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getZapLevel(tt.input)
			if result != tt.expected {
				t.Errorf("getZapLevel(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetOutputPaths(t *testing.T) {
	tests := []struct {
		outputs  []string
		filePath string
		expected []string
	}{
		{[]string{"stdout"}, "", []string{"stdout"}},
		{[]string{"file"}, "test.log", []string{"test.log"}},
		{[]string{"stdout", "file"}, "test.log", []string{"stdout", "test.log"}},
		{[]string{}, "", []string{"stdout"}},
		{[]string{"file"}, "", []string{"stdout"}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getOutputPaths(tt.outputs, tt.filePath)
			if len(result) != len(tt.expected) {
				t.Errorf("getOutputPaths(%v, %s) = %v, want %v", tt.outputs, tt.filePath, result, tt.expected)
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("getOutputPaths(%v, %s) = %v, want %v", tt.outputs, tt.filePath, result, tt.expected)
					break
				}
			}
		})
	}
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{nil, ""},
		{&testError{message: "test error"}, "testError"},
		{&testError{message: "another error"}, "testError"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getErrorType(tt.err)
			if result != tt.expected {
				t.Errorf("getErrorType(%v) = %s, want %s", tt.err, result, tt.expected)
			}
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
