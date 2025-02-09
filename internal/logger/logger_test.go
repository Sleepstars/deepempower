package logger

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	defaultLogger = &Logger{
		level:     INFO,
		logger:    log.New(&buf, "", log.LstdFlags|log.Lmicroseconds),
		component: "test",
	}

	tests := []struct {
		name     string
		level    LogLevel
		logFunc  func(format string, args ...interface{})
		message  string
		wantLog  bool
		contains string
	}{
		{
			name:     "Debug message below INFO level",
			level:    INFO,
			logFunc:  defaultLogger.Debug,
			message:  "debug message",
			wantLog:  false,
			contains: "[DEBUG]",
		},
		{
			name:     "Info message at INFO level",
			level:    INFO,
			logFunc:  defaultLogger.Info,
			message:  "info message",
			wantLog:  true,
			contains: "[INFO]",
		},
		{
			name:     "Warning message above INFO level",
			level:    INFO,
			logFunc:  defaultLogger.Warn,
			message:  "warning message",
			wantLog:  true,
			contains: "[WARN]",
		},
		{
			name:     "Error message above INFO level",
			level:    INFO,
			logFunc:  defaultLogger.Error,
			message:  "error message",
			wantLog:  true,
			contains: "[ERROR]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			defaultLogger.SetLevel(tt.level)
			tt.logFunc(tt.message)

			output := buf.String()
			if tt.wantLog {
				assert.True(t, strings.Contains(output, tt.contains), "log should contain level marker")
				assert.True(t, strings.Contains(output, tt.message), "log should contain message")
				assert.True(t, strings.Contains(output, "[test]"), "log should contain component")
			} else {
				assert.Empty(t, output, "log should be empty")
			}
		})
	}
}

func TestLoggerWithComponent(t *testing.T) {
	logger := GetLogger().WithComponent("test-component")
	assert.Equal(t, "test-component", logger.component)
}

func TestLoggerWithError(t *testing.T) {
	err := assert.AnError
	logger := GetLogger().WithError(err)
	assert.True(t, strings.Contains(logger.component, err.Error()))
}

func TestLogLevelNames(t *testing.T) {
	assert.Equal(t, "DEBUG", levelNames[DEBUG])
	assert.Equal(t, "INFO", levelNames[INFO])
	assert.Equal(t, "WARN", levelNames[WARN])
	assert.Equal(t, "ERROR", levelNames[ERROR])
	assert.Equal(t, "FATAL", levelNames[FATAL])
}

func TestInitLoggerSingleton(t *testing.T) {
	// Reset the singleton
	defaultLogger = nil

	// Initialize multiple times
	for i := 0; i < 3; i++ {
		InitLogger(DEBUG, "test")
	}

	// Verify singleton behavior
	logger1 := GetLogger()
	logger2 := GetLogger()
	assert.Same(t, logger1, logger2, "GetLogger should return the same instance")
	assert.Equal(t, DEBUG, logger1.level)
	assert.Equal(t, "test", logger1.component)
}
