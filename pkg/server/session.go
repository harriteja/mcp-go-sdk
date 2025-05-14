package server

import (
	"sync"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Session represents a client session
type Session struct {
	mu sync.RWMutex

	id        string
	createdAt time.Time
	expiresAt time.Time

	clientInfo         types.Implementation
	clientCapabilities types.ClientCapabilities
	protocolVersion    string
}

// NewSession creates a new session
func NewSession(id string, req *types.InitializeRequest) *Session {
	now := time.Now()
	return &Session{
		id:        id,
		createdAt: now,
		expiresAt: now.Add(24 * time.Hour), // Default 24h expiry

		clientInfo:         req.ClientInfo,
		clientCapabilities: req.Capabilities,
		protocolVersion:    req.ProtocolVersion,
	}
}

// ID returns the session ID
func (s *Session) ID() string {
	return s.id
}

// CreatedAt returns when the session was created
func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

// ExpiresAt returns when the session expires
func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}

// ClientInfo returns the client implementation info
func (s *Session) ClientInfo() types.Implementation {
	return s.clientInfo
}

// ClientCapabilities returns the client capabilities
func (s *Session) ClientCapabilities() types.ClientCapabilities {
	return s.clientCapabilities
}

// ProtocolVersion returns the protocol version
func (s *Session) ProtocolVersion() string {
	return s.protocolVersion
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.expiresAt)
}

// Extend extends the session expiry time
func (s *Session) Extend(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expiresAt = time.Now().Add(duration)
}
