package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/validation/core"
)

// ResourceTemplateVersion represents a semantic version
type ResourceTemplateVersion struct {
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
	Patch      int    `json:"patch"`
	PreRelease string `json:"preRelease,omitempty"`
}

// String returns the version as a string
func (v ResourceTemplateVersion) String() string {
	version := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		version += "-" + v.PreRelease
	}
	return version
}

// ExtendedResourceTemplate extends ResourceTemplate with additional validation features
type ExtendedResourceTemplate struct {
	ResourceTemplate

	// Version represents the template version
	Version ResourceTemplateVersion `json:"version"`

	// Schema defines the JSON schema for resource validation
	Schema json.RawMessage `json:"schema"`

	// ValidationRules defines additional validation rules
	ValidationRules []core.Rule `json:"validationRules"`

	// CreatedAt is when this template was created
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when this template was last updated
	UpdatedAt time.Time `json:"updatedAt"`

	// Metadata contains additional template metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceValidator validates resources against templates
type ResourceValidator interface {
	// ValidateResource validates a resource against a template
	ValidateResource(resource interface{}, template *ExtendedResourceTemplate) (*core.Result, error)

	// AddValidationRule adds a custom validation rule
	AddValidationRule(rule core.Rule) error

	// RemoveValidationRule removes a validation rule
	RemoveValidationRule(ruleID string) error
}

// ResourceTemplateStore manages resource templates
type ResourceTemplateStore interface {
	// GetTemplate gets a template by ID and version
	GetTemplate(id string, version *ResourceTemplateVersion) (*ExtendedResourceTemplate, error)

	// ListTemplates lists all templates
	ListTemplates() ([]*ExtendedResourceTemplate, error)

	// CreateTemplate creates a new template
	CreateTemplate(template *ExtendedResourceTemplate) error

	// UpdateTemplate updates an existing template
	UpdateTemplate(template *ExtendedResourceTemplate) error

	// DeleteTemplate deletes a template
	DeleteTemplate(id string, version *ResourceTemplateVersion) error
}
