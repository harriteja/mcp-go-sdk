package transport

import (
	"fmt"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/server/transport/http"
	"github.com/harriteja/mcp-go-sdk/pkg/server/transport/websocket"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Type represents a transport type
type Type string

const (
	// HTTP transport type
	HTTP Type = "http"
	// WebSocket transport type
	WebSocket Type = "websocket"
)

// Factory creates transport instances
type Factory struct {
	logger types.Logger
}

// NewFactory creates a new transport factory
func NewFactory() *Factory {
	return &Factory{logger: logger.GetDefaultLogger()}
}

// Create creates a new transport instance
func (f *Factory) Create(transportType Type, opts Options) (Transport, error) {
	switch transportType {
	case HTTP:
		return http.New(http.Options{
			Address:      opts.Address,
			ReadTimeout:  defaultTimeout,
			WriteTimeout: defaultTimeout,
			IdleTimeout:  defaultIdleTimeout,
			Logger:       f.logger,
		}), nil

	case WebSocket:
		return websocket.New(websocket.Options{
			ReadBufferSize:   defaultBufferSize,
			WriteBufferSize:  defaultBufferSize,
			HandshakeTimeout: defaultTimeout,
			Logger:           f.logger,
		}), nil

	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transportType)
	}
}

// Default configuration values
const (
	defaultTimeout     = 30 * time.Second
	defaultIdleTimeout = 120 * time.Second
	defaultBufferSize  = 1024
)
