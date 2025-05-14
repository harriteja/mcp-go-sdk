package websocket

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// StreamHandler handles WebSocket streaming
type StreamHandler struct {
	conn    *websocket.Conn
	logger  *zap.Logger
	mu      sync.RWMutex
	closed  atomic.Bool
	writeMu sync.Mutex // Additional mutex for concurrent writes
}

// NewStreamHandler creates a new WebSocket stream handler
func NewStreamHandler(conn *websocket.Conn, logger *zap.Logger) *StreamHandler {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &StreamHandler{
		conn:   conn,
		logger: logger,
	}
}

// WriteChunk writes a chunk to the stream
func (h *StreamHandler) WriteChunk(chunk types.StreamChunk) error {
	if h.closed.Load() {
		return fmt.Errorf("connection closed")
	}

	// Ensure only one goroutine can write at a time
	h.writeMu.Lock()
	defer h.writeMu.Unlock()

	if h.conn == nil {
		return fmt.Errorf("connection closed")
	}

	// Always marshal the entire chunk with type information
	var data []byte
	var err error

	if chunk.Type == types.StreamTypeData {
		// For data chunks, construct the JSON manually to preserve formatting
		data = []byte(fmt.Sprintf(`{"type":"%s","data":%s}`, chunk.Type, chunk.Data))
	} else {
		// For other chunk types, use standard JSON marshaling
		data, err = json.Marshal(chunk)
		if err != nil {
			return fmt.Errorf("failed to marshal chunk: %w", err)
		}
	}

	if err := h.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			h.closed.Store(true)
			return fmt.Errorf("connection closed")
		}
		return fmt.Errorf("failed to write message: %w", err)
	}

	h.logger.Debug("Wrote stream chunk",
		zap.String("type", string(chunk.Type)),
		zap.Int("size", len(data)),
	)

	return nil
}

// Write implements io.Writer
func (h *StreamHandler) Write(data []byte) (n int, err error) {
	if h.closed.Load() {
		return 0, fmt.Errorf("connection closed")
	}

	chunk := types.StreamChunk{
		Type: types.StreamTypeData,
		Data: data,
	}

	if err := h.WriteChunk(chunk); err != nil {
		return 0, err
	}

	return len(data), nil
}

// WriteProgress writes progress information to the stream
func (h *StreamHandler) WriteProgress(progress *types.Progress) error {
	if h.closed.Load() {
		return fmt.Errorf("connection closed")
	}

	chunk := types.StreamChunk{
		Type:     types.StreamTypeProgress,
		Progress: progress,
	}
	return h.WriteChunk(chunk)
}

// WriteError writes an error to the stream
func (h *StreamHandler) WriteError(err error) error {
	if h.closed.Load() {
		return fmt.Errorf("connection closed")
	}

	chunk := types.StreamChunk{
		Type:  types.StreamTypeError,
		Error: types.NewError(500, err.Error()),
	}
	return h.WriteChunk(chunk)
}

// WriteComplete writes a completion message to the stream
func (h *StreamHandler) WriteComplete() error {
	if h.closed.Load() {
		return fmt.Errorf("connection closed")
	}

	chunk := types.StreamChunk{
		Type: types.StreamTypeComplete,
	}
	return h.WriteChunk(chunk)
}

// WriteData writes data to the stream
func (h *StreamHandler) WriteData(data []byte) error {
	if h.closed.Load() {
		return fmt.Errorf("connection closed")
	}

	chunk := types.StreamChunk{
		Type: types.StreamTypeData,
		Data: data,
	}
	return h.WriteChunk(chunk)
}

// Close closes the WebSocket connection
func (h *StreamHandler) Close() error {
	if h.closed.Swap(true) {
		return nil // Already closed
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conn != nil {
		// Send close message with a short deadline
		deadline := time.Now().Add(100 * time.Millisecond)
		_ = h.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			deadline)

		// Wait for the close message to be sent and any pending reads to complete
		time.Sleep(50 * time.Millisecond)

		// Close the underlying connection
		conn := h.conn
		h.conn = nil
		return conn.Close()
	}
	return nil
}

// StreamReader reads stream chunks from a WebSocket connection
type StreamReader struct {
	conn   *websocket.Conn
	logger *zap.Logger
	mu     sync.RWMutex
	closed atomic.Bool
	readMu sync.Mutex // Additional mutex for concurrent reads
}

// NewStreamReader creates a new WebSocket stream reader
func NewStreamReader(conn *websocket.Conn, logger *zap.Logger) *StreamReader {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &StreamReader{
		conn:   conn,
		logger: logger,
	}
}

// Read reads the next chunk from the WebSocket connection
func (r *StreamReader) Read() (*types.StreamChunk, error) {
	if r.closed.Load() {
		return nil, fmt.Errorf("connection closed")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.conn == nil {
		return nil, fmt.Errorf("connection closed")
	}

	// Ensure only one goroutine can read at a time
	r.readMu.Lock()
	defer r.readMu.Unlock()

	_, data, err := r.conn.ReadMessage()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			r.closed.Store(true)
			return nil, fmt.Errorf("connection closed")
		}
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	chunk := &types.StreamChunk{}
	if err := json.Unmarshal(data, chunk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chunk: %w", err)
	}

	r.logger.Debug("Read stream chunk",
		zap.String("type", string(chunk.Type)),
		zap.Int("size", len(data)),
	)

	return chunk, nil
}

// Close closes the WebSocket connection
func (r *StreamReader) Close() error {
	if r.closed.Swap(true) {
		return nil // Already closed
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		// Try to send close message, but don't fail if it doesn't work
		_ = r.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		err := r.conn.Close()
		r.conn = nil
		return err
	}
	return nil
}

// StreamPipe combines a reader and writer for bidirectional streaming
type StreamPipe struct {
	reader *StreamReader
	writer *StreamHandler
}

// NewStreamPipe creates a new WebSocket stream pipe
func NewStreamPipe(conn *websocket.Conn, logger *zap.Logger) types.StreamPipe {
	return &StreamPipe{
		reader: NewStreamReader(conn, logger),
		writer: NewStreamHandler(conn, logger),
	}
}

// Reader returns the stream reader
func (p *StreamPipe) Reader() types.StreamReader { return p.reader }

// Writer returns the stream writer
func (p *StreamPipe) Writer() types.StreamWriter { return p.writer }

// Close closes both the reader and writer
func (p *StreamPipe) Close() error {
	// Close writer first to send close message
	if err := p.writer.Close(); err != nil {
		return err
	}

	// Mark reader as closed but don't close connection again
	p.reader.closed.Store(true)
	p.reader.mu.Lock()
	p.reader.conn = nil
	p.reader.mu.Unlock()

	return nil
}

// IsClosed returns true if the stream is closed
func (h *StreamHandler) IsClosed() bool {
	return h.closed.Load()
}
