package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/validation/core"
)

// PromptVersion represents a semantic version for prompts
type PromptVersion struct {
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
	Patch      int    `json:"patch"`
	PreRelease string `json:"preRelease,omitempty"`
}

// String returns the version as a string
func (v PromptVersion) String() string {
	version := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		version += "-" + v.PreRelease
	}
	return version
}

// Compare compares two versions and returns:
// -1 if v < other
//
//	0 if v == other
//	1 if v > other
func (v PromptVersion) Compare(other PromptVersion) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	if v.PreRelease == other.PreRelease {
		return 0
	}
	if v.PreRelease == "" {
		return 1
	}
	if other.PreRelease == "" {
		return -1
	}
	if v.PreRelease < other.PreRelease {
		return -1
	}
	return 1
}

// PromptArgument represents an argument that can be passed to a prompt
type PromptArgument struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Required    bool            `json:"required"`
	Schema      json.RawMessage `json:"schema,omitempty"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string          `json:"role"` // "user" or "assistant"
	Content json.RawMessage `json:"content"`
}

// ContentType represents the type of content in a message
type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeImage    ContentType = "image"
	ContentTypeResource ContentType = "resource"
)

// TextContent represents text content in a message
type TextContent struct {
	Type ContentType `json:"type"`
	Text string      `json:"text"`
}

// ImageContent represents image content in a message
type ImageContent struct {
	Type   ContentType `json:"type"`
	URI    string      `json:"uri"`
	Format string      `json:"format,omitempty"`
}

// ResourceContent represents an embedded resource in a message
type ResourceContent struct {
	Type     ContentType `json:"type"`
	URI      string      `json:"uri"`
	MimeType string      `json:"mimeType"`
	Content  []byte      `json:"content,omitempty"`
}

// ExtendedPrompt represents a prompt template that can be rendered with parameters
type ExtendedPrompt struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Version     PromptVersion          `json:"version"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`

	// Dependencies are other prompts this prompt depends on
	Dependencies []PromptDependency `json:"dependencies,omitempty"`

	// ValidationRules defines rules for validating prompt parameters
	ValidationRules []core.Rule `json:"validationRules,omitempty"`

	// Schema defines the JSON schema for prompt parameters
	Schema json.RawMessage `json:"schema,omitempty"`
}

// PromptDependency represents a dependency on another prompt
type PromptDependency struct {
	// ID is the ID of the dependent prompt
	ID string `json:"id"`

	// Version specifies the required version
	Version PromptVersion `json:"version"`

	// Optional indicates if this dependency is optional
	Optional bool `json:"optional"`

	// Parameters maps this prompt's parameters to the dependent prompt
	Parameters map[string]string `json:"parameters,omitempty"`

	// OutputMapping maps the dependent prompt's output to this prompt's input
	OutputMapping map[string]string `json:"outputMapping,omitempty"`
}

// PromptChain represents a chain of prompts
type PromptChain struct {
	// ID uniquely identifies this chain
	ID string `json:"id"`

	// Name is a human-readable name for this chain
	Name string `json:"name"`

	// Description describes the purpose of this chain
	Description string `json:"description"`

	// Prompts are the prompts in this chain, in execution order
	Prompts []PromptChainStep `json:"prompts"`

	// Parameters defines the parameters for this chain
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// OutputMapping maps prompt outputs to chain outputs
	OutputMapping map[string]string `json:"outputMapping,omitempty"`

	// CreatedAt is when this chain was created
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when this chain was last updated
	UpdatedAt time.Time `json:"updatedAt"`
}

// PromptChainStep represents a step in a prompt chain
type PromptChainStep struct {
	// PromptID is the ID of the prompt to execute
	PromptID string `json:"promptId"`

	// Version specifies the required prompt version
	Version PromptVersion `json:"version"`

	// Parameters maps chain parameters to prompt parameters
	Parameters map[string]string `json:"parameters,omitempty"`

	// DependsOn lists the IDs of steps this step depends on
	DependsOn []string `json:"dependsOn,omitempty"`

	// Condition specifies when this step should execute
	Condition string `json:"condition,omitempty"`

	// OutputMapping maps this step's output to chain parameters
	OutputMapping map[string]string `json:"outputMapping,omitempty"`
}

// RenderOptions represents options for rendering prompts
type RenderOptions struct {
	// Context is the context for rendering
	Context map[string]interface{} `json:"context,omitempty"`

	// Stream indicates if the response should be streamed
	Stream bool `json:"stream"`

	// MaxTokens limits the number of tokens in the response
	MaxTokens int `json:"maxTokens,omitempty"`

	// Temperature controls response randomness (0-1)
	Temperature float64 `json:"temperature,omitempty"`

	// StopSequences are sequences that stop generation
	StopSequences []string `json:"stopSequences,omitempty"`

	// ModelPreferences specifies model selection preferences
	ModelPreferences map[string]interface{} `json:"modelPreferences,omitempty"`
}

// RenderResult represents the result of rendering a prompt
type RenderResult struct {
	// Messages are the rendered messages
	Messages []PromptMessage `json:"messages"`

	// Usage contains token usage information
	Usage map[string]interface{} `json:"usage,omitempty"`

	// StopReason indicates why generation stopped
	StopReason string `json:"stopReason,omitempty"`

	// Error contains any error that occurred
	Error error `json:"-"`
}

// PromptRenderer defines the interface for rendering prompts
type PromptRenderer interface {
	// RenderPrompt renders a prompt with the given arguments
	RenderPrompt(prompt *ExtendedPrompt, args map[string]interface{}, opts *RenderOptions) (*RenderResult, error)

	// RenderPromptStream renders a prompt and streams the response
	RenderPromptStream(prompt *ExtendedPrompt, args map[string]interface{}, opts *RenderOptions) (<-chan *RenderResult, error)

	// RenderChain renders a prompt chain
	RenderChain(chain *PromptChain, args map[string]interface{}, opts *RenderOptions) (*RenderResult, error)

	// RenderChainStream renders a prompt chain and streams the response
	RenderChainStream(chain *PromptChain, args map[string]interface{}, opts *RenderOptions) (<-chan *RenderResult, error)
}

// PromptStore defines the interface for managing prompts
type PromptStore interface {
	// GetPrompt gets a prompt by ID and version
	GetPrompt(id string, version *PromptVersion) (*ExtendedPrompt, error)

	// ListPrompts lists all prompts
	ListPrompts() ([]*ExtendedPrompt, error)

	// CreatePrompt creates a new prompt
	CreatePrompt(prompt *ExtendedPrompt) error

	// UpdatePrompt updates an existing prompt
	UpdatePrompt(prompt *ExtendedPrompt) error

	// DeletePrompt deletes a prompt
	DeletePrompt(id string, version *PromptVersion) error

	// GetChain gets a prompt chain by ID
	GetChain(id string) (*PromptChain, error)

	// ListChains lists all prompt chains
	ListChains() ([]*PromptChain, error)

	// CreateChain creates a new prompt chain
	CreateChain(chain *PromptChain) error

	// UpdateChain updates an existing prompt chain
	UpdateChain(chain *PromptChain) error

	// DeleteChain deletes a prompt chain
	DeleteChain(id string) error
}
