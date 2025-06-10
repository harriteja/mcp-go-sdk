package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func TestClientSuccess(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Validate request
		assert.Equal(t, "testMethod", req.Method)

		// Send response
		resp := Response{
			ID:     req.ID,
			Result: json.RawMessage(`{"message":"success"}`),
		}
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create client
	client := NewClient(ClientOptions{
		ServerURL:     server.URL,
		Timeout:       time.Second,
		MaxRetries:    3,
		RetryInterval: time.Millisecond,
		Logger:        logger.NewNopLogger(),
	})

	// Make request
	var result map[string]string
	err := client.Call(context.Background(), "testMethod", map[string]string{"key": "value"}, &result)
	require.NoError(t, err)
	assert.Equal(t, "success", result["message"])
}

func TestClientError(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Send error response
		resp := Response{
			ID: req.ID,
			Error: &types.Error{
				Code:    400,
				Message: "Bad request",
			},
		}
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create client
	client := NewClient(ClientOptions{
		ServerURL:     server.URL,
		Timeout:       time.Second,
		MaxRetries:    3,
		RetryInterval: time.Millisecond,
		Logger:        logger.NewNopLogger(),
	})

	// Make request
	var result map[string]string
	err := client.Call(context.Background(), "testMethod", map[string]string{"key": "value"}, &result)
	require.Error(t, err)

	// Verify error type
	mcpErr, ok := types.IsError(err)
	require.True(t, ok)
	assert.Equal(t, 400, mcpErr.Code)
	assert.Equal(t, "Bad request", mcpErr.Message)
}

func TestClientRetry(t *testing.T) {
	attempts := 0

	// Create a test server that fails the first two attempts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Succeed on third attempt
		resp := Response{
			Result: json.RawMessage(`{"message":"success"}`),
		}
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create client with retry
	client := NewClient(ClientOptions{
		ServerURL:     server.URL,
		Timeout:       time.Second,
		MaxRetries:    3,
		RetryInterval: time.Millisecond,
		Logger:        logger.NewNopLogger(),
	})

	// Make request
	var result map[string]string
	err := client.Call(context.Background(), "testMethod", map[string]string{"key": "value"}, &result)
	require.NoError(t, err)
	assert.Equal(t, "success", result["message"])
	assert.Equal(t, 3, attempts)
}
