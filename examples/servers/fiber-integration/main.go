package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	mcpLogger "github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/server"
	transportfiber "github.com/harriteja/mcp-go-sdk/pkg/server/transport/fiber"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// SumRequest is a request to the sum tool
type SumRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

// SumResponse is a response from the sum tool
type SumResponse struct {
	Result int `json:"result"`
}

func main() {
	// Create a new Fiber app - this is your existing Fiber application
	app := fiber.New(fiber.Config{
		AppName:      "Fiber-MCP-Integration",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	})

	// Add Fiber middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Set up your existing Fiber routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to the Fiber + MCP integration example!")
	})

	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now(),
		})
	})

	// Create a new MCP server
	mcpServer, err := server.New(&server.Options{
		Name:    "fiber-mcp-example",
		Version: "1.0.0",
		Logger:  mcpLogger.GetDefaultLogger(),
	})
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Register MCP tools
	mcpServer.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		return []types.Tool{
			{
				Name:        "sum",
				Description: "Add two numbers",
				Parameters: &types.Parameters{
					Type: "object",
					Properties: map[string]types.Parameter{
						"a": {
							Type:        "number",
							Description: "First number",
						},
						"b": {
							Type:        "number",
							Description: "Second number",
						},
					},
				},
			},
		}, nil
	})

	mcpServer.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "sum":
			// Parse arguments
			a, ok := args["a"].(float64)
			if !ok {
				return nil, fmt.Errorf("invalid argument 'a'")
			}
			b, ok := args["b"].(float64)
			if !ok {
				return nil, fmt.Errorf("invalid argument 'b'")
			}

			// Perform calculation
			result := int(a) + int(b)

			// Return response
			return SumResponse{Result: result}, nil
		default:
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
	})

	// Create a Fiber adapter for the MCP server
	fiberAdapter := transportfiber.New(mcpServer, mcpLogger.GetDefaultLogger())

	// Register MCP routes with your existing Fiber app
	// This is the key part - adding MCP routes to your existing app
	fiberAdapter.RegisterRoutes(app)

	// Start the server
	log.Println("Starting server on :3000")
	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
