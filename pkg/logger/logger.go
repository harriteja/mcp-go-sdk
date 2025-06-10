package logger

import (
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// globalLoggerFactory is the global logger factory
var defaultLogger types.Logger = NewStdLogger()

// SetDefaultLogger sets the default logger implementation
func SetDefaultLogger(logger types.Logger) {
	if logger != nil {
		defaultLogger = logger
	}
}

// GetDefaultLogger returns the default logger
func GetDefaultLogger() types.Logger {
	return defaultLogger
}

// New creates a new logger with the given name
func New(name string) types.Logger {
	return defaultLogger.Sub(name)
}

// NewWithVerbosity creates a new logger with the given name and verbosity
func NewWithVerbosity(name string, verbosity int) types.Logger {
	return defaultLogger.SubWithIncrement(name, verbosity)
}
