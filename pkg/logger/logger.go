package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
}

func (l *Logger) Zap() *zap.Logger {
	return l.Logger
}

func New(debug bool) *Logger {
	if debug {
		config := zap.NewProductionConfig()
		config.Development = debug
		config.OutputPaths = []string{"stdout"}
		config.ErrorOutputPaths = []string{"stderr"}

		logger, err := config.Build()
		if err != nil {
			panic(err)
		}

		return &Logger{logger}
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	return &Logger{logger}
}

func NewWithConfig(config zap.Config) *Logger {
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return &Logger{logger}
}

func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

func (l *Logger) Close() error {
	return l.Sync()
}

// NewFromConfig creates a logger from logging configuration
func NewFromConfig(level, format string, outputs []string, filePath string, development bool) *Logger {
	config := zap.NewProductionConfig()

	// Set level
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	// Set development mode
	config.Development = development

	// Set format
	if format == "console" {
		config.Encoding = "console"
	} else {
		config.Encoding = "json"
	}

	// Set outputs
	outputPaths := []string{}
	errorOutputPaths := []string{}

	for _, output := range outputs {
		switch output {
		case "stdout":
			outputPaths = append(outputPaths, "stdout")
			errorOutputPaths = append(errorOutputPaths, "stderr")
		case "file":
			if filePath != "" {
				outputPaths = append(outputPaths, filePath)
				errorOutputPaths = append(errorOutputPaths, filePath)
			}
		}
	}

	// Default to stdout if no outputs specified
	if len(outputPaths) == 0 {
		outputPaths = []string{"stdout"}
		errorOutputPaths = []string{"stderr"}
	}

	config.OutputPaths = outputPaths
	config.ErrorOutputPaths = errorOutputPaths

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return &Logger{logger}
}
