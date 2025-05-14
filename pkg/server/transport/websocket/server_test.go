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
)

type testHandler struct {
	handleFunc func(ctx context.Context, conn *websocket.Conn, msg Message) error
}

func (h *testHandler) HandleMessage(ctx context.Context, conn *websocket.Conn, msg Message) error {
	if h.handleFunc != nil {
		return h.handleFunc(ctx, conn, msg)
	}
	return nil
}

func TestServer_Start(t *testing.T) {
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := server.Start(ctx)
	assert.NoError(t, err)
}

func TestServer_Stop(t *testing.T) {
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := server.Stop(ctx)
	assert.NoError(t, err)
}

func TestServer_HandleConnection(t *testing.T) {
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})

	// Create test server
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Create WebSocket client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Use a channel to synchronize
	handledCh := make(chan bool, 1)

	// Register test handler
	server.RegisterHandler("test", &testHandler{
		handleFunc: func(ctx context.Context, conn *websocket.Conn, msg Message) error {
			handledCh <- true
			return nil
		},
	})

	// Send test message
	err = conn.WriteJSON(Message{
		Type:    "test",
		Payload: json.RawMessage(`{}`),
	})
	assert.NoError(t, err)

	// Wait for handler with timeout
	select {
	case <-handledCh:
		// Successfully handled
	case <-time.After(1 * time.Second):
		t.Fatal("Handler was not called within timeout")
	}
}

func TestServer_UnknownMessageType(t *testing.T) {
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})

	// Create test server
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Create WebSocket client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Send message with unknown type
	err = conn.WriteJSON(Message{
		Type:    "unknown",
		Payload: json.RawMessage(`{}`),
	})
	assert.NoError(t, err)

	// Read error response
	var response Message
	err = conn.ReadJSON(&response)
	assert.NoError(t, err)
	assert.Equal(t, "error", response.Type)
	assert.Contains(t, string(response.Payload), "unknown message type")
}

func TestServer_HandlerError(t *testing.T) {
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})

	// Create test server
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Create WebSocket client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Register handler that returns error
	server.RegisterHandler("test", &testHandler{
		handleFunc: func(ctx context.Context, conn *websocket.Conn, msg Message) error {
			return assert.AnError
		},
	})

	// Send test message
	err = conn.WriteJSON(Message{
		Type:    "test",
		Payload: json.RawMessage(`{}`),
	})
	assert.NoError(t, err)

	// Read error response
	var response Message
	err = conn.ReadJSON(&response)
	assert.NoError(t, err)
	assert.Equal(t, "error", response.Type)
	assert.Contains(t, string(response.Payload), assert.AnError.Error())
}

func TestServer_WriteError(t *testing.T) {
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})

	w := httptest.NewRecorder()
	server.WriteError(w, http.StatusBadRequest, "test error")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "test error")
}

func TestServer_WriteJSON(t *testing.T) {
	server := New(Options{
		Logger: types.NewNoOpLogger(),
	})

	w := httptest.NewRecorder()
	data := map[string]string{"test": "value"}
	server.WriteJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"test":"value"`)
}
