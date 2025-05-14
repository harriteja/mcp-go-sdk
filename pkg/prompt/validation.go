package prompt

import (
	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/harriteja/mcp-go-sdk/pkg/validation/core"
)

// ValidationRule represents a rule for validating prompt parameters
type ValidationRule struct {
	core.Rule
	// ParameterName is the name of the parameter this rule applies to
	ParameterName string `json:"parameterName"`
}

// ValidationError represents a validation error specific to prompts
type ValidationError struct {
	core.Error
	// ParameterName is the name of the invalid parameter
	ParameterName string `json:"parameterName"`
}

// ValidationResult represents the result of validating a prompt
type ValidationResult struct {
	// Valid indicates if the validation passed
	Valid bool `json:"valid"`

	// Errors contains validation errors
	Errors []ValidationError `json:"errors,omitempty"`

	// Warnings contains validation warnings
	Warnings []ValidationError `json:"warnings,omitempty"`

	// Info contains informational validation messages
	Info []ValidationError `json:"info,omitempty"`
}

// Validator defines the interface for validating prompts
type Validator interface {
	// ValidatePrompt validates a prompt definition
	ValidatePrompt(prompt *types.ExtendedPrompt) (*ValidationResult, error)

	// ValidateParameters validates prompt parameters
	ValidateParameters(prompt *types.ExtendedPrompt, parameters map[string]interface{}) (*ValidationResult, error)

	// AddValidationRule adds a custom validation rule
	AddValidationRule(rule ValidationRule) error

	// RemoveValidationRule removes a validation rule
	RemoveValidationRule(ruleID string) error
}

// DefaultValidator provides a default implementation of the Validator interface
type DefaultValidator struct {
	validator core.Validator
}

// NewValidator creates a new DefaultValidator
func NewValidator() Validator {
	return &DefaultValidator{
		validator: core.NewValidator(),
	}
}

// convertToPromptResult converts a base validation result to a prompt validation result
func convertToPromptResult(result *core.Result) *ValidationResult {
	if result == nil {
		return &ValidationResult{Valid: false}
	}

	promptResult := &ValidationResult{
		Valid: result.Valid,
	}

	// Convert errors
	for _, err := range result.Errors {
		promptResult.Errors = append(promptResult.Errors, ValidationError{
			Error: err,
		})
	}

	// Convert warnings
	for _, warn := range result.Warnings {
		promptResult.Warnings = append(promptResult.Warnings, ValidationError{
			Error: warn,
		})
	}

	// Convert info
	for _, info := range result.Info {
		promptResult.Info = append(promptResult.Info, ValidationError{
			Error: info,
		})
	}

	return promptResult
}

// ValidatePrompt validates a prompt definition
func (v *DefaultValidator) ValidatePrompt(prompt *types.ExtendedPrompt) (*ValidationResult, error) {
	result, err := v.validator.ValidateValue(prompt, prompt.ValidationRules)
	if err != nil {
		return nil, err
	}
	return convertToPromptResult(result), nil
}

// ValidateParameters validates prompt parameters
func (v *DefaultValidator) ValidateParameters(prompt *types.ExtendedPrompt, parameters map[string]interface{}) (*ValidationResult, error) {
	result, err := v.validator.ValidateValue(parameters, prompt.ValidationRules)
	if err != nil {
		return nil, err
	}
	return convertToPromptResult(result), nil
}

// AddValidationRule adds a custom validation rule
func (v *DefaultValidator) AddValidationRule(rule ValidationRule) error {
	return v.validator.AddRule(rule.Rule)
}

// RemoveValidationRule removes a validation rule
func (v *DefaultValidator) RemoveValidationRule(ruleID string) error {
	return v.validator.RemoveRule(ruleID)
}
