package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// RecoveryConfig holds configuration for the recovery middleware
type RecoveryConfig struct {
	// Logger is the logger instance to use
	Logger types.Logger
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

					// Get logger
					loggerInstance := config.Logger
					if loggerInstance == nil {
						loggerInstance = logger.GetDefaultLogger()
					}

					// Create error message
					method := r.Method
					path := r.URL.Path
					remoteAddr := r.RemoteAddr
					errStr := fmt.Sprint(err)

					message := fmt.Sprintf("Panic recovered - %s - Method: %s, Path: %s, Remote: %s",
						errStr, method, path, remoteAddr)

					// Log the panic
					ctx := context.Background()
					loggerInstance.Error(ctx, "http", "recovery", message)

					if stack != nil {
						stackMessage := "Stack trace: " + string(stack)
						loggerInstance.Error(ctx, "http", "recovery", stackMessage)
					}

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
						loggerInstance.Error(ctx, "http", "recovery",
							fmt.Sprintf("Failed to write error response: %v", err))
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
