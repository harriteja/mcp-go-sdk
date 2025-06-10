package types

import (
	"context"
)

// LogLevel represents the severity level of a log message
type LogLevel string

const (
	// LogLevelDebug represents debug level logs for detailed debugging information
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo represents informational logs for general operational events
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn represents warning logs for potentially harmful situations
	LogLevelWarn LogLevel = "warn"
	// LogLevelError represents error logs for errors that need attention
	LogLevelError LogLevel = "error"
)

// LogField represents a structured log field with a key-value pair
type LogField struct {
	Key   string
	Value interface{}
}

// LoggerConfig represents configuration options for a logger
type LoggerConfig struct {
	// MinLevel is the minimum log level to output
	MinLevel LogLevel
	// EnableSampling enables log sampling to reduce volume
	EnableSampling bool
	// SampleRate is the fraction of logs to keep when sampling (0.0-1.0)
	SampleRate float64
	// DefaultFields are fields added to all log messages
	DefaultFields []LogField
}

// Logger provides level based logging.
type Logger interface {
	// Access logs all access logs
	Access(ctx context.Context, message string)

	// Info logs all INFO level log messages.
	Info(ctx context.Context, bucket, handler, message string)

	// Warn logs all WARN level log messages.
	Warn(ctx context.Context, bucket, handler, message string)

	// Error logs all ERROR level log messages.
	Error(ctx context.Context, bucket, handler, message string)

	// Panic logs all PANIC level log messages.
	// It raises a panic after logging.
	Panic(ctx context.Context, bucket, handler, message string)

	// V returns true if the verbosity is greater than or equal to n.
	// Default verbosity is 0.
	//
	// Verbosity's set by `LOGGER_LEVEL` environment.
	// To increase or decrease verbosity you can use `SubWithIncrement`
	// which creates a new logger with modified verbosity.
	V(n int) bool

	// Sub returns a sub logger with `name` appended to current logger's bucket.
	//
	// Both Sub is lightweight as it's a clone of current logger with prefix modification.
	// It doesn't create any new file descriptor
	Sub(name string) Logger

	// SubWithIncrement returns a sub logger with increased verbosity by n.
	//
	// SubWithIncrement is lightweight as it's a clone of current logger with modified prefix & verbosity.
	// It doesn't create any new file descriptor
	SubWithIncrement(name string, n int) Logger
}

// LoggerFactory creates and configures logger instances
type LoggerFactory interface {
	// CreateLogger creates a new logger instance with the given name
	CreateLogger(name string) Logger
	// CreateLoggerWithConfig creates a new logger with specific configuration
	CreateLoggerWithConfig(name string, config LoggerConfig) Logger
}

// NoOpLogger implements Logger with no-op operations
type NoOpLogger struct {
	verbosity int
	bucket    string
}

func (l *NoOpLogger) Access(ctx context.Context, message string)                 {}
func (l *NoOpLogger) Info(ctx context.Context, bucket, handler, message string)  {}
func (l *NoOpLogger) Warn(ctx context.Context, bucket, handler, message string)  {}
func (l *NoOpLogger) Error(ctx context.Context, bucket, handler, message string) {}
func (l *NoOpLogger) Panic(ctx context.Context, bucket, handler, message string) { panic(message) }
func (l *NoOpLogger) V(n int) bool                                               { return l.verbosity >= n }

func (l *NoOpLogger) Sub(name string) Logger {
	return &NoOpLogger{
		verbosity: l.verbosity,
		bucket:    l.bucket + "." + name,
	}
}

func (l *NoOpLogger) SubWithIncrement(name string, n int) Logger {
	return &NoOpLogger{
		verbosity: l.verbosity + n,
		bucket:    l.bucket + "." + name,
	}
}

// NewNoOpLogger creates a new no-op logger
func NewNoOpLogger() Logger {
	return &NoOpLogger{
		verbosity: 0,
		bucket:    "root",
	}
}
