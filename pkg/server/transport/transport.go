package transport

import (
	"context"
	"net/http"
)

// Transport defines the interface for MCP transports
type Transport interface {
	http.Handler
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	WriteError(w http.ResponseWriter, code int, message string)
	WriteJSON(w http.ResponseWriter, code int, data interface{})
}

// Handler defines the interface for transport-specific handlers
type Handler interface {
	// Handle processes incoming requests/messages
	Handle(ctx context.Context, data []byte) ([]byte, error)
}

// Middleware defines the interface for transport middleware
type Middleware interface {
	// Wrap wraps a handler with middleware functionality
	Wrap(next Handler) Handler
}

// ErrorHandler defines the interface for custom error handling
type ErrorHandler interface {
	// HandleError handles transport errors
	HandleError(ctx context.Context, err error) (int, interface{})
}

// Options represents common transport configuration options
type Options struct {
	// Address to listen on
	Address string

	// TLS configuration
	TLSCertFile string
	TLSKeyFile  string

	// Handler for processing requests
	Handler Handler

	// Middleware chain
	Middleware []Middleware

	// Custom error handler
	ErrorHandler ErrorHandler

	// Additional transport-specific options
	Options map[string]interface{}
}

// Error represents a transport error
type Error struct {
	Code    int
	Message string
	Details map[string]interface{}
}

func (e *Error) Error() string {
	return e.Message
}

// TransportType represents the type of transport
type TransportType string

const (
	// TransportTypeHTTP represents HTTP transport
	TransportTypeHTTP TransportType = "http"
	// TransportTypeWebSocket represents WebSocket transport
	TransportTypeWebSocket TransportType = "websocket"
	// TransportTypeSSE represents Server-Sent Events transport
	TransportTypeSSE TransportType = "sse"
	// TransportTypeStdIO represents Standard I/O transport
	TransportTypeStdIO TransportType = "stdio"
)

// TransportFactory creates transport instances
type TransportFactory interface {
	// Create creates a new transport instance
	Create(opts Options) (Transport, error)
	// Type returns the transport type
	Type() TransportType
}

// TransportRegistry manages transport factories
type TransportRegistry interface {
	// Register registers a transport factory
	Register(factory TransportFactory) error
	// Get gets a transport factory by type
	Get(transportType TransportType) (TransportFactory, error)
	// List lists all registered transport types
	List() []TransportType
}

// DefaultRegistry provides a default implementation of TransportRegistry
type DefaultRegistry struct {
	factories map[TransportType]TransportFactory
}

// NewRegistry creates a new DefaultRegistry
func NewRegistry() TransportRegistry {
	return &DefaultRegistry{
		factories: make(map[TransportType]TransportFactory),
	}
}

// Register implements TransportRegistry.Register
func (r *DefaultRegistry) Register(factory TransportFactory) error {
	if factory == nil {
		return &Error{
			Code:    http.StatusBadRequest,
			Message: "factory cannot be nil",
		}
	}

	transportType := factory.Type()
	if _, exists := r.factories[transportType]; exists {
		return &Error{
			Code:    http.StatusConflict,
			Message: "transport type already registered",
			Details: map[string]interface{}{
				"transportType": transportType,
			},
		}
	}

	r.factories[transportType] = factory
	return nil
}

// Get implements TransportRegistry.Get
func (r *DefaultRegistry) Get(transportType TransportType) (TransportFactory, error) {
	factory, exists := r.factories[transportType]
	if !exists {
		return nil, &Error{
			Code:    http.StatusNotFound,
			Message: "transport type not found",
			Details: map[string]interface{}{
				"transportType": transportType,
			},
		}
	}
	return factory, nil
}

// List implements TransportRegistry.List
func (r *DefaultRegistry) List() []TransportType {
	types := make([]TransportType, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	return types
}
