package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

type echoHandler struct{}

func (h *echoHandler) HandleMessage(ctx context.Context, conn *websocket.Conn, msg Message) error {
	return conn.WriteJSON(Message{
		Type:    "echo",
		Payload: msg.Payload,
	})
}

type clientTestHandler struct {
	t          *testing.T
	handleFunc func(Message) error
}

func (h *clientTestHandler) HandleMessage(ctx context.Context, conn *websocket.Conn, msg Message) error {
	return h.handleFunc(msg)
}

type customTestHandler struct {
	handleFunc func(context.Context, *websocket.Conn, Message) error
}

func (h *customTestHandler) HandleMessage(ctx context.Context, conn *websocket.Conn, msg Message) error {
	return h.handleFunc(ctx, conn, msg)
}

func TestClient(t *testing.T) {
	// Create test logger
	testLogger := logger.NewNopLogger()

	// Create server
	server := New(Options{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		Logger:          types.NewNoOpLogger(),
	})

	// Register echo handler
	server.RegisterHandler("test", &echoHandler{})

	// Create test server
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Create client
	client, err := NewClient(ClientOptions{
		URL:              wsURL,
		HandshakeTimeout: time.Second,
		Logger:           testLogger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test Send and Receive
	t.Run("Send and Receive", func(t *testing.T) {
		// Create a channel to signal when the message is received
		receivedCh := make(chan bool, 1)

		// Register handler for response
		client.RegisterHandler("echo", &clientTestHandler{
			t: t,
			handleFunc: func(msg Message) error {
				if string(msg.Payload) != `{"data":"test"}` {
					t.Errorf("Expected payload %q, got %q", `{"data":"test"}`, string(msg.Payload))
				}
				// Signal that we received the message
				receivedCh <- true
				return nil
			},
		})

		// Send message
		msg := Message{
			Type:    "test",
			Payload: json.RawMessage(`{"data":"test"}`),
		}
		if err := client.Send(msg); err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}

		// Wait for response with a timeout
		select {
		case <-receivedCh:
			// Successfully received
		case <-time.After(1 * time.Second):
			t.Fatal("Timed out waiting for response")
		}
	})

	// Test SendAndWait
	t.Run("SendAndWait", func(t *testing.T) {
		msg := Message{
			Type:    "test",
			Payload: json.RawMessage(`{"data":"test"}`),
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		response, err := client.SendAndWait(ctx, msg, "echo")
		if err != nil {
			t.Fatalf("SendAndWait failed: %v", err)
		}

		if response.Type != "echo" {
			t.Errorf("Expected response type 'echo', got %q", response.Type)
		}
		if string(response.Payload) != `{"data":"test"}` {
			t.Errorf("Expected payload %q, got %q", `{"data":"test"}`, string(response.Payload))
		}
	})

	// Test SendAndWait timeout
	t.Run("SendAndWait timeout", func(t *testing.T) {
		msg := Message{
			Type:    "test",
			Payload: json.RawMessage(`{"data":"test"}`),
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		_, err := client.SendAndWait(ctx, msg, "non-existent")
		if err == nil {
			t.Error("Expected timeout error")
		}
	})
}

func TestClient_ConnectionError(t *testing.T) {
	// Create test logger
	testLogger := logger.NewNopLogger()

	// Try to connect to non-existent server
	_, err := NewClient(ClientOptions{
		URL:              "ws://localhost:12345",
		HandshakeTimeout: time.Second,
		Logger:           testLogger,
	})

	if err == nil {
		t.Error("Expected connection error")
	}
}

func TestClient_CustomHeaders(t *testing.T) {
	// Create test logger
	testLogger := logger.NewNopLogger()

	// Create server that checks headers
	server := New(Options{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		Logger:          types.NewNoOpLogger(),
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("X-Test") == "test-value"
		},
	})

	// Create test server
	ts := httptest.NewServer(server)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Test with correct headers
	t.Run("Valid headers", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Test", "test-value")

		client, err := NewClient(ClientOptions{
			URL:              wsURL,
			Headers:          headers,
			HandshakeTimeout: time.Second,
			Logger:           testLogger,
		})
		if err != nil {
			t.Fatalf("Failed to connect with valid headers: %v", err)
		}
		client.Close()
	})

	// Test with incorrect headers
	t.Run("Invalid headers", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Test", "wrong-value")

		_, err := NewClient(ClientOptions{
			URL:              wsURL,
			Headers:          headers,
			HandshakeTimeout: time.Second,
			Logger:           testLogger,
		})
		if err == nil {
			t.Error("Expected connection to be rejected")
		}
	})
}

