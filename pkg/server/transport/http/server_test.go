package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func TestServer(t *testing.T) {
	// Create server with custom handler
	server := NewServer(Options{
		Logger:       logger.NewNopLogger(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	// Add handler for test requests
	server.HandleFunc("test", func(w http.ResponseWriter, r *http.Request) {
		// Extract request body
		var request struct {
			Value string `json:"value"`
		}
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Write response
		response := map[string]interface{}{
			"echo": request.Value,
		}
		WriteJSON(w, http.StatusOK, response)
	})

	// Add handler for error responses
	server.HandleFunc("error", func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusBadRequest, "test error")
	})

	// Test requests
	t.Run("Valid request", func(t *testing.T) {
		// Create test request
		requestBody := map[string]interface{}{
			"value": "test value",
		}
		requestBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(string(requestBytes)))
		req.Header.Set("Content-Type", "application/json")

		// Record response
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		// Check response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response body
		var responseBody struct {
			Result struct {
				Echo string `json:"echo"`
			} `json:"result"`
		}
		err := json.NewDecoder(resp.Body).Decode(&responseBody)
		require.NoError(t, err)
		assert.Equal(t, "test value", responseBody.Result.Echo)
	})

	t.Run("Error response", func(t *testing.T) {
		// Create test request
		req := httptest.NewRequest(http.MethodPost, "/error", nil)
		req.Header.Set("Content-Type", "application/json")

		// Record response
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		// Check response
		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Parse response body
		var responseBody struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		err := json.NewDecoder(resp.Body).Decode(&responseBody)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, responseBody.Error.Code)
		assert.Equal(t, "test error", responseBody.Error.Message)
	})

	t.Run("Not found", func(t *testing.T) {
		// Create test request for non-existent endpoint
		req := httptest.NewRequest(http.MethodPost, "/nonexistent", nil)
		req.Header.Set("Content-Type", "application/json")

		// Record response
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		// Check response
		resp := w.Result()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Method not allowed", func(t *testing.T) {
		// Create test request with wrong method
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Content-Type", "application/json")

		// Record response
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		// Check response
		resp := w.Result()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestTransport(t *testing.T) {
	// Create mock MCP server
	mockMCPServer := &mockMCPServer{
		t: t,
		listToolsHandler: func(ctx context.Context) ([]types.Tool, error) {
			return []types.Tool{
				{
					Name:        "test-tool",
					Description: "A test tool",
				},
			}, nil
		},
	}

	// Create HTTP transport
	transport := NewTransport(mockMCPServer, logger.NewNopLogger())

	// Create HTTP server
	handler := transport.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	// Test ListTools request
	t.Run("ListTools", func(t *testing.T) {
		// Create request
		request := Request{
			ID:     "test-id",
			Method: "listTools",
		}
		requestBytes, _ := json.Marshal(request)

		// Send request
		resp, err := http.Post(server.URL, "application/json", strings.NewReader(string(requestBytes)))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check response
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var response Response
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify response
		assert.Equal(t, request.ID, response.ID)
		assert.NotNil(t, response.Result)
		assert.Nil(t, response.Error)

		// Parse result
		var tools []types.Tool
		err = json.Unmarshal(response.Result, &tools)
		require.NoError(t, err)
		assert.Len(t, tools, 1)
		assert.Equal(t, "test-tool", tools[0].Name)
	})
}

func TestServerLifecycle(t *testing.T) {
	opts := Options{
		Address:      ":0", // Use random port
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  5 * time.Second,
		Logger:       logger.NewNopLogger(),
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
