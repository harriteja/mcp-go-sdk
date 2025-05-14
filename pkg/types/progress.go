package types

import (
	"encoding/json"
	"time"
)

// ProgressState represents the state of a progress update
type ProgressState string

const (
	// ProgressStateStarted indicates the operation has started
	ProgressStateStarted ProgressState = "started"
	// ProgressStateInProgress indicates the operation is in progress
	ProgressStateInProgress ProgressState = "in_progress"
	// ProgressStateCompleted indicates the operation has completed successfully
	ProgressStateCompleted ProgressState = "completed"
	// ProgressStateFailed indicates the operation has failed
	ProgressStateFailed ProgressState = "failed"
)

// Progress represents a progress update
type Progress struct {
	// ID uniquely identifies this progress update
	ID string `json:"id"`

	// State represents the current state of the operation
	State ProgressState `json:"state"`

	// Message provides a human-readable description of the progress
	Message string `json:"message"`

	// Percentage indicates the completion percentage (0-100)
	Percentage float64 `json:"percentage"`

	// Details contains additional progress details
	Details map[string]interface{} `json:"details,omitempty"`

	// Error contains error details if State is ProgressStateFailed
	Error *Error `json:"error,omitempty"`

	// Timestamp when this progress update was created
	Timestamp time.Time `json:"timestamp"`
}

// ProgressTracker tracks progress for long-running operations
type ProgressTracker interface {
	// Start starts tracking progress with initial state
	Start(message string) error

	// Update updates the progress state
	Update(percentage float64, message string) error

	// Complete marks the operation as completed
	Complete(message string) error

	// Fail marks the operation as failed with an error
	Fail(err error) error

	// Current returns the current progress state
	Current() *Progress

	// Subscribe returns a channel that receives progress updates
	Subscribe() (<-chan *Progress, error)
}

// MarshalJSON implements json.Marshaler
func (p *Progress) MarshalJSON() ([]byte, error) {
	type Alias Progress
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(p),
		Timestamp: p.Timestamp.Format(time.RFC3339Nano),
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (p *Progress) UnmarshalJSON(data []byte) error {
	type Alias Progress
	aux := &struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	p.Timestamp, err = time.Parse(time.RFC3339Nano, aux.Timestamp)
	return err
}
