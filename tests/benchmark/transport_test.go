package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wsgorilla "github.com/gorilla/websocket"
	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	transporthttp "github.com/harriteja/mcp-go-sdk/pkg/server/transport/http"
	"github.com/harriteja/mcp-go-sdk/pkg/server/transport/websocket"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"go.uber.org/zap"
)

type testHandler struct {
	handleFunc func(ctx context.Context, conn *wsgorilla.Conn, msg websocket.Message) error
}

func (h *testHandler) HandleMessage(ctx context.Context, conn *wsgorilla.Conn, msg websocket.Message) error {
	if h.handleFunc != nil {
		return h.handleFunc(ctx, conn, msg)
	}
	return nil
}

func BenchmarkHTTPTransport(b *testing.B) {
	// Create HTTP server
	server := transporthttp.New(transporthttp.Options{
		Address: ":0",
		Logger:  types.NewNoOpLogger(),
	})

	// Create test server
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := ts.Client().Get(ts.URL)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkWebSocketTransport(b *testing.B) {
	// Create WebSocket server
	server := websocket.New(websocket.Options{
		Logger: logger.NewZapLogger(zap.NewNop(), &types.LoggerConfig{MinLevel: types.LogLevelInfo}),
	})

	// Register echo handler
	server.RegisterHandler("test", &testHandler{
		handleFunc: func(ctx context.Context, conn *wsgorilla.Conn, msg websocket.Message) error {
			return conn.WriteJSON(websocket.Message{
				Type:    "response",
				Payload: msg.Payload,
			})
		},
	})

	// Create test server
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Create WebSocket client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := wsgorilla.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Send message
		err := conn.WriteJSON(websocket.Message{
			Type:    "test",
			Payload: json.RawMessage(`{"test":"data"}`),
		})
		if err != nil {
			b.Fatal(err)
		}

		// Read response
		var response websocket.Message
		err = conn.ReadJSON(&response)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebSocketConcurrent(b *testing.B) {
	// Create WebSocket server
	server := websocket.New(websocket.Options{
		Logger: logger.NewZapLogger(zap.NewNop(), &types.LoggerConfig{MinLevel: types.LogLevelInfo}),
	})

	// Register echo handler
	server.RegisterHandler("test", &testHandler{
		handleFunc: func(ctx context.Context, conn *wsgorilla.Conn, msg websocket.Message) error {
			return conn.WriteJSON(websocket.Message{
				Type:    "response",
				Payload: msg.Payload,
			})
		},
	})

	// Create test server
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Run benchmark with multiple connections
	b.RunParallel(func(pb *testing.PB) {
		// Create WebSocket client
		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
		conn, _, err := wsgorilla.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()

		for pb.Next() {
			// Send message
			err := conn.WriteJSON(websocket.Message{
				Type:    "test",
				Payload: json.RawMessage(`{"test":"data"}`),
			})
			if err != nil {
				b.Fatal(err)
			}

			// Read response
			var response websocket.Message
			err = conn.ReadJSON(&response)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkHTTPConcurrentConnections(b *testing.B) {
	// Create logger
	testLogger := logger.NewZapLogger(zap.NewNop(), &types.LoggerConfig{MinLevel: types.LogLevelInfo})

	server := transporthttp.New(transporthttp.Options{
		Address:      "localhost:0",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Logger:       testLogger,
	})

	ts := httptest.NewServer(server)
	defer ts.Close()

	msg := map[string]string{"content": "test"}
	body, _ := json.Marshal(msg)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		for pb.Next() {
			req, err := http.NewRequest("POST", ts.URL+"/test", bytes.NewReader(body))
			if err != nil {
				b.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}
