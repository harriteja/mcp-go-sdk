package middleware

import (
	"net/http"
	"time"

	httputil "github.com/harriteja/mcp-go-sdk/pkg/server/transport/http"
	"go.uber.org/zap"
)

// LoggingConfig holds configuration for the logging middleware
type LoggingConfig struct {
	// Logger is the zap logger instance to use
	Logger *zap.Logger
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

			// Create fields for logging
			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
			}

			// Add headers (excluding skipped ones)
			headers := make(map[string]string)
			for k, v := range r.Header {
				if !skipHeaders[k] && len(v) > 0 {
					headers[k] = v[0]
				}
			}
			if len(headers) > 0 {
				fields = append(fields, zap.Any("headers", headers))
			}

			// Add query parameters
			if query := r.URL.Query(); len(query) > 0 {
				fields = append(fields, zap.Any("query", query))
			}

			// Process the request
			next.ServeHTTP(ww, r)

			// Add response information
			duration := time.Since(start)
			fields = append(fields,
				zap.Int("status", ww.Status()),
				zap.Int64("bytes_written", ww.BytesWritten()),
				zap.Duration("duration", duration),
			)

			// Log based on status code
			logger := config.Logger
			if logger == nil {
				logger, _ = zap.NewProduction()
			}

			switch {
			case ww.Status() >= 500:
				logger.Error("Server error", fields...)
			case ww.Status() >= 400:
				logger.Warn("Client error", fields...)
			case ww.Status() >= 300:
				logger.Info("Redirection", fields...)
			default:
				logger.Info("Success", fields...)
			}
		})
	}
}
