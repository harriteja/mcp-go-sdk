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
	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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
	logger := zap.NewNop()

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
		Logger:           logger,
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
	logger := zap.NewNop()

	// Try to connect to non-existent server
	_, err := NewClient(ClientOptions{
		URL:              "ws://localhost:12345",
		HandshakeTimeout: time.Second,
		Logger:           logger,
	})

	if err == nil {
		t.Error("Expected connection error")
	}
}

func TestClient_CustomHeaders(t *testing.T) {
	// Create test logger
	logger := zap.NewNop()

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
			Logger:           logger,
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
			Logger:           logger,
		})
		if err == nil {
			t.Error("Expected connection to be rejected")
		}
	})
}

func TestClient_Connect(t *testing.T) {
	// Create test logger
	logger := zap.NewNop()

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
		Logger: logger,
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
	logger := zap.NewNop()

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
		Logger: logger,
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
	logger := zap.NewNop()

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
		Logger: logger,
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
	logger := zap.NewNop()

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
		Logger: logger,
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
