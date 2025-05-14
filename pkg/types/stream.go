package types

import (
	"encoding/json"
	"errors"
	"io"
	"sync"
)

var ErrStreamClosed = errors.New("stream closed")

// StreamType represents the type of stream data
type StreamType string

const (
	// StreamTypeData represents a data chunk
	StreamTypeData StreamType = "data"
	// StreamTypeProgress represents a progress update
	StreamTypeProgress StreamType = "progress"
	// StreamTypeError represents an error
	StreamTypeError StreamType = "error"
	// StreamTypeComplete represents stream completion
	StreamTypeComplete StreamType = "complete"
)

// StreamChunk represents a chunk of stream data
type StreamChunk struct {
	Type     StreamType `json:"type"`
	Data     []byte     `json:"data,omitempty"`
	Progress *Progress  `json:"progress,omitempty"`
	Error    *Error     `json:"error,omitempty"`
}

// StreamWriter is the interface for writing to a stream
type StreamWriter interface {
	io.Writer
	WriteChunk(chunk StreamChunk) error
	WriteData(data []byte) error
	WriteProgress(progress *Progress) error
	WriteError(err error) error
	WriteComplete() error
	Close() error
	IsClosed() bool
}

// StreamReader reads stream chunks
type StreamReader interface {
	// Read reads the next chunk
	Read() (*StreamChunk, error)

	// Close closes the stream
	Close() error
}

// StreamPipe creates a connected reader and writer
type StreamPipe interface {
	// Reader returns the reader end of the pipe
	Reader() StreamReader

	// Writer returns the writer end of the pipe
	Writer() StreamWriter

	// Close closes both ends of the pipe
	Close() error
}

// NewStreamPipe creates a new stream pipe
func NewStreamPipe() StreamPipe {
	pr, pw := io.Pipe()
	return &streamPipe{
		reader: &streamReader{pr: pr},
		writer: &streamWriter{
			writer:   pw,
			doneChan: make(chan struct{}),
		},
	}
}

type streamPipe struct {
	reader *streamReader
	writer *streamWriter
}

func (p *streamPipe) Reader() StreamReader { return p.reader }
func (p *streamPipe) Writer() StreamWriter { return p.writer }
func (p *streamPipe) Close() error {
	p.reader.Close()
	return p.writer.Close()
}

type streamReader struct {
	pr     *io.PipeReader
	dec    *json.Decoder
	closed bool
}

func (r *streamReader) Read() (*StreamChunk, error) {
	if r.closed {
		return nil, io.ErrClosedPipe
	}
	if r.dec == nil {
		r.dec = json.NewDecoder(r.pr)
	}
	chunk := &StreamChunk{}
	if err := r.dec.Decode(chunk); err != nil {
		return nil, err
	}
	return chunk, nil
}

func (r *streamReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	return r.pr.Close()
}

// streamWriter implements the StreamWriter interface
type streamWriter struct {
	writer   io.Writer
	encoder  *json.Encoder
	mu       sync.Mutex
	closed   bool
	doneChan chan struct{}
}

// Write implements io.Writer
func (w *streamWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, ErrStreamClosed
	}

	return w.writer.Write(p)
}

// WriteChunk writes a chunk to the stream
func (w *streamWriter) WriteChunk(chunk StreamChunk) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrStreamClosed
	}

	if w.encoder == nil {
		w.encoder = json.NewEncoder(w.writer)
	}

	return w.encoder.Encode(chunk)
}

// WriteData writes data to the stream
func (w *streamWriter) WriteData(data []byte) error {
	return w.WriteChunk(StreamChunk{
		Type: StreamTypeData,
		Data: data,
	})
}

// WriteProgress writes progress information to the stream
func (w *streamWriter) WriteProgress(progress *Progress) error {
	return w.WriteChunk(StreamChunk{
		Type:     StreamTypeProgress,
		Progress: progress,
	})
}

// WriteError writes an error to the stream
func (w *streamWriter) WriteError(err error) error {
	streamErr := NewError(500, err.Error())
	return w.WriteChunk(StreamChunk{
		Type:  StreamTypeError,
		Error: streamErr,
	})
}

// WriteComplete writes a completion message to the stream
func (w *streamWriter) WriteComplete() error {
	return w.WriteChunk(StreamChunk{
		Type: StreamTypeComplete,
	})
}

// Close closes the stream
func (w *streamWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true
	close(w.doneChan)
	return nil
}

// IsClosed returns true if the stream is closed
func (w *streamWriter) IsClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

// NewStreamWriter creates a new StreamWriter
func NewStreamWriter(writer io.Writer) StreamWriter {
	return &streamWriter{
		writer:   writer,
		doneChan: make(chan struct{}),
	}
}
