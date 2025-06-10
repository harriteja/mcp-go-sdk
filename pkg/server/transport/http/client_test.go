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
					w.WriteHeader(http.StatusBadRequest)
					if err := json.NewEncoder(w).Encode(map[string]interface{}{
						"error": err.Error(),
					}); err != nil {
						http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					}
					return
				}
				result["body"] = body
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}

		case "/error":
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"error": &types.Error{
					Code:    http.StatusBadRequest,
					Message: "test error",
				},
			}); err != nil {
				http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
			}

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create client
	client := NewClient(ClientOptions{
		BaseURL:   server.URL,
		Timeout:   5 * time.Second,
		Logger:    logger.NewNopLogger(),
		Transport: http.DefaultTransport,
	})

	// Test GET request
	t.Run("GET request", func(t *testing.T) {
		headers := map[string]string{
			"X-Test": "test-value",
		}
		resp, err := client.Get(context.Background(), "/test", headers)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Body, &result)
		require.NoError(t, err)

		method, ok := result["method"]
		require.True(t, ok)
		assert.Equal(t, "GET", method)

		headersMap, ok := result["headers"].(map[string]interface{})
		require.True(t, ok)

		xTestValues, ok := headersMap["X-Test"].([]interface{})
		require.True(t, ok)
		require.Len(t, xTestValues, 1)
		assert.Equal(t, "test-value", xTestValues[0])
	})

	// Test POST request
	t.Run("POST request", func(t *testing.T) {
		body := map[string]interface{}{
			"key": "value",
		}
		resp, err := client.Post(context.Background(), "/test", body, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Body, &result)
		require.NoError(t, err)

		method, ok := result["method"]
		require.True(t, ok)
		assert.Equal(t, "POST", method)

		responseBody, ok := result["body"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", responseBody["key"])
	})

	// Test error response
	t.Run("Error response", func(t *testing.T) {
		resp, err := client.Get(context.Background(), "/error", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Body, &result)
		require.NoError(t, err)

		errorObj, ok := result["error"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(http.StatusBadRequest), errorObj["code"])
		assert.Equal(t, "test error", errorObj["message"])
	})

	// Test invalid URL
	t.Run("Invalid URL", func(t *testing.T) {
		invalidClient := NewClient(ClientOptions{
			BaseURL: "://invalid",
			Logger:  logger.NewNopLogger(),
		})

		_, err := invalidClient.Get(context.Background(), "/test", nil)
		require.Error(t, err)
	})
}
