package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Event represents a stored event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      json.RawMessage        `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Store defines the interface for event storage
type Store interface {
	// StoreEvent stores an event
	StoreEvent(ctx context.Context, event *Event) error

	// GetEvents retrieves events since a specific event ID
	GetEvents(ctx context.Context, since string) ([]*Event, error)

	// GetEvent retrieves a specific event
	GetEvent(ctx context.Context, id string) (*Event, error)

	// DeleteEvent deletes a specific event
	DeleteEvent(ctx context.Context, id string) error

	// PurgeEvents deletes events older than the specified duration
	PurgeEvents(ctx context.Context, olderThan time.Duration) error
}

// MemoryStore implements an in-memory event store
type MemoryStore struct {
	events map[string]*Event
	mu     sync.RWMutex
}

// NewMemoryStore creates a new in-memory event store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events: make(map[string]*Event),
	}
}

// StoreEvent implements Store.StoreEvent
func (s *MemoryStore) StoreEvent(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event.ID == "" {
		return fmt.Errorf("event ID is required")
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	s.events[event.ID] = event
	return nil
}

// GetEvents implements Store.GetEvents
func (s *MemoryStore) GetEvents(ctx context.Context, since string) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var events []*Event
	var sinceTime time.Time

	if since != "" {
		sinceEvent, ok := s.events[since]
		if !ok {
			return nil, fmt.Errorf("event not found: %s", since)
		}
		sinceTime = sinceEvent.Timestamp
	}

	// Collect all events after the since time
	for _, event := range s.events {
		if since == "" || event.Timestamp.After(sinceTime) {
			events = append(events, event)
		}
	}

	// Sort events by timestamp
	sortEvents(events)

	return events, nil
}

// GetEvent implements Store.GetEvent
func (s *MemoryStore) GetEvent(ctx context.Context, id string) (*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	event, ok := s.events[id]
	if !ok {
		return nil, fmt.Errorf("event not found: %s", id)
	}

	return event, nil
}

// DeleteEvent implements Store.DeleteEvent
func (s *MemoryStore) DeleteEvent(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[id]; !ok {
		return fmt.Errorf("event not found: %s", id)
	}

	delete(s.events, id)
	return nil
}

// PurgeEvents implements Store.PurgeEvents
func (s *MemoryStore) PurgeEvents(ctx context.Context, olderThan time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	threshold := time.Now().Add(-olderThan)

	for id, event := range s.events {
		if event.Timestamp.Before(threshold) {
			delete(s.events, id)
		}
	}

	return nil
}

// Helper function to sort events by timestamp
func sortEvents(events []*Event) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
}
