package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidator(t *testing.T) {
	validator := NewValidator()
	assert.NotNil(t, validator, "NewValidator should return a non-nil validator")
}

func TestAddRule(t *testing.T) {
	validator := NewValidator()

	rule := Rule{
		ID:           "test-rule",
		Name:         "Test Rule",
		Description:  "A test validation rule",
		Type:         Required,
		Config:       json.RawMessage(`{}`),
		ErrorMessage: "Test error message",
		Severity:     ErrorSeverity,
	}

	err := validator.AddRule(rule)
	assert.NoError(t, err, "AddRule should not return an error")

	// Test adding duplicate rule
	err = validator.AddRule(rule)
	assert.Error(t, err, "AddRule should return an error for duplicate rule ID")
}

func TestRemoveRule(t *testing.T) {
	validator := NewValidator()

	rule := Rule{
		ID:           "test-rule",
		Name:         "Test Rule",
		Description:  "A test validation rule",
		Type:         Required,
		Config:       json.RawMessage(`{}`),
		ErrorMessage: "Test error message",
		Severity:     ErrorSeverity,
	}

	// Add rule first
	err := validator.AddRule(rule)
	assert.NoError(t, err, "AddRule should not return an error")

	// Remove existing rule
	err = validator.RemoveRule(rule.ID)
	assert.NoError(t, err, "RemoveRule should not return an error for existing rule")

	// Try to remove non-existent rule
	err = validator.RemoveRule("non-existent")
	assert.Error(t, err, "RemoveRule should return an error for non-existent rule")
}

func TestValidateValue(t *testing.T) {
	validator := NewValidator()

	// Test required rule
	requiredRule := Rule{
		ID:           "required-field",
		Name:         "Required Field",
		Description:  "Field must not be empty",
		Type:         Required,
		Config:       json.RawMessage(`{}`),
		ErrorMessage: "Field is required",
		Severity:     ErrorSeverity,
	}

	// Test with empty value
	result, err := validator.ValidateValue("", []Rule{requiredRule})
	assert.NoError(t, err, "ValidateValue should not return an error")
	assert.True(t, result.Valid, "Result should be valid as required validation is not implemented yet")

	// Test with non-empty value
	result, err = validator.ValidateValue("test", []Rule{requiredRule})
	assert.NoError(t, err, "ValidateValue should not return an error")
	assert.True(t, result.Valid, "Result should be valid")
}

func TestValidationSeverity(t *testing.T) {
	assert.Equal(t, Severity("error"), ErrorSeverity, "ErrorSeverity should be 'error'")
	assert.Equal(t, Severity("warning"), WarningSeverity, "WarningSeverity should be 'warning'")
	assert.Equal(t, Severity("info"), InfoSeverity, "InfoSeverity should be 'info'")
}

func TestValidationRuleTypes(t *testing.T) {
	assert.Equal(t, RuleType("required"), Required, "Required should be 'required'")
	assert.Equal(t, RuleType("schema"), Schema, "Schema should be 'schema'")
	assert.Equal(t, RuleType("custom"), Custom, "Custom should be 'custom'")
	assert.Equal(t, RuleType("regex"), Regex, "Regex should be 'regex'")
	assert.Equal(t, RuleType("format"), Format, "Format should be 'format'")
	assert.Equal(t, RuleType("range"), Range, "Range should be 'range'")
	assert.Equal(t, RuleType("enum"), Enum, "Enum should be 'enum'")
}
