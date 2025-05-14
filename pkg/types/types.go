package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// Implementation represents server or client implementation information
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities represents capabilities supported by the client
type ClientCapabilities struct {
	Roots        *RootsCapability       `json:"roots,omitempty"`
	Sampling     *SamplingCapability    `json:"sampling,omitempty"`
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

// ServerCapabilities represents capabilities supported by the server
type ServerCapabilities struct {
	Prompts      *PromptsCapability     `json:"prompts,omitempty"`
	Resources    *ResourcesCapability   `json:"resources,omitempty"`
	Tools        *ToolsCapability       `json:"tools,omitempty"`
	Logging      *LoggingCapability     `json:"logging,omitempty"`
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

// RootsCapability represents root-related capabilities
type RootsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// SamplingCapability represents sampling-related capabilities
type SamplingCapability struct {
	// Add sampling-specific fields as needed
}

// PromptsCapability represents prompt-related capabilities
type PromptsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ResourcesCapability represents resource-related capabilities
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

// ToolsCapability represents tool-related capabilities
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// LoggingCapability represents logging-related capabilities
type LoggingCapability struct {
	// Add logging-specific fields as needed
}

// InitializeRequestParams represents parameters for initialization request
type InitializeRequestParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      *Implementation    `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}

// Error represents an MCP error
type Error struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// NewError creates a new Error instance
func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithData creates a new Error instance with additional data
func NewErrorWithData(code int, message string, data map[string]interface{}) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// Error implements the error interface
func (e Error) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("MCP error %d: %s (data: %v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// IsError checks if an error is an MCP error
func IsError(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	if mcpErr, ok := err.(*Error); ok {
		return mcpErr, true
	}
	return nil, false
}

// Parameter represents a parameter in a tool's input schema
type Parameter struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Parameters represents a tool's input schema
type Parameters struct {
	Type       string               `json:"type"`
	Properties map[string]Parameter `json:"properties"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  *Parameters            `json:"parameters,omitempty"`
	InputSchema json.RawMessage        `json:"inputSchema,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType"`
}

// ResourceTemplate represents an MCP resource template
type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Prompt represents an MCP prompt
type Prompt struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// Message represents a message in the MCP protocol
type Message struct {
	Role     string                 `json:"role"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ModelPreferences represents preferences for model selection
type ModelPreferences struct {
	Model      string                 `json:"model,omitempty"`
	Provider   string                 `json:"provider,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// CreateMessageRequest represents a request to create a message
type CreateMessageRequest struct {
	Messages         []Message              `json:"messages"`
	ModelPreferences *ModelPreferences      `json:"modelPreferences,omitempty"`
	SystemPrompt     string                 `json:"systemPrompt,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	MaxTokens        int                    `json:"maxTokens"`
	StopSequences    []string               `json:"stopSequences,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// CreateMessageResponse represents a response to create message request
type CreateMessageResponse struct {
	Message    Message                `json:"message"`
	StopReason string                 `json:"stopReason,omitempty"`
	Usage      map[string]interface{} `json:"usage,omitempty"`
}

// Session represents an MCP session
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

// InitializeRequest represents an initialize request in MCP
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

// InitializeResponse represents an initialize response in MCP
type InitializeResponse struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}
