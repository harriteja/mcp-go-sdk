package validation

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

// DefaultValidator provides a default implementation of the Validator interface
type DefaultValidator struct {
	mu    sync.RWMutex
	rules map[string]Rule
}

// NewValidator creates a new DefaultValidator
func NewValidator() Validator {
	return &DefaultValidator{
		rules: make(map[string]Rule),
	}
}

// ValidateValue validates a value against validation rules
func (v *DefaultValidator) ValidateValue(value interface{}, rules []Rule) (*Result, error) {
	result := &Result{
		Valid: true,
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, rule := range rules {
		if err := v.applyRule(value, rule, result); err != nil {
			return nil, errors.Wrap(err, "failed to apply validation rule")
		}
	}

	return result, nil
}

// AddRule adds a custom validation rule
func (v *DefaultValidator) AddRule(rule Rule) error {
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

// RemoveRule removes a validation rule
func (v *DefaultValidator) RemoveRule(ruleID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, exists := v.rules[ruleID]; !exists {
		return fmt.Errorf("validation rule with ID %s not found", ruleID)
	}

	delete(v.rules, ruleID)
	return nil
}

// applyRule applies a validation rule to a value
func (v *DefaultValidator) applyRule(value interface{}, rule Rule, result *Result) error {
	var err error

	switch rule.Type {
	case Regex:
		err = v.applyRegexRule(value, rule, result)
	case Format:
		err = v.applyFormatRule(value, rule, result)
	case Range:
		err = v.applyRangeRule(value, rule, result)
	case Enum:
		err = v.applyEnumRule(value, rule, result)
	case Custom:
		err = v.applyCustomRule(value, rule, result)
	case Required:
		err = v.applyRequiredRule(value, rule, result)
	case Schema:
		err = v.applySchemaRule(value, rule, result)
	default:
		return fmt.Errorf("unsupported validation rule type: %s", rule.Type)
	}

	if err != nil {
		return errors.Wrap(err, "failed to apply validation rule")
	}

	return nil
}

// applyRegexRule applies a regex validation rule
func (v *DefaultValidator) applyRegexRule(value interface{}, rule Rule, result *Result) error {
	// TODO: Implement regex validation
	return nil
}

// applyFormatRule applies a format validation rule
func (v *DefaultValidator) applyFormatRule(value interface{}, rule Rule, result *Result) error {
	// TODO: Implement format validation
	return nil
}

// applyRangeRule applies a range validation rule
func (v *DefaultValidator) applyRangeRule(value interface{}, rule Rule, result *Result) error {
	// TODO: Implement range validation
	return nil
}

// applyEnumRule applies an enum validation rule
func (v *DefaultValidator) applyEnumRule(value interface{}, rule Rule, result *Result) error {
	// TODO: Implement enum validation
	return nil
}

// applyCustomRule applies a custom validation rule
func (v *DefaultValidator) applyCustomRule(value interface{}, rule Rule, result *Result) error {
	// TODO: Implement custom validation
	return nil
}

// applyRequiredRule applies a required validation rule
func (v *DefaultValidator) applyRequiredRule(value interface{}, rule Rule, result *Result) error {
	// TODO: Implement required validation
	return nil
}

// applySchemaRule applies a schema validation rule
func (v *DefaultValidator) applySchemaRule(value interface{}, rule Rule, result *Result) error {
	// TODO: Implement schema validation
	return nil
}
