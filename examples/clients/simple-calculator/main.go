package main

import (
	"context"
	"fmt"
	"log"

	"github.com/harriteja/mcp-go-sdk/pkg/client"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func main() {
	// Create client
	cli := client.New(client.Options{
		ServerURL: "http://localhost:8080/mcp",
		ClientInfo: types.Implementation{
			Name:    "simple-calculator-client",
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

	// Example calculations
	calculations := []struct {
		operation string
		a, b      float64
	}{
		{"add", 5, 3},
		{"subtract", 10, 4},
		{"multiply", 6, 7},
		{"divide", 15, 3},
	}

	for _, calc := range calculations {
		args := map[string]interface{}{
			"operation": calc.operation,
			"a":         calc.a,
			"b":         calc.b,
		}

		result, err := cli.CallTool(context.Background(), "calculator", args)
		if err != nil {
			log.Printf("Failed to calculate %s: %v", calc.operation, err)
			continue
		}

		// Parse result
		resultMap := result.(map[string]interface{})
		fmt.Printf("%g %s %g = %g\n", calc.a, calc.operation, calc.b, resultMap["result"])
	}
}
