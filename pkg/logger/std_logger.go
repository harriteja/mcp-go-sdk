package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// StdLogger is an implementation of the types.Logger interface using the standard log package
type StdLogger struct {
	logger    *log.Logger
	verbosity int
	bucket    string
}

// NewStdLogger creates a new StdLogger
func NewStdLogger() *StdLogger {
	// Get verbosity from environment variable
	verbosityStr := os.Getenv("LOGGER_LEVEL")
	verbosity := 0
	if v, err := strconv.Atoi(verbosityStr); err == nil {
		verbosity = v
	}

	return &StdLogger{
		logger:    log.New(os.Stderr, "", log.LstdFlags),
		verbosity: verbosity,
		bucket:    "root",
	}
}

// Access implements types.Logger.Access
func (l *StdLogger) Access(ctx context.Context, message string) {
	l.logger.Printf("ACCESS: %s", message)
}

// Info implements types.Logger.Info
func (l *StdLogger) Info(ctx context.Context, bucket, handler, message string) {
	l.logger.Printf("INFO [%s/%s]: %s", bucket, handler, message)
}

// Warn implements types.Logger.Warn
func (l *StdLogger) Warn(ctx context.Context, bucket, handler, message string) {
	l.logger.Printf("WARN [%s/%s]: %s", bucket, handler, message)
}

// Error implements types.Logger.Error
func (l *StdLogger) Error(ctx context.Context, bucket, handler, message string) {
	l.logger.Printf("ERROR [%s/%s]: %s", bucket, handler, message)
}

// Panic implements types.Logger.Panic
func (l *StdLogger) Panic(ctx context.Context, bucket, handler, message string) {
	l.logger.Panicf("PANIC [%s/%s]: %s", bucket, handler, message)
}

// V implements types.Logger.V
func (l *StdLogger) V(n int) bool {
	return l.verbosity >= n
}

// Sub implements types.Logger.Sub
func (l *StdLogger) Sub(name string) types.Logger {
	return &StdLogger{
		logger:    l.logger,
		verbosity: l.verbosity,
		bucket:    fmt.Sprintf("%s.%s", l.bucket, name),
	}
}

// SubWithIncrement implements types.Logger.SubWithIncrement
func (l *StdLogger) SubWithIncrement(name string, n int) types.Logger {
	return &StdLogger{
		logger:    l.logger,
		verbosity: l.verbosity + n,
		bucket:    fmt.Sprintf("%s.%s", l.bucket, name),
	}
}

// Sync flushes any buffered log entries
func (l *StdLogger) Sync() error {
	return nil // standard logger doesn't buffer
}

// NewDefaultLogger creates a new default logger (using standard library)
func NewDefaultLogger() types.Logger {
	return NewStdLogger()
}
