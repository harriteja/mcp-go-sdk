package session

import (
	"sync"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// InitializationState represents the current state of session initialization
type InitializationState int

const (
	NotInitialized InitializationState = iota
	Initializing
	Initialized
)

// ServerSession represents a server-side session with a client
type ServerSession struct {
	mu           sync.RWMutex
	state        InitializationState
	clientParams *types.InitializeRequestParams
	capabilities *types.ServerCapabilities
	serverInfo   *types.Implementation
	instructions string
}

// NewServerSession creates a new server session
func NewServerSession(serverInfo *types.Implementation, capabilities *types.ServerCapabilities, instructions string) *ServerSession {
	return &ServerSession{
		state:        NotInitialized,
		capabilities: capabilities,
		serverInfo:   serverInfo,
		instructions: instructions,
	}
}

// Initialize initializes the session with client parameters
func (s *ServerSession) Initialize(params *types.InitializeRequestParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == Initialized {
		return types.NewError(400, "Session already initialized")
	}

	s.state = Initializing
	s.clientParams = params
	s.state = Initialized
	return nil
}

// CheckClientCapability checks if the client supports a specific capability
func (s *ServerSession) CheckClientCapability(capability types.ClientCapabilities) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.clientParams == nil {
		return false
	}

	clientCaps := s.clientParams.Capabilities

	// Check roots capability
	if capability.Roots != nil {
		if clientCaps.Roots == nil {
			return false
		}
		if capability.Roots.ListChanged && !clientCaps.Roots.ListChanged {
			return false
		}
	}

	// Check sampling capability
	if capability.Sampling != nil {
		if clientCaps.Sampling == nil {
			return false
		}
	}

	// Check experimental capabilities
	if capability.Experimental != nil {
		if clientCaps.Experimental == nil {
			return false
		}
		for expKey, expValue := range capability.Experimental {
			if clientValue, ok := clientCaps.Experimental[expKey]; !ok || clientValue != expValue {
				return false
			}
		}
	}

	return true
}

// GetClientParams returns the client initialization parameters
func (s *ServerSession) GetClientParams() *types.InitializeRequestParams {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clientParams
}

// GetCapabilities returns the server capabilities
func (s *ServerSession) GetCapabilities() *types.ServerCapabilities {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.capabilities
}

// GetServerInfo returns the server implementation info
func (s *ServerSession) GetServerInfo() *types.Implementation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.serverInfo
}

// GetInstructions returns the server instructions
func (s *ServerSession) GetInstructions() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.instructions
}

// IsInitialized returns whether the session is initialized
func (s *ServerSession) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == Initialized
}
