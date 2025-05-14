package streamable

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// EventStore interface for storing and retrieving events
type EventStore interface {
	StoreEvent(event *Event) error
	GetEvents(since string) ([]*Event, error)
}

// Event represents a stored event
type Event struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	Time time.Time       `json:"time"`
}

// Transport implements streamable HTTP transport
type Transport struct {
	sessionID           string
	eventStore          EventStore
	jsonResponseEnabled bool
	logger              *zap.Logger
	upgrader            websocket.Upgrader
	clients             sync.Map
}

// Options for configuring the transport
type Options struct {
	SessionID           string
	EventStore          EventStore
	JSONResponseEnabled bool
	Logger              *zap.Logger
}

// New creates a new streamable HTTP transport
func New(opts Options) *Transport {
	if opts.Logger == nil {
		opts.Logger, _ = zap.NewProduction()
	}

	return &Transport{
		sessionID:           opts.SessionID,
		eventStore:          opts.EventStore,
		jsonResponseEnabled: opts.JSONResponseEnabled,
		logger:              opts.Logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// ServeHTTP implements the http.Handler interface
func (t *Transport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		t.handleGet(w, r)
	case http.MethodPost:
		t.handlePost(w, r)
	case http.MethodDelete:
		t.handleDelete(w, r)
	default:
		t.handleUnsupported(w, r)
	}
}

func (t *Transport) handleGet(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", contentTypeSSE)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get last event ID for resuming
	lastEventID := r.Header.Get(headerLastEvent)
	if lastEventID != "" && t.eventStore != nil {
		if err := t.replayEvents(w, lastEventID); err != nil {
			t.logger.Error("Failed to replay events", zap.Error(err))
			http.Error(w, "Failed to replay events", http.StatusInternalServerError)
			return
		}
	}

	// Create client connection
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	client := &Client{
		ID:   clientID,
		Send: make(chan *Event, 256),
		Done: make(chan struct{}),
	}

	// Store client
	t.clients.Store(clientID, client)
	defer t.clients.Delete(clientID)

	// Start client writer
	go t.writeEvents(w, client)

	// Wait for client to disconnect
	<-client.Done
}

func (t *Transport) handlePost(w http.ResponseWriter, r *http.Request) {
	// Validate session
	if !t.validateSession(r) {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	// Read request body
	var msg json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create event
	event := &Event{
		ID:   fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type: "message",
		Data: msg,
		Time: time.Now(),
	}

	// Store event if event store is configured
	if t.eventStore != nil {
		if err := t.eventStore.StoreEvent(event); err != nil {
			t.logger.Error("Failed to store event", zap.Error(err))
			http.Error(w, "Failed to store event", http.StatusInternalServerError)
			return
		}
	}

	// Broadcast event to all clients
	t.broadcast(event)

	// Send response
	if t.jsonResponseEnabled {
		w.Header().Set("Content-Type", contentTypeJSON)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"id":     event.ID,
		}); err != nil {
			t.logger.Error("Failed to encode JSON response", zap.Error(err))
		}
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
}

func (t *Transport) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Validate session
	if !t.validateSession(r) {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	// Close all client connections
	t.clients.Range(func(key, value interface{}) bool {
		if client, ok := value.(*Client); ok {
			close(client.Done)
		}
		return true
	})

	w.WriteHeader(http.StatusNoContent)
}

func (t *Transport) handleUnsupported(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET, POST, DELETE")
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (t *Transport) validateSession(r *http.Request) bool {
	if t.sessionID == "" {
		return true
	}
	return r.Header.Get(headerSessionID) == t.sessionID
}

func (t *Transport) replayEvents(w http.ResponseWriter, since string) error {
	events, err := t.eventStore.GetEvents(since)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := t.writeEvent(w, event); err != nil {
			return err
		}
	}
	return nil
}

func (t *Transport) broadcast(event *Event) {
	t.clients.Range(func(key, value interface{}) bool {
		if client, ok := value.(*Client); ok {
			select {
			case client.Send <- event:
			default:
				t.logger.Warn("Client send buffer full, dropping event",
					zap.String("client_id", client.ID))
			}
		}
		return true
	})
}

func (t *Transport) writeEvents(w http.ResponseWriter, client *Client) {
	for {
		select {
		case event := <-client.Send:
			if err := t.writeEvent(w, event); err != nil {
				t.logger.Error("Failed to write event",
					zap.Error(err),
					zap.String("client_id", client.ID))
				close(client.Done)
				return
			}
		case <-client.Done:
			return
		}
	}
}

func (t *Transport) writeEvent(w http.ResponseWriter, event *Event) error {
	fmt.Fprintf(w, "id: %s\n", event.ID)
	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", string(event.Data))

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

// Client represents a connected client
type Client struct {
	ID   string
	Send chan *Event
	Done chan struct{}
}

const (
	// Headers
	headerSessionID = "mcp-session-id"
	headerLastEvent = "last-event-id"

	// Content types
	contentTypeJSON = "application/json"
	contentTypeSSE  = "text/event-stream"
)
