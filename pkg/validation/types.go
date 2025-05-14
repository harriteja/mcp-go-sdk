package validation

import (
	"encoding/json"
)

// Severity represents the severity of a validation error
type Severity string

const (
	// ErrorSeverity indicates a validation error
	ErrorSeverity Severity = "error"

	// WarningSeverity indicates a validation warning
	WarningSeverity Severity = "warning"

	// InfoSeverity indicates an informational validation message
	InfoSeverity Severity = "info"
)

// Error represents a validation error
type Error struct {
	// RuleID is the ID of the rule that was violated
	RuleID string `json:"ruleId"`

	// Message describes the validation error
	Message string `json:"message"`

	// Path is the JSON path to the invalid field
	Path string `json:"path"`

	// Value is the invalid value
	Value interface{} `json:"value"`
}

// Result represents the result of validation
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

// RuleType represents the type of validation rule
type RuleType string

const (
	// Required checks if a field is required
	Required RuleType = "required"

	// Schema validates against a JSON schema
	Schema RuleType = "schema"

	// Custom allows custom validation logic
	Custom RuleType = "custom"

	// Regex validates using regular expressions
	Regex RuleType = "regex"

	// Format validates data format
	Format RuleType = "format"

	// Range validates numeric ranges
	Range RuleType = "range"

	// Enum validates against enumerated values
	Enum RuleType = "enum"
)

// Rule represents a rule for validation
type Rule struct {
	// ID uniquely identifies this rule
	ID string `json:"id"`

	// Name is a human-readable name for this rule
	Name string `json:"name"`

	// Description describes what this rule validates
	Description string `json:"description"`

	// Type specifies the type of validation rule
	Type RuleType `json:"type"`

	// Config contains rule-specific configuration
	Config json.RawMessage `json:"config"`

	// ErrorMessage is the message to show when validation fails
	ErrorMessage string `json:"errorMessage"`

	// Severity indicates how severe a violation of this rule is
	Severity Severity `json:"severity"`
}

// Validator defines the interface for validation
type Validator interface {
	// ValidateValue validates a value against validation rules
	ValidateValue(value interface{}, rules []Rule) (*Result, error)

	// AddRule adds a custom validation rule
	AddRule(rule Rule) error

	// RemoveRule removes a validation rule
	RemoveRule(ruleID string) error
}

// PromptRule represents a rule for validating prompt parameters
type PromptRule struct {
	Rule
	// ParameterName is the name of the parameter this rule applies to
	ParameterName string `json:"parameterName"`
}

// PromptError represents a validation error specific to prompts
type PromptError struct {
	Error
	// ParameterName is the name of the invalid parameter
	ParameterName string `json:"parameterName"`
}

// PromptResult represents the result of validating a prompt
type PromptResult struct {
	// Valid indicates if the validation passed
	Valid bool `json:"valid"`

	// Errors contains validation errors
	Errors []PromptError `json:"errors,omitempty"`

	// Warnings contains validation warnings
	Warnings []PromptError `json:"warnings,omitempty"`

	// Info contains informational validation messages
	Info []PromptError `json:"info,omitempty"`
}

// PromptValidator defines the interface for validating prompts
type PromptValidator interface {
	// ValidatePrompt validates a prompt definition
	ValidatePrompt(prompt interface{}) (*PromptResult, error)

	// ValidateParameters validates prompt parameters
	ValidateParameters(prompt interface{}, parameters map[string]interface{}) (*PromptResult, error)

	// AddRule adds a custom validation rule
	AddRule(rule PromptRule) error

	// RemoveRule removes a validation rule
	RemoveRule(ruleID string) error
}
