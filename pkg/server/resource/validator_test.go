package resource

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	resourceTypes "github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/harriteja/mcp-go-sdk/pkg/validation/core"
)

func TestDefaultValidator(t *testing.T) {
	validator := NewValidator()

	t.Run("Validate against JSON schema", func(t *testing.T) {
		schema := []byte(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "number"}
			},
			"required": ["name", "age"]
		}`)

		template := &resourceTypes.ExtendedResourceTemplate{
			ResourceTemplate: resourceTypes.ResourceTemplate{
				URITemplate: "test/{name}",
				Name:        "Test Template",
			},
			Version: resourceTypes.ResourceTemplateVersion{
				Major: 1,
				Minor: 0,
				Patch: 0,
			},
			Schema:          schema,
			ValidationRules: []core.Rule{},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		// Valid resource
		validResource := map[string]interface{}{
			"name": "John",
			"age":  30,
		}
		result, err := validator.ValidateResource(validResource, template)
		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)

		// Invalid resource
		invalidResource := map[string]interface{}{
			"name": "John",
			// Missing age field
		}
		result, err = validator.ValidateResource(invalidResource, template)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("Validate with regex rule", func(t *testing.T) {
		config := map[string]string{
			"pattern": "^[A-Z][a-z]+$",
			"field":   "name",
		}
		configBytes, err := json.Marshal(config)
		assert.NoError(t, err)

		template := &resourceTypes.ExtendedResourceTemplate{
			ResourceTemplate: resourceTypes.ResourceTemplate{
				URITemplate: "test/{name}",
				Name:        "Test Template",
			},
			Version: resourceTypes.ResourceTemplateVersion{
				Major: 1,
				Minor: 0,
				Patch: 0,
			},
			Schema: []byte(`{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}`),
			ValidationRules: []core.Rule{
				{
					ID:           "name-format",
					Name:         "Name Format",
					Description:  "Name must start with uppercase and contain only letters",
					Type:         core.Regex,
					Config:       configBytes,
					ErrorMessage: "Invalid name format",
					Severity:     core.ErrorSeverity,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Valid resource
		validResource := map[string]interface{}{
			"name": "John",
		}
		result, err := validator.ValidateResource(validResource, template)
		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)

		// Invalid resource
		invalidResource := map[string]interface{}{
			"name": "john", // Lowercase first letter
		}
		result, err = validator.ValidateResource(invalidResource, template)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
		assert.Equal(t, "name-format", result.Errors[0].RuleID)
	})

	t.Run("Add and remove validation rules", func(t *testing.T) {
		rule := core.Rule{
			ID:           "test-rule",
			Name:         "Test Rule",
			Description:  "Test validation rule",
			Type:         core.Custom,
			ErrorMessage: "Test error",
			Severity:     core.WarningSeverity,
		}

		// Add rule
		err := validator.AddValidationRule(rule)
		assert.NoError(t, err)

		// Try to add duplicate rule
		err = validator.AddValidationRule(rule)
		assert.Error(t, err)

		// Remove rule
		err = validator.RemoveValidationRule(rule.ID)
		assert.NoError(t, err)

		// Try to remove non-existent rule
		err = validator.RemoveValidationRule("non-existent")
		assert.Error(t, err)
	})
}

func TestValidationSeverity(t *testing.T) {
	validator := NewValidator()

	config := map[string]string{
		"pattern": "^[A-Z][a-z]+$",
		"field":   "name",
	}
	configBytes, err := json.Marshal(config)
	assert.NoError(t, err)

	template := &resourceTypes.ExtendedResourceTemplate{
		ResourceTemplate: resourceTypes.ResourceTemplate{
			URITemplate: "test/{name}",
			Name:        "Test Template",
		},
		Schema: []byte(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"}
			}
		}`),
		ValidationRules: []core.Rule{
			{
				ID:           "error-rule",
				Type:         core.Regex,
				Config:       configBytes,
				ErrorMessage: "Error severity",
				Severity:     core.ErrorSeverity,
			},
			{
				ID:           "warning-rule",
				Type:         core.Regex,
				Config:       configBytes,
				ErrorMessage: "Warning severity",
				Severity:     core.WarningSeverity,
			},
			{
				ID:           "info-rule",
				Type:         core.Regex,
				Config:       configBytes,
				ErrorMessage: "Info severity",
				Severity:     core.InfoSeverity,
			},
		},
	}

	resource := map[string]interface{}{
		"name": "invalid",
	}

	result, err := validator.ValidateResource(resource, template)
	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.NotEmpty(t, result.Warnings)
	assert.NotEmpty(t, result.Info)
}
