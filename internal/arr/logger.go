package arr

import (
	"fmt"
	"log"
	"strings"
)

// LogLevel represents different log levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// StandardLogger implements the Logger interface using Go's standard log package
type StandardLogger struct {
	level  LogLevel
	logger *log.Logger
}

// NewStandardLogger creates a new StandardLogger
func NewStandardLogger(levelStr string) Logger {
	level := parseLogLevel(levelStr)
	return &StandardLogger{
		level:  level,
		logger: log.Default(),
	}
}

// Debug logs a debug message
func (l *StandardLogger) Debug(msg string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		l.log("DEBUG", msg, args...)
	}
}

// Info logs an info message
func (l *StandardLogger) Info(msg string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		l.log("INFO", msg, args...)
	}
}

// Warn logs a warning message
func (l *StandardLogger) Warn(msg string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		l.log("WARN", msg, args...)
	}
}

// Error logs an error message
func (l *StandardLogger) Error(msg string, args ...interface{}) {
	if l.level <= LogLevelError {
		l.log("ERROR", msg, args...)
	}
}

// log is the internal logging method
func (l *StandardLogger) log(level, msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.logger.Printf("[%s] %s", level, msg)
}

// parseLogLevel parses a log level string into LogLevel
func parseLogLevel(levelStr string) LogLevel {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN", "WARNING":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}
