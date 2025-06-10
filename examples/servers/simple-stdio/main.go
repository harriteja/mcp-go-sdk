package main

import (
	"context"
	"encoding/json"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/server"
	"github.com/harriteja/mcp-go-sdk/pkg/server/transport/stdio"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func main() {
	// Create a default logger
	log := logger.New("simple-stdio")

	// Create MCP server
	srv, err := server.New(&server.Options{
		Name:    "simple-stdio-server",
		Version: "1.0.0",
		Logger:  log,
	})
	if err != nil {
		log.Panic(context.Background(), "main", "init", "Failed to create server: "+err.Error())
	}

	// Register tool handlers
	srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		schema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {
					"type": "string"
				}
			},
			"required": ["message"]
		}`)

		return []types.Tool{
			{
				Name:        "echo",
				Description: "Echo back the input message",
				InputSchema: schema,
			},
		}, nil
	})

	srv.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
		if name != "echo" {
			return nil, &types.Error{
				Code:    404,
				Message: "Tool not found",
			}
		}

		message, ok := args["message"].(string)
		if !ok {
			return nil, &types.Error{
				Code:    400,
				Message: "Invalid message parameter",
			}
		}

		return map[string]interface{}{
			"message": message,
			"echo":    true,
		}, nil
	})

	// Create stdio transport
	transport := stdio.New(srv, stdio.Options{
		Logger: log,
	})

	// Start the server
	ctx := context.Background()
	log.Info(ctx, "main", "start", "Starting stdio server")
	if err := transport.Start(); err != nil {
		log.Error(ctx, "main", "serve", "Failed to serve: "+err.Error())
	}
}
