// Package logger provides logging utilities
package logger

// This file is kept for backward compatibility but doesn't use zap anymore
// It provides adapter functions for older code that expects zap-related functionality

import (
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// ZapLoggerShim is a minimal interface that mimics zap.Logger for compatibility
type ZapLoggerShim struct{}

// Info is a compatibility method
func (z *ZapLoggerShim) Info(msg string, fields ...interface{}) {}

// Debug is a compatibility method
func (z *ZapLoggerShim) Debug(msg string, fields ...interface{}) {}

// Warn is a compatibility method
func (z *ZapLoggerShim) Warn(msg string, fields ...interface{}) {}

// Error is a compatibility method
func (z *ZapLoggerShim) Error(msg string, fields ...interface{}) {}

// Fatal is a compatibility method
func (z *ZapLoggerShim) Fatal(msg string, fields ...interface{}) {}

// Panic is a compatibility method
func (z *ZapLoggerShim) Panic(msg string, fields ...interface{}) {}

// Named is a compatibility method
func (z *ZapLoggerShim) Named(name string) *ZapLoggerShim {
	return &ZapLoggerShim{}
}

// With is a compatibility method
func (z *ZapLoggerShim) With(fields ...interface{}) *ZapLoggerShim {
	return &ZapLoggerShim{}
}

// Sync is a compatibility method
func (z *ZapLoggerShim) Sync() error {
	return nil
}

// LegacyToNew adapts a ZapLoggerShim to the new Logger interface
func LegacyToNew(zapLogger *ZapLoggerShim) types.Logger {
	return GetDefaultLogger()
}

// NewToZap converts our new Logger interface to a ZapLoggerShim
func NewToZap(logger types.Logger) *ZapLoggerShim {
	return &ZapLoggerShim{}
}
