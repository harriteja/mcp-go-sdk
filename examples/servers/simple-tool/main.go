package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// CustomLogger is an example of implementing a custom logger
type CustomLogger struct {
	logger *log.Logger
	name   string
}

// NewCustomLogger creates a new custom logger
func NewCustomLogger(name string) *CustomLogger {
	return &CustomLogger{
		logger: log.New(os.Stdout, fmt.Sprintf("[%s] ", name), log.LstdFlags),
		name:   name,
	}
}

// Access implements types.Logger.Access
func (c *CustomLogger) Access(ctx context.Context, message string) {
	c.logger.Printf("[ACCESS] %s", message)
}

// Info implements types.Logger.Info
func (c *CustomLogger) Info(ctx context.Context, bucket, handler, message string) {
	c.logger.Printf("[INFO] [%s:%s] %s", bucket, handler, message)
}

// Warn implements types.Logger.Warn
func (c *CustomLogger) Warn(ctx context.Context, bucket, handler, message string) {
	c.logger.Printf("[WARNING] [%s:%s] %s", bucket, handler, message)
}

// Error implements types.Logger.Error
func (c *CustomLogger) Error(ctx context.Context, bucket, handler, message string) {
	c.logger.Printf("[ERROR] [%s:%s] %s", bucket, handler, message)
}

// Panic implements types.Logger.Panic
func (c *CustomLogger) Panic(ctx context.Context, bucket, handler, message string) {
	c.logger.Printf("[PANIC] [%s:%s] %s", bucket, handler, message)
	panic(message)
}

// V implements types.Logger.V
func (c *CustomLogger) V(n int) bool {
	// Simple verbosity implementation
	return n <= 1 // Only support verbosity level 0 and 1
}

// Sub implements types.Logger.Sub
func (c *CustomLogger) Sub(name string) types.Logger {
	return NewCustomLogger(c.name + "." + name)
}

// SubWithIncrement implements types.Logger.SubWithIncrement
func (c *CustomLogger) SubWithIncrement(name string, n int) types.Logger {
	// For this simple logger, we ignore the increment
	return c.Sub(name)
}

func main() {
	// Parse command line flags
	useCustomLogger := flag.Bool("custom", false, "Use custom logger instead of default")
	flag.Parse()

	// Create context
	ctx := context.Background()

	// Step 1: Get default logger
	defaultLogger := logger.GetDefaultLogger()
	defaultLogger.Info(ctx, "main", "startup", "Starting application with default logger")

	// Step 2: Create a named logger
	namedLogger := logger.New("my-tool")
	namedLogger.Info(ctx, "main", "startup", "Created a named logger")

	// Step 3: Create a sub-logger
	subLogger := namedLogger.Sub("component1")
	subLogger.Info(ctx, "init", "startup", "Created a sub-logger")

	// Step 4: Demonstrate different log levels
	defaultLogger.Info(ctx, "demo", "levels", "This is an info message")
	defaultLogger.Warn(ctx, "demo", "levels", "This is a warning message")
	defaultLogger.Error(ctx, "demo", "levels", "This is an error message")

	// Step 5: If the custom logger flag is set, replace the default logger
	if *useCustomLogger {
		// Create a custom logger
		customLogger := NewCustomLogger("custom")

		// Set it as the default logger
		logger.SetDefaultLogger(customLogger)

		// Get the default logger again (now it should be our custom logger)
		newDefaultLogger := logger.GetDefaultLogger()
		newDefaultLogger.Info(ctx, "main", "custom", "Switched to custom logger")

		// Log using different levels
		newDefaultLogger.Info(ctx, "demo", "levels", "This is an info message from custom logger")
		newDefaultLogger.Warn(ctx, "demo", "levels", "This is a warning message from custom logger")
		newDefaultLogger.Error(ctx, "demo", "levels", "This is an error message from custom logger")
	}

	// Step 6: Demonstrate verbosity
	for i := 0; i < 3; i++ {
		if defaultLogger.V(i) {
			defaultLogger.Info(ctx, "demo", "verbosity", fmt.Sprintf("This log should appear at verbosity level %d", i))
		} else {
			fmt.Printf("Skipping log at verbosity level %d\n", i)
		}
	}

	// Step 7: Simulate some work
	time.Sleep(500 * time.Millisecond)
	defaultLogger.Info(ctx, "main", "shutdown", "Application shutdown")
}
