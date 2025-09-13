package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.SugaredLogger for structured logging
type Logger struct {
	*zap.SugaredLogger
}

// New creates a new logger instance
func New() *Logger {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Check for debug environment variable
	if os.Getenv("KIM_DEBUG") == "true" {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.Development = true
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return &Logger{
		SugaredLogger: logger.Sugar(),
	}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level string) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}

	// Note: This is a simplified approach. In production, you might want to
	// use zap.NewAtomicLevel() for dynamic level changes
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	logger, _ := config.Build()
	l.SugaredLogger = logger.Sugar()
}
