package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func TestRecoveryMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		panicValue     interface{}
		customHandler  bool
		expectResponse bool
	}{
		{
			name:           "Basic panic",
			panicValue:     "test panic",
			customHandler:  false,
			expectResponse: true,
		},
		{
			name:           "Custom handler",
			panicValue:     "custom panic",
			customHandler:  true,
			expectResponse: true,
		},
		{
			name:           "Panic with error",
			panicValue:     error(customError("error panic")),
			customHandler:  false,
			expectResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock logger
			logger := NewMockLogger()

			// Create middleware
			config := RecoveryConfig{
				Logger:     logger,
				StackTrace: true,
			}

			if tt.customHandler {
				config.OnPanic = func(w http.ResponseWriter, r *http.Request, err interface{}) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"custom_error": "handled panic: " + err.(string),
					})
				}
			}

			handler := RecoveryMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(tt.panicValue)
			}))

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			// Process request (should recover from panic)
			handler.ServeHTTP(w, req)

			// Verify response
			if tt.expectResponse {
				if w.Code != http.StatusInternalServerError {
					t.Errorf("handler returned wrong status code: got %v want %v",
						w.Code, http.StatusInternalServerError)
				}

				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("handler returned wrong content type: got %v want %v",
						w.Header().Get("Content-Type"), "application/json")
				}
			}

			// Verify logging
			logs := logger.GetLogs()
			if len(logs) == 0 {
				t.Error("Expected log entry but got none")
			}

			// Check that the panic was logged
			var found bool
			for _, entry := range logs {
				if entry.Level == types.LogLevelError && entry.Bucket == "http" && entry.Handler == "recovery" {
					found = true
					break
				}
			}
			if !found {
				t.Error("No panic recovery log entry found")
			}
		})
	}
}

type customError string

func (e customError) Error() string {
	return string(e)
}
