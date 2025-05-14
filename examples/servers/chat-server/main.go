package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/harriteja/mcp-go-sdk/pkg/server/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type ChatMessage struct {
	Username string `json:"username"`
	Message  string `json:"message"`
	Time     string `json:"time"`
}

type ChatServer struct {
	logger     *zap.Logger
	clients    map[*websocket.Conn]string
	broadcast  chan ChatMessage
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	upgrader   websocket.Upgrader
}

func NewChatServer(logger *zap.Logger) *ChatServer {
	return &ChatServer{
		logger:     logger,
		clients:    make(map[*websocket.Conn]string),
		broadcast:  make(chan ChatMessage),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
	}
}

func (s *ChatServer) Start() {
	for {
		select {
		case client := <-s.register:
			s.clients[client] = ""
			s.logger.Info("New client connected")

		case client := <-s.unregister:
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				s.logger.Info("Client disconnected")
			}

		case msg := <-s.broadcast:
			for client := range s.clients {
				err := client.WriteJSON(msg)
				if err != nil {
					s.logger.Error("Failed to send message", zap.Error(err))
					client.Close()
					delete(s.clients, client)
				}
			}
		}
	}
}

func (s *ChatServer) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	msg.Time = time.Now().Format(time.RFC3339)
	s.broadcast <- msg

	w.WriteHeader(http.StatusOK)
}

func (s *ChatServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	s.register <- conn

	defer func() {
		s.unregister <- conn
		conn.Close()
	}()

	for {
		var msg ChatMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		msg.Time = time.Now().Format(time.RFC3339)
		s.broadcast <- msg
	}
}

func main() {
	// Parse command line flags
	addr := flag.String("addr", ":8080", "Server address")
	flag.Parse()

	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create chat server
	chatServer := NewChatServer(logger)
	go chatServer.Start()

	// Create metrics registry
	registry := prometheus.NewRegistry()

	// Create server mux
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/chat", chatServer.HandleHTTP)
	mux.HandleFunc("/ws", chatServer.HandleWebSocket)

	// Add metrics middleware
	handler := middleware.MetricsMiddleware(middleware.MetricsConfig{
		Registry:     registry,
		Subsystem:    "chat",
		ExcludePaths: []string{"/metrics"},
	})(mux)

	// Create HTTP server
	server := &http.Server{
		Addr:         *addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server
	go func() {
		logger.Info("Starting server", zap.String("addr", *addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Failed to stop server gracefully", zap.Error(err))
	}
}
