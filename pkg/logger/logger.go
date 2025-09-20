package logger

import (
	"go.uber.org/zap"
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
