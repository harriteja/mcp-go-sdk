package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestRecoveryMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		handler       func(http.ResponseWriter, *http.Request)
		customHandler func(http.ResponseWriter, *http.Request, interface{})
		stackTrace    bool
		wantStatus    int
		wantBody      string
		wantPanic     bool
	}{
		{
			name: "No panic",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "success")
			},
			wantStatus: http.StatusOK,
			wantBody:   "success",
			wantPanic:  false,
		},
		{
			name: "Basic panic",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("test panic")
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":{"code":500,"message":"internal server error"}}`,
			wantPanic:  true,
		},
		{
			name: "Custom panic handler",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("custom panic")
			},
			customHandler: func(w http.ResponseWriter, r *http.Request, err interface{}) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				response := struct {
					Error *types.Error `json:"error"`
				}{
					Error: types.NewError(http.StatusServiceUnavailable, fmt.Sprint(err)),
				}
				data, _ := json.Marshal(response)
				_, _ = w.Write(data)
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `{"error":{"code":503,"message":"custom panic"}}`,
			wantPanic:  true,
		},
		{
			name: "Panic with stack trace",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("stack trace panic")
			},
			stackTrace: true,
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":{"code":500,"message":"internal server error"}}`,
			wantPanic:  true,
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
			config := RecoveryConfig{
				Logger:     logger,
				OnPanic:    tt.customHandler,
				StackTrace: tt.stackTrace,
			}
			handler := RecoveryMiddleware(config)(http.HandlerFunc(tt.handler))

			// Create request
			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			w := httptest.NewRecorder()

			// Process request
			handler.ServeHTTP(w, req)

			// Verify response
			if w.Code != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					w.Code, tt.wantStatus)
			}

			if w.Body.String() != tt.wantBody {
				t.Errorf("handler returned wrong body: got %v want %v",
					w.Body.String(), tt.wantBody)
			}

			// Verify logging
			if tt.wantPanic {
				if buf.Len() == 0 {
					t.Error("Expected log entry but got none")
				}

				// Parse log entry
				var logEntry map[string]interface{}
				if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
					t.Fatalf("Failed to parse log entry: %v", err)
				}

				// Verify basic fields
				if level, ok := logEntry["level"].(string); !ok || level != "error" {
					t.Errorf("Wrong log level: got %v want error", level)
				}
				if msg, ok := logEntry["msg"].(string); !ok || msg != "Panic recovered" {
					t.Errorf("Wrong log message: got %v want 'Panic recovered'", msg)
				}

				// Verify stack trace if enabled
				if tt.stackTrace {
					if stack, ok := logEntry["stack"].(string); !ok || len(stack) == 0 {
						t.Error("Expected stack trace but got none")
					}
				} else {
					if _, ok := logEntry["stack"]; ok {
						t.Error("Got stack trace when not requested")
					}
				}
			} else {
				if buf.Len() > 0 {
					t.Error("Got log entry when no panic occurred")
				}
			}
		})
	}
}

func TestDefaultPanicHandler(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://example.com/foo", nil)
	err := fmt.Errorf("test error")

	DefaultPanicHandler(w, r, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			w.Code, http.StatusInternalServerError)
	}

	expected := `{"error":{"code":500,"message":"test error"}}`
	if w.Body.String() != expected {
		t.Errorf("handler returned wrong body: got %v want %v",
			w.Body.String(), expected)
	}
}
