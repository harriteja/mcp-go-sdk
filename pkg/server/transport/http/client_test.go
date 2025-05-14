package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"go.uber.org/zap"
)

func TestClient(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test":
			// Echo request headers and body
			w.Header().Set("Content-Type", "application/json")
			result := map[string]interface{}{
				"method":  r.Method,
				"headers": r.Header,
			}

			if r.Method == http.MethodPost || r.Method == http.MethodPut {
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					WriteError(w, http.StatusBadRequest, err.Error())
					return
				}
				result["body"] = body
			}

			WriteJSON(w, http.StatusOK, result)

		case "/error":
			WriteError(w, http.StatusBadRequest, "test error")

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create client
	client := NewClient(ClientOptions{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		Logger:  zap.NewNop(),
	})

	// Test GET request
	t.Run("GET request", func(t *testing.T) {
		headers := map[string]string{
			"X-Test": "test-value",
		}
		resp, err := client.Get(context.Background(), "/test", headers)
		if err != nil {
			t.Fatalf("Failed to send GET request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var response struct {
			Result struct {
				Method  string                   `json:"method"`
				Headers map[string][]interface{} `json:"headers"`
			} `json:"result"`
		}
		if err := json.Unmarshal(resp.Body, &response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Result.Method != "GET" {
			t.Errorf("Expected method GET, got %v", response.Result.Method)
		}

		xTest := response.Result.Headers["X-Test"]
		if len(xTest) == 0 || xTest[0] != "test-value" {
			t.Errorf("Expected header X-Test=test-value, got %v", xTest)
		}
	})

	// Test POST request
	t.Run("POST request", func(t *testing.T) {
		body := map[string]interface{}{
			"key": "value",
		}
		resp, err := client.Post(context.Background(), "/test", body, nil)
		if err != nil {
			t.Fatalf("Failed to send POST request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var response struct {
			Result struct {
				Method string                 `json:"method"`
				Body   map[string]interface{} `json:"body"`
			} `json:"result"`
		}
		if err := json.Unmarshal(resp.Body, &response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Result.Method != "POST" {
			t.Errorf("Expected method POST, got %v", response.Result.Method)
		}

		if response.Result.Body["key"] != "value" {
			t.Errorf("Expected body key=value, got %v", response.Result.Body["key"])
		}
	})

	// Test error response
	t.Run("Error response", func(t *testing.T) {
		resp, err := client.Get(context.Background(), "/error", nil)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}

		var response struct {
			Error *types.Error `json:"error"`
		}
		if err := json.Unmarshal(resp.Body, &response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Error.Message != "test error" {
			t.Errorf("Expected error message 'test error', got %q", response.Error.Message)
		}
	})

	// Test request timeout
	t.Run("Request timeout", func(t *testing.T) {
		timeoutClient := NewClient(ClientOptions{
			BaseURL: "http://example.com",
			Timeout: 1 * time.Millisecond,
			Logger:  zap.NewNop(),
		})

		_, err := timeoutClient.Get(context.Background(), "/test", nil)
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
	})

	// Test invalid URL
	t.Run("Invalid URL", func(t *testing.T) {
		invalidClient := NewClient(ClientOptions{
			BaseURL: "://invalid",
			Logger:  zap.NewNop(),
		})

		_, err := invalidClient.Get(context.Background(), "/test", nil)
		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
	})
}
