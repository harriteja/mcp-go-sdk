package prompts

import (
	"bytes"
	"errors"
	"text/template"
)

// TemplateRenderer defines the interface for rendering prompt templates
type TemplateRenderer interface {
	// Render processes a template string with the given parameters
	Render(templateStr string, params map[string]interface{}) (string, error)
}

// defaultRenderer implements TemplateRenderer using Go's text/template
type defaultRenderer struct{}

// NewDefaultRenderer creates a new default template renderer
func NewDefaultRenderer() TemplateRenderer {
	return &defaultRenderer{}
}

func (r *defaultRenderer) Render(templateStr string, params map[string]interface{}) (string, error) {
	if templateStr == "" {
		return "", errors.New("empty template")
	}

	tmpl, err := template.New("prompt").Option("missingkey=error").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", err
	}

	return buf.String(), nil
}
