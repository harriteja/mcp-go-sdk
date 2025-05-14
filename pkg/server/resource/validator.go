package resource

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/harriteja/mcp-go-sdk/pkg/validation/core"
)

// DefaultValidator implements the ResourceValidator interface
type DefaultValidator struct {
	mu    sync.RWMutex
	rules map[string]core.Rule
}

// NewValidator creates a new default validator
func NewValidator() types.ResourceValidator {
	return &DefaultValidator{
		rules: make(map[string]core.Rule),
	}
}

// ValidateResource validates a resource against a template
func (v *DefaultValidator) ValidateResource(resource interface{}, template *types.ExtendedResourceTemplate) (*core.Result, error) {
	result := &core.Result{
		Valid: true,
	}

	// Validate against JSON schema
	if err := v.validateSchema(resource, template.Schema); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, core.Error{
			RuleID:  "schema",
			Message: err.Error(),
			Path:    "$",
			Value:   resource,
		})
	}

	// Apply custom validation rules
	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, rule := range template.ValidationRules {
		if err := v.applyRule(resource, rule, result); err != nil {
			return nil, errors.Wrap(err, "failed to apply validation rule")
		}
	}

	return result, nil
}

// AddValidationRule adds a custom validation rule
func (v *DefaultValidator) AddValidationRule(rule core.Rule) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if rule.ID == "" {
		return errors.New("validation rule ID is required")
	}

	if _, exists := v.rules[rule.ID]; exists {
		return fmt.Errorf("validation rule with ID %s already exists", rule.ID)
	}

	v.rules[rule.ID] = rule
	return nil
}

// RemoveValidationRule removes a validation rule
func (v *DefaultValidator) RemoveValidationRule(ruleID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, exists := v.rules[ruleID]; !exists {
		return fmt.Errorf("validation rule with ID %s not found", ruleID)
	}

	delete(v.rules, ruleID)
	return nil
}

// validateSchema validates a resource against a JSON schema
func (v *DefaultValidator) validateSchema(resource interface{}, schema json.RawMessage) error {
	schemaLoader := gojsonschema.NewSchemaLoader()
	schemaLoader.Draft = gojsonschema.Draft7

	sl := gojsonschema.NewStringLoader(string(schema))
	sch, err := schemaLoader.Compile(sl)
	if err != nil {
		return errors.Wrap(err, "failed to compile schema")
	}

	dl := gojsonschema.NewGoLoader(resource)
	result, err := sch.Validate(dl)
	if err != nil {
		return errors.Wrap(err, "failed to validate against schema")
	}

	if !result.Valid() {
		var errMsg string
		for _, err := range result.Errors() {
			errMsg += fmt.Sprintf("%s; ", err.String())
		}
		return errors.New(errMsg)
	}

	return nil
}

// applyRule applies a validation rule to a resource
func (v *DefaultValidator) applyRule(resource interface{}, rule core.Rule, result *core.Result) error {
	var err error

	switch rule.Type {
	case core.Regex:
		err = v.applyRegexRule(resource, rule, result)
	case core.Format:
		err = v.applyFormatRule(resource, rule, result)
	case core.Range:
		err = v.applyRangeRule(resource, rule, result)
	case core.Enum:
		err = v.applyEnumRule(resource, rule, result)
	case core.Custom:
		err = v.applyCustomRule(resource, rule, result)
	default:
		return fmt.Errorf("unsupported validation rule type: %s", rule.Type)
	}

	if err != nil {
		return errors.Wrap(err, "failed to apply validation rule")
	}

	return nil
}

// applyRegexRule applies a regex validation rule
func (v *DefaultValidator) applyRegexRule(resource interface{}, rule core.Rule, result *core.Result) error {
	var config struct {
		Pattern string `json:"pattern"`
		Field   string `json:"field"`
	}

	if err := json.Unmarshal(rule.Config, &config); err != nil {
		return errors.Wrap(err, "failed to unmarshal regex rule config")
	}

	re, err := regexp.Compile(config.Pattern)
	if err != nil {
		return errors.Wrap(err, "failed to compile regex pattern")
	}

	// Extract field value using JSON path
	// This is a simplified example - in practice, you'd want to use a proper JSON path library
	value, ok := resource.(map[string]interface{})[config.Field]
	if !ok {
		return fmt.Errorf("field %s not found", config.Field)
	}

	strValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("field %s is not a string", config.Field)
	}

	if !re.MatchString(strValue) {
		verr := core.Error{
			RuleID:  rule.ID,
			Message: rule.ErrorMessage,
			Path:    config.Field,
			Value:   strValue,
		}

		switch rule.Severity {
		case core.ErrorSeverity:
			result.Valid = false
			result.Errors = append(result.Errors, verr)
		case core.WarningSeverity:
			result.Warnings = append(result.Warnings, verr)
		case core.InfoSeverity:
			result.Info = append(result.Info, verr)
		}
	}

	return nil
}

// applyFormatRule applies a format validation rule
func (v *DefaultValidator) applyFormatRule(resource interface{}, rule core.Rule, result *core.Result) error {
	// Implementation for format validation
	// This would validate things like email, date, URL formats
	return nil
}

// applyRangeRule applies a range validation rule
func (v *DefaultValidator) applyRangeRule(resource interface{}, rule core.Rule, result *core.Result) error {
	// Implementation for range validation
	// This would validate numeric ranges
	return nil
}

// applyEnumRule applies an enum validation rule
func (v *DefaultValidator) applyEnumRule(resource interface{}, rule core.Rule, result *core.Result) error {
	// Implementation for enum validation
	// This would validate against enumerated values
	return nil
}

// applyCustomRule applies a custom validation rule
func (v *DefaultValidator) applyCustomRule(resource interface{}, rule core.Rule, result *core.Result) error {
	// Implementation for custom validation
	// This would allow for custom validation logic
	return nil
}
