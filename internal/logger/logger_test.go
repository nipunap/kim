package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {
	// Test creating logger without debug
	logger := New()
	if logger == nil {
		t.Fatal("Logger should not be nil")
	}

	if logger.SugaredLogger == nil {
		t.Fatal("Internal zap logger should not be nil")
	}

	// Test creating logger with debug enabled via environment variable
	oldDebug := os.Getenv("KIM_DEBUG")
	os.Setenv("KIM_DEBUG", "true")
	defer os.Setenv("KIM_DEBUG", oldDebug)

	debugLogger := New()
	if debugLogger == nil {
		t.Fatal("Debug logger should not be nil")
	}

	if debugLogger.SugaredLogger == nil {
		t.Fatal("Internal debug zap logger should not be nil")
	}
}

func TestLoggerMethods(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a custom logger that writes to our buffer
	config := zap.NewDevelopmentConfig()
	config.OutputPaths = []string{"stdout"}

	// Create encoder config for consistent output
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create core that writes to buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buf),
		zapcore.DebugLevel,
	)

	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test Info logging
	buf.Reset()
	logger.Info("test info message", "key", "value")
	output := buf.String()
	if !strings.Contains(output, "test info message") {
		t.Error("Info message not found in output")
	}
	if !strings.Contains(output, "info") {
		t.Error("Info level not found in output")
	}

	// Test Debug logging
	buf.Reset()
	logger.Debug("test debug message", "debug_key", "debug_value")
	output = buf.String()
	if !strings.Contains(output, "test debug message") {
		t.Error("Debug message not found in output")
	}

	// Test Warn logging
	buf.Reset()
	logger.Warn("test warn message", "warn_key", "warn_value")
	output = buf.String()
	if !strings.Contains(output, "test warn message") {
		t.Error("Warn message not found in output")
	}
	if !strings.Contains(output, "warn") {
		t.Error("Warn level not found in output")
	}

	// Test Error logging
	buf.Reset()
	logger.Error("test error message", "error_key", "error_value")
	output = buf.String()
	if !strings.Contains(output, "test error message") {
		t.Error("Error message not found in output")
	}
	if !strings.Contains(output, "error") {
		t.Error("Error level not found in output")
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buf),
		zapcore.DebugLevel,
	)

	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test logging with multiple key-value pairs
	buf.Reset()
	logger.Info("test message with fields",
		"string_field", "string_value",
		"int_field", 42,
		"bool_field", true,
		"float_field", 3.14)

	output := buf.String()
	if !strings.Contains(output, "string_value") {
		t.Error("String field not found in output")
	}
	if !strings.Contains(output, "42") {
		t.Error("Int field not found in output")
	}
	if !strings.Contains(output, "true") {
		t.Error("Bool field not found in output")
	}
	if !strings.Contains(output, "3.14") {
		t.Error("Float field not found in output")
	}
}

func TestLoggerDebugMode(t *testing.T) {
	// Test that debug mode affects log level
	var debugBuf, infoBuf bytes.Buffer

	// Create debug logger
	debugCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			MessageKey:  "msg",
			LevelKey:    "level",
			EncodeLevel: zapcore.LowercaseLevelEncoder,
		}),
		zapcore.AddSync(&debugBuf),
		zapcore.DebugLevel,
	)
	debugLogger := &Logger{SugaredLogger: zap.New(debugCore).Sugar()}

	// Create info logger (non-debug)
	infoCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			MessageKey:  "msg",
			LevelKey:    "level",
			EncodeLevel: zapcore.LowercaseLevelEncoder,
		}),
		zapcore.AddSync(&infoBuf),
		zapcore.InfoLevel,
	)
	infoLogger := &Logger{SugaredLogger: zap.New(infoCore).Sugar()}

	// Test debug message
	debugLogger.Debug("debug message")
	infoLogger.Debug("debug message")

	debugOutput := debugBuf.String()
	infoOutput := infoBuf.String()

	// Debug logger should capture debug messages
	if !strings.Contains(debugOutput, "debug message") {
		t.Error("Debug logger should capture debug messages")
	}

	// Info logger should not capture debug messages
	if strings.Contains(infoOutput, "debug message") {
		t.Error("Info logger should not capture debug messages")
	}

	// Both should capture info messages
	debugBuf.Reset()
	infoBuf.Reset()

	debugLogger.Info("info message")
	infoLogger.Info("info message")

	debugOutput = debugBuf.String()
	infoOutput = infoBuf.String()

	if !strings.Contains(debugOutput, "info message") {
		t.Error("Debug logger should capture info messages")
	}
	if !strings.Contains(infoOutput, "info message") {
		t.Error("Info logger should capture info messages")
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logger := New()

	// Test setting different levels
	logger.SetLevel("debug")
	logger.SetLevel("info")
	logger.SetLevel("warn")
	logger.SetLevel("error")
	logger.SetLevel("invalid") // Should default to info

	// If we get here without panicking, the test passes
	// The actual level verification would require more complex setup
}

func TestLoggerErrorHandling(t *testing.T) {
	logger := New()

	// Test that logger methods don't panic with nil values
	logger.Info("test with nil", "nil_key", nil)
	logger.Debug("test with nil", "nil_key", nil)
	logger.Warn("test with nil", "nil_key", nil)
	logger.Error("test with nil", "nil_key", nil)

	// Test with empty messages
	logger.Info("")
	logger.Debug("")
	logger.Warn("")
	logger.Error("")

	// Test with odd number of key-value pairs (should handle gracefully)
	logger.Info("test odd pairs", "key1", "value1", "key2")
}

func TestLoggerConcurrency(t *testing.T) {
	logger := New()

	// Test concurrent logging
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info("concurrent log", "goroutine", id)
			logger.Debug("concurrent debug", "goroutine", id)
			logger.Warn("concurrent warn", "goroutine", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we get here without panicking, the test passes
}

func TestLoggerEnvironmentVariable(t *testing.T) {
	// Test that KIM_DEBUG environment variable affects logger creation
	oldDebug := os.Getenv("KIM_DEBUG")
	defer os.Setenv("KIM_DEBUG", oldDebug)

	// Test with debug disabled
	os.Setenv("KIM_DEBUG", "false")
	logger1 := New()
	if logger1 == nil {
		t.Error("Logger should be created even with debug disabled")
	}

	// Test with debug enabled
	os.Setenv("KIM_DEBUG", "true")
	logger2 := New()
	if logger2 == nil {
		t.Error("Logger should be created with debug enabled")
	}

	// Test with invalid debug value
	os.Setenv("KIM_DEBUG", "invalid")
	logger3 := New()
	if logger3 == nil {
		t.Error("Logger should be created with invalid debug value")
	}
}
