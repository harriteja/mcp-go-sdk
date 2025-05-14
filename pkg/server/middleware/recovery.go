package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"go.uber.org/zap"
)

// RecoveryConfig holds configuration for the recovery middleware
type RecoveryConfig struct {
	// Logger is the zap logger instance to use
	Logger *zap.Logger
	// OnPanic is called when a panic occurs, after logging but before writing the response
	OnPanic func(http.ResponseWriter, *http.Request, interface{})
	// StackTrace determines whether to include stack traces in logs
	StackTrace bool
}

// RecoveryMiddleware creates a new recovery middleware that catches panics
func RecoveryMiddleware(config RecoveryConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Ensure the response hasn't been written yet
					if w.Header().Get("Content-Type") != "" {
						return
					}

					// Get stack trace if enabled
					var stack []byte
					if config.StackTrace {
						stack = debug.Stack()
					}

					// Log the panic
					logger := config.Logger
					if logger == nil {
						logger, _ = zap.NewProduction()
					}

					fields := []zap.Field{
						zap.String("error", fmt.Sprint(err)),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("remote_addr", r.RemoteAddr),
					}

					if stack != nil {
						fields = append(fields, zap.ByteString("stack", stack))
					}

					logger.Error("Panic recovered", fields...)

					// Call custom panic handler if provided
					if config.OnPanic != nil {
						config.OnPanic(w, r, err)
						return
					}

					// Default panic response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					response := struct {
						Error *types.Error `json:"error"`
					}{
						Error: types.NewError(http.StatusInternalServerError, "internal server error"),
					}
					data, _ := json.Marshal(response)
					if _, err := w.Write(data); err != nil {
						logger.Error("Failed to write error response",
							zap.Error(err),
						)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultPanicHandler provides a default implementation for handling panics
func DefaultPanicHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	response := struct {
		Error *types.Error `json:"error"`
	}{
		Error: types.NewError(http.StatusInternalServerError, fmt.Sprint(err)),
	}
	data, _ := json.Marshal(response)
	_, _ = w.Write(data)
}
