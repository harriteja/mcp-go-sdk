package logger

import (
	"context"
	"io"
	"log"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// NopLogger is a no-op implementation of the types.Logger interface
// that discards all logs. It's useful for testing.
type NopLogger struct {
	logger *log.Logger
}

// NewNopLogger creates a new NopLogger
func NewNopLogger() *NopLogger {
	return &NopLogger{
		logger: log.New(io.Discard, "", 0),
	}
}

// Access implements types.Logger.Access
func (l *NopLogger) Access(ctx context.Context, message string) {}

// Info implements types.Logger.Info
func (l *NopLogger) Info(ctx context.Context, bucket, handler, message string) {}

// Warn implements types.Logger.Warn
func (l *NopLogger) Warn(ctx context.Context, bucket, handler, message string) {}

// Error implements types.Logger.Error
func (l *NopLogger) Error(ctx context.Context, bucket, handler, message string) {}

// Panic implements types.Logger.Panic
func (l *NopLogger) Panic(ctx context.Context, bucket, handler, message string) {}

// V implements types.Logger.V
func (l *NopLogger) V(n int) bool {
	return false
}

// Sub implements types.Logger.Sub
func (l *NopLogger) Sub(name string) types.Logger {
	return l
}

// SubWithIncrement implements types.Logger.SubWithIncrement
func (l *NopLogger) SubWithIncrement(name string, n int) types.Logger {
	return l
}

// Sync flushes any buffered log entries
func (l *NopLogger) Sync() error {
	return nil
}
