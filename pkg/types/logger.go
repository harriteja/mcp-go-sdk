package types

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

// Logger defines the interface for structured logging
//
// Example usage:
//
//	logger.Info("User logged in",
//	    LogField{Key: "user_id", Value: "123"},
//	    LogField{Key: "ip", Value: "192.168.1.1"})
type Logger interface {
	// Debug logs a debug message with optional structured fields
	Debug(msg string, fields ...LogField)
	// Info logs an informational message with optional structured fields
	Info(msg string, fields ...LogField)
	// Warn logs a warning message with optional structured fields
	Warn(msg string, fields ...LogField)
	// Error logs an error message with optional structured fields
	Error(msg string, fields ...LogField)
	// With returns a new logger with the given fields added to all messages
	With(fields ...LogField) Logger
	// WithSampling returns a new logger with sampling enabled
	WithSampling(rate float64) Logger
	// Flush ensures all logs are written before shutdown
	Flush() error
}

// LoggerFactory creates and configures logger instances
type LoggerFactory interface {
	// CreateLogger creates a new logger instance with the given name
	CreateLogger(name string) Logger
	// CreateLoggerWithConfig creates a new logger with specific configuration
	CreateLoggerWithConfig(name string, config LoggerConfig) Logger
	// WithFields returns a new logger factory with default fields
	WithFields(fields ...LogField) LoggerFactory
}

// NoOpLogger implements Logger with no-op operations
type NoOpLogger struct{}

func (l *NoOpLogger) Debug(msg string, fields ...LogField) {}
func (l *NoOpLogger) Info(msg string, fields ...LogField)  {}
func (l *NoOpLogger) Warn(msg string, fields ...LogField)  {}
func (l *NoOpLogger) Error(msg string, fields ...LogField) {}
func (l *NoOpLogger) With(fields ...LogField) Logger       { return l }
func (l *NoOpLogger) WithSampling(rate float64) Logger     { return l }
func (l *NoOpLogger) Flush() error                         { return nil }

// NewNoOpLogger creates a new no-op logger
func NewNoOpLogger() Logger {
	return &NoOpLogger{}
}
