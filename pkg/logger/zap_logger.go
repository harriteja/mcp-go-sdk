//go:build zap_logger
// +build zap_logger

// Package logger provides logging utilities
package logger

import (
	"context"
	"os"
	"strconv"

	"go.uber.org/zap"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// ZapLogger is an implementation of the types.Logger interface using zap
type ZapLogger struct {
	logger    *zap.Logger
	verbosity int
	bucket    string
}

// NewZapLogger creates a new ZapLogger
func NewZapLogger(zapLogger *zap.Logger) *ZapLogger {
	// Get verbosity from environment variable
	verbosityStr := os.Getenv("LOGGER_LEVEL")
	verbosity := 0
	if v, err := strconv.Atoi(verbosityStr); err == nil {
		verbosity = v
	}

	return &ZapLogger{
		logger:    zapLogger,
		verbosity: verbosity,
		bucket:    "root",
	}
}

// NewProductionLogger creates a new production-configured ZapLogger
func NewProductionLogger() (*ZapLogger, error) {
	config := zap.NewProductionConfig()
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	return NewZapLogger(logger), nil
}

// NewDevelopmentLogger creates a new development-configured ZapLogger
func NewDevelopmentLogger() (*ZapLogger, error) {
	config := zap.NewDevelopmentConfig()
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	return NewZapLogger(logger), nil
}

// Access implements types.Logger.Access
func (l *ZapLogger) Access(ctx context.Context, message string) {
	l.logger.Info("ACCESS: "+message,
		zap.String("type", "access"),
	)
}

// Info implements types.Logger.Info
func (l *ZapLogger) Info(ctx context.Context, bucket, handler, message string) {
	l.logger.Info(message,
		zap.String("bucket", bucket),
		zap.String("handler", handler),
	)
}

// Warn implements types.Logger.Warn
func (l *ZapLogger) Warn(ctx context.Context, bucket, handler, message string) {
	l.logger.Warn(message,
		zap.String("bucket", bucket),
		zap.String("handler", handler),
	)
}

// Error implements types.Logger.Error
func (l *ZapLogger) Error(ctx context.Context, bucket, handler, message string) {
	l.logger.Error(message,
		zap.String("bucket", bucket),
		zap.String("handler", handler),
	)
}

// Panic implements types.Logger.Panic
func (l *ZapLogger) Panic(ctx context.Context, bucket, handler, message string) {
	l.logger.Panic(message,
		zap.String("bucket", bucket),
		zap.String("handler", handler),
	)
}

// V implements types.Logger.V
func (l *ZapLogger) V(n int) bool {
	return l.verbosity >= n
}

// Sub implements types.Logger.Sub
func (l *ZapLogger) Sub(name string) types.Logger {
	return &ZapLogger{
		logger:    l.logger.Named(name),
		verbosity: l.verbosity,
		bucket:    l.bucket + "." + name,
	}
}

// SubWithIncrement implements types.Logger.SubWithIncrement
func (l *ZapLogger) SubWithIncrement(name string, n int) types.Logger {
	return &ZapLogger{
		logger:    l.logger.Named(name),
		verbosity: l.verbosity + n,
		bucket:    l.bucket + "." + name,
	}
}

// Sync flushes any buffered log entries
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

// FromZap converts a zap.Logger to a types.Logger
func FromZap(logger *zap.Logger) types.Logger {
	return NewZapLogger(logger)
}

// ToZap extracts the underlying zap.Logger from a ZapLogger
// Returns nil if the logger is not a ZapLogger
func ToZap(logger types.Logger) *zap.Logger {
	if zapLogger, ok := logger.(*ZapLogger); ok {
		return zapLogger.logger
	}
	return nil
}
