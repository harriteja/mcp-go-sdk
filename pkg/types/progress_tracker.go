package types

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// DefaultTracker implements the ProgressTracker interface
type DefaultTracker struct {
	mu sync.RWMutex

	// Current progress state
	current *Progress

	// Subscribers receiving progress updates
	subscribers map[string]chan *Progress
}

// NewProgressTracker creates a new default progress tracker that implements ProgressTracker
func NewProgressTracker() ProgressTracker {
	return &DefaultTracker{
		subscribers: make(map[string]chan *Progress),
	}
}

// Start implements ProgressTracker.Start
func (t *DefaultTracker) Start(message string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.current != nil && t.current.State != ProgressStateCompleted && t.current.State != ProgressStateFailed {
		return errors.New("progress tracking already started")
	}

	progress := &Progress{
		ID:        uuid.New().String(),
		State:     ProgressStateStarted,
		Message:   message,
		Timestamp: time.Now(),
	}

	t.current = progress
	t.notifySubscribers(progress)
	return nil
}

// Update implements ProgressTracker.Update
func (t *DefaultTracker) Update(percentage float64, message string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.current == nil {
		return errors.New("progress tracking not started")
	}

	if percentage < 0 || percentage > 100 {
		return errors.New("percentage must be between 0 and 100")
	}

	progress := &Progress{
		ID:         t.current.ID,
		State:      ProgressStateInProgress,
		Message:    message,
		Percentage: percentage,
		Timestamp:  time.Now(),
	}

	t.current = progress
	t.notifySubscribers(progress)
	return nil
}

// Complete implements ProgressTracker.Complete
func (t *DefaultTracker) Complete(message string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.current == nil {
		return errors.New("progress tracking not started")
	}

	progress := &Progress{
		ID:         t.current.ID,
		State:      ProgressStateCompleted,
		Message:    message,
		Percentage: 100,
		Timestamp:  time.Now(),
	}

	t.current = progress
	t.notifySubscribers(progress)
	return nil
}

// Fail implements ProgressTracker.Fail
func (t *DefaultTracker) Fail(err error) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.current == nil {
		return errors.New("progress tracking not started")
	}

	progress := &Progress{
		ID:        t.current.ID,
		State:     ProgressStateFailed,
		Message:   err.Error(),
		Error:     NewError(500, err.Error()),
		Timestamp: time.Now(),
	}

	t.current = progress
	t.notifySubscribers(progress)
	return nil
}

// Current implements ProgressTracker.Current
func (t *DefaultTracker) Current() *Progress {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.current
}

// Subscribe implements ProgressTracker.Subscribe
func (t *DefaultTracker) Subscribe() (<-chan *Progress, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ch := make(chan *Progress, 10)
	id := uuid.New().String()
	t.subscribers[id] = ch

	// Send current progress state if available
	if t.current != nil {
		ch <- t.current
	}

	return ch, nil
}

// notifySubscribers sends progress updates to all subscribers
func (t *DefaultTracker) notifySubscribers(progress *Progress) {
	for _, ch := range t.subscribers {
		select {
		case ch <- progress:
		default:
			// Skip if subscriber's channel is full
		}
	}
}

// Unsubscribe removes a subscriber
func (t *DefaultTracker) Unsubscribe(ch <-chan *Progress) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for id, sub := range t.subscribers {
		if sub == ch {
			close(sub)
			delete(t.subscribers, id)
			return
		}
	}
}