func TestClient_Connect(t *testing.T) {
	// Create test logger
	testLogger := logger.NewNopLogger()

	// Create test server
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Create client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	client, err := NewClient(ClientOptions{
		URL:    wsURL,
		Logger: testLogger,
	})
	assert.NoError(t, err)

	// Connect
	err = client.Connect()
	assert.NoError(t, err)
	defer client.Close()

	// Verify connection
	assert.True(t, client.IsConnected())
}

func TestClient_SendMessage(t *testing.T) {
	// Create test logger
	testLogger := logger.NewNopLogger()

	// Create test server
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})

	// Register a handler that responds with "response" type (to match SendMessage expectation)
	// instead of "echo" type
	server.RegisterHandler("test", &customTestHandler{
		handleFunc: func(ctx context.Context, conn *websocket.Conn, msg Message) error {
			return conn.WriteJSON(Message{
				Type:    "response", // Changed from "echo" to "response"
				Payload: msg.Payload,
			})
		},
	})

	ts := httptest.NewServer(server)
	defer ts.Close()

	// Create client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	client, err := NewClient(ClientOptions{
		URL:    wsURL,
		Logger: testLogger,
	})
	assert.NoError(t, err)

	// Connect
	err = client.Connect()
	assert.NoError(t, err)
	defer client.Close()

	// Send message with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payload := json.RawMessage(`{"test":"data"}`)
	response, err := client.SendMessage(ctx, "test", payload)
	assert.NoError(t, err)
	assert.Equal(t, "response", response.Type)
	assert.Equal(t, payload, response.Payload)
}

func TestClient_SendMessageTimeout(t *testing.T) {
	// Create test logger
	testLogger := logger.NewNopLogger()

	// Create test server
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})

	// Register a slow handler that sleeps before responding
	server.RegisterHandler("test", &customTestHandler{
		handleFunc: func(ctx context.Context, conn *websocket.Conn, msg Message) error {
			// Sleep longer than the timeout to ensure it triggers
			time.Sleep(300 * time.Millisecond)
			return conn.WriteJSON(Message{
				Type:    "response",
				Payload: msg.Payload,
			})
		},
	})

	ts := httptest.NewServer(server)
	defer ts.Close()

	// Create client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	client, err := NewClient(ClientOptions{
		URL:    wsURL,
		Logger: testLogger,
	})
	assert.NoError(t, err)

	// Connect
	err = client.Connect()
	assert.NoError(t, err)
	defer client.Close()

	// Send message with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = client.SendMessage(ctx, "test", json.RawMessage(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestClient_Reconnect(t *testing.T) {
	// Create test logger
	testLogger := logger.NewNopLogger()

	// Create test server
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Create client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	client, err := NewClient(ClientOptions{
		URL:    wsURL,
		Logger: testLogger,
	})
	assert.NoError(t, err)

	// Connect
	err = client.Connect()
	assert.NoError(t, err)
	defer client.Close()

	// Force disconnect
	client.Close()
	assert.False(t, client.IsConnected())

	// Reconnect
	err = client.Connect()
	assert.NoError(t, err)
	assert.True(t, client.IsConnected())
}

func TestClientConnection(t *testing.T) {
	// Create WebSocket server
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		// Read first message
		_, p, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read message: %v", err)
			return
		}

		// Echo the message back
		if err := conn.WriteMessage(websocket.TextMessage, p); err != nil {
			t.Fatalf("Failed to write message: %v", err)
			return
		}
	}))
	defer server.Close()

	// Create WebSocket client
	testLogger := logger.NewNopLogger()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, err := NewClient(ClientOptions{
		URL:              wsURL,
		ReconnectBackoff: 100 * time.Millisecond,
		PingInterval:     1 * time.Second,
		Logger:           testLogger,
	})
	require.NoError(t, err)
	defer client.Close()

	// Connect
	err = client.Connect(context.Background())
	require.NoError(t, err)

	// Test sending and receiving
	request := Request{
		ID:     "test-id",
		Method: "test-method",
		Params: json.RawMessage(`{"test":"value"}`),
	}
	response, err := client.Call(context.Background(), request.Method, request.Params)
	require.NoError(t, err)
	assert.NotNil(t, response)

	// Test close
	err = client.Close()
	require.NoError(t, err)
}

