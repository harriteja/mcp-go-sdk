package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// MockLogger is a logger that records log messages for testing
type MockLogger struct {
	mu        sync.Mutex
	logs      []LogEntry
	verbosity int
	bucket    string
}

// LogEntry represents a single log entry
type LogEntry struct {
	Level   types.LogLevel
	Bucket  string
	Handler string
	Message string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs:      []LogEntry{},
		verbosity: 0,
		bucket:    "root",
	}
}

func (l *MockLogger) Access(ctx context.Context, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, LogEntry{
		Level:   types.LogLevelInfo,
		Bucket:  "access",
		Handler: "",
		Message: message,
	})
}

func (l *MockLogger) Info(ctx context.Context, bucket, handler, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, LogEntry{
		Level:   types.LogLevelInfo,
		Bucket:  bucket,
		Handler: handler,
		Message: message,
	})
}

func (l *MockLogger) Warn(ctx context.Context, bucket, handler, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, LogEntry{
		Level:   types.LogLevelWarn,
		Bucket:  bucket,
		Handler: handler,
		Message: message,
	})
}

func (l *MockLogger) Error(ctx context.Context, bucket, handler, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, LogEntry{
		Level:   types.LogLevelError,
		Bucket:  bucket,
		Handler: handler,
		Message: message,
	})
}

func (l *MockLogger) Panic(ctx context.Context, bucket, handler, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, LogEntry{
		Level:   types.LogLevelError,
		Bucket:  bucket,
		Handler: handler,
		Message: message,
	})
	panic(message)
}

func (l *MockLogger) V(n int) bool {
	return l.verbosity >= n
}

func (l *MockLogger) Sub(name string) types.Logger {
	return &MockLogger{
		logs:      l.logs,
		verbosity: l.verbosity,
		bucket:    l.bucket + "." + name,
	}
}

func (l *MockLogger) SubWithIncrement(name string, n int) types.Logger {
	return &MockLogger{
		logs:      l.logs,
		verbosity: l.verbosity + n,
		bucket:    l.bucket + "." + name,
	}
}

func (l *MockLogger) GetLogs() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logs
}

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		method         string
		headers        map[string]string
		skipPaths      []string
		skipHeaders    []string
		expectedStatus int
		expectLogged   bool
	}{
		{
			name:           "Basic request",
			path:           "/test",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectLogged:   true,
		},
		{
			name:   "Request with headers",
			path:   "/test",
			method: "POST",
			headers: map[string]string{
				"X-Test":        "test",
				"Content-Type":  "application/json",
				"Authorization": "Bearer token",
			},
			skipHeaders:    []string{"Authorization"},
			expectedStatus: http.StatusOK,
			expectLogged:   true,
		},
		{
			name:           "Skipped path",
			path:           "/metrics",
			method:         "GET",
			skipPaths:      []string{"/metrics"},
			expectedStatus: http.StatusOK,
			expectLogged:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock logger
			logger := NewMockLogger()

			// Create middleware
			config := LoggingConfig{
				Logger:      logger,
				SkipPaths:   tt.skipPaths,
				SkipHeaders: tt.skipHeaders,
			}
			handler := LoggingMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.expectedStatus)
				_, _ = w.Write([]byte("test"))
			}))

			// Create request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()

			// Process request
			handler.ServeHTTP(w, req)

			// Verify response
			if w.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					w.Code, tt.expectedStatus)
			}

			// Verify logging
			logs := logger.GetLogs()
			if tt.expectLogged {
				if len(logs) == 0 {
					t.Error("Expected log entry but got none")
				}

				// Get the log entry
				logEntry := logs[0]

				// Verify basic fields are included in the message
				if !strings.Contains(logEntry.Message, tt.method) {
					t.Errorf("Log message doesn't contain method: %s", logEntry.Message)
				}
				if !strings.Contains(logEntry.Message, tt.path) {
					t.Errorf("Log message doesn't contain path: %s", logEntry.Message)
				}

				// Verify headers handling
				for k, v := range tt.headers {
					if tt.skipHeaders != nil {
						var skip bool
						for _, sh := range tt.skipHeaders {
							if sh == k {
								skip = true
								break
							}
						}
						if skip {
							// Skipped header should not be in the log message
							if strings.Contains(logEntry.Message, k) && strings.Contains(logEntry.Message, v) {
								t.Errorf("Skipped header %s was logged in: %s", k, logEntry.Message)
							}
							continue
						}
					}
				}
			} else {
				if len(logs) > 0 {
					t.Error("Expected no log entry but got one")
				}
			}
		})
	}
}
