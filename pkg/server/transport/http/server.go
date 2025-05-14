package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/server/transport/response"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Server represents an HTTP transport server
type Server struct {
	server   *http.Server
	handlers map[string]http.Handler
	logger   types.Logger
	mu       sync.RWMutex
	mux      *http.ServeMux
}

// Options represents server configuration options
type Options struct {
	// Address to listen on
	Address string
	// ReadTimeout for requests
	ReadTimeout time.Duration
	// WriteTimeout for responses
	WriteTimeout time.Duration
	// IdleTimeout for keep-alive connections
	IdleTimeout time.Duration
	// Logger instance
	Logger types.Logger
}

// New creates a new HTTP transport server
func New(opts Options) *Server {
	if opts.Logger == nil {
		opts.Logger = types.NewNoOpLogger()
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = 30 * time.Second
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = 30 * time.Second
	}
	if opts.IdleTimeout == 0 {
		opts.IdleTimeout = 120 * time.Second
	}

	s := &Server{
		handlers: make(map[string]http.Handler),
		logger:   opts.Logger,
	}

	mux := http.NewServeMux()
	s.server = &http.Server{
		Addr:         opts.Address,
		Handler:      mux,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		IdleTimeout:  opts.IdleTimeout,
	}
	s.mux = mux

	return s
}

// RegisterHandler registers a new HTTP handler for the given path
func (s *Server) RegisterHandler(path string, handler http.Handler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if path is already registered
	if _, exists := s.handlers[path]; exists {
		return fmt.Errorf("handler already registered for path: %s", path)
	}

	s.handlers[path] = handler
	s.mux.Handle(path, handler)
	return nil
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting HTTP server", types.LogField{Key: "address", Value: s.server.Addr})
	return s.server.ListenAndServe()
}

// StartTLS starts the HTTPS server
func (s *Server) StartTLS(certFile, keyFile string) error {
	s.logger.Info("Starting HTTPS server",
		types.LogField{Key: "address", Value: s.server.Addr},
		types.LogField{Key: "cert_file", Value: certFile},
		types.LogField{Key: "key_file", Value: keyFile},
	)
	return s.server.ListenAndServeTLS(certFile, keyFile)
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}

// WriteError writes an error response
func (s *Server) WriteError(w http.ResponseWriter, code int, message string) {
	response.WriteError(w, code, message)
}

// WriteJSON writes a JSON response
func (s *Server) WriteJSON(w http.ResponseWriter, code int, data interface{}) {
	response.WriteJSON(w, code, data)
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Implement HTTP request handling
	w.WriteHeader(http.StatusOK)
}
