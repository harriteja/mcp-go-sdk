package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/harriteja/mcp-go-sdk/pkg/server"
	ginmcp "github.com/harriteja/mcp-go-sdk/pkg/server/transport/gin"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func main() {
	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create MCP server
	srv := server.New(server.Options{
		Name:         "simple-tool-server",
		Version:      "1.0.0",
		Instructions: "A simple MCP server that provides a calculator tool",
	})

	// Register tool handlers
	srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		schema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"operation": {
					"type": "string",
					"enum": ["add", "subtract", "multiply", "divide"]
				},
				"a": {
					"type": "number"
				},
				"b": {
					"type": "number"
				}
			},
			"required": ["operation", "a", "b"]
		}`)

		return []types.Tool{
			{
				Name:        "calculator",
				Description: "A simple calculator tool",
				InputSchema: schema,
			},
		}, nil
	})

	srv.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
		if name != "calculator" {
			return nil, &types.Error{
				Code:    404,
				Message: "Tool not found",
			}
		}

		operation := args["operation"].(string)
		a := args["a"].(float64)
		b := args["b"].(float64)

		var result float64
		switch operation {
		case "add":
			result = a + b
		case "subtract":
			result = a - b
		case "multiply":
			result = a * b
		case "divide":
			if b == 0 {
				return nil, &types.Error{
					Code:    400,
					Message: "Division by zero",
				}
			}
			result = a / b
		default:
			return nil, &types.Error{
				Code:    400,
				Message: "Invalid operation",
			}
		}

		return map[string]interface{}{
			"result": result,
		}, nil
	})

	// Create Gin engine
	r := gin.Default()

	// Create Gin adapter and register routes
	adapter := ginmcp.New(srv, logger)
	adapter.RegisterRoutes(r)

	// Start server
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
