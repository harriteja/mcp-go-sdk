package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	httputil "github.com/harriteja/mcp-go-sdk/pkg/server/transport/http"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// LoggingConfig holds configuration for the logging middleware
type LoggingConfig struct {
	// Logger is the logger instance to use
	Logger types.Logger
	// SkipPaths are paths that should not be logged
	SkipPaths []string
	// SkipHeaders are headers that should not be logged
	SkipHeaders []string
}

// LoggingMiddleware creates a new logging middleware
func LoggingMiddleware(config LoggingConfig) func(http.Handler) http.Handler {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	skipHeaders := make(map[string]bool)
	for _, header := range config.SkipHeaders {
		skipHeaders[header] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging for specified paths
			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := httputil.NewResponseWriter(w)

			// Process the request
			next.ServeHTTP(ww, r)

			// Use the logger
			loggerInstance := config.Logger
			if loggerInstance == nil {
				loggerInstance = logger.GetDefaultLogger()
			}

			// Create log message
			method := r.Method
			path := r.URL.Path
			status := ww.Status()
			bytesWritten := ww.BytesWritten()
			duration := time.Since(start)

			// Headers (excluding skipped ones)
			headers := make(map[string]string)
			for k, v := range r.Header {
				if !skipHeaders[k] && len(v) > 0 {
					headers[k] = v[0]
				}
			}

			// Log based on status code
			ctx := context.Background()
			logMsg := ""

			if status >= 500 {
				logMsg = "Server error"
			} else if status >= 400 {
				logMsg = "Client error"
			} else if status >= 300 {
				logMsg = "Redirection"
			} else {
				logMsg = "Success"
			}

			// Format the message with all details
			message := fmt.Sprintf("%s - %s %s - Status: %s - Bytes: %d - Duration: %s - Remote: %s - UserAgent: %s",
				logMsg, method, path, http.StatusText(status), bytesWritten, duration.String(), r.RemoteAddr, r.UserAgent())

			// Log with appropriate level based on status code
			if status >= 500 {
				loggerInstance.Error(ctx, "http", "middleware", message)
			} else if status >= 400 {
				loggerInstance.Warn(ctx, "http", "middleware", message)
			} else {
				loggerInstance.Info(ctx, "http", "middleware", message)
			}
		})
	}
}
