package prompts

import "testing"

func TestDefaultRenderer(t *testing.T) {
	renderer := NewDefaultRenderer()

	tests := []struct {
		name       string
		template   string
		params     map[string]interface{}
		want       string
		wantErr    bool
		errMessage string
	}{
		{
			name:     "Simple template",
			template: "Hello {{.name}}!",
			params: map[string]interface{}{
				"name": "World",
			},
			want:    "Hello World!",
			wantErr: false,
		},
		{
			name:     "Multiple parameters",
			template: "{{.greeting}} {{.name}}! How are you {{.time}}?",
			params: map[string]interface{}{
				"greeting": "Hi",
				"name":     "Alice",
				"time":     "today",
			},
			want:    "Hi Alice! How are you today?",
			wantErr: false,
		},
		{
			name:     "Empty template",
			template: "",
			params: map[string]interface{}{
				"name": "World",
			},
			wantErr:    true,
			errMessage: "empty template",
		},
		{
			name:     "Invalid template syntax",
			template: "Hello {{.name!",
			params: map[string]interface{}{
				"name": "World",
			},
			wantErr: true,
		},
		{
			name:     "Missing parameter",
			template: "Hello {{.missing}}!",
			params: map[string]interface{}{
				"name": "World",
			},
			wantErr: true,
		},
		{
			name:     "Nested parameters",
			template: "{{.user.name}} is {{.user.age}} years old",
			params: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Bob",
					"age":  30,
				},
			},
			want:    "Bob is 30 years old",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderer.Render(tt.template, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.errMessage != "" && err.Error() != tt.errMessage {
					t.Errorf("Expected error message %q, got %q", tt.errMessage, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}
