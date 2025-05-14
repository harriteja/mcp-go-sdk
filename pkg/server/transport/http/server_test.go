package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestServer(t *testing.T) {
	// Create test server
	opts := Options{
		Address:      ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
		Logger:       logger.NewZapLogger(zap.NewNop(), &types.LoggerConfig{MinLevel: types.LogLevelInfo}),
	}
	server := New(opts)

	// Test handler registration
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	if err := server.RegisterHandler("/test", testHandler); err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Test request handling
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	server.handlers["/test"].ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response struct {
		Result map[string]string `json:"result"`
		Error  *types.Error      `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}

	if response.Result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %q", response.Result["status"])
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response struct {
		Error *types.Error `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error.Code != http.StatusBadRequest {
		t.Errorf("Expected error code %d, got %d", http.StatusBadRequest, response.Error.Code)
	}

	if response.Error.Message != "test error" {
		t.Errorf("Expected error message %q, got %q", "test error", response.Error.Message)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]interface{}{
		"key": "value",
		"num": 123,
	}
	WriteJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response struct {
		Result map[string]interface{} `json:"result"`
		Error  *types.Error           `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}

	if response.Result["key"] != "value" {
		t.Errorf("Expected key value %q, got %q", "value", response.Result["key"])
	}

	if response.Result["num"].(float64) != 123 {
		t.Errorf("Expected num value %d, got %v", 123, response.Result["num"])
	}
}

func TestServerLifecycle(t *testing.T) {
	opts := Options{
		Address:      ":0", // Use random port
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  5 * time.Second,
		Logger:       logger.NewZapLogger(zap.NewNop(), &types.LoggerConfig{MinLevel: types.LogLevelInfo}),
	}
	server := New(opts)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(context.Background())
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	// Check if server stopped without error
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Server returned unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Server did not stop within timeout")
	}
}

func TestServer_Start(t *testing.T) {
	server := New(Options{
		Address: ":0", // Use random port
		Logger:  types.NewNoOpLogger(),
	})

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(context.Background())
	}()

	// Wait for server to start or timeout
	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("Server failed to start: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
	}

	// Stop server
	err := server.Stop(context.Background())
	assert.NoError(t, err)

	// Wait for server to stop or timeout
	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed to stop: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Server failed to stop")
	}
}

func TestServer_RegisterHandler(t *testing.T) {
	server := New(Options{
		Address: ":0",
		Logger:  types.NewNoOpLogger(),
	})

	// Register first handler
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	err := server.RegisterHandler("/test1", handler1)
	assert.NoError(t, err)

	// Register second handler with different path
	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	err = server.RegisterHandler("/test2", handler2)
	assert.NoError(t, err)

	// Try to register handler with duplicate path
	err = server.RegisterHandler("/test1", handler1)
	assert.Error(t, err)
}

func TestServer_WriteError(t *testing.T) {
	server := New(Options{
		Address: ":0",
		Logger:  types.NewNoOpLogger(),
	})

	w := httptest.NewRecorder()
	server.WriteError(w, http.StatusBadRequest, "test error")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "test error")
}

func TestServer_WriteJSON(t *testing.T) {
	server := New(Options{
		Address: ":0",
		Logger:  types.NewNoOpLogger(),
	})

	w := httptest.NewRecorder()
	data := map[string]string{"test": "value"}
	server.WriteJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"test":"value"`)
}
