package prompts

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrPromptNotFound = errors.New("prompt not found")
	ErrInvalidPrompt  = errors.New("invalid prompt")
)

// PromptTemplate represents a template for generating prompts
type PromptTemplate struct {
	ID          string                 `json:"id"`
	Template    string                 `json:"template"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Description string                 `json:"description,omitempty"`
}

// Manager defines the interface for prompt management
type Manager interface {
	// GetPrompt retrieves a prompt template by ID
	GetPrompt(ctx context.Context, id string) (*PromptTemplate, error)

	// CreatePrompt creates a new prompt template
	CreatePrompt(ctx context.Context, template *PromptTemplate) error

	// UpdatePrompt updates an existing prompt template
	UpdatePrompt(ctx context.Context, template *PromptTemplate) error

	// DeletePrompt removes a prompt template
	DeletePrompt(ctx context.Context, id string) error

	// ListPrompts returns all available prompt templates
	ListPrompts(ctx context.Context) ([]*PromptTemplate, error)

	// RenderPrompt processes a template with given parameters
	RenderPrompt(ctx context.Context, id string, params map[string]interface{}) (string, error)
}

// memoryManager implements Manager interface with in-memory storage
type memoryManager struct {
	mu       sync.RWMutex
	prompts  map[string]*PromptTemplate
	renderer TemplateRenderer
}

// NewMemoryManager creates a new in-memory prompt manager
func NewMemoryManager() Manager {
	return &memoryManager{
		prompts:  make(map[string]*PromptTemplate),
		renderer: NewDefaultRenderer(),
	}
}

func (m *memoryManager) GetPrompt(ctx context.Context, id string) (*PromptTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if prompt, ok := m.prompts[id]; ok {
		return prompt, nil
	}
	return nil, ErrPromptNotFound
}

func (m *memoryManager) CreatePrompt(ctx context.Context, template *PromptTemplate) error {
	if template == nil || template.ID == "" || template.Template == "" {
		return ErrInvalidPrompt
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.prompts[template.ID]; exists {
		return errors.New("prompt already exists")
	}

	now := time.Now()
	template.CreatedAt = now
	template.UpdatedAt = now
	m.prompts[template.ID] = template
	return nil
}

func (m *memoryManager) UpdatePrompt(ctx context.Context, template *PromptTemplate) error {
	if template == nil || template.ID == "" {
		return ErrInvalidPrompt
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.prompts[template.ID]; !exists {
		return ErrPromptNotFound
	}

	template.UpdatedAt = time.Now()
	m.prompts[template.ID] = template
	return nil
}

func (m *memoryManager) DeletePrompt(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.prompts[id]; !exists {
		return ErrPromptNotFound
	}

	delete(m.prompts, id)
	return nil
}

func (m *memoryManager) ListPrompts(ctx context.Context) ([]*PromptTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prompts := make([]*PromptTemplate, 0, len(m.prompts))
	for _, p := range m.prompts {
		prompts = append(prompts, p)
	}
	return prompts, nil
}

func (m *memoryManager) RenderPrompt(ctx context.Context, id string, params map[string]interface{}) (string, error) {
	prompt, err := m.GetPrompt(ctx, id)
	if err != nil {
		return "", err
	}

	return m.renderer.Render(prompt.Template, params)
}
