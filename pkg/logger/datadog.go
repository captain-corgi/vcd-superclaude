package logger

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/captain-corgi/multi-agent-system/pkg/config"
)

// NewDatadogLogger creates a logger configured for Datadog standards
func NewDatadogLogger(logCfg config.LoggingConfig, appCfg config.AppConfig) *Logger {
	// Create Datadog-compatible encoder config
	encoderConfig := zap.NewProductionEncoderConfig()

	// Map Zap keys to Datadog standard field names
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.LevelKey = "status"
	encoderConfig.MessageKey = "message"
	encoderConfig.NameKey = "logger"
	encoderConfig.CallerKey = "logger.source"
	encoderConfig.StacktraceKey = "error.stack"

	// Use lowercase level names (debug, info, warn, error, critical, alert, emergency)
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	// Use RFC3339 timestamp format
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	// Create the base config
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(getZapLevel(logCfg.Level)),
		Development:      logCfg.Development,
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      getOutputPaths(logCfg.Output, logCfg.File),
		ErrorOutputPaths: getErrorOutputPaths(logCfg.Output, logCfg.File),
	}

	// Build the logger
	zapLogger, err := config.Build()
	if err != nil {
		panic(err)
	}

	// Wrap with Datadog field injector
	datadogLogger := &Logger{zapLogger}
	datadogLogger = datadogLogger.withDatadogFields(logCfg, appCfg)

	return datadogLogger
}

// withDatadogFields adds constant Datadog fields to the logger
func (l *Logger) withDatadogFields(logCfg config.LoggingConfig, appCfg config.AppConfig) *Logger {
	// Resolve service name
	service := logCfg.Datadog.Service
	if service == "" {
		service = appCfg.Name
	}

	// Resolve environment
	env := logCfg.Datadog.Env
	if env == "" {
		env = os.Getenv("DD_ENV")
		if env == "" {
			env = "development"
		}
	}

	// Resolve version
	version := logCfg.Datadog.Version
	if version == "" {
		version = os.Getenv("DD_VERSION")
		if version == "" {
			version = appCfg.Version
		}
	}

	// Get hostname
	hostname, _ := os.Hostname()

	// Create a new logger with constant fields
	fields := []zap.Field{
		zap.String("service", service),
		zap.String("env", env),
		zap.String("version", version),
		zap.String("host", hostname),
	}

	// Create a new logger with these fields always present
	newLogger := l.With(fields...)

	return &Logger{newLogger}
}

// getZapLevel converts string level to zap level
func getZapLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "critical":
		return zapcore.FatalLevel
	case "alert":
		return zapcore.FatalLevel
	case "emergency":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// getOutputPaths returns output paths based on config
func getOutputPaths(outputs []string, filePath string) []string {
	paths := []string{}
	for _, output := range outputs {
		switch output {
		case "stdout":
			paths = append(paths, "stdout")
		case "file":
			if filePath != "" {
				paths = append(paths, filePath)
			}
		}
	}

	// Default to stdout if no outputs specified
	if len(paths) == 0 {
		paths = []string{"stdout"}
	}

	return paths
}

// getErrorOutputPaths returns error output paths based on config
func getErrorOutputPaths(outputs []string, filePath string) []string {
	paths := []string{}
	for _, output := range outputs {
		switch output {
		case "stdout":
			paths = append(paths, "stderr")
		case "file":
			if filePath != "" {
				paths = append(paths, filePath)
			}
		}
	}

	// Default to stderr if no outputs specified
	if len(paths) == 0 {
		paths = []string{"stderr"}
	}

	return paths
}

// LogError logs an error with Datadog error fields
func (l *Logger) LogError(err error, message string, fields ...zap.Field) {
	if err == nil {
		// If no error, just log the message with provided fields
		l.Error(message, fields...)
		return
	}

	errorFields := []zap.Field{
		zap.String("error.kind", getErrorType(err)),
		zap.String("error.message", err.Error()),
	}

	// Add stack trace if available
	if stack := getErrorStack(err); stack != "" {
		errorFields = append(errorFields, zap.String("error.stack", stack))
	}

	// Combine with provided fields
	allFields := append(errorFields, fields...)

	l.Error(message, allFields...)
}

// getErrorType extracts error type name
func getErrorType(err error) string {
	if err == nil {
		return ""
	}

	// Get the type name from the error
	errorType := fmt.Sprintf("%T", err)
	// Remove package path and keep only type name
	if idx := strings.LastIndex(errorType, "."); idx >= 0 {
		errorType = errorType[idx+1:]
	}

	return errorType
}

// getErrorStack extracts stack trace if available
func getErrorStack(err error) string {
	// This is a simplified version - in practice, you might want to use
	// runtime.Stack() or a more sophisticated stack trace extraction
	if err == nil {
		return ""
	}

	// For now, return empty string - Zap will handle stack traces automatically
	// when configured with StacktraceKey
	return ""
}

// WithAPMContext creates a logger with APM correlation fields from context
func (l *Logger) WithAPMContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	// Extract trace and span IDs from context
	span, ok := tracer.SpanFromContext(ctx)
	if !ok {
		return l
	}

	// Get trace and span IDs
	traceID := span.Context().TraceID()
	spanID := span.Context().SpanID()

	// Add APM correlation fields
	fields := []zap.Field{
		zap.Uint64("dd.trace_id", traceID),
		zap.Uint64("dd.span_id", spanID),
	}

	return &Logger{l.With(fields...)}
}

// WithAPMFields creates a logger with APM correlation fields directly
func (l *Logger) WithAPMFields(traceID, spanID uint64) *Logger {
	fields := []zap.Field{
		zap.Uint64("dd.trace_id", traceID),
		zap.Uint64("dd.span_id", spanID),
	}

	return &Logger{l.With(fields...)}
}
