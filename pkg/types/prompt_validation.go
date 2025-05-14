package types

import (
	"github.com/harriteja/mcp-go-sdk/pkg/validation/core"
)

// PromptValidationError extends core.Error with prompt-specific fields
type PromptValidationError struct {
	core.Error
	// Parameter is the name of the invalid parameter
	Parameter string `json:"parameter"`
}

// PromptValidationResult extends core.Result with prompt-specific fields
type PromptValidationResult struct {
	// Valid indicates if the prompt is valid
	Valid bool `json:"valid"`

	// Errors contains validation errors
	Errors []PromptValidationError `json:"errors,omitempty"`

	// Warnings contains validation warnings
	Warnings []PromptValidationError `json:"warnings,omitempty"`

	// Info contains informational validation messages
	Info []PromptValidationError `json:"info,omitempty"`
}

// PromptValidationRule extends core.Rule with prompt-specific fields
type PromptValidationRule struct {
	core.Rule
	// Parameter is the name of the parameter this rule applies to
	Parameter string `json:"parameter"`
}

// PromptValidator defines the interface for validating prompts
type PromptValidator interface {
	// ValidatePrompt validates a prompt definition
	ValidatePrompt(prompt *ExtendedPrompt) (*PromptValidationResult, error)

	// ValidateParameters validates prompt parameters
	ValidateParameters(prompt *ExtendedPrompt, parameters map[string]interface{}) (*PromptValidationResult, error)

	// AddValidationRule adds a custom validation rule
	AddValidationRule(rule PromptValidationRule) error

	// RemoveValidationRule removes a validation rule
	RemoveValidationRule(ruleID string) error
}
