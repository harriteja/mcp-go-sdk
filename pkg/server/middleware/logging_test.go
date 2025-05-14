package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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
			// Create a buffer to capture logs
			var buf bytes.Buffer
			encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
				MessageKey:     "msg",
				LevelKey:       "level",
				TimeKey:        "time",
				NameKey:        "name",
				CallerKey:      "caller",
				StacktraceKey:  "stacktrace",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			})
			core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
			logger := zap.New(core)

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
			if tt.expectLogged {
				if buf.Len() == 0 {
					t.Error("Expected log entry but got none")
				}

				// Parse log entry
				var logEntry map[string]interface{}
				if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
					t.Fatalf("Failed to parse log entry: %v", err)
				}

				// Verify basic fields
				if method, ok := logEntry["method"].(string); !ok || method != tt.method {
					t.Errorf("Wrong method in log: got %v want %v", method, tt.method)
				}
				if path, ok := logEntry["path"].(string); !ok || path != tt.path {
					t.Errorf("Wrong path in log: got %v want %v", path, tt.path)
				}

				// Verify headers
				if headers, ok := logEntry["headers"].(map[string]interface{}); ok {
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
								if _, exists := headers[k]; exists {
									t.Errorf("Skipped header %s was logged", k)
								}
								continue
							}
						}
						if headers[k] != v {
							t.Errorf("Wrong header value for %s: got %v want %v", k, headers[k], v)
						}
					}
				}
			} else {
				if buf.Len() > 0 {
					t.Error("Expected no log entry but got one")
				}
			}
		})
	}
}
