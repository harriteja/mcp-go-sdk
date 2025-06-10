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
)

func TestServerBasic(t *testing.T) {
	// Create a server with basic options
	server := New(Options{
		Address:      ":0", // Random port
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		Logger:       logger.NewNopLogger(),
	})

	// Register a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		}); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	})

	err := server.RegisterHandler("/test", testHandler)
	require.NoError(t, err)

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(context.Background())
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Stop the server
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = server.Stop(ctx)
	require.NoError(t, err)

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

func TestHandleRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			Method string                 `json:"method"`
			Params map[string]interface{} `json:"params"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    http.StatusBadRequest,
					"message": err.Error(),
				},
			}); err != nil {
				t.Logf("Failed to encode error response: %v", err)
			}
			return
		}

		// Handle request
		if req.Method == "test" {
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"result": map[string]interface{}{
					"status": "ok",
				},
			}); err != nil {
				t.Logf("Failed to encode response: %v", err)
			}
			return
		}

		// Method not found
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    http.StatusNotFound,
				"message": "method not found",
			},
		}); err != nil {
			t.Logf("Failed to encode error response: %v", err)
		}
	})

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Test successful request
	t.Run("Successful request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"method": "test",
			"params": map[string]interface{}{},
		}

		reqBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(server.URL, "application/json", strings.NewReader(string(reqBytes)))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var respBody map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		result, ok := respBody["result"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "ok", result["status"])
	})

	// Test method not found
	t.Run("Method not found", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"method": "nonexistent",
			"params": map[string]interface{}{},
		}

		reqBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(server.URL, "application/json", strings.NewReader(string(reqBytes)))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var respBody map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		errObj, ok := respBody["error"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(http.StatusNotFound), errObj["code"])
		assert.Equal(t, "method not found", errObj["message"])
	})
}
