package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Handler handles WebSocket messages
type Handler interface {
	// HandleMessage processes incoming messages
	HandleMessage(ctx context.Context, conn *websocket.Conn, msg Message) error
}

// Server represents a WebSocket transport server
type Server struct {
	upgrader websocket.Upgrader
	handlers map[string]Handler
	logger   types.Logger
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	conns    map[*websocket.Conn]struct{}
	connsMu  sync.RWMutex
}

// Options represents server configuration options
type Options struct {
	// ReadBufferSize for WebSocket connections
	ReadBufferSize int
	// WriteBufferSize for WebSocket connections
	WriteBufferSize int
	// HandshakeTimeout for WebSocket upgrade
	HandshakeTimeout time.Duration
	// Logger instance
	Logger types.Logger
	// CheckOrigin function for WebSocket upgrade
	CheckOrigin func(*http.Request) bool
}

// New creates a new WebSocket transport server
func New(opts Options) *Server {
	if opts.Logger == nil {
		opts.Logger = types.NewNoOpLogger()
	}
	if opts.ReadBufferSize == 0 {
		opts.ReadBufferSize = 1024
	}
	if opts.WriteBufferSize == 0 {
		opts.WriteBufferSize = 1024
	}
	if opts.HandshakeTimeout == 0 {
		opts.HandshakeTimeout = 10 * time.Second
	}
	if opts.CheckOrigin == nil {
		opts.CheckOrigin = func(r *http.Request) bool { return true }
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		upgrader: websocket.Upgrader{
			ReadBufferSize:   opts.ReadBufferSize,
			WriteBufferSize:  opts.WriteBufferSize,
			HandshakeTimeout: opts.HandshakeTimeout,
			CheckOrigin:      opts.CheckOrigin,
		},
		handlers: make(map[string]Handler),
		logger:   opts.Logger,
		ctx:      ctx,
		cancel:   cancel,
		conns:    make(map[*websocket.Conn]struct{}),
	}
}

// RegisterHandler registers a handler for a message type
func (s *Server) RegisterHandler(msgType string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[msgType] = handler
}

// Start implements Transport.Start
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info(ctx, "websocket", "start", "Starting WebSocket transport server")
	return nil
}

// Stop implements Transport.Stop
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info(ctx, "websocket", "stop", "Stopping WebSocket transport server")

	// Cancel the server context
	s.cancel()

	// Close all active connections
	s.connsMu.Lock()
	for conn := range s.conns {
		// Send close message with a short deadline
		if err := conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server shutting down"),
			time.Now().Add(time.Second)); err != nil {
			s.logger.Error(ctx, "websocket", "stop", "Failed to send close message: "+err.Error())
		}
		conn.Close()
		delete(s.conns, conn)
	}
	s.connsMu.Unlock()

	return nil
}

// HandleConnection handles a WebSocket connection
func (s *Server) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Check if server is shutting down
	if s.ctx.Err() != nil {
		http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error(r.Context(), "websocket", "upgrade", "Failed to upgrade connection: "+err.Error()+" from "+r.RemoteAddr)
		return
	}

	// Register connection
	s.connsMu.Lock()
	s.conns[conn] = struct{}{}
	s.connsMu.Unlock()

	// Ensure connection is removed when closed
	defer func() {
		s.connsMu.Lock()
		delete(s.conns, conn)
		s.connsMu.Unlock()
		conn.Close()
	}()

	s.logger.Info(r.Context(), "websocket", "connection", "New WebSocket connection from "+r.RemoteAddr)

	// Create context for the connection
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	// Handle messages
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read message
			var msg Message
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					s.logger.Error(ctx, "websocket", "read", "WebSocket read error: "+err.Error()+" from "+r.RemoteAddr)
				}
				return
			}

			// Get handler for message type
			s.mu.RLock()
			handler, ok := s.handlers[msg.Type]
			s.mu.RUnlock()

			if !ok {
				s.logger.Warn(ctx, "websocket", "handler", "Unknown message type: "+msg.Type+" from "+r.RemoteAddr)
				if err := conn.WriteJSON(Message{
					Type:    "error",
					Payload: json.RawMessage(fmt.Sprintf(`{"message":"unknown message type: %s"}`, msg.Type)),
				}); err != nil {
					s.logger.Error(ctx, "websocket", "write", "Failed to write error message: "+err.Error()+" to "+r.RemoteAddr)
				}
				continue
			}

			// Handle message
			if err := handler.HandleMessage(ctx, conn, msg); err != nil {
				s.logger.Error(ctx, "websocket", "handle", "Failed to handle message: "+err.Error()+" of type "+msg.Type+" from "+r.RemoteAddr)
				if err := conn.WriteJSON(Message{
					Type:    "error",
					Payload: json.RawMessage(fmt.Sprintf(`{"message":%q}`, err.Error())),
				}); err != nil {
					s.logger.Error(ctx, "websocket", "write", "Failed to write error message: "+err.Error()+" to "+r.RemoteAddr)
				}
			}
		}
	}
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.HandleConnection(w, r)
}

// WriteError writes an error response
func (s *Server) WriteError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(struct {
		Error *types.Error `json:"error"`
	}{
		Error: types.NewError(code, message),
	}); err != nil {
		s.logger.Error(context.Background(), "websocket", "writeError", "Failed to encode error response: "+err.Error())
	}
}

// WriteJSON writes a JSON response
func (s *Server) WriteJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(struct {
		Result interface{} `json:"result"`
	}{
		Result: data,
	}); err != nil {
		s.logger.Error(context.Background(), "websocket", "writeJSON", "Failed to encode JSON response: "+err.Error())
	}
}
