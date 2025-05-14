package prompts

import (
	"context"
	"testing"
)

func TestMemoryManager(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager()

	t.Run("Create and Get Prompt", func(t *testing.T) {
		template := &PromptTemplate{
			ID:       "test1",
			Template: "Hello {{.name}}!",
			Parameters: map[string]interface{}{
				"name": "string",
			},
			Description: "Test template",
		}

		if err := manager.CreatePrompt(ctx, template); err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		got, err := manager.GetPrompt(ctx, "test1")
		if err != nil {
			t.Fatalf("Failed to get prompt: %v", err)
		}

		if got.ID != template.ID || got.Template != template.Template {
			t.Errorf("Got different prompt than created: got %v, want %v", got, template)
		}
	})

	t.Run("Update Prompt", func(t *testing.T) {
		template := &PromptTemplate{
			ID:          "test2",
			Template:    "Original",
			Description: "Original description",
		}

		if err := manager.CreatePrompt(ctx, template); err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		template.Template = "Updated"
		if err := manager.UpdatePrompt(ctx, template); err != nil {
			t.Fatalf("Failed to update prompt: %v", err)
		}

		got, err := manager.GetPrompt(ctx, "test2")
		if err != nil {
			t.Fatalf("Failed to get prompt: %v", err)
		}

		if got.Template != "Updated" {
			t.Errorf("Update failed: got %v, want Updated", got.Template)
		}
	})

	t.Run("Delete Prompt", func(t *testing.T) {
		template := &PromptTemplate{
			ID:       "test3",
			Template: "To be deleted",
		}

		if err := manager.CreatePrompt(ctx, template); err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		if err := manager.DeletePrompt(ctx, "test3"); err != nil {
			t.Fatalf("Failed to delete prompt: %v", err)
		}

		if _, err := manager.GetPrompt(ctx, "test3"); err != ErrPromptNotFound {
			t.Errorf("Expected ErrPromptNotFound, got %v", err)
		}
	})

	t.Run("List Prompts", func(t *testing.T) {
		// Clear existing prompts
		templates := []*PromptTemplate{
			{ID: "list1", Template: "Template 1"},
			{ID: "list2", Template: "Template 2"},
		}

		for _, tmpl := range templates {
			if err := manager.CreatePrompt(ctx, tmpl); err != nil {
				t.Fatalf("Failed to create prompt: %v", err)
			}
		}

		got, err := manager.ListPrompts(ctx)
		if err != nil {
			t.Fatalf("Failed to list prompts: %v", err)
		}

		// We expect 4 prompts total:
		// - test1 from "Create and Get Prompt"
		// - test2 from "Update Prompt"
		// - list1 and list2 from this test
		// Note: test3 was deleted in "Delete Prompt"
		if len(got) != 4 {
			t.Errorf("Expected 4 prompts, got %d", len(got))
		}
	})

	t.Run("Render Prompt", func(t *testing.T) {
		template := &PromptTemplate{
			ID:       "render1",
			Template: "Hello {{.name}}!",
		}

		if err := manager.CreatePrompt(ctx, template); err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		params := map[string]interface{}{
			"name": "World",
		}

		rendered, err := manager.RenderPrompt(ctx, "render1", params)
		if err != nil {
			t.Fatalf("Failed to render prompt: %v", err)
		}

		if rendered != "Hello World!" {
			t.Errorf("Wrong render output: got %q, want %q", rendered, "Hello World!")
		}
	})
}
