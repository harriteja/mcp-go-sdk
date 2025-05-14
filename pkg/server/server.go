package server

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// HandlerFunc represents a generic handler function type
type HandlerFunc[T any] func(context.Context) (T, error)

// Server represents an MCP server instance
type Server struct {
	mu sync.RWMutex

	name         string
	version      string
	instructions string
	logger       *zap.Logger

	// Handlers
	listToolsHandler             HandlerFunc[[]types.Tool]
	callToolHandler              func(context.Context, string, map[string]interface{}) (interface{}, error)
	listPromptsHandler           HandlerFunc[[]types.Prompt]
	getPromptHandler             func(context.Context, string, map[string]interface{}) (*types.Prompt, error)
	listResourcesHandler         HandlerFunc[[]types.Resource]
	readResourceHandler          func(context.Context, string) ([]byte, string, error)
	listResourceTemplatesHandler HandlerFunc[[]types.ResourceTemplate]

	// Session management
	sessions map[string]*Session
}

// Options represents server configuration options
type Options struct {
	Name         string
	Version      string
	Instructions string
	Logger       *zap.Logger
}

// New creates a new MCP server instance
func New(opts Options) *Server {
	logger := opts.Logger
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}

	return &Server{
		name:         opts.Name,
		version:      opts.Version,
		instructions: opts.Instructions,
		logger:       logger,
		sessions:     make(map[string]*Session),
	}
}

// OnListTools registers a handler for listing tools
func (s *Server) OnListTools(handler HandlerFunc[[]types.Tool]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listToolsHandler = handler
}

// OnCallTool registers a handler for calling tools
func (s *Server) OnCallTool(handler func(context.Context, string, map[string]interface{}) (interface{}, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callToolHandler = handler
}

// OnListPrompts registers a handler for listing prompts
func (s *Server) OnListPrompts(handler HandlerFunc[[]types.Prompt]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listPromptsHandler = handler
}

// OnGetPrompt registers a handler for getting a prompt
func (s *Server) OnGetPrompt(handler func(context.Context, string, map[string]interface{}) (*types.Prompt, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getPromptHandler = handler
}

// OnListResources registers a handler for listing resources
func (s *Server) OnListResources(handler HandlerFunc[[]types.Resource]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listResourcesHandler = handler
}

// OnReadResource registers a handler for reading resources
func (s *Server) OnReadResource(handler func(context.Context, string) ([]byte, string, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readResourceHandler = handler
}

// OnListResourceTemplates registers a handler for listing resource templates
func (s *Server) OnListResourceTemplates(handler HandlerFunc[[]types.ResourceTemplate]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listResourceTemplatesHandler = handler
}

// Initialize handles client initialization
func (s *Server) Initialize(ctx context.Context, req *types.InitializeRequest) (*types.InitializeResponse, error) {
	s.logger.Info("Initializing server",
		zap.String("client_name", req.ClientInfo.Name),
		zap.String("client_version", req.ClientInfo.Version),
		zap.String("protocol_version", req.ProtocolVersion))

	// Create new session
	sessionID := uuid.New().String()
	session := NewSession(sessionID, req)

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	// Return server capabilities
	return &types.InitializeResponse{
		ProtocolVersion: req.ProtocolVersion,
		ServerInfo: types.Implementation{
			Name:    s.name,
			Version: s.version,
		},
		Capabilities: types.ServerCapabilities{
			Tools:     &types.ToolsCapability{},
			Prompts:   &types.PromptsCapability{},
			Resources: &types.ResourcesCapability{},
		},
		Instructions: s.instructions,
	}, nil
}

// getSession retrieves a session by ID
func (s *Server) getSession(sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	return session, nil
}

// ListTools handles the list tools request
func (s *Server) ListTools(ctx context.Context) ([]types.Tool, error) {
	if s.listToolsHandler == nil {
		return nil, errors.New("list tools handler not registered")
	}
	return s.listToolsHandler(ctx)
}

// CallTool handles the call tool request
func (s *Server) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	if s.callToolHandler == nil {
		return nil, errors.New("call tool handler not registered")
	}
	return s.callToolHandler(ctx, name, args)
}

// ListPrompts handles the list prompts request
func (s *Server) ListPrompts(ctx context.Context) ([]types.Prompt, error) {
	if s.listPromptsHandler == nil {
		return nil, errors.New("list prompts handler not registered")
	}
	return s.listPromptsHandler(ctx)
}

// GetPrompt handles the get prompt request
func (s *Server) GetPrompt(ctx context.Context, name string, args map[string]interface{}) (*types.Prompt, error) {
	if s.getPromptHandler == nil {
		return nil, errors.New("get prompt handler not registered")
	}
	return s.getPromptHandler(ctx, name, args)
}

// ListResources handles the list resources request
func (s *Server) ListResources(ctx context.Context) ([]types.Resource, error) {
	if s.listResourcesHandler == nil {
		return nil, errors.New("list resources handler not registered")
	}
	return s.listResourcesHandler(ctx)
}

// ReadResource handles the read resource request
func (s *Server) ReadResource(ctx context.Context, uri string) ([]byte, string, error) {
	if s.readResourceHandler == nil {
		return nil, "", errors.New("read resource handler not registered")
	}
	return s.readResourceHandler(ctx, uri)
}

// ListResourceTemplates handles the list resource templates request
func (s *Server) ListResourceTemplates(ctx context.Context) ([]types.ResourceTemplate, error) {
	if s.listResourceTemplatesHandler == nil {
		return nil, errors.New("list resource templates handler not registered")
	}
	return s.listResourceTemplatesHandler(ctx)
}
