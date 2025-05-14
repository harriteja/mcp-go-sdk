package context

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/server/session"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Context represents the context for MCP operations
type Context struct {
	ctx       context.Context
	session   *session.ServerSession
	requestID string
	clientID  string
	progress  *progressInfo
	mu        sync.RWMutex
	tracker   types.ProgressTracker
	stream    types.StreamPipe
}

type progressInfo struct {
	current float64
	total   *float64
}

// NewContext creates a new MCP context
func NewContext(ctx context.Context, sess *session.ServerSession, requestID, clientID string) *Context {
	return &Context{
		ctx:       ctx,
		session:   sess,
		requestID: requestID,
		clientID:  clientID,
		tracker:   types.NewProgressTracker(),
		stream:    types.NewStreamPipe(),
	}
}

// Context returns the underlying context.Context
func (c *Context) Context() context.Context {
	return c.ctx
}

// Session returns the server session
func (c *Context) Session() *session.ServerSession {
	return c.session
}

// RequestID returns the current request ID
func (c *Context) RequestID() string {
	return c.requestID
}

// ClientID returns the client ID
func (c *Context) ClientID() string {
	return c.clientID
}

// ReportProgress reports the current progress
func (c *Context) ReportProgress(current float64, total *float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.progress = &progressInfo{
		current: current,
		total:   total,
	}

	// Convert to percentage for the tracker
	var percentage float64
	if total != nil && *total > 0 {
		percentage = (current / *total) * 100
	} else {
		percentage = current
	}

	// Update the tracker
	if err := c.tracker.Update(percentage, fmt.Sprintf("Progress: %.2f", current)); err != nil {
		return err
	}

	// Write progress to stream if available
	if c.stream != nil {
		return c.stream.Writer().WriteProgress(c.tracker.Current())
	}

	return nil
}

// GetProgress returns the current progress
func (c *Context) GetProgress() *progressInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.progress
}

// Stream returns the streaming pipe
func (c *Context) Stream() types.StreamPipe {
	return c.stream
}

// Write writes data to the stream
func (c *Context) Write(data []byte) (n int, err error) {
	if c.stream == nil {
		return 0, fmt.Errorf("no stream available")
	}
	if err := c.stream.Writer().WriteData(data); err != nil {
		return 0, err
	}
	return len(data), nil
}

// WriteError writes an error to the stream
func (c *Context) WriteError(err error) error {
	if c.stream == nil {
		return fmt.Errorf("no stream available")
	}
	return c.stream.Writer().WriteError(err)
}

// CompleteStream marks the stream as complete
func (c *Context) CompleteStream() error {
	if c.stream == nil {
		return fmt.Errorf("no stream available")
	}
	return c.stream.Writer().WriteComplete()
}

// CloseStream closes the stream
func (c *Context) CloseStream() error {
	if c.stream == nil {
		return nil
	}
	return c.stream.Close()
}

// ReadResource reads a resource by URI
func (c *Context) ReadResource(uri string) (io.ReadCloser, string, error) {
	if c.session == nil {
		return nil, "", fmt.Errorf("no session available")
	}

	// Implementation depends on how resources are stored and accessed
	// This is a placeholder that should be implemented based on your needs
	return nil, "", fmt.Errorf("not implemented")
}

// Info logs an informational message
func (c *Context) Info(format string, args ...interface{}) {
	// Implementation depends on how we want to handle logging
	// This could be through the session, a logger, or other mechanisms
}

// Debug logs a debug message
func (c *Context) Debug(format string, args ...interface{}) {
	// Implementation depends on how we want to handle logging
}

// Warning logs a warning message
func (c *Context) Warning(format string, args ...interface{}) {
	// Implementation depends on how we want to handle logging
}

// Error logs an error message
func (c *Context) Error(format string, args ...interface{}) {
	// Implementation depends on how we want to handle logging
}

// WithValue returns a new Context with the provided key-value pair
func (c *Context) WithValue(key, val interface{}) *Context {
	newCtx := &Context{
		ctx:       context.WithValue(c.ctx, key, val),
		session:   c.session,
		requestID: c.requestID,
		clientID:  c.clientID,
		progress:  c.progress,
		tracker:   c.tracker,
		stream:    c.stream,
	}
	return newCtx
}

// Value returns the value associated with the given key
func (c *Context) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}

// Done returns a channel that's closed when the context is done
func (c *Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err returns the error that caused the Done channel to close
func (c *Context) Err() error {
	return c.ctx.Err()
}

// Deadline returns the time when this context will be cancelled, if any
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

// StartProgress starts progress tracking
func (c *Context) StartProgress(message string) error {
	if err := c.tracker.Start(message); err != nil {
		return err
	}

	// Write initial progress to stream if available
	if c.stream != nil {
		return c.stream.Writer().WriteProgress(c.tracker.Current())
	}

	return nil
}

// UpdateProgress updates the progress state
func (c *Context) UpdateProgress(percentage float64, message string) error {
	if err := c.tracker.Update(percentage, message); err != nil {
		return err
	}

	// Write progress to stream if available
	if c.stream != nil {
		return c.stream.Writer().WriteProgress(c.tracker.Current())
	}

	return nil
}

// CompleteProgress marks the progress as completed
func (c *Context) CompleteProgress(message string) error {
	if err := c.tracker.Complete(message); err != nil {
		return err
	}

	// Write final progress to stream if available
	if c.stream != nil {
		return c.stream.Writer().WriteProgress(c.tracker.Current())
	}

	return nil
}

// FailProgress marks the progress as failed
func (c *Context) FailProgress(err error) error {
	if err := c.tracker.Fail(err); err != nil {
		return err
	}

	// Write failed progress to stream if available
	if c.stream != nil {
		return c.stream.Writer().WriteProgress(c.tracker.Current())
	}

	return nil
}

// Progress returns the current progress state
func (c *Context) Progress() *types.Progress {
	return c.tracker.Current()
}

// SubscribeProgress subscribes to progress updates
func (c *Context) SubscribeProgress() (<-chan *types.Progress, error) {
	return c.tracker.Subscribe()
}
