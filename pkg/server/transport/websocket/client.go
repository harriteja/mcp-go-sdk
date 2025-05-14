package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Client represents a WebSocket client
type Client struct {
	conn     *websocket.Conn
	handlers map[string]Handler
	logger   *zap.Logger
	mu       sync.RWMutex
	done     chan struct{}
	url      string
	headers  http.Header
	dialer   websocket.Dialer
}

// ClientOptions represents client configuration options
type ClientOptions struct {
	// URL of the WebSocket server
	URL string
	// Headers to include in the connection request
	Headers http.Header
	// HandshakeTimeout for the connection
	HandshakeTimeout time.Duration
	// Logger instance
	Logger *zap.Logger
}

// NewClient creates a new WebSocket client
func NewClient(opts ClientOptions) (*Client, error) {
	if opts.Logger == nil {
		opts.Logger, _ = zap.NewProduction()
	}
	if opts.HandshakeTimeout == 0 {
		opts.HandshakeTimeout = 10 * time.Second
	}

	client := &Client{
		handlers: make(map[string]Handler),
		logger:   opts.Logger,
		done:     make(chan struct{}),
		url:      opts.URL,
		headers:  opts.Headers,
		dialer: websocket.Dialer{
			HandshakeTimeout: opts.HandshakeTimeout,
		},
	}

	// Connect to server
	if err := client.Connect(); err != nil {
		return nil, err
	}

	return client, nil
}

// Connect establishes a connection to the WebSocket server
func (c *Client) Connect() error {
	c.mu.Lock()
	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close()
	}

	// Create fresh done channel
	c.done = make(chan struct{})
	c.mu.Unlock()

	// Connect to server - do this outside the lock to avoid deadlock
	conn, _, err := c.dialer.Dial(c.url, c.headers)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}

	// Now that we have a connection, lock again to update the state
	c.mu.Lock()
	c.conn = conn
	doneCh := c.done // Capture the done channel while holding the lock
	c.mu.Unlock()

	// Start message handler with the captured done channel
	go c.handleMessages(doneCh, conn)

	return nil
}

// IsConnected returns true if the client is connected to the server
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil
}

// SendMessage sends a message of a specific type with payload and waits for a response
func (c *Client) SendMessage(ctx context.Context, msgType string, payload json.RawMessage) (*Message, error) {
	msg := Message{
		Type:    msgType,
		Payload: payload,
	}
	return c.SendAndWait(ctx, msg, "response")
}

// RegisterHandler registers a handler for a message type
func (c *Client) RegisterHandler(msgType string, handler Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[msgType] = handler
}

// Send sends a message to the server
func (c *Client) Send(msg Message) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	c.logger.Debug("Sent message",
		zap.String("type", msg.Type),
		zap.String("payload", string(msg.Payload)),
	)

	return nil
}

// Close closes the WebSocket connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		// Already closed
		return nil
	default:
		close(c.done)
	}

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// handleMessages handles incoming messages
func (c *Client) handleMessages(doneCh chan struct{}, conn *websocket.Conn) {
	for {
		select {
		case <-doneCh:
			return
		default:
			var msg Message
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.logger.Error("WebSocket read error",
						zap.Error(err),
					)
				}
				return
			}

			c.logger.Debug("Received message",
				zap.String("type", msg.Type),
				zap.String("payload", string(msg.Payload)),
			)

			// Get handler for message type
			c.mu.RLock()
			handler, ok := c.handlers[msg.Type]
			c.mu.RUnlock()

			if !ok {
				c.logger.Warn("Unknown message type",
					zap.String("type", msg.Type),
				)
				continue
			}

			// Handle message in a goroutine
			go func(msgCopy Message, handlerCopy Handler) {
				if err := handlerCopy.HandleMessage(context.Background(), conn, msgCopy); err != nil {
					c.logger.Error("Failed to handle message",
						zap.Error(err),
						zap.String("type", msgCopy.Type),
					)
				}
			}(msg, handler)
		}
	}
}

// SendAndWait sends a message and waits for a response
func (c *Client) SendAndWait(ctx context.Context, msg Message, responseType string) (*Message, error) {
	// Create response channel
	responseCh := make(chan *Message, 1)
	errCh := make(chan error, 1)

	// Create temporary handler for response
	handler := &responseHandler{
		responseCh: responseCh,
		errCh:      errCh,
	}

	// Register handler
	c.RegisterHandler(responseType, handler)
	defer func() {
		c.mu.Lock()
		delete(c.handlers, responseType)
		c.mu.Unlock()
	}()

	// Send message
	if err := c.Send(msg); err != nil {
		return nil, err
	}

	// Wait for response or context cancellation
	select {
	case response := <-responseCh:
		return response, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// responseHandler handles response messages for SendAndWait
type responseHandler struct {
	responseCh chan *Message
	errCh      chan error
}

func (h *responseHandler) HandleMessage(ctx context.Context, conn *websocket.Conn, msg Message) error {
	select {
	case h.responseCh <- &msg:
	default:
		h.errCh <- fmt.Errorf("response channel full")
	}
	return nil
}
