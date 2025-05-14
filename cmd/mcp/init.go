package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// Project templates
const (
	mainTemplate = `package main

import (
	"context"
	"log"

	"github.com/harriteja/mcp-go-sdk/pkg/server"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func main() {
	// Create server
	srv := server.New(server.Options{
		Name:    "{{.Name}}",
		Version: "1.0.0",
	})

	// Register handlers
	srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		return []types.Tool{
			{
				Name:        "example",
				Description: "An example tool",
			},
		}, nil
	})

	// Start server
	if err := srv.ListenAndServe(":8080"); err != nil {
		log.Fatal(err)
	}
}`

	goModTemplate = `module {{.Name}}

go 1.22

require github.com/harriteja/mcp-go-sdk v1.0.0
`
)

type projectData struct {
	Name string
}

func initProject(name string) error {
	// Create project directory
	if err := os.MkdirAll(name, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create main.go
	if err := createFileFromTemplate(
		filepath.Join(name, "main.go"),
		mainTemplate,
		projectData{Name: name},
	); err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}

	// Create go.mod
	if err := createFileFromTemplate(
		filepath.Join(name, "go.mod"),
		goModTemplate,
		projectData{Name: name},
	); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	fmt.Printf("Successfully created project %s\n", name)
	fmt.Println("To get started:")
	fmt.Printf("  cd %s\n", name)
	fmt.Println("  go mod tidy")
	fmt.Println("  go run main.go")

	return nil
}

func createFileFromTemplate(path, tmpl string, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	t := template.Must(template.New("").Parse(tmpl))
	return t.Execute(f, data)
}
