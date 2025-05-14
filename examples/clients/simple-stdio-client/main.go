package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/harriteja/mcp-go-sdk/pkg/client"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func main() {
	// Start server process
	serverCmd := exec.Command("go", "run", "../../servers/simple-stdio/main.go")
	serverIn, err := serverCmd.StdinPipe()
	if err != nil {
		log.Fatal("Failed to get server stdin:", err)
	}
	serverOut, err := serverCmd.StdoutPipe()
	if err != nil {
		log.Fatal("Failed to get server stdout:", err)
	}
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
	defer func() {
		if err := serverCmd.Process.Kill(); err != nil {
			log.Printf("Error killing server process: %v", err)
		}
	}()

	// Create client
	cli := client.New(client.Options{
		Reader: serverOut,
		Writer: serverIn,
		ClientInfo: types.Implementation{
			Name:    "simple-stdio-client",
			Version: "1.0.0",
		},
	})

	// Initialize client
	if err := cli.Initialize(context.Background()); err != nil {
		log.Fatal("Failed to initialize client:", err)
	}

	// List available tools
	tools, err := cli.ListTools(context.Background())
	if err != nil {
		log.Fatal("Failed to list tools:", err)
	}

	fmt.Println("Available tools:")
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

	// Test echo tool
	args := map[string]interface{}{
		"message": "Hello, World!",
		"number":  42,
		"nested": map[string]interface{}{
			"key": "value",
		},
	}

	result, err := cli.CallTool(context.Background(), "echo", args)
	if err != nil {
		log.Fatal("Failed to call echo tool:", err)
	}

	fmt.Println("\nEcho result:")
	fmt.Printf("%+v\n", result)
}
