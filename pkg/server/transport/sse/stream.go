package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Stream implements a Server-Sent Events stream
type Stream struct {
	clients      map[string]*Client
	register     chan *Client
	unregister   chan *Client
	broadcast    chan Event
	mu           sync.RWMutex
	logger       types.Logger
	maxClients   int
	clientCount  int
	clientIDFunc func(*http.Request) string
}

// StreamOptions defines options for the SSE stream
type StreamOptions struct {
	// Logger is the logger to use
	Logger types.Logger
	// MaxClients is the maximum number of clients allowed
	MaxClients int
	// ClientIDFunc generates a client ID from the request
	ClientIDFunc func(*http.Request) string
}

// NewStream creates a new SSE stream
func NewStream(opts StreamOptions) *Stream {
	if opts.Logger == nil {
		opts.Logger = logger.GetDefaultLogger()
	}
	if opts.MaxClients <= 0 {
		opts.MaxClients = 100
	}
	if opts.ClientIDFunc == nil {
		opts.ClientIDFunc = defaultClientIDFunc
	}

	s := &Stream{
		clients:      make(map[string]*Client),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		broadcast:    make(chan Event),
		logger:       opts.Logger,
		maxClients:   opts.MaxClients,
		clientIDFunc: opts.ClientIDFunc,
	}

	// Start the event handling loop
	go s.run()

	return s
}

// Event represents an SSE event
type Event struct {
	ID    string          `json:"id,omitempty"`
	Type  string          `json:"type,omitempty"`
	Data  json.RawMessage `json:"data"`
	Retry int             `json:"retry,omitempty"`
}

// Client represents a connected SSE client
type Client struct {
	ID      string
	Send    chan Event
	Closed  chan struct{}
	Writer  http.ResponseWriter
	Flush   http.Flusher
	Done    <-chan struct{}
	Request *http.Request
}

// run handles client connections and broadcasts
func (s *Stream) run() {
	ctx := context.Background()
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			if s.clientCount >= s.maxClients {
				s.logger.Warn(ctx, "sse", "stream", fmt.Sprintf("Rejecting client %s: max clients reached", client.ID))
				close(client.Closed)
				s.mu.Unlock()
				continue
			}

			s.clients[client.ID] = client
			s.clientCount++
			s.mu.Unlock()

			s.logger.Info(ctx, "sse", "stream", fmt.Sprintf("Client connected: %s (total: %d)", client.ID, s.clientCount))

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client.ID]; ok {
				delete(s.clients, client.ID)
				s.clientCount--
				close(client.Send)
				s.logger.Info(ctx, "sse", "stream", fmt.Sprintf("Client disconnected: %s (total: %d)", client.ID, s.clientCount))
			}
			s.mu.Unlock()

		case event := <-s.broadcast:
			s.mu.RLock()
			for id, client := range s.clients {
				select {
				case client.Send <- event:
					// Event sent successfully
				default:
					// Client buffer full, close connection
					s.logger.Warn(ctx, "sse", "stream", fmt.Sprintf("Client %s buffer full, closing connection", id))
					close(client.Closed)
				}
			}
			s.mu.RUnlock()
		}
	}
}

// Handler returns an HTTP handler for the SSE stream
func (s *Stream) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Check if streaming is supported
		flusher, ok := w.(http.Flusher)
		if !ok {
			s.logger.Error(ctx, "sse", "stream", "Streaming not supported")
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Generate client ID
		clientID := s.clientIDFunc(r)

		// Create client
		client := &Client{
			ID:      clientID,
			Send:    make(chan Event, 256),
			Closed:  make(chan struct{}),
			Writer:  w,
			Flush:   flusher,
			Done:    r.Context().Done(),
			Request: r,
		}

		// Register client
		s.register <- client

		// Start client writer
		go s.writeEvents(client)

		// Block until the client is done or closed
		select {
		case <-client.Done:
		case <-client.Closed:
		}

		// Unregister client
		s.unregister <- client
	}
}

// Broadcast sends an event to all clients
func (s *Stream) Broadcast(event Event) {
	s.broadcast <- event
}

// BroadcastData sends data to all clients
func (s *Stream) BroadcastData(eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	s.broadcast <- Event{
		Type: eventType,
		Data: jsonData,
	}

	return nil
}

// ClientCount returns the current number of connected clients
func (s *Stream) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clientCount
}

// writeEvents writes events to a client
func (s *Stream) writeEvents(client *Client) {
	ctx := client.Request.Context()
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error(ctx, "sse", "stream", fmt.Sprintf("Panic in writeEvents: %v", r))
		}
	}()

	for {
		select {
		case event, ok := <-client.Send:
			if !ok {
				return
			}

			if err := writeEvent(client.Writer, event); err != nil {
				s.logger.Error(ctx, "sse", "stream", fmt.Sprintf("Error writing event: %v", err))
				close(client.Closed)
				return
			}

			client.Flush.Flush()

		case <-client.Done:
			return

		case <-client.Closed:
			return
		}
	}
}

// writeEvent writes a single event to the response writer
func writeEvent(w http.ResponseWriter, event Event) error {
	if event.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
			return err
		}
	}

	if event.Type != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event.Type); err != nil {
			return err
		}
	}

	if event.Retry > 0 {
		if _, err := fmt.Fprintf(w, "retry: %d\n", event.Retry); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "data: %s\n\n", event.Data); err != nil {
		return err
	}

	return nil
}

// defaultClientIDFunc generates a default client ID
func defaultClientIDFunc(r *http.Request) string {
	return fmt.Sprintf("client-%p", r)
}
