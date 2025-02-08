package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	// DEBUG level for detailed debugging information
	DEBUG LogLevel = iota
	// INFO level for general operational information
	INFO
	// WARN level for warning messages
	WARN
	// ERROR level for error messages
	ERROR
	// FATAL level for fatal errors that require immediate attention
	FATAL
)

var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// Logger represents our custom logger with levels
type Logger struct {
	level     LogLevel
	logger    *log.Logger
	mu        sync.Mutex
	component string
}

var (
	defaultLogger *Logger
	once         sync.Once
)

// InitLogger initializes the default logger
func InitLogger(level LogLevel, component string) {
	once.Do(func() {
		defaultLogger = &Logger{
			level:     level,
			logger:    log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
			component: component,
		}
	})
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		InitLogger(INFO, "default")
	}
	return defaultLogger
}

// WithComponent creates a new logger with the specified component name
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		level:     l.level,
		logger:    l.logger,
		component: component,
	}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// log performs the actual logging
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[%s][%s] %s", levelNames[level], l.component, msg)

	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs debug level messages
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs info level messages
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs warning level messages
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs error level messages
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal logs fatal level messages and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
}

// WithError creates an error message with stack trace
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		level:     l.level,
		logger:    l.logger,
		component: fmt.Sprintf("%s: %v", l.component, err),
	}
}
