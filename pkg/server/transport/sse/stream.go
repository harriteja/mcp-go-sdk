package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// StreamHandler handles SSE streaming
type StreamHandler struct {
	w       http.ResponseWriter
	flusher http.Flusher
	logger  *zap.Logger
	mu      sync.RWMutex
	closed  bool
}

// NewStreamHandler creates a new SSE stream handler
func NewStreamHandler(w http.ResponseWriter, logger *zap.Logger) (*StreamHandler, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	return &StreamHandler{
		w:       w,
		flusher: flusher,
		logger:  logger,
	}, nil
}

// Write implements io.Writer
func (h *StreamHandler) Write(data []byte) (n int, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return 0, fmt.Errorf("stream closed")
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

// WriteChunk writes a chunk as an SSE event
func (h *StreamHandler) WriteChunk(chunk types.StreamChunk) error {
	if h.closed {
		return fmt.Errorf("stream closed")
	}

	data, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk: %w", err)
	}

	// Write SSE event
	fmt.Fprintf(h.w, "event: %s\n", chunk.Type)
	fmt.Fprintf(h.w, "data: %s\n\n", data)
	h.flusher.Flush()

	h.logger.Debug("Wrote SSE event",
		zap.String("type", string(chunk.Type)),
		zap.Int("size", len(data)),
	)

	return nil
}

// WriteData writes data to the stream
func (h *StreamHandler) WriteData(data []byte) error {
	if h.closed {
		return fmt.Errorf("stream closed")
	}

	chunk := types.StreamChunk{
		Type: types.StreamTypeData,
		Data: data,
	}
	return h.WriteChunk(chunk)
}

// WriteProgress writes a progress update as an SSE event
func (h *StreamHandler) WriteProgress(progress *types.Progress) error {
	if h.closed {
		return fmt.Errorf("stream closed")
	}

	chunk := types.StreamChunk{
		Type:     types.StreamTypeProgress,
		Progress: progress,
	}
	return h.WriteChunk(chunk)
}

// WriteError writes an error as an SSE event
func (h *StreamHandler) WriteError(err error) error {
	if h.closed {
		return fmt.Errorf("stream closed")
	}

	chunk := types.StreamChunk{
		Type:  types.StreamTypeError,
		Error: types.NewError(500, err.Error()),
	}
	return h.WriteChunk(chunk)
}

// WriteComplete writes a completion message as an SSE event
func (h *StreamHandler) WriteComplete() error {
	if h.closed {
		return fmt.Errorf("stream closed")
	}

	chunk := types.StreamChunk{
		Type: types.StreamTypeComplete,
	}
	return h.WriteChunk(chunk)
}

// IsClosed returns true if the stream is closed
func (h *StreamHandler) IsClosed() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.closed
}

// Complete marks the stream as complete
func (h *StreamHandler) Complete() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return fmt.Errorf("stream closed")
	}

	chunk := types.StreamChunk{
		Type: types.StreamTypeComplete,
	}
	return h.writeChunk(&chunk)
}

// Close closes the stream
func (h *StreamHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil
	}

	h.closed = true
	return nil
}

// writeChunk writes a chunk as an SSE event
func (h *StreamHandler) writeChunk(chunk *types.StreamChunk) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk: %w", err)
	}

	// Write SSE event
	fmt.Fprintf(h.w, "event: %s\n", chunk.Type)
	fmt.Fprintf(h.w, "data: %s\n\n", data)
	h.flusher.Flush()

	h.logger.Debug("Wrote SSE event",
		zap.String("type", string(chunk.Type)),
		zap.Int("size", len(data)),
	)

	return nil
}

// StreamReader reads SSE events
type StreamReader struct {
	r      *http.Request
	logger *zap.Logger
	mu     sync.RWMutex
	closed bool
}

// NewStreamReader creates a new SSE stream reader
func NewStreamReader(r *http.Request, logger *zap.Logger) *StreamReader {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &StreamReader{
		r:      r,
		logger: logger,
	}
}

// Read reads the next SSE event
func (r *StreamReader) Read() (*types.StreamChunk, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, fmt.Errorf("stream closed")
	}

	// SSE reading is handled by the client-side EventSource
	// This is just a placeholder as the server doesn't read SSE events
	return nil, fmt.Errorf("SSE reading not supported on server side")
}

// Close closes the stream
func (r *StreamReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true
	return nil
}

// StreamPipe creates an SSE stream pipe
type StreamPipe struct {
	reader *StreamReader
	writer *StreamHandler
}

// NewStreamPipe creates a new SSE stream pipe
func NewStreamPipe(w http.ResponseWriter, r *http.Request, logger *zap.Logger) (types.StreamPipe, error) {
	writer, err := NewStreamHandler(w, logger)
	if err != nil {
		return nil, err
	}

	return &StreamPipe{
		reader: NewStreamReader(r, logger),
		writer: writer,
	}, nil
}

func (p *StreamPipe) Reader() types.StreamReader { return p.reader }
func (p *StreamPipe) Writer() types.StreamWriter { return p.writer }
func (p *StreamPipe) Close() error {
	p.reader.Close()
	return p.writer.Close()
}
