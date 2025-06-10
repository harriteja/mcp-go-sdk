package websocket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// ClientOptions represents client configuration options
type ClientOptions struct {
	// URL for the WebSocket server
	URL string
	// Headers to include in the connection request
	Headers http.Header
	// Timeout for connection attempts
	ConnectTimeout time.Duration
	// Logger instance
	Logger types.Logger
	// ReadBufferSize is the size of the buffer for reading messages
	ReadBufferSize int
	// WriteBufferSize is the size of the buffer for writing messages
	WriteBufferSize int
	// EnableCompression enables per-message compression
	EnableCompression bool
}

// Client represents a WebSocket client
type Client struct {
	url               *url.URL
	headers           http.Header
	conn              *websocket.Conn
	logger            types.Logger
	connectTimeout    time.Duration
	mutex             sync.Mutex
	readBufferSize    int
	writeBufferSize   int
	enableCompression bool
}

// NewClient creates a new WebSocket client
func NewClient(opts ClientOptions) (*Client, error) {
	if opts.Logger == nil {
		opts.Logger = logger.GetDefaultLogger()
	}
	if opts.ConnectTimeout == 0 {
		opts.ConnectTimeout = 10 * time.Second
	}
	if opts.ReadBufferSize == 0 {
		opts.ReadBufferSize = 1024
	}
	if opts.WriteBufferSize == 0 {
		opts.WriteBufferSize = 1024
	}

	u, err := url.Parse(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	return &Client{
		url:               u,
		headers:           opts.Headers,
		logger:            opts.Logger,
		connectTimeout:    opts.ConnectTimeout,
		readBufferSize:    opts.ReadBufferSize,
		writeBufferSize:   opts.WriteBufferSize,
		enableCompression: opts.EnableCompression,
	}, nil
}

// Connect establishes a WebSocket connection
func (c *Client) Connect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		return nil
	}

	c.logger.Info(ctx, "websocket", "client", fmt.Sprintf("Connecting to %s", c.url.String()))

	dialer := &websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  c.connectTimeout,
		ReadBufferSize:    c.readBufferSize,
		WriteBufferSize:   c.writeBufferSize,
		EnableCompression: c.enableCompression,
	}

	conn, resp, err := dialer.DialContext(ctx, c.url.String(), c.headers)
	if err != nil {
		if resp != nil {
			c.logger.Error(ctx, "websocket", "client", fmt.Sprintf("Failed to connect: %v, status: %d", err, resp.StatusCode))
		} else {
			c.logger.Error(ctx, "websocket", "client", fmt.Sprintf("Failed to connect: %v", err))
		}
		return err
	}

	c.conn = conn
	c.logger.Info(ctx, "websocket", "client", "Successfully connected")
	return nil
}

// Close closes the WebSocket connection
func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn == nil {
		return nil
	}

	ctx := context.Background()
	c.logger.Info(ctx, "websocket", "client", "Closing connection")

	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		c.logger.Warn(ctx, "websocket", "client", fmt.Sprintf("Error sending close message: %v", err))
	}

	if err := c.conn.Close(); err != nil {
		c.logger.Error(ctx, "websocket", "client", fmt.Sprintf("Error closing connection: %v", err))
		return err
	}

	c.conn = nil
	return nil
}

// ReadMessage reads a message from the WebSocket connection
func (c *Client) ReadMessage(ctx context.Context) (int, []byte, error) {
	c.mutex.Lock()
	if c.conn == nil {
		c.mutex.Unlock()
		return 0, nil, fmt.Errorf("not connected")
	}
	conn := c.conn
	c.mutex.Unlock()

	messageType, data, err := conn.ReadMessage()
	if err != nil {
		c.logger.Error(ctx, "websocket", "client", fmt.Sprintf("Error reading message: %v", err))
		return 0, nil, err
	}

	c.logger.Info(ctx, "websocket", "client", fmt.Sprintf("Received message type %d with %d bytes", messageType, len(data)))
	return messageType, data, nil
}

// WriteMessage writes a message to the WebSocket connection
func (c *Client) WriteMessage(ctx context.Context, messageType int, data []byte) error {
	c.mutex.Lock()
	if c.conn == nil {
		c.mutex.Unlock()
		return fmt.Errorf("not connected")
	}
	conn := c.conn
	c.mutex.Unlock()

	if err := conn.WriteMessage(messageType, data); err != nil {
		c.logger.Error(ctx, "websocket", "client", fmt.Sprintf("Error writing message: %v", err))
		return err
	}

	c.logger.Info(ctx, "websocket", "client", fmt.Sprintf("Sent message type %d with %d bytes", messageType, len(data)))
	return nil
}

// WriteJSON writes a JSON message to the WebSocket connection
func (c *Client) WriteJSON(ctx context.Context, v interface{}) error {
	c.mutex.Lock()
	if c.conn == nil {
		c.mutex.Unlock()
		return fmt.Errorf("not connected")
	}
	conn := c.conn
	c.mutex.Unlock()

	if err := conn.WriteJSON(v); err != nil {
		c.logger.Error(ctx, "websocket", "client", fmt.Sprintf("Error writing JSON: %v", err))
		return err
	}

	c.logger.Info(ctx, "websocket", "client", "Sent JSON message")
	return nil
}

// ReadJSON reads a JSON message from the WebSocket connection
func (c *Client) ReadJSON(ctx context.Context, v interface{}) error {
	c.mutex.Lock()
	if c.conn == nil {
		c.mutex.Unlock()
		return fmt.Errorf("not connected")
	}
	conn := c.conn
	c.mutex.Unlock()

	if err := conn.ReadJSON(v); err != nil {
		c.logger.Error(ctx, "websocket", "client", fmt.Sprintf("Error reading JSON: %v", err))
		return err
	}

	c.logger.Info(ctx, "websocket", "client", "Received JSON message")
	return nil
}