func TestClientReconnect(t *testing.T) {
	// Skip if running in CI
	if testing.Short() {
		t.Skip("Skipping reconnection test in short mode")
	}

	// Create a channel to track connections
	connections := make(chan struct{}, 2)

	// Create WebSocket server that closes connection after first message
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connections <- struct{}{}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
			return
		}

		// Read one message, then close
		_, _, err = conn.ReadMessage()
		if err != nil {
			return
		}

		// Close connection after first message
		conn.Close()
	}))
	defer server.Close()

	// Create WebSocket client
	testLogger := logger.NewNopLogger()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, err := NewClient(ClientOptions{
		URL:              wsURL,
		ReconnectBackoff: 100 * time.Millisecond,
		PingInterval:     1 * time.Second,
		MaxReconnect:     2,
		Logger:           testLogger,
	})
	require.NoError(t, err)
	defer client.Close()

	// Connect
	err = client.Connect(context.Background())
	require.NoError(t, err)

	// Send message to trigger connection close
	go func() {
		client.Call(context.Background(), "test-method", json.RawMessage(`{}`))
	}()

	// Wait for reconnect
	select {
	case <-connections:
		// First connection established
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for first connection")
	}

	select {
	case <-connections:
		// Reconnection occurred
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for reconnection")
	}

	// Close the client
	err = client.Close()
	require.NoError(t, err)
}

func TestClientSendBeforeConnect(t *testing.T) {
	testLogger := logger.NewNopLogger()
	client, err := NewClient(ClientOptions{
		URL:              "ws://localhost:12345", // Invalid URL to prevent actual connection
		ReconnectBackoff: 100 * time.Millisecond,
		Logger:           testLogger,
	})
	require.NoError(t, err)
	defer client.Close()

	// Try to send before connecting
	_, err = client.Call(context.Background(), "test-method", json.RawMessage(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestClientContextCancellation(t *testing.T) {
	// Create WebSocket server that never responds
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		// Hang forever, never responding
		select {}
	}))
	defer server.Close()

	// Create WebSocket client
	testLogger := logger.NewNopLogger()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, err := NewClient(ClientOptions{
		URL:              wsURL,
		ReconnectBackoff: 100 * time.Millisecond,
		Logger:           testLogger,
	})
	require.NoError(t, err)
	defer client.Close()

	// Connect
	err = client.Connect(context.Background())
	require.NoError(t, err)

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Send request with cancellable context
	_, err = client.Call(ctx, "test-method", json.RawMessage(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestClientCallTimeout(t *testing.T) {
	// Create WebSocket server that never responds to specific methods
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req Request
			if err := json.Unmarshal(p, &req); err != nil {
				return
			}

			// Only respond to non-timeout methods
			if req.Method != "timeout-method" {
				resp := Response{
					ID:     req.ID,
					Result: json.RawMessage(`{"success":true}`),
				}
				respData, _ := json.Marshal(resp)
				conn.WriteMessage(messageType, respData)
			}
			// Otherwise, don't respond to simulate timeout
		}
	}))
	defer server.Close()

	// Create WebSocket client with short timeout
	testLogger := logger.NewNopLogger()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, err := NewClient(ClientOptions{
		URL:              wsURL,
		ReconnectBackoff: 100 * time.Millisecond,
		CallTimeout:      100 * time.Millisecond,
		Logger:           testLogger,
	})
	require.NoError(t, err)
	defer client.Close()

	// Connect
	err = client.Connect(context.Background())
	require.NoError(t, err)

	// Call method that will time out
	_, err = client.Call(context.Background(), "timeout-method", json.RawMessage(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")

	// Call method that will succeed
	resp, err := client.Call(context.Background(), "normal-method", json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClientErrorResponse(t *testing.T) {
	// Create WebSocket server that returns error for specific methods
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req Request
			if err := json.Unmarshal(p, &req); err != nil {
				return
			}

			var resp Response
			if req.Method == "error-method" {
				resp = Response{
					ID: req.ID,
					Error: &types.Error{
						Code:    400,
						Message: "Test error",
					},
				}
			} else {
				resp = Response{
					ID:     req.ID,
					Result: json.RawMessage(`{"success":true}`),
				}
			}

			respData, _ := json.Marshal(resp)
			conn.WriteMessage(messageType, respData)
		}
	}))
	defer server.Close()

	// Create WebSocket client
	testLogger := logger.NewNopLogger()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, err := NewClient(ClientOptions{
		URL:              wsURL,
		ReconnectBackoff: 100 * time.Millisecond,
		Logger:           testLogger,
	})
	require.NoError(t, err)
	defer client.Close()

	// Connect
	err = client.Connect(context.Background())
	require.NoError(t, err)

	// Call method that will return error
	_, err = client.Call(context.Background(), "error-method", json.RawMessage(`{}`))
	require.Error(t, err)
	mcpErr, ok := types.IsError(err)
	require.True(t, ok)
	assert.Equal(t, 400, mcpErr.Code)
	assert.Equal(t, "Test error", mcpErr.Message)

	// Call method that will succeed
	resp, err := client.Call(context.Background(), "normal-method", json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
