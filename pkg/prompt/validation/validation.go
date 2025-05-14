package validation

import (
	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/harriteja/mcp-go-sdk/pkg/validation/core"
)

// Rule represents a rule for validating prompt parameters
type Rule struct {
	core.Rule
	// ParameterName is the name of the parameter this rule applies to
	ParameterName string `json:"parameterName"`
}

// Error represents a validation error specific to prompts
type Error struct {
	core.Error
	// ParameterName is the name of the invalid parameter
	ParameterName string `json:"parameterName"`
}

// Result represents the result of validating a prompt
type Result struct {
	// Valid indicates if the validation passed
	Valid bool `json:"valid"`

	// Errors contains validation errors
	Errors []Error `json:"errors,omitempty"`

	// Warnings contains validation warnings
	Warnings []Error `json:"warnings,omitempty"`

	// Info contains informational validation messages
	Info []Error `json:"info,omitempty"`
}

// PromptValidator defines the interface for validating prompts
type PromptValidator interface {
	// ValidatePrompt validates a prompt definition
	ValidatePrompt(prompt *types.ExtendedPrompt) (*core.Result, error)

	// ValidateParameters validates prompt parameters
	ValidateParameters(prompt *types.ExtendedPrompt, parameters map[string]interface{}) (*core.Result, error)

	// AddRule adds a custom validation rule
	AddRule(rule core.Rule) error

	// RemoveRule removes a validation rule
	RemoveRule(ruleID string) error
}

// DefaultPromptValidator provides a default implementation of PromptValidator
type DefaultPromptValidator struct {
	validator core.Validator
	registry  core.ValidationRegistry
}

// NewPromptValidator creates a new DefaultPromptValidator
func NewPromptValidator(validator core.Validator, registry core.ValidationRegistry) PromptValidator {
	return &DefaultPromptValidator{
		validator: validator,
		registry:  registry,
	}
}

// ValidatePrompt implements PromptValidator.ValidatePrompt
func (v *DefaultPromptValidator) ValidatePrompt(prompt *types.ExtendedPrompt) (*core.Result, error) {
	// Get prompt-specific validation rules
	rules := append([]core.Rule{}, prompt.ValidationRules...)

	// Add built-in prompt validation rules
	rules = append(rules, v.getBuiltInPromptRules()...)

	return v.validator.ValidateValue(prompt, rules)
}

// ValidateParameters implements PromptValidator.ValidateParameters
func (v *DefaultPromptValidator) ValidateParameters(prompt *types.ExtendedPrompt, parameters map[string]interface{}) (*core.Result, error) {
	// Get parameter-specific validation rules
	rules := v.getParameterRules(prompt)

	return v.validator.ValidateValue(parameters, rules)
}

// AddRule implements PromptValidator.AddRule
func (v *DefaultPromptValidator) AddRule(rule core.Rule) error {
	return v.validator.AddRule(rule)
}

// RemoveRule implements PromptValidator.RemoveRule
func (v *DefaultPromptValidator) RemoveRule(ruleID string) error {
	return v.validator.RemoveRule(ruleID)
}

// getBuiltInPromptRules returns built-in validation rules for prompts
func (v *DefaultPromptValidator) getBuiltInPromptRules() []core.Rule {
	return []core.Rule{
		{
			ID:           "prompt.name.required",
			Name:         "Prompt Name Required",
			Description:  "Validates that the prompt has a name",
			Type:         core.Required,
			Config:       nil,
			ErrorMessage: "Prompt name is required",
			Severity:     core.ErrorSeverity,
		},
		{
			ID:           "prompt.version.valid",
			Name:         "Valid Version",
			Description:  "Validates that the prompt version is valid",
			Type:         core.Custom,
			Config:       nil,
			ErrorMessage: "Invalid prompt version",
			Severity:     core.ErrorSeverity,
		},
	}
}

// getParameterRules returns validation rules for prompt parameters
func (v *DefaultPromptValidator) getParameterRules(prompt *types.ExtendedPrompt) []core.Rule {
	var rules []core.Rule

	// Add schema validation if schema is defined
	if prompt.Schema != nil {
		rules = append(rules, core.Rule{
			ID:           "parameters.schema",
			Name:         "Parameter Schema",
			Description:  "Validates parameters against JSON schema",
			Type:         core.Schema,
			Config:       prompt.Schema,
			ErrorMessage: "Parameters do not match schema",
			Severity:     core.ErrorSeverity,
		})
	}

	// Add rules for required arguments
	for _, arg := range prompt.Arguments {
		if arg.Required {
			rules = append(rules, core.Rule{
				ID:           "parameters." + arg.Name + ".required",
				Name:         arg.Name + " Required",
				Description:  "Validates that required parameter is present",
				Type:         core.Required,
				Config:       nil,
				ErrorMessage: "Required parameter " + arg.Name + " is missing",
				Severity:     core.ErrorSeverity,
				Context: map[string]interface{}{
					"parameterName": arg.Name,
				},
			})
		}

		// Add schema validation for argument if schema is defined
		if arg.Schema != nil {
			rules = append(rules, core.Rule{
				ID:           "parameters." + arg.Name + ".schema",
				Name:         arg.Name + " Schema",
				Description:  "Validates parameter against JSON schema",
				Type:         core.Schema,
				Config:       arg.Schema,
				ErrorMessage: "Parameter " + arg.Name + " does not match schema",
				Severity:     core.ErrorSeverity,
				Context: map[string]interface{}{
					"parameterName": arg.Name,
				},
			})
		}
	}

	return rules
}
